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
	"io"
	"os"
)

// progressReader wraps an io.Reader and calls onRead after every chunk,
// reporting cumulative bytes read. Used to drive progress display during
// simple (single-PUT) uploads.
type progressReader struct {
	r      io.Reader
	done   int64
	total  int64
	onRead func(done, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.done += int64(n)
		p.onRead(p.done, p.total)
	}
	return n, err
}

// printUploadProgress writes a single-line progress update to stderr using
// carriage-return to overwrite the previous line in a terminal.
func printUploadProgress(done, total int64) {
	if total > 0 {
		pct := float64(done) / float64(total) * 100
		fmt.Fprintf(os.Stderr, "\r  %s / %s (%.0f%%)",
			formatBytes(done), formatBytes(total), pct)
	} else {
		fmt.Fprintf(os.Stderr, "\r  %s", formatBytes(done))
	}
}

// printMultipartProgress writes a progress line that also shows the part number.
func printMultipartProgress(partNum int, done, total int64) {
	if total > 0 {
		pct := float64(done) / float64(total) * 100
		fmt.Fprintf(os.Stderr, "\r  part %d — %s / %s (%.0f%%)",
			partNum, formatBytes(done), formatBytes(total), pct)
	} else {
		fmt.Fprintf(os.Stderr, "\r  part %d — %s", partNum, formatBytes(done))
	}
}

// formatBytes returns a human-readable byte count (e.g. "42.3 MiB").
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
