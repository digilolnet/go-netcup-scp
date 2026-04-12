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
