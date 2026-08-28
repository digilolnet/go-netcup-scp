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

func TestListTasks(t *testing.T) {
	uuid := "task-abc"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tasks" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.TaskInfoMinimal{{Uuid: &uuid}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	tasks, err := client.ListTasks(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	if tasks[0].Uuid == nil || *tasks[0].Uuid != "task-abc" {
		t.Errorf("unexpected task UUID: %v", tasks[0].Uuid)
	}
}

func TestGetTask(t *testing.T) {
	uuid := "task-abc"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tasks/task-abc" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.GetTask(context.Background(), "task-abc")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if task.Uuid == nil || *task.Uuid != "task-abc" {
		t.Errorf("unexpected task UUID: %v", task.Uuid)
	}
}

func TestCancelTask(t *testing.T) {
	uuid := "task-abc"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tasks/task-abc:cancel" && r.Method == http.MethodPut {
			writeJSON(w, http.StatusAccepted, generated.TaskInfo{Uuid: &uuid})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	task, err := client.CancelTask(context.Background(), "task-abc")
	if err != nil {
		t.Fatalf("CancelTask() error = %v", err)
	}

	if task == nil || task.Uuid == nil || *task.Uuid != "task-abc" {
		t.Errorf("unexpected task: %v", task)
	}
}
