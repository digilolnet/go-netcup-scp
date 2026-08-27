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

package scp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// traceBodyLimit caps recorded bodies so ISO/image uploads don't end up on disk.
const traceBodyLimit = 1 << 20

// WithTraceDir records every API exchange (method, path, status, content type,
// request/response bodies) as one JSON file per call in dir. Authorization
// headers are never recorded and bodies above 1 MiB are elided. Intended for
// debugging and for capturing real API responses as test fixtures.
func WithTraceDir(dir string) ClientOption {
	return func(c *Client) {
		c.traceDir = dir
	}
}

// traceExchange is the on-disk schema of one recorded API call.
type traceExchange struct {
	Method       string          `json:"method"`
	URL          string          `json:"url"`
	Status       int             `json:"status"`
	ContentType  string          `json:"contentType,omitempty"`
	RequestBody  json.RawMessage `json:"requestBody,omitempty"`
	ResponseBody json.RawMessage `json:"responseBody,omitempty"`
	Note         string          `json:"note,omitempty"`
}

// traceTransport wraps another RoundTripper and writes one file per exchange.
type traceTransport struct {
	base http.RoundTripper
	dir  string
	seq  atomic.Int64
}

func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ex := traceExchange{Method: req.Method, URL: req.URL.String()}

	if req.Body != nil && req.ContentLength >= 0 && req.ContentLength <= traceBodyLimit {
		body, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		ex.RequestBody = asRawJSON(body)
	} else if req.Body != nil {
		ex.Note = "request body elided (streaming or >1MiB)"
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		ex.Note = strings.TrimSpace(ex.Note + " transport error: " + err.Error())
		t.write(&ex)
		return resp, err
	}

	ex.Status = resp.StatusCode
	ex.ContentType = resp.Header.Get("Content-Type")
	if resp.Body != nil {
		body, rerr := io.ReadAll(io.LimitReader(resp.Body, traceBodyLimit+1))
		resp.Body.Close()
		if rerr != nil {
			return nil, rerr
		}
		if len(body) > traceBodyLimit {
			ex.Note = "response body truncated at 1MiB"
			body = body[:traceBodyLimit]
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		ex.ResponseBody = asRawJSON(body)
	}

	t.write(&ex)
	return resp, nil
}

// asRawJSON returns b as-is when it is valid JSON, else as a JSON string so
// the fixture file stays parseable.
func asRawJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	quoted, _ := json.Marshal(string(b))
	return json.RawMessage(quoted)
}

func (t *traceTransport) write(ex *traceExchange) {
	seq := t.seq.Add(1)
	name := fmt.Sprintf("%04d_%s_%s.json", seq, ex.Method, sanitizePath(ex.URL))
	data, err := json.MarshalIndent(ex, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(t.dir, name), data, 0o644)
}

// sanitizePath turns a URL into a short filesystem-safe fixture name.
func sanitizePath(rawURL string) string {
	s := rawURL
	if i := strings.Index(s, "/api/v1/"); i >= 0 {
		s = s[i+len("/api/v1/"):]
	} else if i := strings.Index(s, "://"); i >= 0 {
		if j := strings.Index(s[i+3:], "/"); j >= 0 {
			s = s[i+3+j+1:]
		}
	}
	if i := strings.IndexByte(s, '?'); i >= 0 {
		s = s[:i]
	}
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, s)
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}
