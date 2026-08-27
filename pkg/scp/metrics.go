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

// MetricsOptions configures the metrics retrieval operations.
type MetricsOptions struct {
	// Hours limits the metrics to the last N hours. If nil, the API default is used.
	Hours *int32
}

// GetCPUMetrics retrieves CPU usage metrics for a server.
// The response schema is untyped (map[string]any) as it varies by API version.
func (c *Client) GetCPUMetrics(ctx context.Context, serverID int32, opts *MetricsOptions) (map[string]any, error) {
	params := &generated.GetApiV1ServersServerIdMetricsCpuParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsCpuWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get cpu metrics: %w", err)
	}

	return pickBodyVal("get cpu metrics", resp, resp.JSON200, resp.HALJSON200, 200)
}

// GetDiskMetrics retrieves disk I/O metrics for a server.
// The response schema is untyped (map[string]any) as it varies by API version.
func (c *Client) GetDiskMetrics(ctx context.Context, serverID int32, opts *MetricsOptions) (map[string]any, error) {
	params := &generated.GetApiV1ServersServerIdMetricsDiskParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsDiskWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get disk metrics: %w", err)
	}

	return pickBodyVal("get disk metrics", resp, resp.JSON200, resp.HALJSON200, 200)
}

// GetNetworkMetrics retrieves network throughput metrics for a server.
// The response schema is untyped (map[string]any) as it varies by API version.
func (c *Client) GetNetworkMetrics(ctx context.Context, serverID int32, opts *MetricsOptions) (map[string]any, error) {
	params := &generated.GetApiV1ServersServerIdMetricsNetworkParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsNetworkWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get network metrics: %w", err)
	}

	return pickBodyVal("get network metrics", resp, resp.JSON200, resp.HALJSON200, 200)
}

// GetNetworkPacketMetrics retrieves network packet count metrics for a server.
// The response schema is untyped (map[string]any) as it varies by API version.
func (c *Client) GetNetworkPacketMetrics(
	ctx context.Context,
	serverID int32,
	opts *MetricsOptions,
) (map[string]any, error) {
	params := &generated.GetApiV1ServersServerIdMetricsNetworkPacketParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsNetworkPacketWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get network packet metrics: %w", err)
	}

	return pickBodyVal("get network packet metrics", resp, resp.JSON200, resp.HALJSON200, 200)
}
