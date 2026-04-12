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

// FirewallPolicySave is the request body for creating or updating a firewall policy.
type FirewallPolicySave = generated.FirewallPolicySave

// FirewallRule defines a single firewall rule within a policy.
type FirewallRule = generated.FirewallRule

// FirewallRuleDirection is the traffic direction for a firewall rule.
type FirewallRuleDirection = generated.FirewallRuleDirection

// FirewallProtocol is the network protocol for a firewall rule.
type FirewallProtocol = generated.FirewallProtocol

// FirewallAction is the action taken when a firewall rule matches.
type FirewallAction = generated.FirewallAction

// ListFirewallPoliciesOptions configures the ListFirewallPolicies operation.
type ListFirewallPoliciesOptions struct {
	// Limit sets the maximum number of policies to return.
	Limit *int32
	// Offset sets the starting position for pagination.
	Offset *int32
	// Q searches policies by name or description (case-insensitive).
	Q *string
}

// ListFirewallPolicies retrieves all firewall policies for a user.
func (c *Client) ListFirewallPolicies(ctx context.Context, userID int32, opts *ListFirewallPoliciesOptions) ([]generated.FirewallPolicy, error) {
	params := &generated.GetApiV1UsersUserIdFirewallPoliciesParams{}
	if opts != nil {
		params.Limit = opts.Limit
		params.Offset = opts.Offset
		params.Q = opts.Q
	}

	resp, err := c.api.GetApiV1UsersUserIdFirewallPoliciesWithResponse(ctx, userID, params)
	if err != nil {
		return nil, fmt.Errorf("list firewall policies: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list firewall policies: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list firewall policies: empty response")
	}

	return *resp.JSON200, nil
}

// CreateFirewallPolicy creates a new firewall policy for a user.
func (c *Client) CreateFirewallPolicy(ctx context.Context, userID int32, policy FirewallPolicySave) (*generated.FirewallPolicy, error) {
	resp, err := c.api.PostApiV1UsersUserIdFirewallPoliciesWithResponse(ctx, userID, policy)
	if err != nil {
		return nil, fmt.Errorf("create firewall policy: %w", err)
	}

	if err := checkResponse(resp, 201); err != nil {
		return nil, fmt.Errorf("create firewall policy: %w", err)
	}

	if resp.JSON201 == nil {
		return nil, fmt.Errorf("create firewall policy: empty response")
	}

	return resp.JSON201, nil
}

// GetFirewallPolicy retrieves a specific firewall policy by ID.
func (c *Client) GetFirewallPolicy(ctx context.Context, userID, policyID int32) (*generated.FirewallPolicy, error) {
	resp, err := c.api.GetApiV1UsersUserIdFirewallPoliciesIdWithResponse(ctx, userID, policyID)
	if err != nil {
		return nil, fmt.Errorf("get firewall policy: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get firewall policy: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get firewall policy: empty response")
	}

	return resp.JSON200, nil
}

// UpdateFirewallPolicy updates an existing firewall policy.
// If the policy is applied to servers, the update is asynchronous and a task is returned.
func (c *Client) UpdateFirewallPolicy(ctx context.Context, userID, policyID int32, policy FirewallPolicySave) (*generated.FirewallPolicyUpdateResult, error) {
	resp, err := c.api.PutApiV1UsersUserIdFirewallPoliciesIdWithResponse(ctx, userID, policyID, policy)
	if err != nil {
		return nil, fmt.Errorf("update firewall policy: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("update firewall policy: %w", err)
	}

	return resp.JSON202, nil
}

// AddFirewallRule appends a rule to an existing firewall policy. The API always
// requires the full rule list, so this fetches the current policy first and
// re-sends all existing rules with the new one appended.
func (c *Client) AddFirewallRule(ctx context.Context, userID, policyID int32, rule FirewallRule) (*generated.FirewallPolicyUpdateResult, error) {
	existing, err := c.GetFirewallPolicy(ctx, userID, policyID)
	if err != nil {
		return nil, fmt.Errorf("add firewall rule: %w", err)
	}

	var rules []FirewallRule
	if existing.Rules != nil {
		rules = *existing.Rules
	}
	rules = append(rules, rule)

	result, err := c.UpdateFirewallPolicy(ctx, userID, policyID, FirewallPolicySave{
		Name:        deref(existing.Name),
		Description: existing.Description,
		Rules:       &rules,
	})
	if err != nil {
		return nil, fmt.Errorf("add firewall rule: %w", err)
	}
	return result, nil
}

// DeleteFirewallPolicy permanently deletes a firewall policy.
// The policy must not be in use by any server interface.
func (c *Client) DeleteFirewallPolicy(ctx context.Context, userID, policyID int32) error {
	resp, err := c.api.DeleteApiV1UsersUserIdFirewallPoliciesIdWithResponse(ctx, userID, policyID)
	if err != nil {
		return fmt.Errorf("delete firewall policy: %w", err)
	}

	if err := checkResponse(resp, 204); err != nil {
		return fmt.Errorf("delete firewall policy: %w", err)
	}

	return nil
}
