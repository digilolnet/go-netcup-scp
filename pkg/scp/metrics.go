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
	"sort"
	"time"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// MetricsOptions configures the metrics retrieval operations.
type MetricsOptions struct {
	// Hours limits the metrics to the last N hours. If nil, the API default is used.
	Hours *int32
}

// MetricPoint is one timestamped sample across all series of a metric.
// A series absent from Values means the API reported no sample for it at this
// point in time — callers should treat that as a gap, not as zero.
type MetricPoint struct {
	Time time.Time
	// Values maps a series name to its sample. Series names come from the
	// API and are already descriptive: "CPU0", "vda Read", "<mac> IN", ...
	Values map[string]float64
}

// Series returns the sorted union of series names across all points.
func Series(points []MetricPoint) []string {
	set := map[string]bool{}
	for _, p := range points {
		for name := range p.Values {
			set[name] = true
		}
	}
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// parseMetricPoints converts the API's untyped metric response (the spec
// declares only "object"; the real shape is ISO-8601 timestamp -> series ->
// value, verified against recorded live responses) into sorted MetricPoints.
func parseMetricPoints(op string, raw map[string]any) ([]MetricPoint, error) {
	points := make([]MetricPoint, 0, len(raw))
	for ts, v := range raw {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("%s: unexpected timestamp key %q: %w", op, ts, err)
		}
		nested, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: unexpected value type %T for %q", op, v, ts)
		}
		vals := make(map[string]float64, len(nested))
		for name, nv := range nested {
			f, ok := nv.(float64)
			if !ok {
				continue // non-numeric sample: treat as missing
			}
			vals[name] = f
		}
		points = append(points, MetricPoint{Time: t, Values: vals})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Time.Before(points[j].Time) })
	return points, nil
}

// getMetrics is the shared fetch+parse path for the four metric endpoints.
func getMetrics(
	op string,
	resp interface{ StatusCode() int },
	jsonBody, halBody *map[string]any,
) ([]MetricPoint, error) {
	raw, err := pickBodyVal(op, resp, jsonBody, halBody, 200)
	if err != nil {
		return nil, err
	}
	return parseMetricPoints(op, raw)
}

// GetCPUMetrics retrieves per-vCPU usage as percent of a single core.
// The API reports nanoseconds of CPU time consumed per second per vCPU
// (1e9 = a fully busy core); values are converted so 100 means one core
// fully utilized.
func (c *Client) GetCPUMetrics(ctx context.Context, serverID int32, opts *MetricsOptions) ([]MetricPoint, error) {
	params := &generated.GetApiV1ServersServerIdMetricsCpuParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsCpuWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get cpu metrics: %w", err)
	}

	points, err := getMetrics("get cpu metrics", resp, resp.JSON200, resp.HALJSON200)
	if err != nil {
		return nil, err
	}
	for _, p := range points {
		for name, v := range p.Values {
			p.Values[name] = v / 1e7 // ns of CPU time per second -> percent
		}
	}
	return points, nil
}

// GetDiskMetrics retrieves disk I/O operations per second, one series per
// disk and direction ("vda Read", "vda Write", ...).
func (c *Client) GetDiskMetrics(ctx context.Context, serverID int32, opts *MetricsOptions) ([]MetricPoint, error) {
	params := &generated.GetApiV1ServersServerIdMetricsDiskParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsDiskWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get disk metrics: %w", err)
	}

	return getMetrics("get disk metrics", resp, resp.JSON200, resp.HALJSON200)
}

// GetNetworkMetrics retrieves network throughput in bytes per second, one
// series per interface and direction ("<mac> IN", "<mac> OUT").
func (c *Client) GetNetworkMetrics(ctx context.Context, serverID int32, opts *MetricsOptions) ([]MetricPoint, error) {
	params := &generated.GetApiV1ServersServerIdMetricsNetworkParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsNetworkWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get network metrics: %w", err)
	}

	return getMetrics("get network metrics", resp, resp.JSON200, resp.HALJSON200)
}

// GetNetworkPacketMetrics retrieves network packets per second, one series
// per interface and direction ("<mac> IN", "<mac> OUT").
func (c *Client) GetNetworkPacketMetrics(
	ctx context.Context,
	serverID int32,
	opts *MetricsOptions,
) ([]MetricPoint, error) {
	params := &generated.GetApiV1ServersServerIdMetricsNetworkPacketParams{}
	if opts != nil {
		params.Hours = opts.Hours
	}

	resp, err := c.api.GetApiV1ServersServerIdMetricsNetworkPacketWithResponse(ctx, serverID, params)
	if err != nil {
		return nil, fmt.Errorf("get network packet metrics: %w", err)
	}

	return getMetrics("get network packet metrics", resp, resp.JSON200, resp.HALJSON200)
}
