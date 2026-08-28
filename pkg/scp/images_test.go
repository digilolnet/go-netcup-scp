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
	"context"
	"net/http"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

func TestListUserImages(t *testing.T) {
	key := "ubuntu.img"
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/images" && r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, []generated.S3Object{{Key: &key}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	images, err := client.ListUserImages(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListUserImages() error = %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	if images[0].Key == nil || *images[0].Key != "ubuntu.img" {
		t.Errorf("unexpected key: %v", images[0].Key)
	}
}

func TestDeleteUserImage(t *testing.T) {
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/images/ubuntu.img" && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	if err := client.DeleteUserImage(context.Background(), 1, "ubuntu.img"); err != nil {
		t.Errorf("DeleteUserImage() error = %v", err)
	}
}

func TestInitiateImageUpload(t *testing.T) {
	presignedURL := "https://s3.example.com/upload?sig=abc"
	uploadID := ""
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/users/1/images/ubuntu.img" && r.Method == http.MethodPost {
			writeJSON(w, http.StatusCreated, generated.S3Upload{PresignedUrl: &presignedURL, UploadId: &uploadID})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	url, err := client.InitiateImageUpload(context.Background(), 1, "ubuntu.img")
	if err != nil {
		t.Fatalf("InitiateImageUpload() error = %v", err)
	}

	if url != presignedURL {
		t.Errorf("expected %q, got %q", presignedURL, url)
	}
}
