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

// ListVLansOptions configures the ListVLans operation.
type ListVLansOptions struct {
	// ServerId filters VLANs by server ID
	ServerId *int32
}

// ListVLans retrieves all VLANs for a user.
func (c *Client) ListVLans(ctx context.Context, userID int32, opts *ListVLansOptions) ([]generated.VLan, error) {
	params := &generated.GetApiV1UsersUserIdVlansParams{}
	if opts != nil {
		params.ServerId = opts.ServerId
	}

	resp, err := c.api.GetApiV1UsersUserIdVlansWithResponse(ctx, userID, params)
	if err != nil {
		return nil, fmt.Errorf("list vlans: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list vlans: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list vlans: empty response")
	}

	return *resp.JSON200, nil
}

// GetVLan retrieves information about a specific VLAN.
func (c *Client) GetVLan(ctx context.Context, userID int32, vlanID int32) (*generated.VLan, error) {
	resp, err := c.api.GetApiV1UsersUserIdVlansVlanIdWithResponse(ctx, userID, vlanID)
	if err != nil {
		return nil, fmt.Errorf("get vlan: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get vlan: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get vlan: empty response")
	}

	return resp.JSON200, nil
}

// GetVLanByID retrieves VLAN information by VLAN ID (without user ID).
func (c *Client) GetVLanByID(ctx context.Context, vlanID int32) (*generated.VLan, error) {
	resp, err := c.api.GetApiV1VlansVlanIdWithResponse(ctx, vlanID)
	if err != nil {
		return nil, fmt.Errorf("get vlan by id: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get vlan by id: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get vlan by id: empty response")
	}

	return resp.JSON200, nil
}

// UpdateVLan updates a VLAN's properties (currently only name).
func (c *Client) UpdateVLan(ctx context.Context, userID int32, vlanID int32, name string) error {
	vlanSave := generated.VLanSave{
		Name: &name,
	}

	resp, err := c.api.PutApiV1UsersUserIdVlansVlanIdWithResponse(ctx, userID, vlanID, vlanSave)
	if err != nil {
		return fmt.Errorf("update vlan: %w", err)
	}

	if err := checkResponse(resp, 200, 204); err != nil {
		return fmt.Errorf("update vlan: %w", err)
	}

	return nil
}
