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

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

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

// taskPollInterval is how often --wait polls the task state.
const taskPollInterval = 2 * time.Second

// defaultWaitTimeout bounds --wait polling; generous because OS installs and
// snapshot exports legitimately take many minutes.
const defaultWaitTimeout = 30 * time.Minute

// registerWaitFlags adds the --wait/--timeout pair to a command that returns
// an async task.
func registerWaitFlags(cmd *cobra.Command, wait *bool) {
	cmd.Flags().BoolVar(wait, "wait", false, "wait for task to complete")
	cmd.Flags().DurationVar(
		&rootFlags.waitTimeout,
		"timeout",
		defaultWaitTimeout,
		"give up waiting after this long (with --wait; 0 = no limit)",
	)
}

// waitTask polls a task by UUID until it reaches a terminal state or
// --timeout elapses. On a terminal it shows a spinner with progress and
// elapsed time on stderr (stdout stays clean); otherwise it prints a single
// line and stays quiet until done.
func waitTask(cc *cmdContext, uuid string) error {
	start := time.Now()
	interactive := !cc.jsonOut && term.IsTerminal(int(os.Stderr.Fd()))
	if !interactive {
		fmt.Fprintf(os.Stderr, "Waiting for task %s\n", uuid)
	}

	frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	frame := 0
	progress := ""
	render := func() {
		if !interactive {
			return
		}
		elapsed := time.Since(start).Round(time.Second)
		fmt.Fprintf(
			os.Stderr,
			"\r\033[K%c Waiting for task %s  %s%s",
			frames[frame%len(frames)],
			uuid,
			progress,
			elapsed,
		)
		frame++
	}
	clearLine := func() {
		if interactive {
			fmt.Fprint(os.Stderr, "\r\033[K")
		}
	}

	for {
		// Animate the spinner while the next poll interval elapses.
		for end := time.Now().Add(taskPollInterval); time.Now().Before(end); {
			render()
			time.Sleep(100 * time.Millisecond)
		}

		task, err := cc.client.GetTask(cc.ctx, uuid)
		if err != nil {
			clearLine()
			return err
		}
		switch derefStr((*string)(task.State)) {
		case string(scp.TaskStateFINISHED):
			clearLine()
			fmt.Fprintf(os.Stderr, "Task %s finished after %s\n", uuid, time.Since(start).Round(time.Second))
			return nil
		case string(scp.TaskStateERROR):
			clearLine()
			return fmt.Errorf("task failed: %s", derefStr(task.Message))
		case string(scp.TaskStateCANCELED):
			clearLine()
			return fmt.Errorf("task canceled")
		default:
			if t := rootFlags.waitTimeout; t > 0 && time.Since(start) > t {
				clearLine()
				return fmt.Errorf(
					"timed out after %s waiting for task %s (the task keeps running; check 'tasks get %s')",
					t, uuid, uuid,
				)
			}
			progress = ""
			if p := task.TaskProgress; p != nil && p.ProgressInPercent != nil { //nolint:staticcheck
				progress = fmt.Sprintf("%.0f%%  ", *p.ProgressInPercent)
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
