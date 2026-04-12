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

// GetFirewallOptions configures the GetFirewall operation.
type GetFirewallOptions struct {
	// ConsistencyCheck verifies that the firewall rules have been applied, setting the
	// Consistent field on the result to true or false.
	ConsistencyCheck *bool
}

// GetFirewall retrieves the firewall configuration for a network interface.
func (c *Client) GetFirewall(ctx context.Context, serverID int32, mac string, opts *GetFirewallOptions) (*generated.ServerFirewall, error) {
	params := &generated.GetApiV1ServersServerIdInterfacesMacFirewallParams{}
	if opts != nil {
		params.ConsistencyCheck = opts.ConsistencyCheck
	}

	resp, err := c.api.GetApiV1ServersServerIdInterfacesMacFirewallWithResponse(ctx, serverID, mac, params)
	if err != nil {
		return nil, fmt.Errorf("get firewall: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get firewall: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get firewall: empty response")
	}

	return resp.JSON200, nil
}

// UpdateFirewall replaces the firewall configuration for a network interface.
// The body must include all desired policies; omitted policies will be removed.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) UpdateFirewall(ctx context.Context, serverID int32, mac string, body generated.ServerFirewallSave) (*generated.TaskInfo, error) {
	resp, err := c.api.PutApiV1ServersServerIdInterfacesMacFirewallWithResponse(ctx, serverID, mac, body)
	if err != nil {
		return nil, fmt.Errorf("update firewall: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("update firewall: %w", err)
	}

	return resp.JSON202, nil
}

// ReapplyFirewall re-applies the firewall rules to a network interface without changing the configuration.
// Use this to recover from inconsistencies reported by GetFirewall.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) ReapplyFirewall(ctx context.Context, serverID int32, mac string) (*generated.TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdInterfacesMacFirewallReapplyWithResponse(ctx, serverID, mac, setContentTypeJSON)
	if err != nil {
		return nil, fmt.Errorf("reapply firewall: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("reapply firewall: %w", err)
	}

	return resp.JSON202, nil
}

// RestoreCopiedFirewallPolicies re-applies copied firewall policies for a network interface.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) RestoreCopiedFirewallPolicies(ctx context.Context, serverID int32, mac string) (*generated.TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdInterfacesMacFirewallRestoreCopiedPoliciesWithResponse(ctx, serverID, mac, setContentTypeJSON)
	if err != nil {
		return nil, fmt.Errorf("restore copied firewall policies: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("restore copied firewall policies: %w", err)
	}

	return resp.JSON202, nil
}
