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
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func newInterfacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interfaces",
		Short: "Manage network interfaces",
	}
	cmd.AddCommand(
		newInterfacesListCmd(),
		newInterfacesGetCmd(),
		newInterfacesDeleteCmd(),
		newInterfacesUpdateDriverCmd(),
		newInterfacesCreateVLANCmd(),
	)
	return cmd
}

func newInterfacesListCmd() *cobra.Command {
	var loadRdns bool
	cmd := &cobra.Command{
		Use:               "list <server-id>",
		Short:             "List network interfaces",
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
			opts := &scp.ListInterfacesOptions{}
			if loadRdns {
				opts.LoadRdns = new(true)
			}
			ifaces, err := cc.client.ListInterfaces(cc.ctx, id, opts)
			if err != nil {
				return err
			}
			return printResult(cc, ifaces, func() {
				t := newTable("MAC", "SPEED (Mbit/s)", "DRIVER")
				for _, iface := range ifaces {
					t.AppendRow(table.Row{derefStr(iface.Mac), derefInt32(iface.SpeedInMBits), derefStr((*string)(iface.Driver))})
				}
				t.Render()
			})
		},
	}
	cmd.Flags().BoolVar(&loadRdns, "load-rdns", false, "include rDNS entries")
	return cmd
}

func newInterfacesGetCmd() *cobra.Command {
	var loadRdns bool
	cmd := &cobra.Command{
		Use:               "get <server-id> <mac>",
		Short:             "Get interface details",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions),
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
			opts := &scp.GetInterfaceOptions{}
			if loadRdns {
				opts.LoadRdns = new(true)
			}
			iface, err := cc.client.GetInterface(cc.ctx, id, args[1], opts)
			if err != nil {
				return err
			}
			return printResult(cc, iface, func() {
				printKV(
					"MAC", derefStr(iface.Mac),
					"Speed (Mbit/s)", derefInt32(iface.SpeedInMBits),
					"Driver", derefStr((*string)(iface.Driver)),
				)
				if iface.Ipv4Addresses != nil && len(*iface.Ipv4Addresses) > 0 {
					fmt.Println("\nIPv4:")
					t := newTable("ID", "CIDR", "GATEWAY / ROUTING TO", "RDNS")
					for _, ip := range *iface.Ipv4Addresses {
						gwOrDst := derefStr(ip.Gateway)
						if ip.Type != nil && *ip.Type == scp.ServerIpTypeROUTEDIP {
							gwOrDst = "→ " + derefStr(ip.DestinationIp)
						}
						t.AppendRow(table.Row{
							derefInt32(ip.Id),
							derefStr(ip.Cidr),
							gwOrDst,
							derefStr(ip.Rdns),
						})
					}
					t.Render()
				}
				if iface.Ipv6Addresses != nil && len(*iface.Ipv6Addresses) > 0 {
					fmt.Println("\nIPv6:")
					t := newTable("ID", "CIDR", "GATEWAY / ROUTING TO", "LINK-LOCAL")
					var ipv6WithRDNS []scp.ServerIpv6
					for _, ip := range *iface.Ipv6Addresses {
						gwOrDst := derefStr(ip.Gateway)
						if ip.Type != nil && *ip.Type == scp.ServerIpTypeROUTEDIP {
							gwOrDst = "→ " + derefStr(ip.DestinationIp)
						}
						t.AppendRow(table.Row{
							derefInt32(ip.Id),
							derefStr(ip.Cidr),
							gwOrDst,
							deref(ip.LinkLocal),
						})
						if ip.Rdns != nil && len(*ip.Rdns) > 0 {
							ipv6WithRDNS = append(ipv6WithRDNS, ip)
						}
					}
					t.Render()
					if len(ipv6WithRDNS) > 0 {
						fmt.Println("\nIPv6 rDNS:")
						rdnsTable := newTable("PREFIX", "ADDRESS", "HOSTNAME")
						for _, ip := range ipv6WithRDNS {
							keys := make([]string, 0, len(*ip.Rdns))
							for k := range *ip.Rdns {
								keys = append(keys, k)
							}
							sort.Strings(keys)
							for _, addr := range keys {
								rdnsTable.AppendRow(table.Row{derefStr(ip.Cidr), addr, (*ip.Rdns)[addr]}, table.RowConfig{AutoMerge: true})
							}
						}
						rdnsTable.Render()
					}
				}
			})
		},
	}
	cmd.Flags().BoolVar(&loadRdns, "load-rdns", false, "include rDNS entries")
	return cmd
}

func newInterfacesDeleteCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "delete <server-id> <mac>",
		Short:             "Delete a network interface",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, deletableInterfaceMACCompletions),
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

			iface, err := cc.client.GetInterface(cc.ctx, id, args[1], nil)
			if err != nil {
				return err
			}
			if scp.IsPrimaryInterface(iface) {
				return fmt.Errorf("interface %s has provider-assigned IP addresses and cannot be recreated via the API — deletion blocked", args[1])
			}

			task, err := cc.client.DeleteInterface(cc.ctx, id, args[1])
			if err != nil {
				return err
			}
			cc.invalidateCompletionCache(
				serverKey("interfaces-", id),
				serverKey("interfaces-deletable-", id),
			)
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	return cmd
}

func newInterfacesUpdateDriverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-driver <server-id> <mac> <driver>",
		Short: "Change interface driver",
		Args:  cobra.ExactArgs(3),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions, func(_ *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return networkDriverCompletions(), cobra.ShellCompDirectiveNoFileComp
		}),
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
			// The API's driver enum is uppercase, but `interfaces get`
			// displays lowercase — normalize so our own output is valid input.
			driver := scp.NetworkDriver(strings.ToUpper(args[2]))
			task, err := cc.client.UpdateInterfaceDriver(cc.ctx, id, args[1], driver)
			if err != nil {
				return err
			}
			if task != nil {
				return printTaskAndWait(cc, task, false)
			}
			printOK(cc)
			return nil
		},
	}
}

func newInterfacesCreateVLANCmd() *cobra.Command {
	var driver string
	var wait bool
	cmd := &cobra.Command{
		Use:               "create-vlan <server-id> <vlan-id>",
		Short:             "Create a VLAN interface",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, vlanIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			serverID, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			vlanID, err := parseID(args[1], "vlan-id")
			if err != nil {
				return err
			}
			task, err := cc.client.CreateVLanInterface(cc.ctx, serverID, vlanID, scp.NetworkDriver(driver))
			if err != nil {
				return err
			}
			cc.invalidateCompletionCache(
				serverKey("interfaces-", serverID),
				serverKey("interfaces-deletable-", serverID),
			)
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().StringVar(&driver, "driver", string(scp.NetworkDriverVIRTIO), "network driver")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	registerFlagCompleter(cmd, "driver", func(_ *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return networkDriverCompletions(), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// --- rdns-v4 ---

func newRDNSv4Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rdns-v4",
		Short: "Manage IPv4 reverse DNS entries",
	}
	cmd.AddCommand(
		newRDNSGetCmd("rdns-v4 get <ip>", func(cc *cmdContext, ip string) (string, error) {
			rdns, err := cc.client.GetRDNSv4(cc.ctx, ip)
			if err != nil {
				return "", err
			}
			return derefStr(rdns.Rdns), nil
		}),
		newRDNSSetCmd("rdns-v4 set <ip> <hostname>", func(cc *cmdContext, ip, hostname string) error {
			return cc.client.SetRDNSv4(cc.ctx, ip, hostname)
		}),
		newRDNSDeleteCmd("rdns-v4 delete <ip>", func(cc *cmdContext, ip string) error {
			return cc.client.DeleteRDNSv4(cc.ctx, ip)
		}),
	)
	return cmd
}

// --- rdns-v6 ---

func newRDNSv6Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rdns-v6",
		Short: "Manage IPv6 reverse DNS entries",
	}
	cmd.AddCommand(
		newRDNSGetCmd("rdns-v6 get <ip>", func(cc *cmdContext, ip string) (string, error) {
			rdns, err := cc.client.GetRDNSv6(cc.ctx, ip)
			if err != nil {
				return "", err
			}
			return derefStr(rdns.Rdns), nil
		}),
		newRDNSSetCmd("rdns-v6 set <ip> <hostname>", func(cc *cmdContext, ip, hostname string) error {
			return cc.client.SetRDNSv6(cc.ctx, ip, hostname)
		}),
		newRDNSDeleteCmd("rdns-v6 delete <ip>", func(cc *cmdContext, ip string) error {
			return cc.client.DeleteRDNSv6(cc.ctx, ip)
		}),
	)
	return cmd
}

func newRDNSGetCmd(_ string, fn func(*cmdContext, string) (string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "get <ip>",
		Short: "Get rDNS entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			hostname, err := fn(cc, args[0])
			if err != nil {
				return err
			}
			if cc.jsonOut {
				return printJSON(map[string]string{"hostname": hostname})
			}
			printKV("Hostname", hostname)
			return nil
		},
	}
}

func newRDNSSetCmd(_ string, fn func(*cmdContext, string, string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "set <ip> <hostname>",
		Short: "Set rDNS entry",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			if err := fn(cc, args[0], args[1]); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}

func newRDNSDeleteCmd(_ string, fn func(*cmdContext, string) error) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <ip>",
		Short: "Delete rDNS entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			if err := fn(cc, args[0]); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}

func networkDriverCompletions() []string {
	return []string{
		string(scp.NetworkDriverVIRTIO) + "\tparavirtual (recommended)",
		string(scp.NetworkDriverE1000) + "\tIntel Gigabit",
		string(scp.NetworkDriverE1000E) + "\tIntel Gigabit (e1000e)",
		string(scp.NetworkDriverRTL8139) + "\tRealtek Fast Ethernet",
		string(scp.NetworkDriverVMXNET3) + "\tVMware paravirtual",
	}
}
