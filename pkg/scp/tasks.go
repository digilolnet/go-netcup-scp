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

// ListTasksOptions configures the ListTasks operation.
type ListTasksOptions struct {
	// Limit sets the maximum number of tasks to return.
	Limit *int32
	// Offset sets the starting position for pagination.
	Offset *int32
	// Q searches tasks by name, UUID, server name, server nickname, or server UUID (case-insensitive).
	Q *string
	// ServerId filters tasks by server ID.
	ServerId *int32
	// State filters tasks by state (ROLLBACK is not supported).
	State *generated.TaskState
}

// ListTasks retrieves the list of async tasks.
func (c *Client) ListTasks(ctx context.Context, opts *ListTasksOptions) ([]generated.TaskInfoMinimal, error) {
	params := &generated.GetApiV1TasksParams{}
	if opts != nil {
		params.Limit = opts.Limit
		params.Offset = opts.Offset
		params.Q = opts.Q
		params.ServerId = opts.ServerId
		params.State = opts.State
	}

	resp, err := c.api.GetApiV1TasksWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list tasks: empty response")
	}

	return *resp.JSON200, nil
}

// GetTask retrieves detailed information about a specific async task.
func (c *Client) GetTask(ctx context.Context, uuid string) (*generated.TaskInfo, error) {
	resp, err := c.api.GetApiV1TasksUuidWithResponse(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get task: empty response")
	}

	return resp.JSON200, nil
}

// CancelTask cancels a running async task.
// Returns the updated task info; the task may still be in progress if cancellation is pending.
func (c *Client) CancelTask(ctx context.Context, uuid string) (*generated.TaskInfo, error) {
	resp, err := c.api.PutApiV1TasksUuidCancelWithResponse(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("cancel task: %w", err)
	}

	if err := checkResponse(resp, 202, 204); err != nil {
		return nil, fmt.Errorf("cancel task: %w", err)
	}

	if resp.JSON202 != nil {
		return resp.JSON202, nil
	}

	return resp.JSON204, nil
}
