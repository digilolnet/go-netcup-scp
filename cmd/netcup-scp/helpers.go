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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func ptr[T any](v T) *T { return &v }

// fmtTime formats a *time.Time as "2006-01-02 15:04:05" (UTC), or "" if nil.
func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}

// parseID parses a string as int32 for use as an API identifier.
func parseID(s, name string) (int32, error) {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: must be an integer", name, s)
	}
	return int32(n), nil
}

// parseBool accepts true/false, on/off, yes/no, 1/0.
func parseBool(s string) (bool, error) {
	switch s {
	case "true", "on", "yes", "1", "enable", "enabled":
		return true, nil
	case "false", "off", "no", "0", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q: use true/false or on/off", s)
	}
}

// readJSONStdin reads all of stdin and unmarshals it into v.
func readJSONStdin(v any) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse JSON from stdin: %w", err)
	}
	return nil
}

// waitTask polls a task by UUID until it reaches a terminal state, printing progress to stderr.
func waitTask(cc *cmdContext, uuid string) error {
	fmt.Fprintf(os.Stderr, "Waiting for task %s", uuid)
	for {
		time.Sleep(2 * time.Second)
		task, err := cc.client.GetTask(cc.ctx, uuid)
		if err != nil {
			fmt.Fprintln(os.Stderr)
			return err
		}
		switch derefStr((*string)(task.State)) {
		case string(scp.TaskStateFINISHED):
			fmt.Fprintln(os.Stderr, " done")
			return nil
		case string(scp.TaskStateERROR):
			fmt.Fprintln(os.Stderr, " failed")
			return fmt.Errorf("task failed: %s", derefStr(task.Message))
		case string(scp.TaskStateCANCELED):
			fmt.Fprintln(os.Stderr, " canceled")
			return fmt.Errorf("task canceled")
		default:
			if p := task.TaskProgress; p != nil && p.ProgressInPercent != nil { //nolint:staticcheck
				fmt.Fprintf(os.Stderr, "\rWaiting for task %s  %.0f%%", uuid, *p.ProgressInPercent)
			} else {
				fmt.Fprint(os.Stderr, ".")
			}
		}
	}
}

// printTaskAndWait prints a task result and optionally waits for completion.
func printTaskAndWait(cc *cmdContext, task *scp.TaskInfo, wait bool) error {
	if err := printResult(cc, task, func() {
		if task != nil {
			printKV("Task UUID", derefStr(task.Uuid), "State", derefStr((*string)(task.State)))
		}
	}); err != nil {
		return err
	}
	if wait && task != nil && task.Uuid != nil {
		return waitTask(cc, *task.Uuid)
	}
	return nil
}

// splitCSV splits a comma-separated string into a trimmed slice, filtering empty entries.
func splitCSV(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
