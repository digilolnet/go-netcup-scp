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

// IsoImage is a standard ISO image offered by netcup.
type IsoImage = generated.IsoImage

// S3Object is an uploaded user file (ISO or image) in netcup's object store.
type S3Object = generated.S3Object

// ListAvailableISOs retrieves all available ISO images for a server.
func (c *Client) ListAvailableISOs(ctx context.Context, serverID int32) ([]IsoImage, error) {
	resp, err := c.api.GetApiV1ServersServerIdIsoimagesWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("list available isos: %w", err)
	}

	return pickBodyVal("list available isos", resp, resp.JSON200, resp.HALJSON200, 200)
}

// GetAttachedISO retrieves information about the currently attached ISO for a server.
// Returns (nil, nil) if no ISO is attached (the API answers 200 with
// {"iso": null, "isoAttached": false}).
func (c *Client) GetAttachedISO(ctx context.Context, serverID int32) (*generated.Iso, error) {
	resp, err := c.api.GetApiV1ServersServerIdIsoWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("get attached iso: %w", err)
	}

	iso, err := pickBody("get attached iso", resp, resp.JSON200, resp.HALJSON200, 200)
	if err != nil || !deref(iso.IsoAttached) {
		return nil, err
	}
	return iso, nil
}

// AttachISOOptions configures the AttachISO operation.
type AttachISOOptions struct {
	// IsoID is the ID of the ISO image to attach (from ListAvailableISOs)
	IsoID *int32
	// UserIsoName is the name of a user-uploaded ISO to attach
	UserIsoName *string
	// ChangeBootDeviceToCdrom changes the boot device to CDROM after attaching
	ChangeBootDeviceToCdrom *bool
}

// AttachISO attaches an ISO image to a server.
// Either IsoID or UserIsoName must be provided in opts.
// Returns a TaskInfo when the API responds with 202 (async operation started), or nil
// when it responds synchronously. Use the task UUID with GetTask / WaitForTask to track completion.
func (c *Client) AttachISO(ctx context.Context, serverID int32, opts *AttachISOOptions) (*TaskInfo, error) {
	if opts == nil {
		return nil, fmt.Errorf("attach iso: options cannot be nil")
	}

	if opts.IsoID == nil && opts.UserIsoName == nil {
		return nil, fmt.Errorf("attach iso: either IsoID or UserIsoName must be provided")
	}

	body := generated.ServerAttachIso{
		IsoId:                   opts.IsoID,
		UserIsoName:             opts.UserIsoName,
		ChangeBootDeviceToCdrom: opts.ChangeBootDeviceToCdrom,
	}

	resp, err := c.api.PostApiV1ServersServerIdIsoWithResponse(ctx, serverID, body)
	if err != nil {
		return nil, fmt.Errorf("attach iso: %w", err)
	}

	if err := checkResponse(resp, 200, 201, 202); err != nil {
		return nil, fmt.Errorf("attach iso: %w", err)
	}

	if resp.JSON202 != nil {
		return resp.JSON202, nil
	}
	if resp.HALJSON202 != nil {
		return resp.HALJSON202, nil
	}
	return nil, nil
}

// DetachISO detaches the currently attached ISO from a server.
func (c *Client) DetachISO(ctx context.Context, serverID int32) error {
	resp, err := c.api.DeleteApiV1ServersServerIdIsoWithResponse(ctx, serverID)
	if err != nil {
		return fmt.Errorf("detach iso: %w", err)
	}

	if err := checkResponse(resp, 200, 204); err != nil {
		return fmt.Errorf("detach iso: %w", err)
	}

	return nil
}

// ListUserISOs retrieves all user-uploaded ISOs.
func (c *Client) ListUserISOs(ctx context.Context, userID int32) ([]S3Object, error) {
	resp, err := c.api.GetApiV1UsersUserIdIsosWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user isos: %w", err)
	}

	return pickBodyVal("list user isos", resp, resp.JSON200, resp.HALJSON200, 200)
}

// GetUserISO retrieves information about a specific user-uploaded ISO.
func (c *Client) GetUserISO(ctx context.Context, userID int32, key string) (*generated.S3DownloadInfos, error) {
	resp, err := c.api.GetApiV1UsersUserIdIsosKeyWithResponse(ctx, userID, key)
	if err != nil {
		return nil, fmt.Errorf("get user iso: %w", err)
	}

	return pickBody("get user iso", resp, resp.JSON200, resp.HALJSON200, 200)
}

// DeleteUserISO deletes a user-uploaded ISO.
func (c *Client) DeleteUserISO(ctx context.Context, userID int32, key string) error {
	resp, err := c.api.DeleteApiV1UsersUserIdIsosKeyWithResponse(ctx, userID, key)
	if err != nil {
		return fmt.Errorf("delete user iso: %w", err)
	}

	if err := checkResponse(resp, 200, 204); err != nil {
		return fmt.Errorf("delete user iso: %w", err)
	}

	return nil
}

// setContentTypeJSON is a request editor that sets Content-Type: application/json.
// The SCP API requires this header on upload-initiation POSTs even though the
// request body is empty.
func setContentTypeJSON(_ context.Context, req *http.Request) error {
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// InitiateISOUpload initiates a simple (non-multipart) ISO upload.
// Returns a presigned URL that you can upload the ISO file to using HTTP PUT.
func (c *Client) InitiateISOUpload(ctx context.Context, userID int32, key string) (string, error) {
	multipart := false
	params := &generated.PostApiV1UsersUserIdIsosKeyParams{
		Multipart: &multipart,
	}

	resp, err := c.api.PostApiV1UsersUserIdIsosKeyWithResponse(ctx, userID, key, params, setContentTypeJSON)
	if err != nil {
		return "", fmt.Errorf("initiate iso upload: %w", err)
	}

	if err := checkResponse(resp, 201); err != nil {
		return "", fmt.Errorf("initiate iso upload: %w", err)
	}

	if resp.JSON201 == nil || resp.JSON201.PresignedUrl == nil {
		return "", fmt.Errorf("initiate iso upload: missing presigned URL in response")
	}

	return *resp.JSON201.PresignedUrl, nil
}

// InitiateMultipartISOUpload initiates a multipart ISO upload.
// Returns an upload ID that should be used for uploading parts and completing the upload.
func (c *Client) InitiateMultipartISOUpload(ctx context.Context, userID int32, key string) (string, error) {
	multipart := true
	params := &generated.PostApiV1UsersUserIdIsosKeyParams{
		Multipart: &multipart,
	}

	resp, err := c.api.PostApiV1UsersUserIdIsosKeyWithResponse(ctx, userID, key, params, setContentTypeJSON)
	if err != nil {
		return "", fmt.Errorf("initiate multipart iso upload: %w", err)
	}

	if err := checkResponse(resp, 201); err != nil {
		return "", fmt.Errorf("initiate multipart iso upload: %w", err)
	}

	if resp.JSON201 == nil || resp.JSON201.UploadId == nil {
		return "", fmt.Errorf("initiate multipart iso upload: missing upload ID in response")
	}

	return *resp.JSON201.UploadId, nil
}

// GetISOUploadPartURL retrieves a presigned URL for uploading a specific part.
// Part numbers start at 1.
func (c *Client) GetISOUploadPartURL(
	ctx context.Context,
	userID int32,
	key string,
	uploadID string,
	partNumber int32,
) (string, error) {
	resp, err := c.api.GetApiV1UsersUserIdIsosKeyUploadIdPartsPartNumberWithResponse(
		ctx,
		userID,
		key,
		uploadID,
		partNumber,
	)
	if err != nil {
		return "", fmt.Errorf("get iso upload part url: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return "", fmt.Errorf("get iso upload part url: %w", err)
	}

	if resp.JSON200 == nil || resp.JSON200.Url == nil {
		return "", fmt.Errorf("get iso upload part url: missing URL in response")
	}

	return *resp.JSON200.Url, nil
}

// CompleteMultipartISOUpload completes a multipart upload with the list of uploaded parts.
// The parts slice should contain all uploaded parts with their ETags and part numbers.
func (c *Client) CompleteMultipartISOUpload(
	ctx context.Context,
	userID int32,
	key string,
	uploadID string,
	parts []generated.S3CompletedPart,
) error {
	resp, err := c.api.PutApiV1UsersUserIdIsosKeyUploadIdWithResponse(ctx, userID, key, uploadID, parts)
	if err != nil {
		return fmt.Errorf("complete multipart iso upload: %w", err)
	}

	if err := checkResponse(resp, 200, 204); err != nil {
		return fmt.Errorf("complete multipart iso upload: %w", err)
	}

	return nil
}

// UploadISOMultipart uploads an ISO using S3 multipart upload.
// partSize is the size of each part in bytes (S3 minimum is 5 MiB; use 50+ MiB in practice).
// totalSize is the file size and is used only for progress reporting (pass 0 if unknown).
// progress is called after each part completes and may be nil.
func (c *Client) UploadISOMultipart(
	ctx context.Context,
	userID int32,
	key string,
	r io.Reader,
	totalSize,
	partSize int64,
	progress UploadProgress,
) error {
	uploadID, err := c.InitiateMultipartISOUpload(ctx, userID, key)
	if err != nil {
		return err
	}

	getPartURL := func(ctx context.Context, partNum int32) (string, error) {
		return c.GetISOUploadPartURL(ctx, userID, key, uploadID, partNum)
	}
	complete := func(ctx context.Context, parts []generated.S3CompletedPart) error {
		return c.CompleteMultipartISOUpload(ctx, userID, key, uploadID, parts)
	}

	return c.multipartUpload(ctx, r, totalSize, partSize, getPartURL, complete, progress)
}

// UploadISO uploads an ISO file using the simple (non-multipart) method.
// This is a convenience function that initiates the upload and performs the upload in one call.
// contentLength must be set to the exact file size; S3 presigned PUT URLs require a known
// Content-Length and reject requests that use chunked transfer encoding (HTTP 501).
func (c *Client) UploadISO(ctx context.Context, userID int32, key string, reader io.Reader, contentLength int64) error {
	presignedURL, err := c.InitiateISOUpload(ctx, userID, key)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, reader)
	if err != nil {
		return fmt.Errorf("upload iso: %w", err)
	}
	req.ContentLength = contentLength

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload iso: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("upload iso: unexpected status code: %d", httpResp.StatusCode)
	}

	return nil
}
