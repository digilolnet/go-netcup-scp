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

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
	"github.com/digilolnet/go-netcup-scp/pkg/scp/auth"
)

// newTestContext returns a cmdContext whose client talks to an httptest
// server running handler. The completion cache is isolated per test by
// pointing XDG_CACHE_HOME at a temp dir.
func newTestContext(t *testing.T, handler http.Handler) *cmdContext {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	mgr := auth.NewManager(auth.WithAutoRefresh(false))
	mgr.LoadToken(&auth.TokenResponse{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		ObtainedAt:  time.Now(),
	})
	t.Cleanup(mgr.Close)

	client, err := scp.NewClient(mgr, scp.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	t.Cleanup(client.Close)

	return &cmdContext{
		ctx:       context.Background(),
		client:    client,
		authMgr:   mgr,
		tokenFile: filepath.Join(t.TempDir(), "token.json"),
	}
}
