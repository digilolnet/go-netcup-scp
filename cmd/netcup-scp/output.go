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
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// printJSON writes v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// printOK prints a success indicator. In JSON mode: {"status":"ok"}.
func printOK(cc *cmdContext) {
	if cc.jsonOut {
		_ = printJSON(map[string]string{"status": "ok"})
	}
}

// printResult prints v as JSON when --json is set, otherwise calls textFn.
func printResult(cc *cmdContext, v any, textFn func()) error {
	if cc.jsonOut {
		return printJSON(v)
	}
	textFn()
	return nil
}

// deref returns *p if p != nil, else zero value.
func deref[T any](p *T) T {
	if p == nil {
		var z T
		return z
	}
	return *p
}

// derefStr returns *p if non-nil, else "".
func derefStr(p *string) string { return deref(p) }

// derefInt32 returns *p if non-nil, else 0.
func derefInt32(p *int32) int32 { return deref(p) }

// newTable creates a table writer with a consistent light-border style.
// Call t.AppendRow(...) then t.Render() to produce output.
func newTable(headers ...any) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row(headers))
	style := table.StyleLight
	style.Format.Header = text.FormatDefault // preserve header case as written
	t.SetStyle(style)
	return t
}

// printKV prints "key: value\n" lines. Values with nil pointers are printed as "-".
func printKV(pairs ...any) {
	for i := 0; i+1 < len(pairs); i += 2 {
		fmt.Printf("%-24s %v\n", fmt.Sprintf("%v:", pairs[i]), pairs[i+1])
	}
}
