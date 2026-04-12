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

// GetRescueSystem retrieves the rescue system status for a server.
func (c *Client) GetRescueSystem(ctx context.Context, serverID int32) (*generated.RescueSystemStatus, error) {
	resp, err := c.api.GetApiV1ServersServerIdRescuesystemWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("get rescue system: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get rescue system: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get rescue system: empty response")
	}

	return resp.JSON200, nil
}

// ActivateRescueSystem boots the server into rescue mode.
// The operation is asynchronous; use the returned task to track progress.
// The response includes a one-time password for the rescue system.
func (c *Client) ActivateRescueSystem(ctx context.Context, serverID int32) (*generated.TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdRescuesystemWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("activate rescue system: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("activate rescue system: %w", err)
	}

	return resp.JSON202, nil
}

// DeactivateRescueSystem exits rescue mode and boots the server normally.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) DeactivateRescueSystem(ctx context.Context, serverID int32) (*generated.TaskInfo, error) {
	resp, err := c.api.DeleteApiV1ServersServerIdRescuesystemWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("deactivate rescue system: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("deactivate rescue system: %w", err)
	}

	return resp.JSON202, nil
}
