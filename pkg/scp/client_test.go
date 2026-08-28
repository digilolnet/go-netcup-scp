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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/digilolnet/go-netcup-scp/pkg/scp/auth"
)

// mockRefresher implements tokenRefresher for tests.
type mockRefresher struct {
	newToken *auth.TokenResponse
	err      error
	calls    int
}

func (m *mockRefresher) GetRefreshToken() (string, error) {
	return "test-refresh-token", nil
}

func (m *mockRefresher) RefreshToken(_ context.Context, _ string) (*auth.TokenResponse, error) {
	m.calls++
	return m.newToken, m.err
}

func TestRetryTransport_NoRetryOn200(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rt := &retryTransport{base: srv.Client().Transport, auth: &mockRefresher{}}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if calls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", calls.Load())
	}
}

func TestRetryTransport_RefreshesOn401AndRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Second call must carry the refreshed token.
		if got := r.Header.Get("Authorization"); got != "Bearer new-access-token" {
			t.Errorf("retry Authorization = %q, want %q", got, "Bearer new-access-token")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	refresher := &mockRefresher{
		newToken: &auth.TokenResponse{AccessToken: "new-access-token", ExpiresIn: 300},
	}
	rt := &retryTransport{base: srv.Client().Transport, auth: refresher}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer old-access-token")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if calls.Load() != 2 {
		t.Errorf("expected 2 calls, got %d", calls.Load())
	}
}

func TestRetryTransport_PostBodyRestoredOnRetry(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	refresher := &mockRefresher{
		newToken: &auth.TokenResponse{AccessToken: "new-access-token", ExpiresIn: 300},
	}
	rt := &retryTransport{base: srv.Client().Transport, auth: refresher}

	body := `{"hello":"world"}`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL,
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer stale")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if len(bodies) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(bodies))
	}
	if bodies[1] != body {
		t.Errorf("retry body = %q, want %q", bodies[1], body)
	}
}

func TestRetryTransport_NoRetryWhenRefreshFails(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	refresher := &mockRefresher{err: auth.ErrTokenRefreshFailed}
	rt := &retryTransport{base: srv.Client().Transport, auth: refresher}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer stale")
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error when refresh fails, got nil")
	}
	// Only one call — no retry after a failed refresh.
	if calls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", calls.Load())
	}
}

func TestRetryTransport_NoRefreshForUnauthenticatedRequests(t *testing.T) {
	// Presigned S3 requests carry no Authorization header; a 401 from them
	// must pass through untouched — no OAuth refresh, no token attached.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	refresher := &mockRefresher{
		newToken: &auth.TokenResponse{AccessToken: "should-not-be-used", ExpiresIn: 300},
	}
	rt := &retryTransport{base: srv.Client().Transport, auth: refresher}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want passthrough 401", resp.StatusCode)
	}
	if refresher.calls != 0 {
		t.Errorf("refresh called %d times for unauthenticated request", refresher.calls)
	}
}

func TestRetryTransport_NoRetryWithConsumedStreamingBody(t *testing.T) {
	// A body without GetBody (e.g. a raw io.Reader upload) is consumed by the
	// first attempt; retrying would send it empty or truncated.
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	refresher := &mockRefresher{
		newToken: &auth.TokenResponse{AccessToken: "fresh", ExpiresIn: 300},
	}
	rt := &retryTransport{base: srv.Client().Transport, auth: refresher}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPut, srv.URL,
		io.NopCloser(strings.NewReader("streaming-payload")))
	req.Header.Set("Authorization", "Bearer stale")
	req.GetBody = nil

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if got := requests.Load(); got != 1 {
		t.Errorf("server saw %d requests, want 1 (no retry)", got)
	}
}
