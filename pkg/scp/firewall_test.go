package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestGetFirewall(t *testing.T) {
	active := true
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces/aa:bb:cc:dd:ee:ff/firewall" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, generated.ServerFirewall{Active: &active})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	fw, err := client.GetFirewall(context.Background(), 123, "aa:bb:cc:dd:ee:ff", nil)
	if err != nil {
		t.Fatalf("GetFirewall() error = %v", err)
	}

	if fw.Active == nil || !*fw.Active {
		t.Errorf("expected Active=true, got %v", fw.Active)
	}
}

func TestUpdateFirewall(t *testing.T) {
	uuid := "fw-task-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces/aa:bb:cc:dd:ee:ff/firewall" && r.Method == http.MethodPut {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.UpdateFirewall(context.Background(), 123, "aa:bb:cc:dd:ee:ff", generated.ServerFirewallSave{
		CopiedPolicies: []generated.IdentifierInt{},
		UserPolicies:   []generated.IdentifierInt{},
	})
	if err != nil {
		t.Fatalf("UpdateFirewall() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "fw-task-1" {
		t.Errorf("unexpected task: %v", task)
	}
}

func TestReapplyFirewall(t *testing.T) {
	uuid := "fw-task-2"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces/aa:bb:cc:dd:ee:ff/firewall:reapply" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.ReapplyFirewall(context.Background(), 123, "aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("ReapplyFirewall() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "fw-task-2" {
		t.Errorf("unexpected task: %v", task)
	}
}

func TestRestoreCopiedFirewallPolicies(t *testing.T) {
	uuid := "fw-task-3"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces/aa:bb:cc:dd:ee:ff/firewall:restore-copied-policies" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.RestoreCopiedFirewallPolicies(context.Background(), 123, "aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("RestoreCopiedFirewallPolicies() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "fw-task-3" {
		t.Errorf("unexpected task: %v", task)
	}
}
