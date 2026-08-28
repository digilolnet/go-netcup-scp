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
	"strings"
	"testing"
)

func TestConfirmForceBypasses(t *testing.T) {
	rootFlags.force = true
	defer func() { rootFlags.force = false }()

	if err := confirm("delete everything"); err != nil {
		t.Errorf("confirm with --force: %v", err)
	}
	// --force must short-circuit before cc is touched.
	if err := confirmRetype(nil, 1, "format the disk"); err != nil {
		t.Errorf("confirmRetype with --force: %v", err)
	}
}

func TestConfirmNonInteractiveRefuses(t *testing.T) {
	// Test stdin is not a terminal, so confirm must refuse, not hang.
	err := confirm("delete everything")
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("non-interactive confirm = %v, want --force hint", err)
	}
	err = confirmRetype(nil, 1, "format the disk")
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Errorf("non-interactive confirmRetype = %v, want --force hint", err)
	}
}

// withPromptInput fakes an interactive terminal whose user types input.
func withPromptInput(t *testing.T, input string) {
	t.Helper()
	oldIn, oldInteractive := confirmInput, confirmInteractive
	confirmInput = strings.NewReader(input)
	confirmInteractive = func() bool { return true }
	t.Cleanup(func() { confirmInput, confirmInteractive = oldIn, oldInteractive })
}

func TestConfirmAnswers(t *testing.T) {
	tests := []struct {
		answer string
		ok     bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"Y\n", true},
		{"n\n", false},
		{"\n", false},
		{"nah\n", false},
	}
	for _, tc := range tests {
		t.Run(strings.TrimSpace(tc.answer), func(t *testing.T) {
			withPromptInput(t, tc.answer)
			err := confirm("delete everything")
			if tc.ok && err != nil {
				t.Errorf("answer %q: got %v, want accepted", tc.answer, err)
			}
			if !tc.ok && err == nil {
				t.Errorf("answer %q: accepted, want abort", tc.answer)
			}
		})
	}
}

func TestConfirmRetypeAnswers(t *testing.T) {
	// Seed the completion cache so serverLabelByID resolves offline.
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cc := &cmdContext{tokenFile: "retype-test-token"}
	newCompletionCache(cc.tokenFile).set("servers", []string{
		encodeServerRef(serverRef{ID: 7, Name: "v9007", Nickname: "web-prod"}),
	})

	withPromptInput(t, "web-prod\n")
	if err := confirmRetype(cc, 7, "format the disk"); err != nil {
		t.Errorf("correct retype: %v", err)
	}

	withPromptInput(t, "web-prod-oops\n")
	err := confirmRetype(cc, 7, "format the disk")
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("wrong retype: got %v, want mismatch abort", err)
	}
}

func TestFindServerLabel(t *testing.T) {
	if label, ok := findServerLabel(testServers, 1); !ok || label != "web-prod" {
		t.Errorf("id 1: got %q, %v; want nickname web-prod", label, ok)
	}
	if label, ok := findServerLabel(testServers, 3); !ok || label != "v9003" {
		t.Errorf("id 3: got %q, %v; want name fallback v9003", label, ok)
	}
	if _, ok := findServerLabel(testServers, 99); ok {
		t.Error("id 99: want not found")
	}
}
