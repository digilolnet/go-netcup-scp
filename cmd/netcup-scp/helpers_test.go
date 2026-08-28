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

package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// shrinkWaitTimings speeds up waitTask's poll loop for tests.
func shrinkWaitTimings(t *testing.T) {
	t.Helper()
	oldPoll, oldTick := taskPollInterval, spinnerTick
	taskPollInterval, spinnerTick = 5*time.Millisecond, time.Millisecond
	t.Cleanup(func() { taskPollInterval, spinnerTick = oldPoll, oldTick })
}

// taskServer serves GET /api/v1/tasks/<uuid>, walking through states on
// successive polls and sticking on the last one.
func taskServer(uuid string, states ...string) (http.Handler, *int) {
	polls := new(int)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks/"+uuid, func(w http.ResponseWriter, _ *http.Request) {
		state := states[min(*polls, len(states)-1)]
		*polls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"uuid":%q,"state":%q,"message":"boom"}`, uuid, state)
	})
	return mux, polls
}

func TestWaitTaskFinishes(t *testing.T) {
	shrinkWaitTimings(t)
	handler, polls := taskServer("u-1", "PENDING", "RUNNING", "FINISHED")
	cc := newTestContext(t, handler)

	if err := waitTask(cc, "u-1"); err != nil {
		t.Fatalf("waitTask: %v", err)
	}
	if *polls != 3 {
		t.Errorf("polled %d times, want 3", *polls)
	}
}

func TestWaitTaskError(t *testing.T) {
	shrinkWaitTimings(t)
	handler, _ := taskServer("u-2", "RUNNING", "ERROR")
	cc := newTestContext(t, handler)

	err := waitTask(cc, "u-2")
	if err == nil || !strings.Contains(err.Error(), "task failed: boom") {
		t.Fatalf("waitTask on ERROR: got %v", err)
	}
}

func TestWaitTaskCanceled(t *testing.T) {
	shrinkWaitTimings(t)
	handler, _ := taskServer("u-3", "CANCELED")
	cc := newTestContext(t, handler)

	err := waitTask(cc, "u-3")
	if err == nil || !strings.Contains(err.Error(), "task canceled") {
		t.Fatalf("waitTask on CANCELED: got %v", err)
	}
}

func TestWaitTaskTimeout(t *testing.T) {
	shrinkWaitTimings(t)
	rootFlags.waitTimeout = 25 * time.Millisecond
	defer func() { rootFlags.waitTimeout = 0 }()

	handler, _ := taskServer("u-4", "RUNNING")
	cc := newTestContext(t, handler)

	err := waitTask(cc, "u-4")
	if err == nil ||
		!strings.Contains(err.Error(), "timed out") ||
		!strings.Contains(err.Error(), "tasks get u-4") {
		t.Fatalf("waitTask timeout: got %v", err)
	}
}
