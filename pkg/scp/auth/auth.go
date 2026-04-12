// Copyright 2026 Laurynas Četyrkinas <laurynas@digilol.net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/digilolnet/go-netcup-scp/internal/version"
)

const (
	authURL  = "https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect"
	clientID = "scp"
)

var (
	ErrDeviceAuthorizationFailed = errors.New("device authorization failed")
	ErrTokenRefreshFailed        = errors.New("token refresh failed")
	ErrTokenRevokeFailed         = errors.New("token revoke failed")
)

type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

type Manager struct {
	httpClient   *http.Client
	token        *TokenResponse
	tokenExpiry  time.Time
	mu           sync.RWMutex
	onRefresh    func(*TokenResponse)
	autoRefresh  bool
	refreshTimer *time.Timer
}

type Option func(*Manager)

func WithHTTPClient(client *http.Client) Option {
	return func(am *Manager) {
		am.httpClient = client
	}
}

func WithTokenRefreshCallback(callback func(*TokenResponse)) Option {
	return func(am *Manager) {
		am.onRefresh = callback
	}
}

func WithAutoRefresh(enable bool) Option {
	return func(am *Manager) {
		am.autoRefresh = enable
	}
}

func NewManager(opts ...Option) *Manager {
	am := &Manager{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		autoRefresh: true,
	}

	for _, opt := range opts {
		opt(am)
	}

	return am
}

func (am *Manager) InitiateDeviceAuth(ctx context.Context) (*DeviceAuthResponse, error) {
	data := url.Values{
		"client_id": {clientID},
		"scope":     {"offline_access openid"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL+"/auth/device", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrDeviceAuthorizationFailed, resp.StatusCode, string(body))
	}

	var authResp DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &authResp, nil
}

func (am *Manager) PollForToken(ctx context.Context, deviceCode string, interval time.Duration) (*TokenResponse, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			token, err := am.requestToken(ctx, deviceCode)
			if err != nil {
				var authErr *authError
				if errors.As(err, &authErr) {
					switch authErr.ErrorCode {
					case "authorization_pending":
						continue
					case "slow_down":
						ticker.Reset(interval * 2)
						continue
					default:
						return nil, err
					}
				}
				return nil, err
			}

			am.setToken(token)
			if am.autoRefresh {
				am.scheduleRefresh()
			}
			return token, nil
		}
	}
}

func (am *Manager) requestToken(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceCode},
		"client_id":   {clientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL+"/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp authError
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return nil, &errResp
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed: status %d: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &token, nil
}

func (am *Manager) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL+"/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrTokenRefreshFailed, resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	am.setToken(&token)
	if am.onRefresh != nil {
		am.onRefresh(&token)
	}
	if am.autoRefresh {
		am.scheduleRefresh()
	}

	return &token, nil
}

func (am *Manager) RevokeToken(ctx context.Context, refreshToken string) error {
	data := url.Values{
		"client_id":       {clientID},
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL+"/revoke", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", ErrTokenRevokeFailed, resp.StatusCode, string(body))
	}

	am.mu.Lock()
	am.token = nil
	am.tokenExpiry = time.Time{}
	if am.refreshTimer != nil {
		am.refreshTimer.Stop()
		am.refreshTimer = nil
	}
	am.mu.Unlock()

	return nil
}

func (am *Manager) LoadToken(token *TokenResponse) {
	am.setToken(token)
	if am.autoRefresh {
		am.scheduleRefresh()
	}
}

func (am *Manager) GetAccessToken() (string, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.token == nil {
		return "", errors.New("no token available")
	}

	if time.Now().After(am.tokenExpiry) {
		return "", errors.New("token expired")
	}

	return am.token.AccessToken, nil
}

func (am *Manager) GetRefreshToken() (string, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.token == nil {
		return "", errors.New("no token available")
	}

	return am.token.RefreshToken, nil
}

func (am *Manager) setToken(token *TokenResponse) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.token = token
	am.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
}

func (am *Manager) scheduleRefresh() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.refreshTimer != nil {
		am.refreshTimer.Stop()
	}

	if am.token == nil {
		return
	}

	refreshIn := time.Until(am.tokenExpiry) - 30*time.Second
	if refreshIn < 0 {
		refreshIn = 0
	}

	am.refreshTimer = time.AfterFunc(refreshIn, func() {
		am.mu.RLock()
		refreshToken := am.token.RefreshToken
		am.mu.RUnlock()

		_, _ = am.RefreshToken(context.Background(), refreshToken)
	})
}

func (am *Manager) Close() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.refreshTimer != nil {
		am.refreshTimer.Stop()
		am.refreshTimer = nil
	}
}

type authError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (e *authError) Error() string {
	if e.ErrorDescription != "" {
		return fmt.Sprintf("%s: %s", e.ErrorCode, e.ErrorDescription)
	}
	return e.ErrorCode
}
