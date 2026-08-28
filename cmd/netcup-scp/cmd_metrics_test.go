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

package main

import (
	"testing"
	"time"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func metricPointsEndingAt(last time.Time, interval time.Duration, n int) []scp.MetricPoint {
	points := make([]scp.MetricPoint, n)
	for i := range points {
		points[i] = scp.MetricPoint{
			Time:   last.Add(-time.Duration(n-1-i) * interval),
			Values: map[string]float64{"s": 1},
		}
	}
	return points
}

func TestTrimOpenBucket(t *testing.T) {
	now := time.Now()

	// Last bucket still open (its minute has not elapsed): dropped.
	open := metricPointsEndingAt(now.Truncate(time.Minute), time.Minute, 5)
	if got := trimOpenBucket(open); len(got) != 4 {
		t.Errorf("open bucket: want 4 points, got %d", len(got))
	}

	// Last bucket closed (older than one interval): kept.
	closed := metricPointsEndingAt(now.Add(-2*time.Minute), time.Minute, 5)
	if got := trimOpenBucket(closed); len(got) != 5 {
		t.Errorf("closed bucket: want 5 points, got %d", len(got))
	}

	// Too few points to judge: untouched.
	if got := trimOpenBucket(open[:2]); len(got) != 2 {
		t.Errorf("short series: want 2 points, got %d", len(got))
	}
}

func TestFilterSeriesAndTotal(t *testing.T) {
	points := []scp.MetricPoint{
		{Values: map[string]float64{"CPU0": 1, "CPU1": 2, "vda Read": 5}},
		{Values: map[string]float64{"CPU0": 3}},
		{Values: map[string]float64{}},
	}

	got, err := filterSeries(points, []string{"cpu*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got[0].Values) != 2 || len(got[1].Values) != 1 {
		t.Errorf("glob filter wrong: %+v", got)
	}

	if _, err := filterSeries(points, []string{"gpu*"}); err == nil {
		t.Error("want error when nothing matches")
	}

	tot := totalSeries(points)
	if tot[0].Values["TOTAL"] != 8 || tot[1].Values["TOTAL"] != 3 {
		t.Errorf("totals wrong: %+v", tot)
	}
	if len(tot[2].Values) != 0 {
		t.Error("empty point must stay a gap, not TOTAL=0")
	}
}
