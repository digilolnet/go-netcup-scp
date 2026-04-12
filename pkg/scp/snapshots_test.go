package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestCreateSnapshot(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/snapshots" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.CreateSnapshot(context.Background(), 123, "snap1", "my snapshot"); err != nil {
		t.Errorf("CreateSnapshot() error = %v", err)
	}
}

func TestListSnapshots(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/snapshots" && r.Method == http.MethodGet {
			name := "snap1"
			writeJSON(w, http.StatusOK, []generated.SnapshotMinimal{{Name: &name}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	snaps, err := client.ListSnapshots(context.Background(), 123)
	if err != nil {
		t.Fatalf("ListSnapshots() error = %v", err)
	}

	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}

	if snaps[0].Name == nil || *snaps[0].Name != "snap1" {
		t.Errorf("unexpected snapshot name: %v", snaps[0].Name)
	}
}

func TestGetSnapshot(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/snapshots/snap1" && r.Method == http.MethodGet {
			name := "snap1"
			writeJSON(w, http.StatusOK, generated.Snapshot{Name: &name})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	snap, err := client.GetSnapshot(context.Background(), 123, "snap1")
	if err != nil {
		t.Fatalf("GetSnapshot() error = %v", err)
	}

	if snap.Name == nil || *snap.Name != "snap1" {
		t.Errorf("unexpected snapshot name: %v", snap.Name)
	}
}

func TestDeleteSnapshot(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/snapshots/snap1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteSnapshot(context.Background(), 123, "snap1"); err != nil {
		t.Errorf("DeleteSnapshot() error = %v", err)
	}
}

func TestRevertSnapshot(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/snapshots/snap1/revert" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.RevertSnapshot(context.Background(), 123, "snap1"); err != nil {
		t.Errorf("RevertSnapshot() error = %v", err)
	}
}

func TestDryRunSnapshot(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/snapshots:dryrun" && r.Method == http.MethodPost {
			// Empty list means snapshot can be created.
			writeJSON(w, http.StatusOK, []generated.ResponseError{})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	errs, err := client.DryRunSnapshot(context.Background(), 123, true, nil)
	if err != nil {
		t.Fatalf("DryRunSnapshot() error = %v", err)
	}

	if len(errs) != 0 {
		t.Errorf("expected no blocking errors, got %v", errs)
	}
}

func TestExportSnapshot(t *testing.T) {
	uuid := "export-task-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/snapshots/snap1/export" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.ExportSnapshot(context.Background(), 123, "snap1")
	if err != nil {
		t.Fatalf("ExportSnapshot() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "export-task-1" {
		t.Errorf("unexpected task: %v", task)
	}
}
