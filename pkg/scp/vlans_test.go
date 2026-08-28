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

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestListVLans(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/vlans" && r.Method == http.MethodGet {
			vlanID := int32(10)
			name := "test-vlan"
			writeJSON(w, http.StatusOK, []generated.VLan{{VlanId: &vlanID, Name: &name}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	vlans, err := client.ListVLans(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("ListVLans() error = %v", err)
	}

	if len(vlans) != 1 {
		t.Fatalf("expected 1 vlan, got %d", len(vlans))
	}

	if vlans[0].Name == nil || *vlans[0].Name != "test-vlan" {
		t.Errorf("unexpected VLAN name: %v", vlans[0].Name)
	}
}

func TestGetVLan(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/vlans/10" && r.Method == http.MethodGet {
			vlanID := int32(10)
			name := "test-vlan"
			writeJSON(w, http.StatusOK, generated.VLan{VlanId: &vlanID, Name: &name})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	vlan, err := client.GetVLan(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("GetVLan() error = %v", err)
	}

	if vlan.VlanId == nil || *vlan.VlanId != 10 {
		t.Errorf("unexpected VLAN ID: %v", vlan.VlanId)
	}
}

func TestGetVLanByID(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/vlans/10" && r.Method == http.MethodGet {
			vlanID := int32(10)
			name := "test-vlan"
			writeJSON(w, http.StatusOK, generated.VLan{VlanId: &vlanID, Name: &name})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	vlan, err := client.GetVLanByID(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetVLanByID() error = %v", err)
	}

	if vlan.Name == nil || *vlan.Name != "test-vlan" {
		t.Errorf("unexpected VLAN name: %v", vlan.Name)
	}
}

func TestUpdateVLan(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/vlans/10" && r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.UpdateVLan(context.Background(), 1, 10, "new-name"); err != nil {
		t.Errorf("UpdateVLan() error = %v", err)
	}
}
