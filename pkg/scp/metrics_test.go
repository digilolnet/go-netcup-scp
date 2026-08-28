// Copyright 2026 Laurynas Četyrkinas <laurynas@digilol.net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scp

import (
	"context"
	"net/http"
	"testing"
)

func TestGetCPUMetrics(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/metrics/cpu" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]any{"usage": 42.5})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	metrics, err := client.GetCPUMetrics(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("GetCPUMetrics() error = %v", err)
	}

	if _, ok := metrics["usage"]; !ok {
		t.Errorf("expected 'usage' key in metrics, got %v", metrics)
	}
}

func TestGetDiskMetrics(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/metrics/disk" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]any{"read_bytes": 1024})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	metrics, err := client.GetDiskMetrics(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("GetDiskMetrics() error = %v", err)
	}

	if _, ok := metrics["read_bytes"]; !ok {
		t.Errorf("expected 'read_bytes' key in metrics, got %v", metrics)
	}
}

func TestGetNetworkMetrics(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/metrics/network" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]any{"rx_bytes": 2048})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	metrics, err := client.GetNetworkMetrics(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("GetNetworkMetrics() error = %v", err)
	}

	if _, ok := metrics["rx_bytes"]; !ok {
		t.Errorf("expected 'rx_bytes' key in metrics, got %v", metrics)
	}
}

func TestGetNetworkPacketMetrics(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/servers/123/metrics/network/packet" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]any{"rx_packets": 512})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	metrics, err := client.GetNetworkPacketMetrics(context.Background(), 123, nil)
	if err != nil {
		t.Fatalf("GetNetworkPacketMetrics() error = %v", err)
	}

	if _, ok := metrics["rx_packets"]; !ok {
		t.Errorf("expected 'rx_packets' key in metrics, got %v", metrics)
	}
}
