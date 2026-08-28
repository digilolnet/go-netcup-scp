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
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// confirmInput is where confirmation answers are read from, and
// confirmInteractive reports whether prompting is possible at all.
// Variables so tests can drive the prompts without a real terminal.
var (
	confirmInput       io.Reader = os.Stdin
	confirmInteractive           = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
)

// readLine reads one line of user input.
func readLine() (string, error) {
	line, err := bufio.NewReader(confirmInput).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read confirmation: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// confirm guards a destructive action behind a y/N prompt. --force skips it;
// without a terminal on stdin it refuses rather than guess. The prompt goes
// to stderr so stdout stays clean for -j output.
func confirm(action string) error {
	if rootFlags.force {
		return nil
	}
	if !confirmInteractive() {
		return fmt.Errorf("%q requires confirmation; re-run with --force", action)
	}
	fmt.Fprintf(os.Stderr, "Are you sure you want to %s? (y/N) ", action)
	answer, err := readLine()
	if err != nil {
		return err
	}
	switch strings.ToLower(answer) {
	case "y", "yes":
		return nil
	default:
		return fmt.Errorf("aborted")
	}
}

// confirmRetype guards a catastrophic action: the user must type the server's
// nickname (or name when it has none) back before it proceeds. --force skips.
func confirmRetype(cc *cmdContext, serverID int32, action string) error {
	if rootFlags.force {
		return nil
	}
	if !confirmInteractive() {
		return fmt.Errorf("%q requires confirmation; re-run with --force", action)
	}
	label := serverLabelByID(cc, serverID)
	fmt.Fprintf(os.Stderr, "This will %s.\nType %q to confirm (or re-run with --force): ", action, label)
	answer, err := readLine()
	if err != nil {
		return err
	}
	if answer != label {
		return fmt.Errorf("confirmation %q does not match %q; aborted", answer, label)
	}
	return nil
}

// serverLabelByID returns the server's nickname (or name) for prompts,
// falling back to the id when the server cannot be found.
func serverLabelByID(cc *cmdContext, id int32) string {
	servers, cached, err := cc.serverRefs()
	if err != nil {
		return strconv.Itoa(int(id))
	}
	if label, ok := findServerLabel(servers, id); ok {
		return label
	}
	if cached {
		if servers, err = cc.fetchServerRefs(); err == nil {
			if label, ok := findServerLabel(servers, id); ok {
				return label
			}
		}
	}
	return strconv.Itoa(int(id))
}

func findServerLabel(servers []serverRef, id int32) (string, bool) {
	for _, s := range servers {
		if s.ID != id {
			continue
		}
		if s.Nickname != "" {
			return s.Nickname, true
		}
		if s.Name != "" {
			return s.Name, true
		}
		return strconv.Itoa(int(id)), true
	}
	return "", false
}
