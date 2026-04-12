package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestListAvailableISOs(t *testing.T) {
	id := int32(1)
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/isoimages" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.IsoImage{{Id: &id, Name: "debian-12.iso"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	isos, err := client.ListAvailableISOs(context.Background(), 123)
	if err != nil {
		t.Fatalf("ListAvailableISOs() error = %v", err)
	}

	if len(isos) != 1 {
		t.Fatalf("expected 1 ISO, got %d", len(isos))
	}

	if isos[0].Name != "debian-12.iso" {
		t.Errorf("unexpected ISO name: %q", isos[0].Name)
	}
}

func TestAttachISO(t *testing.T) {
	isoID := int32(1)
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/iso" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.AttachISO(context.Background(), 123, &AttachISOOptions{IsoID: &isoID})
	if err != nil {
		t.Errorf("AttachISO() error = %v", err)
	}
	if task != nil {
		t.Errorf("expected nil task for 200 response, got %v", task)
	}
}

func TestAttachISOAsync(t *testing.T) {
	isoID := int32(1)
	uuid := "attach-iso-task-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/iso" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.AttachISO(context.Background(), 123, &AttachISOOptions{IsoID: &isoID})
	if err != nil {
		t.Errorf("AttachISO() error = %v", err)
	}
	if task == nil || task.Uuid == nil || *task.Uuid != uuid {
		t.Errorf("unexpected task: %v", task)
	}
}

func TestAttachISOValidation(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if _, err := client.AttachISO(context.Background(), 123, nil); err == nil {
		t.Error("expected error for nil opts")
	}

	if _, err := client.AttachISO(context.Background(), 123, &AttachISOOptions{}); err == nil {
		t.Error("expected error when neither IsoID nor UserIsoName is set")
	}
}

func TestDetachISO(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/iso" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DetachISO(context.Background(), 123); err != nil {
		t.Errorf("DetachISO() error = %v", err)
	}
}

func TestListUserISOs(t *testing.T) {
	key := "my.iso"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/isos" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.S3Object{{Key: &key}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	isos, err := client.ListUserISOs(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListUserISOs() error = %v", err)
	}

	if len(isos) != 1 {
		t.Fatalf("expected 1 ISO, got %d", len(isos))
	}

	if isos[0].Key == nil || *isos[0].Key != "my.iso" {
		t.Errorf("unexpected key: %v", isos[0].Key)
	}
}

func TestDeleteUserISO(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/isos/my.iso" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteUserISO(context.Background(), 1, "my.iso"); err != nil {
		t.Errorf("DeleteUserISO() error = %v", err)
	}
}
