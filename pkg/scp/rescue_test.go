package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestGetRescueSystem(t *testing.T) {
	active := true
	password := "s3cr3t"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/rescuesystem" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, generated.RescueSystemStatus{Active: &active, Password: &password})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	status, err := client.GetRescueSystem(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetRescueSystem() error = %v", err)
	}

	if status.Active == nil || !*status.Active {
		t.Errorf("expected Active=true, got %v", status.Active)
	}
}

func TestActivateRescueSystem(t *testing.T) {
	uuid := "rescue-task-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/rescuesystem" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.ActivateRescueSystem(context.Background(), 123)
	if err != nil {
		t.Fatalf("ActivateRescueSystem() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "rescue-task-1" {
		t.Errorf("unexpected task: %v", task)
	}
}

func TestDeactivateRescueSystem(t *testing.T) {
	uuid := "rescue-task-2"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/rescuesystem" && r.Method == http.MethodDelete {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.DeactivateRescueSystem(context.Background(), 123)
	if err != nil {
		t.Fatalf("DeactivateRescueSystem() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "rescue-task-2" {
		t.Errorf("unexpected task: %v", task)
	}
}
