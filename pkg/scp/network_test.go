package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestListInterfaces(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces" && r.Method == http.MethodGet {
			mac := "aa:bb:cc:dd:ee:ff"
			writeJSON(w, http.StatusOK, []generated.Interface{{Mac: &mac}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	ifaces, err := client.ListInterfaces(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("ListInterfaces() error = %v", err)
	}

	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}

	if ifaces[0].Mac == nil || *ifaces[0].Mac != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("unexpected MAC: %v", ifaces[0].Mac)
	}
}

func TestGetInterface(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces/aa:bb:cc:dd:ee:ff" && r.Method == http.MethodGet {
			mac := "aa:bb:cc:dd:ee:ff"
			writeJSON(w, http.StatusOK, generated.Interface{Mac: &mac})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	iface, err := client.GetInterface(context.Background(), 123, "aa:bb:cc:dd:ee:ff", nil)
	if err != nil {
		t.Fatalf("GetInterface() error = %v", err)
	}

	if iface.Mac == nil || *iface.Mac != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("unexpected MAC: %v", iface.Mac)
	}
}

func TestSetRDNSv4(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/rdns/ipv4" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.SetRDNSv4(context.Background(), "1.2.3.4", "host.example.com"); err != nil {
		t.Errorf("SetRDNSv4() error = %v", err)
	}
}

func TestSetRDNSv6(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/rdns/ipv6" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.SetRDNSv6(context.Background(), "2001:db8::1", "host.example.com"); err != nil {
		t.Errorf("SetRDNSv6() error = %v", err)
	}
}

func TestGetRDNSv4(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/rdns/ipv4/1.2.3.4" && r.Method == http.MethodGet {
			rdns := "host.example.com"
			writeJSON(w, http.StatusOK, generated.Rdns{Rdns: &rdns})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	rdns, err := client.GetRDNSv4(context.Background(), "1.2.3.4")
	if err != nil {
		t.Fatalf("GetRDNSv4() error = %v", err)
	}

	if rdns.Rdns == nil || *rdns.Rdns != "host.example.com" {
		t.Errorf("unexpected RDNS: %v", rdns.Rdns)
	}
}

func TestGetRDNSv6(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/rdns/ipv6/2001:db8::1" && r.Method == http.MethodGet {
			rdns := "host6.example.com"
			writeJSON(w, http.StatusOK, generated.Rdns{Rdns: &rdns})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	rdns, err := client.GetRDNSv6(context.Background(), "2001:db8::1")
	if err != nil {
		t.Fatalf("GetRDNSv6() error = %v", err)
	}

	if rdns.Rdns == nil || *rdns.Rdns != "host6.example.com" {
		t.Errorf("unexpected RDNS: %v", rdns.Rdns)
	}
}

func TestDeleteRDNSv4(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/rdns/ipv4/1.2.3.4" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteRDNSv4(context.Background(), "1.2.3.4"); err != nil {
		t.Errorf("DeleteRDNSv4() error = %v", err)
	}
}

func TestDeleteRDNSv6(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/rdns/ipv6/2001:db8::1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteRDNSv6(context.Background(), "2001:db8::1"); err != nil {
		t.Errorf("DeleteRDNSv6() error = %v", err)
	}
}

func TestCreateVLanInterface(t *testing.T) {
	uuid := "vlan-task-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.CreateVLanInterface(context.Background(), 123, 10, generated.NetworkDriverVIRTIO)
	if err != nil {
		t.Errorf("CreateVLanInterface() error = %v", err)
	}
	if task == nil || task.Uuid == nil || *task.Uuid != uuid {
		t.Errorf("unexpected task: %v", task)
	}
}

func TestDeleteInterface(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces/aa:bb:cc:dd:ee:ff" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteInterface(context.Background(), 123, "aa:bb:cc:dd:ee:ff"); err != nil {
		t.Errorf("DeleteInterface() error = %v", err)
	}
}

func TestUpdateInterfaceDriver(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/interfaces/aa:bb:cc:dd:ee:ff" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.UpdateInterfaceDriver(context.Background(), 123, "aa:bb:cc:dd:ee:ff", generated.NetworkDriverVIRTIO); err != nil {
		t.Errorf("UpdateInterfaceDriver() error = %v", err)
	}
}
