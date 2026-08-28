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

func TestListFirewallPolicies(t *testing.T) {
	id := int32(1)
	name := "allow-ssh"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/firewall-policies" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.FirewallPolicy{{Id: &id, Name: &name}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	policies, err := client.ListFirewallPolicies(context.Background(), 1, nil)
	if err != nil {
		t.Fatalf("ListFirewallPolicies() error = %v", err)
	}

	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}

	if policies[0].Name == nil || *policies[0].Name != "allow-ssh" {
		t.Errorf("unexpected policy name: %v", policies[0].Name)
	}
}

func TestCreateFirewallPolicy(t *testing.T) {
	id := int32(2)
	name := "new-policy"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/firewall-policies" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusCreated, generated.FirewallPolicy{Id: &id, Name: &name})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	policy, err := client.CreateFirewallPolicy(context.Background(), 1, generated.FirewallPolicySave{Name: "new-policy"})
	if err != nil {
		t.Fatalf("CreateFirewallPolicy() error = %v", err)
	}

	if policy.Name == nil || *policy.Name != "new-policy" {
		t.Errorf("unexpected policy name: %v", policy.Name)
	}
}

func TestGetFirewallPolicy(t *testing.T) {
	id := int32(1)
	name := "allow-ssh"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/firewall-policies/1" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, generated.FirewallPolicy{Id: &id, Name: &name})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	policy, err := client.GetFirewallPolicy(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("GetFirewallPolicy() error = %v", err)
	}

	if policy.Name == nil || *policy.Name != "allow-ssh" {
		t.Errorf("unexpected policy name: %v", policy.Name)
	}
}

func TestUpdateFirewallPolicy(t *testing.T) {
	uuid := "fw-policy-task-1"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/firewall-policies/1" && r.Method == http.MethodPut {
			writeJSON(w, http.StatusAccepted, generated.FirewallPolicyUpdateResult{
				TaskInfo: &generated.TaskInfo{Uuid: &uuid},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	result, err := client.UpdateFirewallPolicy(context.Background(), 1, 1, generated.FirewallPolicySave{Name: "updated"})
	if err != nil {
		t.Fatalf("UpdateFirewallPolicy() error = %v", err)
	}

	if result == nil || result.TaskInfo == nil || result.TaskInfo.Uuid == nil || *result.TaskInfo.Uuid != "fw-policy-task-1" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestDeleteFirewallPolicy(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/firewall-policies/1" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteFirewallPolicy(context.Background(), 1, 1); err != nil {
		t.Errorf("DeleteFirewallPolicy() error = %v", err)
	}
}
