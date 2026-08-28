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
	"math"
	"testing"
	"time"
)

func TestParseMetricPoints(t *testing.T) {
	raw := map[string]any{
		"2026-08-27T09:51:00Z": map[string]any{"CPU0": 2.0, "CPU1": 3.0},
		"2026-08-27T09:50:00Z": map[string]any{"CPU0": 1.0, "note": "not-a-number"},
	}
	points, err := parseMetricPoints("test", raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 {
		t.Fatalf("want 2 points, got %d", len(points))
	}
	if !points[0].Time.Before(points[1].Time) {
		t.Error("points not sorted by time")
	}
	// Non-numeric samples are dropped; missing series are absent, not zero.
	if _, ok := points[0].Values["note"]; ok {
		t.Error("non-numeric sample should be dropped")
	}
	if _, ok := points[0].Values["CPU1"]; ok {
		t.Error("missing series must be absent from Values, not present")
	}
	if got := Series(points); len(got) != 2 || got[0] != "CPU0" || got[1] != "CPU1" {
		t.Errorf("Series() = %v", got)
	}
}

func TestParseMetricPointsRejectsBadShape(t *testing.T) {
	if _, err := parseMetricPoints("test", map[string]any{"not-a-time": map[string]any{}}); err == nil {
		t.Error("want error for non-timestamp key")
	}
	if _, err := parseMetricPoints("test", map[string]any{"2026-08-27T09:50:00Z": "flat"}); err == nil {
		t.Error("want error for non-object value")
	}
}

// TestMetricsFromLiveFixtures runs every metric endpoint against its recorded
// production response and checks the typed result, including the CPU
// nanoseconds-to-percent conversion.
func TestMetricsFromLiveFixtures(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		fixture string
		call    func(c *Client) ([]MetricPoint, error)
		series  string // one series name that must exist
	}{
		{
			fixture: "GET_servers_111007_metrics_cpu.json",
			call:    func(c *Client) ([]MetricPoint, error) { return c.GetCPUMetrics(ctx, 111007, nil) },
			series:  "CPU0",
		},
		{
			fixture: "GET_servers_111007_metrics_disk.json",
			call:    func(c *Client) ([]MetricPoint, error) { return c.GetDiskMetrics(ctx, 111007, nil) },
			series:  "vda Read",
		},
		{
			fixture: "GET_servers_111007_metrics_network.json",
			call:    func(c *Client) ([]MetricPoint, error) { return c.GetNetworkMetrics(ctx, 111007, nil) },
			series:  "02:00:00:97:47:9f IN",
		},
		{
			fixture: "GET_servers_111007_metrics_network-packet.json",
			call: func(c *Client) ([]MetricPoint, error) {
				return c.GetNetworkPacketMetrics(ctx, 111007, nil)
			},
			series: "02:00:00:97:47:9f IN",
		},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			client, cleanup := newTestClient(t, fixtureHandler(t, tc.fixture))
			defer cleanup()
			points, err := tc.call(client)
			if err != nil {
				t.Fatal(err)
			}
			if len(points) == 0 {
				t.Fatal("no points parsed")
			}
			for i := 1; i < len(points); i++ {
				if points[i].Time.Before(points[i-1].Time) {
					t.Fatal("points not sorted")
				}
			}
			if _, ok := points[0].Values[tc.series]; !ok {
				t.Errorf("series %q missing; have %v", tc.series, Series(points))
			}
		})
	}
}

func TestCPUMetricsConvertedToPercent(t *testing.T) {
	client, cleanup := newTestClient(t, fixtureHandler(t, "GET_servers_111007_metrics_cpu.json"))
	defer cleanup()
	points, err := client.GetCPUMetrics(context.Background(), 111007, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The recorded response has CPU0 = 18666666.67 ns/s at 09:50:00Z, which
	// is 1.87% of one core.
	ts, _ := time.Parse(time.RFC3339, "2026-08-27T09:50:00Z")
	for _, p := range points {
		if p.Time.Equal(ts) {
			if got := p.Values["CPU0"]; math.Abs(got-1.8666666) > 0.001 {
				t.Errorf("CPU0 = %v, want ~1.867 percent", got)
			}
			return
		}
	}
	t.Fatal("expected timestamp not found in fixture")
}
