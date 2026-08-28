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
	// DefaultAuthURL is the production netcup SCP OpenID Connect endpoint base.
	DefaultAuthURL = "https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect"
	clientID       = "scp"
)

// slowDownIncrement is how much each slow_down response grows the polling
// interval (RFC 8628 §3.5 mandates 5 seconds); a variable so tests can shrink it.
var slowDownIncrement = 5 * time.Second

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
	// ObtainedAt is set by this library (not the IdP) when the token is
	// received, so that a token reloaded from disk expires at the right
	// absolute time instead of ExpiresIn seconds after loading.
	ObtainedAt time.Time `json:"obtained_at,omitempty"`
}

type Manager struct {
	httpClient   *http.Client
	authURL      string
	token        *TokenResponse
	tokenExpiry  time.Time
	mu           sync.RWMutex
	onRefresh    func(*TokenResponse)
	autoRefresh  bool
	refreshTimer *time.Timer
	closed       bool
	// refreshMu single-flights token refreshes: concurrent callers needing a
	// fresh token wait for the in-flight refresh and reuse its result instead
	// of racing the IdP (fatal under refresh-token rotation). It also keeps
	// each setToken+onRefresh pair atomic so the persisted token can never be
	// older than the in-memory one.
	refreshMu sync.Mutex
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

// WithAuthURL overrides the OpenID Connect endpoint base (default
// DefaultAuthURL) — for a staging realm, a mock, or a local proxy.
func WithAuthURL(url string) Option {
	return func(am *Manager) {
		am.authURL = strings.TrimSuffix(url, "/")
	}
}

func NewManager(opts ...Option) *Manager {
	am := &Manager{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		authURL:     DefaultAuthURL,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, am.authURL+"/auth/device", strings.NewReader(data.Encode()))
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
						interval += slowDownIncrement
						ticker.Reset(interval)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, am.authURL+"/token", strings.NewReader(data.Encode()))
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

// RefreshToken exchanges refreshToken for a fresh token pair. Concurrent
// calls are serialized; prefer ValidAccessToken, which also skips the network
// round-trip when the cached token is still valid.
func (am *Manager) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	am.refreshMu.Lock()
	defer am.refreshMu.Unlock()
	return am.refresh(ctx, refreshToken)
}

// refresh performs the token refresh. Callers must hold refreshMu.
func (am *Manager) refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	am.mu.RLock()
	closed := am.closed
	am.mu.RUnlock()
	if closed {
		return nil, errors.New("auth manager is closed")
	}

	data := url.Values{
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, am.authURL+"/token", strings.NewReader(data.Encode()))
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

// ValidAccessToken returns a currently-valid access token, refreshing once
// (single-flight across goroutines) when the cached token has expired.
func (am *Manager) ValidAccessToken(ctx context.Context) (string, error) {
	if tok, err := am.GetAccessToken(); err == nil {
		return tok, nil
	}
	am.refreshMu.Lock()
	defer am.refreshMu.Unlock()
	// Another caller may have refreshed while we waited for the lock.
	if tok, err := am.GetAccessToken(); err == nil {
		return tok, nil
	}
	refreshToken, err := am.GetRefreshToken()
	if err != nil {
		return "", fmt.Errorf("no valid access token: %w", err)
	}
	tok, err := am.refresh(ctx, refreshToken)
	if err != nil {
		return "", fmt.Errorf("refresh access token: %w", err)
	}
	return tok.AccessToken, nil
}

func (am *Manager) RevokeToken(ctx context.Context, refreshToken string) error {
	data := url.Values{
		"client_id":       {clientID},
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, am.authURL+"/revoke", strings.NewReader(data.Encode()))
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
	if token.ObtainedAt.IsZero() {
		// Token files written before ObtainedAt existed carry no issue
		// time; backdate so the token counts as already expired and the
		// first use refreshes instead of sending a possibly-expired token.
		// Backdating (rather than fixing up expiry afterwards) keeps the
		// whole load in one lock section, so no reader can observe a
		// spuriously-fresh expiry in between.
		token.ObtainedAt = time.Now().Add(-time.Duration(token.ExpiresIn)*time.Second - time.Second)
	}
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

	if token.ObtainedAt.IsZero() {
		token.ObtainedAt = time.Now()
	}
	// Store a copy: callers keep using their pointer after LoadToken/refresh,
	// and aliasing it would let them mutate manager state without the lock.
	t := *token
	am.token = &t
	am.tokenExpiry = token.ObtainedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
}

func (am *Manager) scheduleRefresh() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.refreshTimer != nil {
		am.refreshTimer.Stop()
	}

	if am.token == nil || am.closed {
		return
	}

	refreshIn := time.Until(am.tokenExpiry) - 30*time.Second
	// Floor the interval: an already-expired or very short-lived token would
	// otherwise refresh in a hot loop against the IdP. The floor also paces
	// retries when a refresh fails (see the callback below).
	if refreshIn < 15*time.Second {
		refreshIn = 15 * time.Second
	}

	am.refreshTimer = time.AfterFunc(refreshIn, func() {
		// The timer may fire concurrently with Close or RevokeToken;
		// Stop() does not wait for a running callback, so re-check state
		// under the lock instead of assuming a token is present.
		am.mu.RLock()
		closed := am.closed
		refreshToken := ""
		if am.token != nil {
			refreshToken = am.token.RefreshToken
		}
		am.mu.RUnlock()
		if closed || refreshToken == "" {
			return
		}

		if _, err := am.RefreshToken(context.Background(), refreshToken); err != nil {
			// Transient failure must not silently kill auto-refresh for a
			// long-lived process: re-arm (the floor above paces retries).
			// The request paths refresh on demand regardless.
			am.scheduleRefresh()
		}
	})
}

func (am *Manager) Close() {
	am.mu.Lock()
	defer am.mu.Unlock()

	// The flag, not the Stop call, is what actually ends auto-refresh: a
	// callback already in flight re-checks it before refreshing, and refresh
	// and scheduleRefresh refuse to run once it is set.
	am.closed = true
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
