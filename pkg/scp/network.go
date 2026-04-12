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

// NetworkDriver is the virtual NIC driver type for VLAN interfaces.
type NetworkDriver = generated.NetworkDriver

// Network driver constants for use with CreateVLanInterface.
const (
	NetworkDriverVIRTIO  NetworkDriver = generated.NetworkDriverVIRTIO
	NetworkDriverE1000   NetworkDriver = generated.NetworkDriverE1000
	NetworkDriverE1000E  NetworkDriver = generated.NetworkDriverE1000E
	NetworkDriverVMXNET3 NetworkDriver = generated.NetworkDriverVMXNET3
	NetworkDriverRTL8139 NetworkDriver = generated.NetworkDriverRTL8139
)

// Interface represents a server network interface.
type Interface = generated.Interface

// ServerIpv6 represents an IPv6 address attached to a network interface.
type ServerIpv6 = generated.ServerIpv6

// ServerIpType is the classification of an IP address attached to an interface.
type ServerIpType = generated.ServerIpType

// ServerIpType constants for filtering IP address types.
const (
	ServerIpTypeIP       ServerIpType = generated.ServerIpTypeIP
	ServerIpTypeROUTEDIP ServerIpType = generated.ServerIpTypeROUTEDIP
)

// ListInterfacesOptions configures the ListInterfaces operation.
type ListInterfacesOptions struct {
	// LoadRdns includes reverse DNS entries for each IP address.
	LoadRdns *bool
}

// ListInterfaces retrieves all network interfaces for a server.
func (c *Client) ListInterfaces(ctx context.Context, serverID int32, opts *ListInterfacesOptions) ([]generated.Interface, error) {
	params := &generated.GetApiV1ServersServerIdInterfacesParams{}
	if opts != nil {
		params.LoadRdns = opts.LoadRdns
	}

	resp, err := c.api.GetApiV1ServersServerIdInterfacesWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list interfaces: empty response")
	}

	return *resp.JSON200, nil
}

// GetInterfaceOptions configures the GetInterface operation.
type GetInterfaceOptions struct {
	// LoadRdns includes reverse DNS entries for the interface's IP addresses.
	LoadRdns *bool
}

// GetInterface retrieves information about a specific network interface.
func (c *Client) GetInterface(ctx context.Context, serverID int32, mac string, opts *GetInterfaceOptions) (*generated.Interface, error) {
	params := &generated.GetApiV1ServersServerIdInterfacesMacParams{}
	if opts != nil {
		params.LoadRdns = opts.LoadRdns
	}

	resp, err := c.api.GetApiV1ServersServerIdInterfacesMacWithResponse(ctx, serverID, mac, params)
	if err != nil {
		return nil, fmt.Errorf("get interface: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get interface: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get interface: empty response")
	}

	return resp.JSON200, nil
}

// SetRDNSv4 sets the reverse DNS entry for an IPv4 address.
func (c *Client) SetRDNSv4(ctx context.Context, ip, hostname string) error {
	resp, err := c.api.PostApiV1RdnsIpv4WithResponse(
		ctx,
		generated.SetRdnsIpv4{
			Ip:   ip,
			Rdns: hostname,
		},
	)
	if err != nil {
		return fmt.Errorf("set rdns ipv4: %w", err)
	}

	if err := checkResponse(resp, 200, 201); err != nil {
		return fmt.Errorf("set rdns ipv4: %w", err)
	}

	return nil
}

// SetRDNSv6 sets the reverse DNS entry for an IPv6 address.
func (c *Client) SetRDNSv6(ctx context.Context, ip, hostname string) error {
	resp, err := c.api.PostApiV1RdnsIpv6WithResponse(
		ctx,
		generated.SetRdnsIpv6{
			Ip:   ip,
			Rdns: hostname,
		},
	)
	if err != nil {
		return fmt.Errorf("set rdns ipv6: %w", err)
	}

	if err := checkResponse(resp, 200, 201); err != nil {
		return fmt.Errorf("set rdns ipv6: %w", err)
	}

	return nil
}

// DeleteRDNSv4 removes the reverse DNS entry for an IPv4 address.
func (c *Client) DeleteRDNSv4(ctx context.Context, ip string) error {
	resp, err := c.api.DeleteApiV1RdnsIpv4IpWithResponse(ctx, ip)
	if err != nil {
		return fmt.Errorf("delete rdns ipv4: %w", err)
	}

	if err := checkResponse(resp, 200, 204); err != nil {
		return fmt.Errorf("delete rdns ipv4: %w", err)
	}

	return nil
}

// DeleteRDNSv6 removes the reverse DNS entry for an IPv6 address.
func (c *Client) DeleteRDNSv6(ctx context.Context, ip string) error {
	resp, err := c.api.DeleteApiV1RdnsIpv6IpWithResponse(ctx, ip)
	if err != nil {
		return fmt.Errorf("delete rdns ipv6: %w", err)
	}

	if err := checkResponse(resp, 200, 204); err != nil {
		return fmt.Errorf("delete rdns ipv6: %w", err)
	}

	return nil
}

// CreateVLanInterface creates a new VLAN network interface for a server.
func (c *Client) CreateVLanInterface(ctx context.Context, serverID int32, vlanID int32, driver NetworkDriver) (*TaskInfo, error) {
	patch := generated.ServerCreateNicVlan{
		VlanId:        vlanID,
		NetworkDriver: driver,
	}
	body, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("create vlan interface: %w", err)
	}

	resp, err := c.api.PostApiV1ServersServerIdInterfacesWithBodyWithResponse(
		ctx,
		serverID,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create vlan interface: %w", err)
	}

	if err := checkResponse(resp, 202); err != nil {
		return nil, fmt.Errorf("create vlan interface: %w", err)
	}

	return resp.JSON202, nil
}

// IsPrimaryInterface reports whether iface has provider-assigned (non-editable) IP
// addresses. Such interfaces cannot be safely deleted because they cannot be
// recreated via the API.
func IsPrimaryInterface(iface *Interface) bool {
	if iface.Ipv4Addresses != nil {
		for _, ip := range *iface.Ipv4Addresses {
			if ip.Type != nil && *ip.Type == ServerIpTypeIP && !deref(ip.Editable) {
				return true
			}
		}
	}
	if iface.Ipv6Addresses != nil {
		for _, ip := range *iface.Ipv6Addresses {
			if ip.Type != nil && *ip.Type == ServerIpTypeIP && !deref(ip.LinkLocal) && !deref(ip.Editable) {
				return true
			}
		}
	}
	return false
}

// DeleteInterface removes a network interface from a server.
func (c *Client) DeleteInterface(ctx context.Context, serverID int32, mac string) error {
	resp, err := c.api.DeleteApiV1ServersServerIdInterfacesMacWithResponse(ctx, serverID, mac)
	if err != nil {
		return fmt.Errorf("delete interface: %w", err)
	}

	if err := checkResponse(resp, 200, 202, 204); err != nil {
		return fmt.Errorf("delete interface: %w", err)
	}

	return nil
}

// UpdateInterfaceDriver updates a network interface's driver.
func (c *Client) UpdateInterfaceDriver(ctx context.Context, serverID int32, mac string, driver NetworkDriver) error {
	update := generated.ServerInterfaceUpdate{
		Driver: &driver,
	}

	resp, err := c.api.PutApiV1ServersServerIdInterfacesMacWithResponse(ctx, serverID, mac, update)
	if err != nil {
		return fmt.Errorf("update interface driver: %w", err)
	}

	if err := checkResponse(resp, 200, 204); err != nil {
		return fmt.Errorf("update interface driver: %w", err)
	}

	return nil
}

// GetRDNSv4 retrieves the reverse DNS entry for an IPv4 address.
func (c *Client) GetRDNSv4(ctx context.Context, ip string) (*generated.Rdns, error) {
	resp, err := c.api.GetApiV1RdnsIpv4IpWithResponse(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("get rdns ipv4: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get rdns ipv4: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get rdns ipv4: empty response")
	}

	return resp.JSON200, nil
}

// GetRDNSv6 retrieves the reverse DNS entry for an IPv6 address.
func (c *Client) GetRDNSv6(ctx context.Context, ip string) (*generated.Rdns, error) {
	resp, err := c.api.GetApiV1RdnsIpv6IpWithResponse(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("get rdns ipv6: %w", err)
	}

	if err := checkResponse(resp, 200); err != nil {
		return nil, fmt.Errorf("get rdns ipv6: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get rdns ipv6: empty response")
	}

	return resp.JSON200, nil
}
