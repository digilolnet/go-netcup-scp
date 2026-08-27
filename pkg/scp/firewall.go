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

// ServerFirewallSave is the request body for UpdateFirewall.
type ServerFirewallSave = generated.ServerFirewallSave

// IdentifierInt is an integer ID reference used in firewall policy lists.
type IdentifierInt = generated.IdentifierInt

// GetFirewallOptions configures the GetFirewall operation.
type GetFirewallOptions struct {
	// ConsistencyCheck verifies that the firewall rules have been applied, setting the
	// Consistent field on the result to true or false.
	ConsistencyCheck *bool
}

// GetFirewall retrieves the firewall configuration for a network interface.
func (c *Client) GetFirewall(
	ctx context.Context,
	serverID int32,
	mac string,
	opts *GetFirewallOptions,
) (*generated.ServerFirewall, error) {
	params := &generated.GetApiV1ServersServerIdInterfacesMacFirewallParams{}
	if opts != nil {
		params.ConsistencyCheck = opts.ConsistencyCheck
	}

	resp, err := c.api.GetApiV1ServersServerIdInterfacesMacFirewallWithResponse(ctx, serverID, mac, params)
	if err != nil {
		return nil, fmt.Errorf("get firewall: %w", err)
	}

	return pickBody("get firewall", resp, resp.JSON200, resp.HALJSON200, 200)
}

// UpdateFirewall replaces the firewall configuration for a network interface.
// The body must include all desired policies; omitted policies will be removed.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) UpdateFirewall(
	ctx context.Context,
	serverID int32,
	mac string,
	body ServerFirewallSave,
) (*TaskInfo, error) {
	resp, err := c.api.PutApiV1ServersServerIdInterfacesMacFirewallWithResponse(ctx, serverID, mac, body)
	if err != nil {
		return nil, fmt.Errorf("update firewall: %w", err)
	}

	return taskBody("update firewall", resp, resp.JSON202, resp.HALJSON202, 202)
}

// ReapplyFirewall re-applies the firewall rules to a network interface without changing the configuration.
// Use this to recover from inconsistencies reported by GetFirewall.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) ReapplyFirewall(ctx context.Context, serverID int32, mac string) (*TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdInterfacesMacFirewallReapplyWithResponse(
		ctx,
		serverID,
		mac,
		setContentTypeJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("reapply firewall: %w", err)
	}

	return taskBody("reapply firewall", resp, resp.JSON202, resp.HALJSON202, 202)
}

// SetFirewallActive enables or disables the firewall for a network interface while
// preserving all existing user and copied policies. The API requires the full policy
// list to be re-sent on every update, so this fetches the current config first.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) SetFirewallActive(ctx context.Context, serverID int32, mac string, active bool) (*TaskInfo, error) {
	fw, err := c.GetFirewall(ctx, serverID, mac, nil)
	if err != nil {
		return nil, fmt.Errorf("set firewall active: %w", err)
	}

	body := ServerFirewallSave{
		Active:         &active,
		UserPolicies:   []IdentifierInt{},
		CopiedPolicies: []IdentifierInt{},
	}
	if fw.UserPolicies != nil {
		for _, p := range *fw.UserPolicies {
			body.UserPolicies = append(body.UserPolicies, IdentifierInt{Id: deref(p.Id)})
		}
	}
	if fw.CopiedPolicies != nil {
		for _, p := range *fw.CopiedPolicies {
			body.CopiedPolicies = append(body.CopiedPolicies, IdentifierInt{Id: deref(p.Id)})
		}
	}

	task, err := c.UpdateFirewall(ctx, serverID, mac, body)
	if err != nil {
		return nil, fmt.Errorf("set firewall active: %w", err)
	}
	return task, nil
}

// ClearFirewall removes all policies from an interface.
//
// Contract note: on partial failure (the clear succeeded but restoring copied
// policies did not) it returns BOTH the tasks already issued and a non-nil
// error — discard neither: the wipe is in flight even though the call failed.
// ClearFirewall removes all user and copied policies from a network interface.
// If restoreCopied is true, netcup's copied policies are restored afterwards.
// Returns the tasks issued; two tasks are returned when restoreCopied is true and
// copied policies existed. The operations are asynchronous.
func (c *Client) ClearFirewall(
	ctx context.Context,
	serverID int32,
	mac string,
	restoreCopied bool,
) ([]*TaskInfo, error) {
	var hasCopied bool
	if restoreCopied {
		fw, err := c.GetFirewall(ctx, serverID, mac, nil)
		if err != nil {
			return nil, fmt.Errorf("clear firewall: %w", err)
		}
		hasCopied = fw.CopiedPolicies != nil && len(*fw.CopiedPolicies) > 0
	}

	task, err := c.UpdateFirewall(ctx, serverID, mac, ServerFirewallSave{
		UserPolicies:   []IdentifierInt{},
		CopiedPolicies: []IdentifierInt{},
	})
	if err != nil {
		return nil, fmt.Errorf("clear firewall: %w", err)
	}
	tasks := []*TaskInfo{task}

	if restoreCopied && hasCopied {
		task2, err := c.RestoreCopiedFirewallPolicies(ctx, serverID, mac)
		if err != nil {
			return tasks, fmt.Errorf("clear firewall: restore copied policies: %w", err)
		}
		tasks = append(tasks, task2)
	}
	return tasks, nil
}

// RestoreCopiedFirewallPolicies re-applies copied firewall policies for a network interface.
// The operation is asynchronous; use the returned task to track progress.
func (c *Client) RestoreCopiedFirewallPolicies(ctx context.Context, serverID int32, mac string) (*TaskInfo, error) {
	resp, err := c.api.PostApiV1ServersServerIdInterfacesMacFirewallRestoreCopiedPoliciesWithResponse(
		ctx,
		serverID,
		mac,
		setContentTypeJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("restore copied firewall policies: %w", err)
	}

	return taskBody("restore copied firewall policies", resp, resp.JSON202, resp.HALJSON202, 202)
}
