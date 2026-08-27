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

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System-level operations",
	}
	cmd.AddCommand(
		newSystemPingCmd(),
		newSystemMaintenanceCmd(),
	)
	return cmd
}

func newSystemPingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check connectivity to the SCP API",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			if err := cc.client.Ping(cc.ctx); err != nil {
				return err
			}
			if cc.jsonOut {
				return printJSON(map[string]string{"status": "ok"})
			}
			fmt.Println("pong")
			return nil
		},
	}
}

func newSystemMaintenanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "maintenance",
		Short: "List upcoming maintenance windows",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			windows, err := cc.client.GetMaintenance(cc.ctx)
			if err != nil {
				return err
			}
			return printResult(cc, windows, func() {
				if len(windows) == 0 {
					fmt.Println("no maintenance scheduled")
					return
				}
				t := newTable("START (UTC)", "END (UTC)")
				for _, w := range windows {
					t.AppendRow(table.Row{fmtTime(w.StartAt), fmtTime(w.FinishAt)})
				}
				t.Render()
			})
		},
	}
}
