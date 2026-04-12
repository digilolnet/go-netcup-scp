package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestListDisks(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/disks" && r.Method == http.MethodGet {
			name := "vda"
			writeJSON(w, http.StatusOK, []generated.Disk{{Name: &name}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	disks, err := client.ListDisks(context.Background(), 123)
	if err != nil {
		t.Fatalf("ListDisks() error = %v", err)
	}

	if len(disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(disks))
	}

	if disks[0].Name == nil || *disks[0].Name != "vda" {
		t.Errorf("expected disk name 'vda', got %v", disks[0].Name)
	}
}

func TestGetDisk(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/disks/vda" && r.Method == http.MethodGet {
			name := "vda"
			writeJSON(w, http.StatusOK, generated.Disk{Name: &name})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	disk, err := client.GetDisk(context.Background(), 123, "vda")
	if err != nil {
		t.Fatalf("GetDisk() error = %v", err)
	}

	if disk.Name == nil || *disk.Name != "vda" {
		t.Errorf("expected disk name 'vda', got %v", disk.Name)
	}
}

func TestFormatDisk(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/disks/vda:format" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.FormatDisk(context.Background(), 123, "vda"); err != nil {
		t.Errorf("FormatDisk() error = %v", err)
	}
}

func TestSetDiskDriver(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/disks" && r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.SetDiskDriver(context.Background(), 123, generated.StorageDriverVIRTIO); err != nil {
		t.Errorf("SetDiskDriver() error = %v", err)
	}
}

func TestGetSupportedDiskDrivers(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/disks/supported-drivers" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.StorageDriver{"virtio", "ide"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	drivers, err := client.GetSupportedDiskDrivers(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetSupportedDiskDrivers() error = %v", err)
	}

	if len(drivers) != 2 {
		t.Fatalf("expected 2 drivers, got %d", len(drivers))
	}
}
