package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestGetUser(t *testing.T) {
	id := int32(1)
	email := "user@example.com"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, generated.User{Id: &id, Email: &email})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	user, err := client.GetUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}

	if user.Email == nil || *user.Email != "user@example.com" {
		t.Errorf("unexpected email: %v", user.Email)
	}
}

func TestUpdateUser(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1" && r.Method == http.MethodPut {
			writeJSON(w, http.StatusOK, generated.UserSave{Language: "en"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	result, err := client.UpdateUser(context.Background(), 1, generated.UserSave{Language: "en"})
	if err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}

	if result.Language != "en" {
		t.Errorf("unexpected language: %q", result.Language)
	}
}

func TestGetUserLogs(t *testing.T) {
	msg := "login"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/logs" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.Log{{Message: &msg}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	logs, err := client.GetUserLogs(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("GetUserLogs() error = %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].Message == nil || *logs[0].Message != "login" {
		t.Errorf("unexpected message: %v", logs[0].Message)
	}
}
