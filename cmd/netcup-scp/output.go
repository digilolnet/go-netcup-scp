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
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
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

// col is one column of a resource table: its --format name, rendered header,
// and row extractor.
type col[T any] struct {
	name   string
	header string
	value  func(T) any
}

// column builds a col with T inferred from the extractor.
func column[T any](name, header string, value func(T) any) col[T] {
	return col[T]{name: name, header: header, value: value}
}

// displayer renders a list of resources honoring --format, --no-header and
// -q/--quiet. The first column doubles as the --quiet value.
type displayer[T any] struct {
	cols []col[T]
}

func newDisplayer[T any](cols ...col[T]) *displayer[T] {
	return &displayer[T]{cols: cols}
}

func (d *displayer[T]) columnNames() []string {
	names := make([]string, len(d.cols))
	for i, c := range d.cols {
		names[i] = c.name
	}
	return names
}

// addFlags registers --format, --no-header and -q/--quiet on cmd; the
// --format help lists this resource's columns.
func (d *displayer[T]) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&rootFlags.format,
		"format",
		"",
		"comma-separated columns to show: "+strings.Join(d.columnNames(), ", "),
	)
	cmd.Flags().BoolVar(&rootFlags.noHeader, "no-header", false, "plain tab-separated rows without header or borders")
	cmd.Flags().BoolVarP(&rootFlags.quiet, "quiet", "q", false, "print only the "+d.cols[0].name+" column")
	_ = cmd.RegisterFlagCompletionFunc(
		"format",
		cobra.FixedCompletions(d.columnNames(), cobra.ShellCompDirectiveNoFileComp),
	)
}

// selectCols resolves --format to an ordered column subset (all when unset).
func (d *displayer[T]) selectCols() ([]col[T], error) {
	if rootFlags.format == "" {
		return d.cols, nil
	}
	names := splitCSV(rootFlags.format)
	cols := make([]col[T], 0, len(names))
	for _, name := range names {
		found := false
		for _, c := range d.cols {
			if strings.EqualFold(c.name, name) {
				cols = append(cols, c)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf(
				"unknown column %q (valid: %s)", name, strings.Join(d.columnNames(), ", "),
			)
		}
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("--format selected no columns (valid: %s)", strings.Join(d.columnNames(), ", "))
	}
	return cols, nil
}

// print renders items: JSON with -j (shape unchanged), first-column values
// with -q, plain tab-separated rows with --no-header, else the usual table.
func (d *displayer[T]) print(cc *cmdContext, items []T) error {
	if cc.jsonOut {
		return printJSON(items)
	}
	return d.render(os.Stdout, items)
}

func (d *displayer[T]) render(w io.Writer, items []T) error {
	if rootFlags.quiet {
		for _, it := range items {
			fmt.Fprintf(w, "%v\n", d.cols[0].value(it))
		}
		return nil
	}

	cols, err := d.selectCols()
	if err != nil {
		return err
	}

	if rootFlags.noHeader {
		for _, it := range items {
			parts := make([]string, len(cols))
			for i, c := range cols {
				parts[i] = fmt.Sprintf("%v", c.value(it))
			}
			fmt.Fprintln(w, strings.Join(parts, "\t"))
		}
		return nil
	}

	headers := make(table.Row, len(cols))
	for i, c := range cols {
		headers[i] = c.header
	}
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.AppendHeader(headers)
	style := table.StyleLight
	style.Format.Header = text.FormatDefault // preserve header case as written
	t.SetStyle(style)
	for _, it := range items {
		row := make(table.Row, len(cols))
		for i, c := range cols {
			row[i] = c.value(it)
		}
		t.AppendRow(row)
	}
	t.Render()
	return nil
}

// printKV prints aligned "key: value" lines; callers pre-deref pointers.
func printKV(pairs ...any) {
	for i := 0; i+1 < len(pairs); i += 2 {
		fmt.Printf("%-24s %v\n", fmt.Sprintf("%v:", pairs[i]), pairs[i+1])
	}
}
