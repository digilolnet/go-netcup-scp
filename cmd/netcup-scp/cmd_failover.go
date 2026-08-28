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

func newFailoverV4Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "failover-v4",
		Short: "Manage IPv4 failover addresses",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newFailoverV4ListCmd(),
		newFailoverV4RouteCmd(),
		newFailoverV4UnrouteCmd(),
	)
	return cmd
}

func newFailoverV4ListCmd() *cobra.Command {
	var ip string
	var serverID int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your IPv4 failover addresses",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			opts := &scp.ListFailoverIPsOptions{}
			if ip != "" {
				opts.Ip = &ip
			}
			if serverID > 0 {
				opts.ServerId = new(int32(serverID))
			}
			ips, err := cc.client.ListFailoverIPv4(cc.ctx, cc.userID, opts)
			if err != nil {
				return err
			}
			return printResult(cc, ips, func() {
				t := newTable("ID", "IP", "SITE", "ROUTED TO SERVER")
				for _, f := range ips {
					serverName := ""
					if f.Server != nil {
						serverName = fmt.Sprintf("%d (%s)", derefInt32(f.Server.Id), derefStr(f.Server.Name))
					}
					site := ""
					if f.Site != nil {
						site = f.Site.City
					}
					ip := fmt.Sprintf("%s/%d", derefStr(f.Ip), derefInt32(f.CidrSuffix))
					t.AppendRow(table.Row{derefInt32(f.Id), ip, site, serverName})
				}
				t.Render()
			})
		},
	}
	cmd.Flags().StringVar(&ip, "ip", "", "filter by IP address")
	cmd.Flags().IntVar(&serverID, "server-id", 0, "filter by server ID")
	return cmd
}

func newFailoverV4RouteCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "route <failover-id> <server-id>",
		Short:             "Route a failover IPv4 to a server",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(failoverV4IDCompletions, serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			failoverID, err := parseID(args[0], "failover-id")
			if err != nil {
				return err
			}
			serverID, err := parseID(args[1], "server-id")
			if err != nil {
				return err
			}
			task, err := cc.client.RouteFailoverIPv4(cc.ctx, cc.userID, failoverID, serverID)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	return cmd
}

func newFailoverV4UnrouteCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "unroute <failover-id>",
		Short:             "Unroute a failover IPv4 from its server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(failoverV4IDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			failoverID, err := parseID(args[0], "failover-id")
			if err != nil {
				return err
			}
			task, err := cc.client.UnrouteFailoverIPv4(cc.ctx, cc.userID, failoverID)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	return cmd
}

func newFailoverV6Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "failover-v6",
		Short: "Manage IPv6 failover prefixes",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newFailoverV6ListCmd(),
		newFailoverV6RouteCmd(),
		newFailoverV6UnrouteCmd(),
	)
	return cmd
}

func newFailoverV6ListCmd() *cobra.Command {
	var ip string
	var serverID int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your IPv6 failover prefixes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			opts := &scp.ListFailoverIPsOptions{}
			if ip != "" {
				opts.Ip = &ip
			}
			if serverID > 0 {
				opts.ServerId = new(int32(serverID))
			}
			ips, err := cc.client.ListFailoverIPv6(cc.ctx, cc.userID, opts)
			if err != nil {
				return err
			}
			return printResult(cc, ips, func() {
				t := newTable("ID", "PREFIX", "SITE", "ROUTED TO SERVER")
				for _, f := range ips {
					serverName := ""
					if f.Server != nil {
						serverName = fmt.Sprintf("%d (%s)", derefInt32(f.Server.Id), derefStr(f.Server.Name))
					}
					site := ""
					if f.Site != nil {
						site = f.Site.City
					}
					prefix := fmt.Sprintf("%s/%d", derefStr(f.NetworkPrefix), derefInt32(f.NetworkPrefixLength))
					t.AppendRow(table.Row{derefInt32(f.Id), prefix, site, serverName})
				}
				t.Render()
			})
		},
	}
	cmd.Flags().StringVar(&ip, "ip", "", "filter by IP address")
	cmd.Flags().IntVar(&serverID, "server-id", 0, "filter by server ID")
	return cmd
}

func newFailoverV6UnrouteCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "unroute <failover-id>",
		Short:             "Unroute a failover IPv6 from its server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(failoverV6IDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			failoverID, err := parseID(args[0], "failover-id")
			if err != nil {
				return err
			}
			task, err := cc.client.UnrouteFailoverIPv6(cc.ctx, cc.userID, failoverID)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	return cmd
}

func newFailoverV6RouteCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "route <failover-id> <server-id>",
		Short:             "Route a failover IPv6 to a server",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(failoverV6IDCompletions, serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			failoverID, err := parseID(args[0], "failover-id")
			if err != nil {
				return err
			}
			serverID, err := parseID(args[1], "server-id")
			if err != nil {
				return err
			}
			task, err := cc.client.RouteFailoverIPv6(cc.ctx, cc.userID, failoverID, serverID)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	return cmd
}
