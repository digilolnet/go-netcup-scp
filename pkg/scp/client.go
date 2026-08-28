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

package scp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"slices"
	"time"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
	"github.com/digilolnet/go-netcup-scp/internal/version"
	"github.com/digilolnet/go-netcup-scp/pkg/scp/auth"
)

// Maintenance is an announced host maintenance window.
type Maintenance = generated.Maintenance

const (
	BaseURL = "https://www.servercontrolpanel.de/scp-core"
)

type Client struct {
	auth       *auth.Manager
	httpClient *http.Client
	api        *generated.ClientWithResponses
	baseURL    string
	traceDir   string
}

type ClientOption func(*Client)

// WithBaseURL overrides the default API base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// tokenRefresher is the subset of auth.Manager used by retryTransport.
// Using an interface keeps the transport testable without a real auth server.
type tokenRefresher interface {
	GetRefreshToken() (string, error)
	RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenResponse, error)
}

// retryTransport is an http.RoundTripper that transparently refreshes the
// OAuth2 access token on 401 responses and retries the request once.
// The auth manager's existing onRefresh callback handles persisting the new token.
type retryTransport struct {
	base http.RoundTripper
	auth tokenRefresher
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	// Only requests that carried our Bearer token are candidates: a 401 from
	// anywhere else (e.g. a presigned S3 upload URL) must not trigger an
	// OAuth refresh, and retrying would attach the API token to a
	// third-party host.
	if req.Header.Get("Authorization") == "" {
		return resp, nil
	}
	// A consumed streaming body without GetBody cannot be replayed; a retry
	// would silently send an empty or truncated body.
	if req.Body != nil && req.GetBody == nil {
		return resp, nil
	}

	// Drain and discard the 401 body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Refresh the access token.
	refreshToken, err := t.auth.GetRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("auto-refresh: %w", err)
	}
	newTok, err := t.auth.RefreshToken(req.Context(), refreshToken)
	if err != nil {
		return nil, fmt.Errorf("auto-refresh: %w", err)
	}

	// Clone the request so we don't mutate the original, then update the token.
	retryReq := req.Clone(req.Context())
	retryReq.Header.Set("Authorization", "Bearer "+newTok.AccessToken)

	// Restore the request body if it was consumed (e.g. POST/PATCH).
	if req.GetBody != nil {
		retryReq.Body, err = req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("auto-refresh: restore body: %w", err)
		}
	}

	return t.base.RoundTrip(retryReq)
}

func NewClient(authManager *auth.Manager, opts ...ClientOption) (*Client, error) {
	client := &Client{
		auth:    authManager,
		baseURL: BaseURL,
	}

	for _, opt := range opts {
		opt(client)
	}

	base := http.RoundTripper(http.DefaultTransport)
	if client.traceDir != "" {
		if err := os.MkdirAll(client.traceDir, 0o755); err != nil {
			return nil, fmt.Errorf("create trace dir: %w", err)
		}
		base = &traceTransport{base: base, dir: client.traceDir}
	}
	client.httpClient = &http.Client{
		Transport: &retryTransport{
			base: base,
			auth: authManager,
		},
	}

	requestEditor := func(ctx context.Context, req *http.Request) error {
		token, err := client.validAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", version.UserAgent())
		return nil
	}

	api, err := generated.NewClientWithResponses(
		client.baseURL,
		generated.WithHTTPClient(client.httpClient),
		generated.WithRequestEditorFn(requestEditor),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	client.api = api
	return client, nil
}

func (c *Client) API() *generated.ClientWithResponses {
	return c.api
}

func (c *Client) Auth() *auth.Manager {
	return c.auth
}

func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.api.GetApiPingWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	return nil
}

func (c *Client) Close() {
	c.auth.Close()
}

// GetMaintenance retrieves upcoming maintenance windows.
// Returns an empty slice if no maintenance is scheduled.
//
// Deprecated: the upstream endpoint is deprecated and will be removed by 2026-12-31.
// Note: the generated parser expects a JSON array but the live API returns a single
// object, so we bypass the generated parser and handle both formats here.
func (c *Client) GetMaintenance(ctx context.Context) ([]Maintenance, error) {
	rawResp, err := c.api.GetApiV1Maintenance(ctx)
	if err != nil {
		return nil, fmt.Errorf("get maintenance: %w", err)
	}
	defer rawResp.Body.Close()

	if rawResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get maintenance: unexpected status code: %d", rawResp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(rawResp.Body)
	if err != nil {
		return nil, fmt.Errorf("get maintenance: read body: %w", err)
	}

	// Try array first (spec says it should be an array).
	var windows []Maintenance
	if err := json.Unmarshal(bodyBytes, &windows); err == nil {
		if windows == nil {
			windows = []Maintenance{}
		}
		return windows, nil
	}

	// Fall back to single object (actual live API behaviour).
	// Use pointer fields so we can detect JSON null vs. absent: the API uses
	// {"finishAt":null,"startAt":null} to signal "no maintenance scheduled".
	var single struct {
		FinishAt *time.Time `json:"finishAt"`
		StartAt  *time.Time `json:"startAt"`
	}
	if err := json.Unmarshal(bodyBytes, &single); err != nil {
		return nil, fmt.Errorf("get maintenance: parse response: %w", err)
	}
	if single.StartAt == nil || single.FinishAt == nil {
		return []Maintenance{}, nil
	}
	return []Maintenance{{StartAt: single.StartAt, FinishAt: single.FinishAt}}, nil
}

// requireID rejects empty path identifiers: an empty segment would silently
// address the collection endpoint instead (e.g. GET /disks/ is the list), which
// surfaces as a confusing unmarshal error or a 405 rather than a clear message.
func requireID(op, name, value string) error {
	if value == "" {
		return fmt.Errorf("%s: %s must not be empty", op, name)
	}
	return nil
}

// deref returns the dereferenced value of p, or the zero value of T if p is nil.
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// checkResponse validates that the response status code matches one of the expected codes.
// If the status code is unexpected and the response body contains a JSON error message, it is included in the error.
func checkResponse(resp interface{ StatusCode() int }, expectedCodes ...int) error {
	if slices.Contains(expectedCodes, resp.StatusCode()) {
		return nil
	}
	if body := responseBody(resp); len(body) > 0 {
		var errResp struct {
			Message *string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != nil {
			return fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode(), *errResp.Message)
		}
	}
	return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
}

// responseBody extracts the raw body bytes from a generated response struct using reflection.
// All generated *WithResponse types have a Body []byte field.
func responseBody(resp interface{ StatusCode() int }) []byte {
	v := reflect.ValueOf(resp)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if f := v.FieldByName("Body"); f.IsValid() && f.Kind() == reflect.Slice {
		return f.Bytes()
	}
	return nil
}

// pickBody validates the status code and returns the response body of a 2xx
// response, which the API may serve as either application/json or
// application/hal+json (server-side content negotiation): whichever of the two
// generated body fields is populated wins. A successful status with neither
// body set is an error.
func pickBody[T any](op string, resp interface{ StatusCode() int }, jsonBody, halBody *T, codes ...int) (*T, error) {
	if err := checkResponse(resp, codes...); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if halBody != nil {
		return halBody, nil
	}
	if jsonBody != nil {
		return jsonBody, nil
	}
	return nil, fmt.Errorf("%s: empty response", op)
}

// pickBodyVal is pickBody for wrappers that return the dereferenced value.
func pickBodyVal[T any](op string, resp interface{ StatusCode() int }, jsonBody, halBody *T, codes ...int) (T, error) {
	body, err := pickBody(op, resp, jsonBody, halBody, codes...)
	if err != nil {
		var zero T
		return zero, err
	}
	return *body, nil
}

// taskBody is pickBody for task-returning mutations, where a bodiless success
// (e.g. 204 no-op) legitimately yields a nil result.
func taskBody[T any](op string, resp interface{ StatusCode() int }, jsonBody, halBody *T, codes ...int) (*T, error) {
	if err := checkResponse(resp, codes...); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if halBody != nil {
		return halBody, nil
	}
	return jsonBody, nil
}
