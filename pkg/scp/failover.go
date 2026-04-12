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

// ListFailoverIPsOptions configures ListFailoverIPv4 and ListFailoverIPv6.
type ListFailoverIPsOptions struct {
	// Ip filters failover IPs by address.
	Ip *string
	// ServerId filters failover IPs by the server they are routed to.
	ServerId *int32
}

// ListFailoverIPv4 retrieves all IPv4 failover addresses for a user.
func (c *Client) ListFailoverIPv4(ctx context.Context, userID int32, opts *ListFailoverIPsOptions) ([]generated.FailoverIPv4, error) {
	params := &generated.GetApiV1UsersUserIdFailoveripsV4Params{}
	if opts != nil {
		params.Ip = opts.Ip
		params.ServerId = opts.ServerId
	}

	resp, err := c.api.GetApiV1UsersUserIdFailoveripsV4WithResponse(ctx, userID, params)
	if err != nil {
		return nil, fmt.Errorf("list failover ipv4: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list failover ipv4: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list failover ipv4: empty response")
	}

	return *resp.JSON200, nil
}

// RouteFailoverIPv4 routes a failover IPv4 address to a different server.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) RouteFailoverIPv4(ctx context.Context, userID, failoverID, serverID int32) (*generated.TaskInfo, error) {
	body := generated.RouteFailoverIp{ServerId: &serverID}

	resp, err := c.api.PatchApiV1UsersUserIdFailoveripsV4IdWithResponse(ctx, userID, failoverID, body)
	if err != nil {
		return nil, fmt.Errorf("route failover ipv4: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("route failover ipv4: %w", err)
	}

	return resp.JSON202, nil
}

// UnrouteFailoverIPv4 removes the routing of a failover IPv4 address.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) UnrouteFailoverIPv4(ctx context.Context, userID, failoverID int32) (*generated.TaskInfo, error) {
	body := generated.RouteFailoverIp{} // omit serverId to unroute

	resp, err := c.api.PatchApiV1UsersUserIdFailoveripsV4IdWithResponse(ctx, userID, failoverID, body)
	if err != nil {
		return nil, fmt.Errorf("unroute failover ipv4: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("unroute failover ipv4: %w", err)
	}

	return resp.JSON202, nil
}

// ListFailoverIPv6 retrieves all IPv6 failover prefixes for a user.
func (c *Client) ListFailoverIPv6(ctx context.Context, userID int32, opts *ListFailoverIPsOptions) ([]generated.FailoverIPv6, error) {
	params := &generated.GetApiV1UsersUserIdFailoveripsV6Params{}
	if opts != nil {
		params.Ip = opts.Ip
		params.ServerId = opts.ServerId
	}

	resp, err := c.api.GetApiV1UsersUserIdFailoveripsV6WithResponse(ctx, userID, params)
	if err != nil {
		return nil, fmt.Errorf("list failover ipv6: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list failover ipv6: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list failover ipv6: empty response")
	}

	return *resp.JSON200, nil
}

// RouteFailoverIPv6 routes a failover IPv6 prefix to a different server.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) RouteFailoverIPv6(ctx context.Context, userID, failoverID, serverID int32) (*generated.TaskInfo, error) {
	body := generated.RouteFailoverIp{ServerId: &serverID}

	resp, err := c.api.PatchApiV1UsersUserIdFailoveripsV6IdWithResponse(ctx, userID, failoverID, body)
	if err != nil {
		return nil, fmt.Errorf("route failover ipv6: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("route failover ipv6: %w", err)
	}

	return resp.JSON202, nil
}

// UnrouteFailoverIPv6 removes the routing of a failover IPv6 prefix.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) UnrouteFailoverIPv6(ctx context.Context, userID, failoverID int32) (*generated.TaskInfo, error) {
	body := generated.RouteFailoverIp{} // omit serverId to unroute

	resp, err := c.api.PatchApiV1UsersUserIdFailoveripsV6IdWithResponse(ctx, userID, failoverID, body)
	if err != nil {
		return nil, fmt.Errorf("unroute failover ipv6: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("unroute failover ipv6: %w", err)
	}

	return resp.JSON202, nil
}
