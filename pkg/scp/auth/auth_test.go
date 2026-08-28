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
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testServer creates a mock OAuth2 server and returns it along with a Manager
// whose httpClient points at the mock server.
func testServer(t *testing.T, mux *http.ServeMux) (*httptest.Server, *Manager) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mgr := NewManager(
		WithAutoRefresh(false),
		WithHTTPClient(&http.Client{Timeout: 5 * time.Second}),
	)
	// Point the manager at the test server by overriding authURL for tests.
	// We do this by patching the httpClient transport to rewrite requests.
	mgr.httpClient.Transport = rewriteTransport{base: http.DefaultTransport, from: DefaultAuthURL, to: srv.URL}
	return srv, mgr
}

// rewriteTransport rewrites the scheme+host of every request from `from` to `to`.
type rewriteTransport struct {
	base http.RoundTripper
	from string
	to   string
}

func (rt rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	toURL, _ := url.Parse(rt.to)
	cloned.URL.Host = toURL.Host
	// Strip the authURL prefix from the path, keep the path suffix
	path := req.URL.String()
	if strings.HasPrefix(path, rt.from) {
		cloned.URL.Path = path[len(rt.from):]
	}
	return rt.base.RoundTrip(cloned)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// --- NewManager / GetAccessToken / GetRefreshToken ---

func TestGetAccessToken_NoToken(t *testing.T) {
	mgr := NewManager(WithAutoRefresh(false))
	_, err := mgr.GetAccessToken()
	if err == nil {
		t.Fatal("expected error when no token is set")
	}
}

func TestGetRefreshToken_NoToken(t *testing.T) {
	mgr := NewManager(WithAutoRefresh(false))
	_, err := mgr.GetRefreshToken()
	if err == nil {
		t.Fatal("expected error when no token is set")
	}
}

func TestLoadToken_GetAccessToken(t *testing.T) {
	mgr := NewManager(WithAutoRefresh(false))
	tok := &TokenResponse{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresIn:    3600,
		ObtainedAt:   time.Now(),
	}
	mgr.LoadToken(tok)

	got, err := mgr.GetAccessToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "access-abc" {
		t.Errorf("got %q, want %q", got, "access-abc")
	}
}

func TestLoadToken_StaleAndLegacyExpired(t *testing.T) {
	// A token obtained longer ago than ExpiresIn must not be served.
	mgr := NewManager(WithAutoRefresh(false))
	mgr.LoadToken(&TokenResponse{
		AccessToken: "stale",
		ExpiresIn:   300,
		ObtainedAt:  time.Now().Add(-10 * time.Minute),
	})
	if _, err := mgr.GetAccessToken(); err == nil {
		t.Fatal("expected error for stale token")
	}

	// Legacy token files carry no obtained_at; treat them as expired so the
	// first use refreshes instead of sending a possibly-expired token.
	mgr = NewManager(WithAutoRefresh(false))
	mgr.LoadToken(&TokenResponse{AccessToken: "legacy", ExpiresIn: 300})
	if _, err := mgr.GetAccessToken(); err == nil {
		t.Fatal("expected error for legacy token without obtained_at")
	}
}

func TestLoadToken_GetRefreshToken(t *testing.T) {
	mgr := NewManager(WithAutoRefresh(false))
	tok := &TokenResponse{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresIn:    3600,
	}
	mgr.LoadToken(tok)

	got, err := mgr.GetRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "refresh-xyz" {
		t.Errorf("got %q, want %q", got, "refresh-xyz")
	}
}

func TestGetAccessToken_Expired(t *testing.T) {
	mgr := NewManager(WithAutoRefresh(false))
	tok := &TokenResponse{
		AccessToken:  "old-token",
		RefreshToken: "refresh-xyz",
		ExpiresIn:    -1, // already expired
	}
	mgr.LoadToken(tok)

	_, err := mgr.GetAccessToken()
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

// --- InitiateDeviceAuth ---

func TestInitiateDeviceAuth_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/device", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, DeviceAuthResponse{
			DeviceCode:              "dev-code-123",
			UserCode:                "USER-CODE",
			VerificationURI:         "https://example.com/activate",
			VerificationURIComplete: "https://example.com/activate?user_code=USER-CODE",
			ExpiresIn:               300,
			Interval:                5,
		})
	})

	_, mgr := testServer(t, mux)
	resp, err := mgr.InitiateDeviceAuth(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DeviceCode != "dev-code-123" {
		t.Errorf("DeviceCode = %q, want %q", resp.DeviceCode, "dev-code-123")
	}
	if resp.UserCode != "USER-CODE" {
		t.Errorf("UserCode = %q, want %q", resp.UserCode, "USER-CODE")
	}
}

func TestWithAuthURL_RedirectsRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/device", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, DeviceAuthResponse{DeviceCode: "dev-code-456"})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	// No transport rewriting: the option alone must point requests at srv.
	// The trailing slash must be tolerated.
	mgr := NewManager(WithAutoRefresh(false), WithAuthURL(srv.URL+"/"))
	resp, err := mgr.InitiateDeviceAuth(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DeviceCode != "dev-code-456" {
		t.Errorf("DeviceCode = %q, want %q", resp.DeviceCode, "dev-code-456")
	}
}

func TestInitiateDeviceAuth_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/device", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})

	_, mgr := testServer(t, mux)
	_, err := mgr.InitiateDeviceAuth(context.Background())
	if err == nil {
		t.Fatal("expected error on server error")
	}
}

// --- PollForToken ---

func TestPollForToken_Success(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 2 {
			// First poll returns authorization_pending
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{
				"error":             "authorization_pending",
				"error_description": "waiting for user",
			})
			return
		}
		writeJSON(w, TokenResponse{
			AccessToken:  "polled-access",
			RefreshToken: "polled-refresh",
			ExpiresIn:    3600,
		})
	})

	_, mgr := testServer(t, mux)
	tok, err := mgr.PollForToken(context.Background(), "dev-code", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "polled-access" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "polled-access")
	}
	if callCount < 2 {
		t.Errorf("expected at least 2 poll calls, got %d", callCount)
	}
}

func TestPollForToken_ContextCancelled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "authorization_pending"})
	})

	_, mgr := testServer(t, mux)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := mgr.PollForToken(ctx, "dev-code", 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error on context cancellation")
	}
}

func TestPollForToken_SlowDown(t *testing.T) {
	old := slowDownIncrement
	slowDownIncrement = 20 * time.Millisecond
	defer func() { slowDownIncrement = old }()

	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"error": "slow_down"})
			return
		}
		writeJSON(w, TokenResponse{
			AccessToken:  "slow-access",
			RefreshToken: "slow-refresh",
			ExpiresIn:    3600,
		})
	})

	_, mgr := testServer(t, mux)
	tok, err := mgr.PollForToken(context.Background(), "dev-code", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "slow-access" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "slow-access")
	}
}

func TestPollForToken_FatalError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{
			"error":             "expired_token",
			"error_description": "device code expired",
		})
	})

	_, mgr := testServer(t, mux)
	_, err := mgr.PollForToken(context.Background(), "dev-code", 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error on fatal auth error")
	}
}

// --- RefreshToken ---

func TestRefreshToken_Success(t *testing.T) {
	var refreshCallback *TokenResponse
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, TokenResponse{
			AccessToken:  "refreshed-access",
			RefreshToken: "refreshed-refresh",
			ExpiresIn:    3600,
		})
	})

	_, mgr := testServer(t, mux)
	mgr.onRefresh = func(tok *TokenResponse) { refreshCallback = tok }

	tok, err := mgr.RefreshToken(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "refreshed-access" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "refreshed-access")
	}
	if refreshCallback == nil || refreshCallback.AccessToken != "refreshed-access" {
		t.Errorf("onRefresh callback not called or received wrong token")
	}

	// Token should be stored
	got, err := mgr.GetAccessToken()
	if err != nil {
		t.Fatalf("GetAccessToken after refresh: %v", err)
	}
	if got != "refreshed-access" {
		t.Errorf("stored token = %q, want %q", got, "refreshed-access")
	}
}

func TestRefreshToken_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	_, mgr := testServer(t, mux)
	_, err := mgr.RefreshToken(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error on server error")
	}
}

// --- RevokeToken ---

func TestRevokeToken_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/revoke", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	_, mgr := testServer(t, mux)
	mgr.LoadToken(&TokenResponse{
		AccessToken:  "acc",
		RefreshToken: "ref",
		ExpiresIn:    3600,
	})

	if err := mgr.RevokeToken(context.Background(), "ref"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Token should be cleared
	if _, err := mgr.GetAccessToken(); err == nil {
		t.Fatal("expected error after token revocation")
	}
}

func TestRevokeToken_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/revoke", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})

	_, mgr := testServer(t, mux)
	err := mgr.RevokeToken(context.Background(), "some-token")
	if err == nil {
		t.Fatal("expected error on revoke server error")
	}
}

// --- Close ---

func TestClose_StopsTimer(t *testing.T) {
	mgr := NewManager(WithAutoRefresh(true))
	mgr.LoadToken(&TokenResponse{
		AccessToken:  "acc",
		RefreshToken: "ref",
		ExpiresIn:    3600,
	})
	// scheduleRefresh should have set a timer; Close should stop it without panic.
	mgr.Close()

	mgr.mu.Lock()
	hasTimer := mgr.refreshTimer != nil
	mgr.mu.Unlock()

	if hasTimer {
		t.Error("expected refreshTimer to be nil after Close")
	}
}

// --- WithTokenRefreshCallback option ---

func TestWithTokenRefreshCallback(t *testing.T) {
	called := false
	cb := func(_ *TokenResponse) { called = true }
	mgr := NewManager(WithAutoRefresh(false), WithTokenRefreshCallback(cb))

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, TokenResponse{AccessToken: "x", RefreshToken: "y", ExpiresIn: 100})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mgr.httpClient.Transport = rewriteTransport{base: http.DefaultTransport, from: DefaultAuthURL, to: srv.URL}

	if _, err := mgr.RefreshToken(context.Background(), "ref"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected refresh callback to be called")
	}
}

func TestValidAccessToken_SingleFlight(t *testing.T) {
	var refreshes atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		refreshes.Add(1)
		time.Sleep(50 * time.Millisecond) // widen the race window
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "fresh",
			RefreshToken: "refresh-2",
			ExpiresIn:    300,
		})
	})
	_, mgr := testServer(t, mux)
	mgr.LoadToken(&TokenResponse{
		AccessToken:  "stale",
		RefreshToken: "refresh-1",
		ExpiresIn:    300,
		ObtainedAt:   time.Now().Add(-10 * time.Minute),
	})

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tok, err := mgr.ValidAccessToken(context.Background())
			if err != nil || tok != "fresh" {
				t.Errorf("ValidAccessToken = %q, %v", tok, err)
			}
		}()
	}
	wg.Wait()
	if got := refreshes.Load(); got != 1 {
		t.Errorf("want exactly 1 refresh request, got %d", got)
	}
}

func TestValidAccessToken_CachedNoRefresh(t *testing.T) {
	mux := http.NewServeMux() // any request would 404 and fail the refresh
	_, mgr := testServer(t, mux)
	mgr.LoadToken(&TokenResponse{AccessToken: "live", ExpiresIn: 300, ObtainedAt: time.Now()})
	tok, err := mgr.ValidAccessToken(context.Background())
	if err != nil || tok != "live" {
		t.Fatalf("ValidAccessToken = %q, %v", tok, err)
	}
}

func TestRefreshToken_AfterCloseRefused(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(TokenResponse{AccessToken: "x", ExpiresIn: 300})
	})
	_, mgr := testServer(t, mux)
	mgr.LoadToken(&TokenResponse{AccessToken: "a", RefreshToken: "r", ExpiresIn: 300, ObtainedAt: time.Now()})
	mgr.Close()
	if _, err := mgr.RefreshToken(context.Background(), "r"); err == nil {
		t.Fatal("want error refreshing after Close")
	}
}

func TestSetToken_DoesNotAliasCaller(t *testing.T) {
	mgr := NewManager(WithAutoRefresh(false))
	tok := &TokenResponse{AccessToken: "a", ExpiresIn: 300, ObtainedAt: time.Now()}
	mgr.LoadToken(tok)
	tok.AccessToken = "mutated-by-caller"
	got, err := mgr.GetAccessToken()
	if err != nil || got != "a" {
		t.Fatalf("manager state affected by caller mutation: %q, %v", got, err)
	}
}
