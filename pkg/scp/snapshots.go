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

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// CreateSnapshot creates a snapshot of a server.
// The snapshot can be created online (while server is running) or offline.
// Returns a TaskInfo when the API responds with 202 (async), or nil for 200/201.
func (c *Client) CreateSnapshot(ctx context.Context, serverID int32, name, description string) (*TaskInfo, error) {
	onlineSnapshot := true
	resp, err := c.api.PostApiV1ServersServerIdSnapshotsWithResponse(
		ctx,
		serverID,
		generated.ServerSnapshotCreate{
			Name:           name,
			Description:    &description,
			OnlineSnapshot: &onlineSnapshot,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}

	if err := checkResponse(resp, 200, 201, 202); err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}

	if resp.JSON202 != nil {
		return resp.JSON202, nil
	}
	return resp.HALJSON202, nil
}

// ListSnapshots retrieves all snapshots for a server.
func (c *Client) ListSnapshots(ctx context.Context, serverID int32) ([]generated.SnapshotMinimal, error) {
	resp, err := c.api.GetApiV1ServersServerIdSnapshotsWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list snapshots: empty response")
	}

	return *resp.JSON200, nil
}

// GetSnapshot retrieves information about a specific snapshot.
func (c *Client) GetSnapshot(ctx context.Context, serverID int32, name string) (*generated.Snapshot, error) {
	resp, err := c.api.GetApiV1ServersServerIdSnapshotsNameWithResponse(ctx, serverID, name)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get snapshot: empty response")
	}

	return resp.JSON200, nil
}

// DeleteSnapshot deletes a snapshot.
// This operation cannot be undone.
// Returns a TaskInfo when the API responds with 202 (async), or nil for 200/204.
func (c *Client) DeleteSnapshot(ctx context.Context, serverID int32, name string) (*TaskInfo, error) {
	resp, err := c.api.DeleteApiV1ServersServerIdSnapshotsNameWithResponse(ctx, serverID, name)
	if err != nil {
		return nil, fmt.Errorf("delete snapshot: %w", err)
	}

	if err := checkResponse(resp, 200, 202, 204); err != nil {
		return nil, fmt.Errorf("delete snapshot: %w", err)
	}

	if resp.JSON202 != nil {
		return resp.JSON202, nil
	}
	return resp.HALJSON202, nil
}

// RevertSnapshot reverts a server to a previous snapshot state.
// This operation will revert the server to the state it was in when the snapshot was created.
// Returns a TaskInfo when the API responds with 202 (async), or nil for 200.
func (c *Client) RevertSnapshot(ctx context.Context, serverID int32, name string) (*TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdSnapshotsNameRevertWithResponse(ctx, serverID, name)
	if err != nil {
		return nil, fmt.Errorf("revert snapshot: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return nil, fmt.Errorf("revert snapshot: %w", err)
	}

	if resp.JSON202 != nil {
		return resp.JSON202, nil
	}
	return resp.HALJSON202, nil
}

// DryRunSnapshot checks whether a snapshot can be created without actually creating one.
// Returns a list of blocking errors; an empty slice means the snapshot can be created.
func (c *Client) DryRunSnapshot(ctx context.Context, serverID int32, onlineSnapshot bool, diskName *string) ([]generated.ResponseError, error) {
	body := generated.ServerSnapshotCreateCheck{
		OnlineSnapshot: &onlineSnapshot,
		DiskName:       diskName,
	}

	resp, err := c.api.PostApiV1ServersServerIdSnapshotsDryrunWithResponse(ctx, serverID, body)
	if err != nil {
		return nil, fmt.Errorf("dry run snapshot: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("dry run snapshot: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dry run snapshot: empty response")
	}

	return *resp.JSON200, nil
}

// ExportSnapshot exports a snapshot.
// The operation is asynchronous; check the returned task for completion.
func (c *Client) ExportSnapshot(ctx context.Context, serverID int32, name string) (*generated.TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdSnapshotsNameExportWithResponse(ctx, serverID, name)
	if err != nil {
		return nil, fmt.Errorf("export snapshot: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("export snapshot: %w", err)
	}

	return resp.JSON202, nil
}
