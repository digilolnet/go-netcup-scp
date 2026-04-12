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

// ServerImageSetup is the request body for InstallImage.
type ServerImageSetup = generated.ServerImageSetup

// ServerUserImageSetup is the request body for InstallUserImage.
type ServerUserImageSetup = generated.ServerUserImageSetup

// ListServersOptions contains optional parameters for listing servers.
type ListServersOptions struct {
	// Ip filters servers by IP address.
	Ip *string
	// Limit sets the maximum number of servers to return.
	Limit *int32
	// Name filters servers by name.
	Name *string
	// Offset sets the starting position for pagination.
	Offset *int32
	// Q searches within name, nickname, or IPv4 addresses (case-insensitive).
	Q *string
}

// GetServerOptions contains optional parameters for getting a server.
type GetServerOptions struct {
	// LoadServerLiveInfo controls whether to include live server information.
	// If nil, the API default is used.
	LoadServerLiveInfo *bool
}

// ListServers retrieves a list of all servers.
func (c *Client) ListServers(ctx context.Context, opts *ListServersOptions) ([]generated.ServerListMinimal, error) {
	params := &generated.GetApiV1ServersParams{}
	if opts != nil {
		params.Ip = opts.Ip
		params.Limit = opts.Limit
		params.Name = opts.Name
		params.Offset = opts.Offset
		params.Q = opts.Q
	}

	resp, err := c.api.GetApiV1ServersWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list servers: empty response")
	}

	return *resp.JSON200, nil
}

// GetServer retrieves detailed information about a specific server.
func (c *Client) GetServer(ctx context.Context, serverID int32, opts *GetServerOptions) (*generated.Server, error) {
	params := &generated.GetApiV1ServersServerIdParams{}
	if opts != nil {
		params.LoadServerLiveInfo = opts.LoadServerLiveInfo
	}

	resp, err := c.api.GetApiV1ServersServerIdWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get server: empty response")
	}

	return resp.JSON200, nil
}

// StartServer powers on the server.
// Returns immediately; server may take time to fully boot.
func (c *Client) StartServer(ctx context.Context, serverID int32) error {
	state := generated.ServerState1ON
	patch := &generated.ServerStatePatch{State: &state}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdWithBodyWithResponse(
		ctx,
		serverID,
		nil,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

// StopServer shuts down the server.
// If powerOff is true, performs hard power off; otherwise attempts graceful shutdown.
// Returns immediately; server may take time to fully stop.
func (c *Client) StopServer(ctx context.Context, serverID int32, powerOff bool) error {
	state := generated.ServerState1OFF
	patch := &generated.ServerStatePatch{State: &state}
	params := &generated.PatchApiV1ServersServerIdParams{}

	if powerOff {
		stateOption := "POWEROFF"
		params.StateOption = &stateOption
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdWithBodyWithResponse(
		ctx,
		serverID,
		params,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	return nil
}

// RestartServer reboots the server.
// If reset is true, performs a hard reset; otherwise performs a power cycle.
// Returns immediately; server may take time to restart.
func (c *Client) RestartServer(ctx context.Context, serverID int32, reset bool) error {
	state := generated.ServerState1ON
	patch := &generated.ServerStatePatch{State: &state}

	stateOption := "POWERCYCLE"
	if reset {
		stateOption = "RESET"
	}
	params := &generated.PatchApiV1ServersServerIdParams{
		StateOption: &stateOption,
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("restart server: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdWithBodyWithResponse(
		ctx,
		serverID,
		params,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("restart server: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return fmt.Errorf("restart server: %w", err)
	}

	return nil
}

// SetAutostart configures whether the server starts automatically.
func (c *Client) SetAutostart(ctx context.Context, serverID int32, enabled bool) error {
	patch := &generated.ServerAutostartPatch{Autostart: &enabled}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("set autostart: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdWithBodyWithResponse(
		ctx,
		serverID,
		nil,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("set autostart: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return fmt.Errorf("set autostart: %w", err)
	}

	return nil
}

// SetUEFI configures whether the server uses UEFI boot mode.
func (c *Client) SetUEFI(ctx context.Context, serverID int32, enabled bool) error {
	patch := &generated.ServerUEFIPatch{Uefi: &enabled}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("set uefi: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdWithBodyWithResponse(
		ctx,
		serverID,
		nil,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("set uefi: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return fmt.Errorf("set uefi: %w", err)
	}

	return nil
}

// UpdateNickname sets a custom nickname for the server.
func (c *Client) UpdateNickname(ctx context.Context, serverID int32, nickname string) error {
	patch := &generated.ServerNicknamePatch{Nickname: &nickname}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("update nickname: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdWithBodyWithResponse(
		ctx,
		serverID,
		nil,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("update nickname: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return fmt.Errorf("update nickname: %w", err)
	}

	return nil
}

// GetServerLogsOptions configures the GetServerLogs operation.
type GetServerLogsOptions struct {
	// Limit sets the maximum number of log entries to return.
	Limit *int32
	// Offset sets the starting position for pagination.
	Offset *int32
}

// OptimizeStorageOptions configures the OptimizeStorage operation.
type OptimizeStorageOptions struct {
	// Disks restricts optimization to specific disk names; nil means all disks.
	Disks *[]string
	// StartAfterOptimization starts the server after the optimization completes.
	StartAfterOptimization *bool
}

// GetGuestAgent retrieves guest agent information from the server.
// The guest agent must be running inside the VM for this to return data.
func (c *Client) GetGuestAgent(ctx context.Context, serverID int32) (*generated.GuestAgentData, error) {
	resp, err := c.api.GetApiV1ServersServerIdGuestAgentWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("get guest agent: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get guest agent: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get guest agent: empty response")
	}

	return resp.JSON200, nil
}

// GetServerLogs retrieves the event log for a server.
func (c *Client) GetServerLogs(ctx context.Context, serverID int32, opts *GetServerLogsOptions) ([]generated.Log, error) {
	params := &generated.GetApiV1ServersServerIdLogsParams{}
	if opts != nil {
		params.Limit = opts.Limit
		params.Offset = opts.Offset
	}

	resp, err := c.api.GetApiV1ServersServerIdLogsWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get server logs: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get server logs: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get server logs: empty response")
	}

	return *resp.JSON200, nil
}

// ListImageFlavours retrieves the available OS image flavours for a server.
func (c *Client) ListImageFlavours(ctx context.Context, serverID int32) ([]generated.ImageFlavour, error) {
	resp, err := c.api.GetApiV1ServersServerIdImageflavoursWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("list image flavours: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list image flavours: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list image flavours: empty response")
	}

	return *resp.JSON200, nil
}

// InstallImage reinstalls the server with the specified OS image configuration.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) InstallImage(ctx context.Context, serverID int32, setup ServerImageSetup) (*TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdImageWithResponse(ctx, serverID, setup)
	if err != nil {
		return nil, fmt.Errorf("install image: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("install image: %w", err)
	}

	return resp.JSON202, nil
}

// InstallUserImage installs a user-uploaded image onto the server.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) InstallUserImage(ctx context.Context, serverID int32, setup ServerUserImageSetup) (*TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdUserImageWithResponse(ctx, serverID, setup)
	if err != nil {
		return nil, fmt.Errorf("install user image: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("install user image: %w", err)
	}

	return resp.JSON202, nil
}

// OptimizeStorage runs storage optimization on a server's disks.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) OptimizeStorage(ctx context.Context, serverID int32, opts *OptimizeStorageOptions) (*TaskInfo, error) {
	params := &generated.PostApiV1ServersServerIdStorageoptimizationParams{}
	if opts != nil {
		params.Disks = opts.Disks
		params.StartAfterOptimization = opts.StartAfterOptimization
	}

	resp, err := c.api.PostApiV1ServersServerIdStorageoptimizationWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("optimize storage: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("optimize storage: %w", err)
	}

	return resp.JSON202, nil
}

// SetCPUTopology configures the CPU topology (sockets and cores per socket).
func (c *Client) SetCPUTopology(ctx context.Context, serverID int32, sockets, cores int32) error {
	topology := &generated.CpuTopology{
		SocketCount:         &sockets,
		CoresPerSocketCount: &cores,
	}
	patch := &generated.ServerCpuTopologyPatch{CpuTopology: topology}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("set cpu topology: %w", err)
	}

	resp, err := c.api.PatchApiV1ServersServerIdWithBodyWithResponse(
		ctx,
		serverID,
		nil,
		"application/merge-patch+json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("set cpu topology: %w", err)
	}

	if err := checkResponse(resp, 200, 202); err != nil {
		return fmt.Errorf("set cpu topology: %w", err)
	}

	return nil
}

// GetGPUDriver retrieves a presigned S3 download URL for the GPU driver for a server.
func (c *Client) GetGPUDriver(ctx context.Context, serverID int32) (*generated.S3DownloadInfos, error) {
	resp, err := c.api.GetApiV1ServersServerIdGpuDriverWithResponse(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("get gpu driver: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get gpu driver: %w", err)
	}

	if resp.HALJSON200 != nil {
		return resp.HALJSON200, nil
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, fmt.Errorf("get gpu driver: empty response")
}
