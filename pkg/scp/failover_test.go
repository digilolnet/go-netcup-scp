package scp

import (
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestListFailoverIPv4(t *testing.T) {
	id := int32(1)
	ip := "203.0.113.1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/failoverips/v4" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.FailoverIPv4{{Id: &id, Ip: &ip}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	ips, err := client.ListFailoverIPv4(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("ListFailoverIPv4() error = %v", err)
	}

	if len(ips) != 1 {
		t.Fatalf("expected 1 IP, got %d", len(ips))
	}

	if ips[0].Ip == nil || *ips[0].Ip != "203.0.113.1" {
		t.Errorf("unexpected IP: %v", ips[0].Ip)
	}
}

func TestRouteFailoverIPv4(t *testing.T) {
	uuid := "failover-task-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/failoverips/v4/1" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.RouteFailoverIPv4(context.Background(), 1, 1, 123)
	if err != nil {
		t.Fatalf("RouteFailoverIPv4() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "failover-task-1" {
		t.Errorf("unexpected task: %v", task)
	}
}

func TestListFailoverIPv6(t *testing.T) {
	id := int32(1)
	prefix := "2001:db8::/32"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/failoverips/v6" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.FailoverIPv6{{Id: &id, NetworkPrefix: &prefix}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	ips, err := client.ListFailoverIPv6(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("ListFailoverIPv6() error = %v", err)
	}

	if len(ips) != 1 {
		t.Fatalf("expected 1 prefix, got %d", len(ips))
	}

	if ips[0].NetworkPrefix == nil || *ips[0].NetworkPrefix != "2001:db8::/32" {
		t.Errorf("unexpected prefix: %v", ips[0].NetworkPrefix)
	}
}

func TestRouteFailoverIPv6(t *testing.T) {
	uuid := "failover-task-2"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/failoverips/v6/1" && r.Method == http.MethodPatch {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.RouteFailoverIPv6(context.Background(), 1, 1, 123)
	if err != nil {
		t.Fatalf("RouteFailoverIPv6() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "failover-task-2" {
		t.Errorf("unexpected task: %v", task)
	}
}
