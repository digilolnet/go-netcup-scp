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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// partSrv is a test S3-like server that records uploaded part bodies and
// returns a fake ETag per part.
type partSrv struct {
	parts []string // bodies received, in order
}

func (s *partSrv) handler() http.HandlerFunc {
	var mu int32
	return func(w http.ResponseWriter, r *http.Request) {
		_ = atomic.AddInt32(&mu, 1)
		body, _ := io.ReadAll(r.Body)
		s.parts = append(s.parts, string(body))
		w.Header().Set("ETag", `"etag"`)
		w.WriteHeader(http.StatusOK)
	}
}

func TestMultipartUpload_SplitsCorrectly(t *testing.T) {
	srv := &partSrv{}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	client := &Client{httpClient: ts.Client()}

	data := strings.Repeat("A", 7) // 7 bytes, partSize=3 → parts: 3,3,1
	partSize := int64(3)

	var progressCalls []int
	getPartURL := func(_ context.Context, partNum int32) (string, error) {
		return ts.URL, nil
	}
	complete := func(_ context.Context, parts []generated.S3CompletedPart) error {
		return nil
	}
	progress := func(partNum int, done, total int64) {
		progressCalls = append(progressCalls, partNum)
	}

	err := client.multipartUpload(context.Background(), strings.NewReader(data), int64(len(data)), partSize, getPartURL, complete, progress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(srv.parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(srv.parts))
	}
	if srv.parts[0] != "AAA" || srv.parts[1] != "AAA" || srv.parts[2] != "A" {
		t.Errorf("unexpected part bodies: %v", srv.parts)
	}
	if len(progressCalls) != 3 {
		t.Errorf("expected 3 progress calls, got %d", len(progressCalls))
	}
}

func TestMultipartUpload_ExactMultiple(t *testing.T) {
	srv := &partSrv{}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	client := &Client{httpClient: ts.Client()}
	data := strings.Repeat("B", 6) // 6 bytes, partSize=3 → parts: 3,3

	err := client.multipartUpload(
		context.Background(),
		strings.NewReader(data), int64(len(data)), 3,
		func(_ context.Context, _ int32) (string, error) { return ts.URL, nil },
		func(_ context.Context, _ []generated.S3CompletedPart) error { return nil },
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(srv.parts) != 2 {
		t.Errorf("expected 2 parts, got %d: %v", len(srv.parts), srv.parts)
	}
}

func TestMultipartUpload_SinglePart(t *testing.T) {
	srv := &partSrv{}
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	client := &Client{httpClient: ts.Client()}
	data := "hello"

	err := client.multipartUpload(
		context.Background(),
		strings.NewReader(data), int64(len(data)), 100,
		func(_ context.Context, _ int32) (string, error) { return ts.URL, nil },
		func(_ context.Context, _ []generated.S3CompletedPart) error { return nil },
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(srv.parts) != 1 || srv.parts[0] != data {
		t.Errorf("unexpected parts: %v", srv.parts)
	}
}

func TestMultipartUpload_ETags(t *testing.T) {
	partNum := 0
	etags := []string{`"etag-1"`, `"etag-2"`}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Header().Set("ETag", etags[partNum])
		partNum++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &Client{httpClient: ts.Client()}

	var completedParts []generated.S3CompletedPart
	err := client.multipartUpload(
		context.Background(),
		bytes.NewReader(make([]byte, 6)), 6, 3,
		func(_ context.Context, _ int32) (string, error) { return ts.URL, nil },
		func(_ context.Context, parts []generated.S3CompletedPart) error {
			completedParts = parts
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(completedParts) != 2 {
		t.Fatalf("expected 2 completed parts, got %d", len(completedParts))
	}
	if *completedParts[0].ETag != `"etag-1"` || *completedParts[1].ETag != `"etag-2"` {
		t.Errorf("unexpected ETags: %v %v", *completedParts[0].ETag, *completedParts[1].ETag)
	}
}

func TestMultipartUpload_EmptyInput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &Client{httpClient: ts.Client()}
	err := client.multipartUpload(
		context.Background(),
		strings.NewReader(""), 0, 100,
		func(_ context.Context, _ int32) (string, error) { return ts.URL, nil },
		func(_ context.Context, _ []generated.S3CompletedPart) error { return nil },
		nil,
	)
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}
