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
	"encoding/json"
	"fmt"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// StorageDriver is the disk storage driver type.
type StorageDriver = generated.StorageDriver

// ListDisks retrieves all disks attached to a server.
func (c *Client) ListDisks(ctx context.Context, serverID int32) ([]generated.Disk, error) {
	resp, err := c.api.GetApiV1ServersServerIdDisksWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("list disks: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list disks: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list disks: empty response")
	}

	return *resp.JSON200, nil
}

// GetDisk retrieves information about a specific disk.
func (c *Client) GetDisk(ctx context.Context, serverID int32, diskName string) (*generated.Disk, error) {
	resp, err := c.api.GetApiV1ServersServerIdDisksDiskNameWithResponse(ctx, serverID, diskName)
	if err != nil {
		return nil, fmt.Errorf("get disk: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get disk: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get disk: empty response")
	}

	return resp.JSON200, nil
}

// FormatDisk formats a disk, destroying all data on it.
// This operation cannot be undone.
// Returns a TaskInfo when the API responds with 202 (async), or nil for 200.
func (c *Client) FormatDisk(ctx context.Context, serverID int32, diskName string) (*TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdDisksDiskNameFormatWithResponse(ctx, serverID, diskName, setContentTypeJSON)
	if err != nil {
		return nil, fmt.Errorf("format disk: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return nil, fmt.Errorf("format disk: %w", err)
	}

	if resp.JSON202 != nil {
		return resp.JSON202, nil
	}
	return resp.HALJSON202, nil
}

// GetSupportedDiskDrivers retrieves the list of storage drivers supported by a server.
func (c *Client) GetSupportedDiskDrivers(ctx context.Context, serverID int32) ([]generated.StorageDriver, error) {
	resp, err := c.api.GetApiV1ServersServerIdDisksSupportedDriversWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("get supported disk drivers: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get supported disk drivers: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get supported disk drivers: empty response")
	}

	return *resp.JSON200, nil
}

// SetDiskDriver changes the storage driver for all disks on a server.
// Returns a TaskInfo when the API responds with 202 (async), or nil for 200.
func (c *Client) SetDiskDriver(ctx context.Context, serverID int32, driver StorageDriver) (*TaskInfo, error) {
	patch := &generated.EditDisksDriver{Driver: driver}

	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("set disk driver: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdDisksWithBodyWithResponse(
		ctx,
		serverID,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("set disk driver: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return nil, fmt.Errorf("set disk driver: %w", err)
	}

	if resp.JSON202 != nil {
		return resp.JSON202, nil
	}
	return resp.HALJSON202, nil
}
