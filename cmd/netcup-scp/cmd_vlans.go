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

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func newVLansCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vlans",
		Short: "Manage VLANs",
	}
	cmd.AddCommand(
		newVLansListCmd(),
		newVLansGetByIDCmd(),
		newVLansUpdateCmd(),
	)
	return cmd
}

func newVLansListCmd() *cobra.Command {
	var serverID int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your VLANs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			opts := &scp.ListVLansOptions{}
			if serverID > 0 {
				opts.ServerId = new(int32(serverID))
			}
			vlans, err := cc.client.ListVLans(cc.ctx, cc.userID, opts)
			if err != nil {
				return err
			}
			return printResult(cc, vlans, func() {
				t := newTable("VLAN ID", "NAME", "SITE", "BANDWIDTH (Mbit/s)")
				for _, v := range vlans {
					site := ""
					if v.Site != nil {
						site = v.Site.City
					}
					bw := ""
					if v.BandwidthClass != nil {
						bw = fmt.Sprintf("%d", v.BandwidthClass.SpeedInMBit)
					}
					t.AppendRow(table.Row{derefInt32(v.VlanId), derefStr(v.Name), site, bw})
				}
				t.Render()
			})
		},
	}
	cmd.Flags().IntVar(&serverID, "server-id", 0, "filter by server ID")
	return cmd
}

func newVLansGetByIDCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <vlan-id>",
		Short:             "Get VLAN details by ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(vlanIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			vlanID, err := parseID(args[0], "vlan-id")
			if err != nil {
				return err
			}
			vlan, err := cc.client.GetVLanByID(cc.ctx, vlanID)
			if err != nil {
				return err
			}
			return printResult(cc, vlan, func() {
				site := ""
				if vlan.Site != nil {
					site = vlan.Site.City
				}
				bw := ""
				if vlan.BandwidthClass != nil {
					bw = fmt.Sprintf("%d", vlan.BandwidthClass.SpeedInMBit)
				}
				printKV(
					"VLAN ID", derefInt32(vlan.VlanId),
					"Name", derefStr(vlan.Name),
					"Site", site,
					"Bandwidth (Mbit/s)", bw,
				)
			})
		},
	}
}

func newVLansUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "update <vlan-id> <name>",
		Short:             "Update VLAN name",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(vlanIDCompletions, nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			vlanID, err := parseID(args[0], "vlan-id")
			if err != nil {
				return err
			}
			if err := cc.client.UpdateVLan(cc.ctx, cc.userID, vlanID, args[1]); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}
