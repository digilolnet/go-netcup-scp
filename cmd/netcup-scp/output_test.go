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

type testRow struct {
	ID   int32
	Name string
}

var testDisplayer = newDisplayer(
	column("id", "ID", func(r testRow) any { return r.ID }),
	column("name", "NAME", func(r testRow) any { return r.Name }),
)

var testRows = []testRow{{1, "alpha"}, {2, "beta"}}

func resetOutputFlags() {
	rootFlags.format = ""
	rootFlags.noHeader = false
	rootFlags.quiet = false
}

func TestDisplayerTable(t *testing.T) {
	defer resetOutputFlags()
	var b strings.Builder
	if err := testDisplayer.render(&b, testRows); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"ID", "NAME", "alpha", "beta"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}

func TestDisplayerFormat(t *testing.T) {
	defer resetOutputFlags()
	rootFlags.format = "name"
	rootFlags.noHeader = true
	var b strings.Builder
	if err := testDisplayer.render(&b, testRows); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != "alpha\nbeta\n" {
		t.Errorf("format=name no-header: got %q", got)
	}

	// Column order follows --format, not the displayer.
	rootFlags.format = "name,id"
	b.Reset()
	if err := testDisplayer.render(&b, testRows); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != "alpha\t1\nbeta\t2\n" {
		t.Errorf("format=name,id no-header: got %q", got)
	}
}

func TestDisplayerUnknownColumn(t *testing.T) {
	defer resetOutputFlags()
	rootFlags.format = "bogus"
	var b strings.Builder
	err := testDisplayer.render(&b, testRows)
	if err == nil || !strings.Contains(err.Error(), "id, name") {
		t.Errorf("unknown column: got %v, want error listing valid columns", err)
	}
}

func TestDisplayerQuiet(t *testing.T) {
	defer resetOutputFlags()
	rootFlags.quiet = true
	var b strings.Builder
	if err := testDisplayer.render(&b, testRows); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != "1\n2\n" {
		t.Errorf("quiet: got %q", got)
	}
}
