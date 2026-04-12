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
}

func (m *mockRefresher) GetRefreshToken() (string, error) {
	return "test-refresh-token", nil
}

func (m *mockRefresher) RefreshToken(_ context.Context, _ string) (*auth.TokenResponse, error) {
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
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error when refresh fails, got nil")
	}
	// Only one call — no retry after a failed refresh.
	if calls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", calls.Load())
	}
}
