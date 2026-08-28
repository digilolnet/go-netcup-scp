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
	"math"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

// metricSpec describes one metrics subcommand; the four commands differ only
// in endpoint, unit and SI base.
type metricSpec struct {
	use    string
	short  string
	unit   string
	siBase float64
	fetch  func(cc *cmdContext, id int32, opts *scp.MetricsOptions) ([]scp.MetricPoint, error)
}

var metricSpecs = []metricSpec{
	{
		use: "cpu", short: "Per-vCPU usage (percent of one core)", unit: "%", siBase: 1000,
		fetch: func(cc *cmdContext, id int32, opts *scp.MetricsOptions) ([]scp.MetricPoint, error) {
			return cc.client.GetCPUMetrics(cc.ctx, id, opts)
		},
	},
	{
		use: "disk", short: "Disk I/O operations per second", unit: "OP/s", siBase: 1000,
		fetch: func(cc *cmdContext, id int32, opts *scp.MetricsOptions) ([]scp.MetricPoint, error) {
			return cc.client.GetDiskMetrics(cc.ctx, id, opts)
		},
	},
	{
		use: "network", short: "Network throughput", unit: "B/s", siBase: 1024,
		fetch: func(cc *cmdContext, id int32, opts *scp.MetricsOptions) ([]scp.MetricPoint, error) {
			return cc.client.GetNetworkMetrics(cc.ctx, id, opts)
		},
	},
	{
		use: "network-packet", short: "Network packets per second", unit: "P/s", siBase: 1000,
		fetch: func(cc *cmdContext, id int32, opts *scp.MetricsOptions) ([]scp.MetricPoint, error) {
			return cc.client.GetNetworkPacketMetrics(cc.ctx, id, opts)
		},
	},
}

func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Retrieve server performance metrics",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	for _, spec := range metricSpecs {
		cmd.AddCommand(newMetricsSubCmd(spec))
	}
	return cmd
}

func newMetricsSubCmd(spec metricSpec) *cobra.Command {
	var hours int32
	cmd := &cobra.Command{
		Use:               spec.use + " <server-id>",
		Short:             spec.short,
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
			points, err := spec.fetch(cc, id, &scp.MetricsOptions{Hours: &hours})
			if err != nil {
				return err
			}
			return printResult(cc, points, func() {
				printMetricsSparklines(points, spec.unit, spec.siBase)
			})
		},
	}
	cmd.Flags().Int32Var(&hours, "hours", 6, "last N hours of data")
	return cmd
}

// printMetricsSparklines renders one colored sparkline per series. Samples a
// series is missing are plotted as NaN, which asciigraph draws as gaps —
// never as fake zeros.
func printMetricsSparklines(points []scp.MetricPoint, baseUnit string, siBase float64) {
	if len(points) == 0 {
		fmt.Println("no data")
		return
	}
	series := scp.Series(points)

	seriesData := make([][]float64, len(series))
	globalMax := 0.0
	for si, name := range series {
		vals := make([]float64, len(points))
		for i, p := range points {
			v, ok := p.Values[name]
			if !ok {
				vals[i] = math.NaN()
				continue
			}
			vals[i] = v
			if v > globalMax {
				globalMax = v
			}
		}
		seriesData[si] = vals
	}

	// Scale to the largest SI prefix the data reaches.
	scale, unit := 1.0, baseUnit
	for _, prefix := range []string{"K", "M", "G"} {
		if globalMax >= scale*siBase {
			scale, unit = scale*siBase, prefix+baseUnit
		}
	}
	for si := range seriesData {
		for i, v := range seriesData[si] {
			seriesData[si][i] = v / scale
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

	xMin := float64(points[0].Time.Unix())
	xMax := float64(points[len(points)-1].Time.Unix())

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
