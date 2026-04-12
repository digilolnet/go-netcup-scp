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
	"fmt"
	"io"
	"net/http"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// UploadProgress is called after each successfully uploaded part.
// partNum is 1-based. done is cumulative bytes uploaded; total is the file
// size (0 when unknown).
type UploadProgress func(partNum int, done, total int64)

// multipartUpload performs an S3 multipart upload, reading from r in chunks of
// partSize bytes. For each chunk it fetches a presigned URL via getPartURL,
// PUTs the data, records the ETag from the response, then calls complete with
// all collected parts.
func (c *Client) multipartUpload(
	ctx context.Context,
	r io.Reader,
	totalSize, partSize int64,
	getPartURL func(ctx context.Context, partNum int32) (string, error),
	complete func(ctx context.Context, parts []generated.S3CompletedPart) error,
	progress UploadProgress,
) error {
	var (
		parts   []generated.S3CompletedPart
		partNum int32 = 1
		done    int64
	)

	buf := make([]byte, partSize)
	for {
		n, readErr := io.ReadFull(r, buf)
		if n == 0 {
			break // clean EOF before reading anything
		}

		partURL, err := getPartURL(ctx, partNum)
		if err != nil {
			return err
		}

		// bytes.NewReader gives http a known Content-Length, required by S3.
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, partURL, bytes.NewReader(buf[:n]))
		if err != nil {
			return fmt.Errorf("part %d: create request: %w", partNum, err)
		}
		req.ContentLength = int64(n)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("part %d: %w", partNum, err)
		}
		etag := resp.Header.Get("ETag")
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("part %d: unexpected status %d", partNum, resp.StatusCode)
		}

		n32 := partNum
		parts = append(parts, generated.S3CompletedPart{
			ETag:       &etag,
			PartNumber: &n32,
		})

		done += int64(n)
		if progress != nil {
			progress(int(partNum), done, totalSize)
		}

		if readErr != nil {
			// io.ErrUnexpectedEOF = final partial chunk; stop looping.
			break
		}
		partNum++
	}

	if len(parts) == 0 {
		return fmt.Errorf("no data to upload")
	}

	return complete(ctx, parts)
}
