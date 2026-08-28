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
