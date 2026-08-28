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

	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func newVLansCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vlans",
		Short: "Manage VLANs",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newVLansListCmd(),
		newVLansGetByIDCmd(),
		newVLansUpdateCmd(),
	)
	return cmd
}

var vlanDisplayer = newDisplayer(
	column("id", "VLAN ID", func(v scp.VLan) any { return derefInt32(v.VlanId) }),
	column("name", "NAME", func(v scp.VLan) any { return derefStr(v.Name) }),
	column("site", "SITE", func(v scp.VLan) any {
		if v.Site != nil {
			return v.Site.City
		}
		return ""
	}),
	column("bandwidth", "BANDWIDTH (Mbit/s)", func(v scp.VLan) any {
		if v.BandwidthClass != nil {
			return fmt.Sprintf("%d", v.BandwidthClass.SpeedInMBit)
		}
		return ""
	}),
)

func newVLansListCmd() *cobra.Command {
	var server string
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
			if server != "" {
				id, err := resolveServerArg(cc, server)
				if err != nil {
					return err
				}
				opts.ServerId = &id
			}
			vlans, err := cc.client.ListVLans(cc.ctx, cc.userID, opts)
			if err != nil {
				return err
			}
			return vlanDisplayer.print(cc, vlans)
		},
	}
	cmd.Flags().StringVar(&server, "server", "", "filter by server (id, nickname, name, or hostname)")
	registerFlagCompleter(cmd, "server", serverIDCompletions)
	vlanDisplayer.addFlags(cmd)
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
			cc.invalidateCompletionCache("vlans")
			printOK(cc)
			return nil
		},
	}
}
