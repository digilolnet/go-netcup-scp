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
	"fmt"
	"sort"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Retrieve server performance metrics",
	}
	cmd.AddCommand(
		newMetricsCPUCmd(),
		newMetricsDiskCmd(),
		newMetricsNetworkCmd(),
		newMetricsNetworkPacketsCmd(),
	)
	return cmd
}

func newMetricsCPUCmd() *cobra.Command {
	var hours int
	cmd := &cobra.Command{
		Use:               "cpu <server-id>",
		Short:             "Get CPU usage metrics",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			data, err := cc.client.GetCPUMetrics(cc.ctx, id, &scp.MetricsOptions{Hours: ptr(int32(hours))})
			if err != nil {
				return err
			}
			return printResult(cc, data, func() {
				printMetricsSparklines(data, "OP/s", 1000)
			})
		},
	}
	cmd.Flags().IntVar(&hours, "hours", 6, "last N hours of data")
	return cmd
}

func newMetricsDiskCmd() *cobra.Command {
	var hours int
	cmd := &cobra.Command{
		Use:               "disk <server-id>",
		Short:             "Get disk I/O metrics",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			data, err := cc.client.GetDiskMetrics(cc.ctx, id, &scp.MetricsOptions{Hours: ptr(int32(hours))})
			if err != nil {
				return err
			}
			return printResult(cc, data, func() {
				printMetricsSparklines(data, "OP/s", 1000)
			})
		},
	}
	cmd.Flags().IntVar(&hours, "hours", 6, "last N hours of data")
	return cmd
}

func newMetricsNetworkCmd() *cobra.Command {
	var hours int
	cmd := &cobra.Command{
		Use:               "network <server-id>",
		Short:             "Get network throughput metrics",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			data, err := cc.client.GetNetworkMetrics(cc.ctx, id, &scp.MetricsOptions{Hours: ptr(int32(hours))})
			if err != nil {
				return err
			}
			return printResult(cc, data, func() {
				printMetricsSparklines(data, "B/s", 1024)
			})
		},
	}
	cmd.Flags().IntVar(&hours, "hours", 6, "last N hours of data")
	return cmd
}

func newMetricsNetworkPacketsCmd() *cobra.Command {
	var hours int
	cmd := &cobra.Command{
		Use:               "network-packet <server-id>",
		Short:             "Get network packet count metrics",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			data, err := cc.client.GetNetworkPacketMetrics(cc.ctx, id, &scp.MetricsOptions{Hours: ptr(int32(hours))})
			if err != nil {
				return err
			}
			return printResult(cc, data, func() {
				printMetricsSparklines(data, "PPS", 1000)
			})
		},
	}
	cmd.Flags().IntVar(&hours, "hours", 6, "last N hours of data")
	return cmd
}

func printMetricsSparklines(data map[string]any, baseUnit string, siBase float64) {
	type point struct {
		t    time.Time
		vals map[string]float64
	}

	var points []point
	seriesSet := map[string]bool{}

	for k, v := range data {
		t, err := time.Parse(time.RFC3339, k)
		if err != nil {
			_ = printJSON(data)
			return
		}
		nested, ok := v.(map[string]any)
		if !ok {
			_ = printJSON(data)
			return
		}
		vals := map[string]float64{}
		for nk, nv := range nested {
			if f, ok := nv.(float64); ok {
				vals[nk] = f
				seriesSet[nk] = true
			}
		}
		points = append(points, point{t, vals})
	}

	sort.Slice(points, func(i, j int) bool { return points[i].t.Before(points[j].t) })

	series := make([]string, 0, len(seriesSet))
	for k := range seriesSet {
		series = append(series, k)
	}
	sort.Strings(series)

	// Build one float64 slice per series and find global max for scaling
	seriesData := make([][]float64, len(series))
	globalMax := 0.0
	for si, name := range series {
		vals := make([]float64, len(points))
		for i, p := range points {
			vals[i] = p.vals[name]
			if vals[i] > globalMax {
				globalMax = vals[i]
			}
		}
		seriesData[si] = vals
	}

	// Determine SI scale
	var scale float64
	var unit string
	switch {
	case globalMax >= siBase*siBase*siBase:
		scale, unit = siBase*siBase*siBase, "G"+baseUnit
	case globalMax >= siBase*siBase:
		scale, unit = siBase*siBase, "M"+baseUnit
	case globalMax >= siBase:
		scale, unit = siBase, "K"+baseUnit
	default:
		scale, unit = 1, baseUnit
	}
	for si := range seriesData {
		for i := range seriesData[si] {
			seriesData[si][i] /= scale
		}
	}

	colors := []asciigraph.AnsiColor{
		asciigraph.Red, asciigraph.Green, asciigraph.Yellow, asciigraph.Blue,
		asciigraph.Magenta, asciigraph.Cyan, asciigraph.White, asciigraph.Orange,
	}
	graphColors := make([]asciigraph.AnsiColor, len(series))
	for i := range series {
		graphColors[i] = colors[i%len(colors)]
	}

	var xMin, xMax float64
	if len(points) > 0 {
		xMin = float64(points[0].t.Unix())
		xMax = float64(points[len(points)-1].t.Unix())
	}

	graph := asciigraph.PlotMany(seriesData,
		asciigraph.Height(15),
		asciigraph.Width(100),
		asciigraph.Precision(2),
		asciigraph.Caption(unit),
		asciigraph.XAxisRange(xMin, xMax),
		asciigraph.XAxisTickCount(6),
		asciigraph.XAxisValueFormatter(func(v float64) string {
			return time.Unix(int64(v), 0).UTC().Format("15:04")
		}),
		asciigraph.SeriesColors(graphColors...),
		asciigraph.SeriesLegends(series...),
	)
	fmt.Println(graph)
}
