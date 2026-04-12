package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestListSSHKeys(t *testing.T) {
	id := int32(1)
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/ssh-keys" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.SSHKey{
				{Id: &id, Name: "my-key", Key: "ssh-ed25519 AAAA..."},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	keys, err := client.ListSSHKeys(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListSSHKeys() error = %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	if keys[0].Name != "my-key" {
		t.Errorf("unexpected key name: %q", keys[0].Name)
	}
}

func TestCreateSSHKey(t *testing.T) {
	id := int32(2)
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/ssh-keys" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusCreated, generated.SSHKey{
				Id: &id, Name: "new-key", Key: "ssh-ed25519 AAAA...",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	key, err := client.CreateSSHKey(context.Background(), 1, generated.SSHKey{
		Name: "new-key",
		Key:  "ssh-ed25519 AAAA...",
	})
	if err != nil {
		t.Fatalf("CreateSSHKey() error = %v", err)
	}

	if key.Name != "new-key" {
		t.Errorf("unexpected key name: %q", key.Name)
	}
}

func TestDeleteSSHKey(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/ssh-keys/2" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteSSHKey(context.Background(), 1, 2); err != nil {
		t.Errorf("DeleteSSHKey() error = %v", err)
	}
}
