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
	"fmt"
	"io"
	"net/http"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// ListUserImages retrieves all user-uploaded disk images.
func (c *Client) ListUserImages(ctx context.Context, userID int32) ([]generated.S3Object, error) {
	resp, err := c.api.GetApiV1UsersUserIdImagesWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user images: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list user images: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list user images: empty response")
	}

	return *resp.JSON200, nil
}

// GetUserImage retrieves download information for a specific user-uploaded image.
func (c *Client) GetUserImage(ctx context.Context, userID int32, key string) (*generated.S3DownloadInfos, error) {
	resp, err := c.api.GetApiV1UsersUserIdImagesKeyWithResponse(ctx, userID, key)
	if err != nil {
		return nil, fmt.Errorf("get user image: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get user image: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get user image: empty response")
	}

	return resp.JSON200, nil
}

// DeleteUserImage deletes a user-uploaded disk image.
func (c *Client) DeleteUserImage(ctx context.Context, userID int32, key string) error {
	resp, err := c.api.DeleteApiV1UsersUserIdImagesKeyWithResponse(ctx, userID, key)
	if err != nil {
		return fmt.Errorf("delete user image: %w", err)
	}

	if err := checkResponse(resp, 204); err != nil {
		return fmt.Errorf("delete user image: %w", err)
	}

	return nil
}

// InitiateImageUpload initiates a simple (non-multipart) image upload.
// Returns a presigned URL that you can upload the image file to using HTTP PUT.
func (c *Client) InitiateImageUpload(ctx context.Context, userID int32, key string) (string, error) {
	multipart := false
	params := &generated.PostApiV1UsersUserIdImagesKeyParams{
		Multipart: &multipart,
	}

	resp, err := c.api.PostApiV1UsersUserIdImagesKeyWithResponse(ctx, userID, key, params)
	if err != nil {
		return "", fmt.Errorf("initiate image upload: %w", err)
	}

	if err := checkResponse(resp, 201); err != nil {
		return "", fmt.Errorf("initiate image upload: %w", err)
	}

	if resp.JSON201 == nil || resp.JSON201.PresignedUrl == nil {
		return "", fmt.Errorf("initiate image upload: missing presigned URL in response")
	}

	return *resp.JSON201.PresignedUrl, nil
}

// InitiateMultipartImageUpload initiates a multipart image upload.
// Returns an upload ID that should be used for uploading parts and completing the upload.
func (c *Client) InitiateMultipartImageUpload(ctx context.Context, userID int32, key string) (string, error) {
	multipart := true
	params := &generated.PostApiV1UsersUserIdImagesKeyParams{
		Multipart: &multipart,
	}

	resp, err := c.api.PostApiV1UsersUserIdImagesKeyWithResponse(ctx, userID, key, params)
	if err != nil {
		return "", fmt.Errorf("initiate multipart image upload: %w", err)
	}

	if err := checkResponse(resp, 201); err != nil {
		return "", fmt.Errorf("initiate multipart image upload: %w", err)
	}

	if resp.JSON201 == nil || resp.JSON201.UploadId == nil {
		return "", fmt.Errorf("initiate multipart image upload: missing upload ID in response")
	}

	return *resp.JSON201.UploadId, nil
}

// GetImageUploadPartURL retrieves a presigned URL for uploading a specific part of an image.
// Part numbers start at 1.
func (c *Client) GetImageUploadPartURL(ctx context.Context, userID int32, key, uploadID string, partNumber int32) (string, error) {
	resp, err := c.api.GetApiV1UsersUserIdImagesKeyUploadIdPartsPartNumberWithResponse(ctx, userID, key, uploadID, partNumber)
	if err != nil {
		return "", fmt.Errorf("get image upload part url: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return "", fmt.Errorf("get image upload part url: %w", err)
	}

	if resp.JSON200 == nil || resp.JSON200.Url == nil {
		return "", fmt.Errorf("get image upload part url: missing URL in response")
	}

	return *resp.JSON200.Url, nil
}

// CompleteMultipartImageUpload completes a multipart upload with the list of uploaded parts.
// The parts slice should contain all uploaded parts with their ETags and part numbers.
func (c *Client) CompleteMultipartImageUpload(ctx context.Context, userID int32, key, uploadID string, parts []generated.S3CompletedPart) error {
	resp, err := c.api.PutApiV1UsersUserIdImagesKeyUploadIdWithResponse(ctx, userID, key, uploadID, parts)
	if err != nil {
		return fmt.Errorf("complete multipart image upload: %w", err)
	}

	if err := checkResponse(resp, 204); err != nil {
		return fmt.Errorf("complete multipart image upload: %w", err)
	}

	return nil
}

// UploadImage uploads an image file using the simple (non-multipart) method.
// This is a convenience function that initiates the upload and performs the HTTP PUT in one call.
func (c *Client) UploadImage(ctx context.Context, userID int32, key string, reader io.Reader) error {
	presignedURL, err := c.InitiateImageUpload(ctx, userID, key)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, reader)
	if err != nil {
		return fmt.Errorf("upload image: %w", err)
	}

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload image: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("upload image: unexpected status code: %d", httpResp.StatusCode)
	}

	return nil
}
