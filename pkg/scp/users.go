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

// UserSave is the request/response body for UpdateUser.
type UserSave = generated.UserSave

// GetUserLogsOptions configures the GetUserLogs operation.
type GetUserLogsOptions struct {
	// Limit sets the maximum number of log entries to return.
	Limit *int32
	// Offset sets the starting position for pagination.
	Offset *int32
}

// GetUser retrieves account information for the specified user.
func (c *Client) GetUser(ctx context.Context, userID int32) (*generated.User, error) {
	resp, err := c.api.GetApiV1UsersUserIdWithResponse(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return pickBody("get user", resp, resp.JSON200, resp.HALJSON200, 200)
}

// UpdateUser updates account settings for the specified user.
// Returns the updated user save data.
func (c *Client) UpdateUser(ctx context.Context, userID int32, body UserSave) (*UserSave, error) {
	resp, err := c.api.PutApiV1UsersUserIdWithResponse(ctx, userID, body)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return pickBody("update user", resp, resp.JSON200, resp.HALJSON200, 200)
}

// GetUserLogs retrieves the audit log for the specified user.
func (c *Client) GetUserLogs(ctx context.Context, userID int32, opts *GetUserLogsOptions) ([]generated.Log, error) {
	params := &generated.GetApiV1UsersUserIdLogsParams{}
	if opts != nil {
		params.Limit = opts.Limit
		params.Offset = opts.Offset
	}

	resp, err := c.api.GetApiV1UsersUserIdLogsWithResponse(ctx, userID, params)
	if err != nil {
		return nil, fmt.Errorf("get user logs: %w", err)
	}

	return pickBodyVal("get user logs", resp, resp.JSON200, resp.HALJSON200, 200)
}
