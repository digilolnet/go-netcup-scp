package scp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
	"github.com/digilolnet/go-netcup-scp/pkg/scp/auth"
)

// newTestClient creates a mock HTTP server and returns a matching Client and a cleanup func.
// The handler is called for every request made via the returned client.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, func()) {
	t.Helper()

	srv := httptest.NewServer(handler)

	authManager := auth.NewManager()
	authManager.LoadToken(&auth.TokenResponse{
		AccessToken:  "test-token",
		RefreshToken: "test-refresh",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	})

	rawClient, err := generated.NewClientWithResponses(
		srv.URL,
		generated.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer test-token")
			return nil
		}),
	)
	if err != nil {
		srv.Close()
		t.Fatalf("newTestClient: %v", err)
	}

	client := &Client{
		auth:       authManager,
		httpClient: srv.Client(),
		api:        rawClient,
		baseURL:    srv.URL,
	}

	return client, srv.Close
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func TestCheckResponse(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedCodes []int
		wantErr       bool
	}{
		{
			name:          "success with single expected code",
			statusCode:    200,
			expectedCodes: []int{200},
			wantErr:       false,
		},
		{
			name:          "success with multiple expected codes",
			statusCode:    202,
			expectedCodes: []int{200, 202},
			wantErr:       false,
		},
		{
			name:          "error when code not expected",
			statusCode:    404,
			expectedCodes: []int{200, 202},
			wantErr:       true,
		},
		{
			name:          "error with empty expected codes",
			statusCode:    200,
			expectedCodes: []int{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &mockResponse{statusCode: tt.statusCode}
			err := checkResponse(resp, tt.expectedCodes...)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockResponse struct {
	statusCode int
}

func (m *mockResponse) StatusCode() int {
	return m.statusCode
}

func TestListServers(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/servers" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		id := int32(123)
		name := "test-server"
		writeJSON(w, http.StatusOK, []generated.ServerListMinimal{
			{Id: &id, Name: &name},
		})
	})
	defer cleanup()

	servers, err := client.ListServers(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	if servers[0].Id == nil || *servers[0].Id != 123 {
		t.Errorf("expected server ID 123, got %v", servers[0].Id)
	}

	if servers[0].Name == nil || *servers[0].Name != "test-server" {
		t.Errorf("expected server name 'test-server', got %v", servers[0].Name)
	}
}

func TestGetServer(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/servers/42" || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		id := int32(42)
		hostname := "host.example.com"
		writeJSON(w, http.StatusOK, generated.Server{Id: &id, Hostname: &hostname})
	})
	defer cleanup()

	server, err := client.GetServer(context.Background(), 42, nil)
	if err != nil {
		t.Fatalf("GetServer() error = %v", err)
	}

	if server.Id == nil || *server.Id != 42 {
		t.Errorf("expected server ID 42, got %v", server.Id)
	}

	if server.Hostname == nil || *server.Hostname != "host.example.com" {
		t.Errorf("expected hostname 'host.example.com', got %v", server.Hostname)
	}
}

func TestStartServer(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusAccepted, map[string]any{"id": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.StartServer(context.Background(), 123); err != nil {
		t.Errorf("StartServer() error = %v", err)
	}
}

func TestStopServer(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusAccepted, map[string]any{"id": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.StopServer(context.Background(), 123, false); err != nil {
		t.Errorf("StopServer() error = %v", err)
	}
}

func TestRestartServer(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusAccepted, map[string]any{"id": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.RestartServer(context.Background(), 123, false); err != nil {
		t.Errorf("RestartServer() error = %v", err)
	}
}

func TestSetAutostart(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusOK, map[string]any{"id": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.SetAutostart(context.Background(), 123, true); err != nil {
		t.Errorf("SetAutostart() error = %v", err)
	}
}

func TestSetUEFI(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusOK, map[string]any{"id": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.SetUEFI(context.Background(), 123, true); err != nil {
		t.Errorf("SetUEFI() error = %v", err)
	}
}

func TestUpdateNickname(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusOK, map[string]any{"id": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.UpdateNickname(context.Background(), 123, "my-server"); err != nil {
		t.Errorf("UpdateNickname() error = %v", err)
	}
}

func TestSetCPUTopology(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusOK, map[string]any{"id": 123})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.SetCPUTopology(context.Background(), 123, 2, 4); err != nil {
		t.Errorf("SetCPUTopology() error = %v", err)
	}
}

func TestGetGuestAgent(t *testing.T) {
	available := true
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/guest-agent" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, generated.GuestAgentData{GuestAgentAvailable: &available})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	data, err := client.GetGuestAgent(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetGuestAgent() error = %v", err)
	}

	if data.GuestAgentAvailable == nil || !*data.GuestAgentAvailable {
		t.Errorf("expected GuestAgentAvailable=true, got %v", data.GuestAgentAvailable)
	}
}

func TestListImageFlavours(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/imageflavours" && r.Method == http.MethodGet {
			id := int32(1)
			writeJSON(w, http.StatusOK, []generated.ImageFlavour{
				{Id: &id, Alias: "debian-12", Name: "Debian 12", Text: "Debian Bookworm"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	flavours, err := client.ListImageFlavours(context.Background(), 123)
	if err != nil {
		t.Fatalf("ListImageFlavours() error = %v", err)
	}

	if len(flavours) != 1 {
		t.Fatalf("expected 1 flavour, got %d", len(flavours))
	}

	if flavours[0].Alias != "debian-12" {
		t.Errorf("expected alias 'debian-12', got %q", flavours[0].Alias)
	}
}

func TestGetServerLogs(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/logs" && r.Method == http.MethodGet {
			msg := "server started"
			writeJSON(w, http.StatusOK, []generated.Log{{Message: &msg}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	logs, err := client.GetServerLogs(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("GetServerLogs() error = %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	if logs[0].Message == nil || *logs[0].Message != "server started" {
		t.Errorf("unexpected log message: %v", logs[0].Message)
	}
}

func TestOptimizeStorage(t *testing.T) {
	uuid := "task-uuid-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/storageoptimization" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.OptimizeStorage(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("OptimizeStorage() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "task-uuid-1" {
		t.Errorf("unexpected task: %v", task)
	}
}
