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
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func newServersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servers",
		Short: "Manage servers",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newServersListCmd(),
		newServersGetCmd(),
		newServersStartCmd(),
		newServersStopCmd(),
		newServersRestartCmd(),
		newServersAutostartCmd(),
		newServersUEFICmd(),
		newServersNicknameCmd(),
		newServersLogsCmd(),
		newServersCPUTopologyCmd(),
		newServersGuestAgentCmd(),
		newServersVNCCmd(),
		newServersRescueCmd(),
		newServersImageFlavoursCmd(),
		newServersInstallImageCmd(),
		newServersInstallUserImageCmd(),
		newServersOptimizeStorageCmd(),
		newServersQemuStatusCmd(),
		newServersGPUDriverCmd(),
	)
	return cmd
}

var serverListDisplayer = newDisplayer(
	column("id", "ID", func(s scp.ServerListMinimal) any { return derefInt32(s.Id) }),
	column("name", "NAME", func(s scp.ServerListMinimal) any { return derefStr(s.Name) }),
	column("nickname", "NICKNAME", func(s scp.ServerListMinimal) any { return derefStr(s.Nickname) }),
	column("hostname", "HOSTNAME", func(s scp.ServerListMinimal) any { return derefStr(s.Hostname) }),
	column("template", "TEMPLATE", func(s scp.ServerListMinimal) any {
		if s.Template != nil {
			return s.Template.Name
		}
		return ""
	}),
	column("disabled", "DISABLED", func(s scp.ServerListMinimal) any { return deref(s.Disabled) }),
)

func newServersListCmd() *cobra.Command {
	var limit, offset int
	var name, ip, q string
	var sort []string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			opts := &scp.ListServersOptions{}
			if limit > 0 {
				opts.Limit = new(int32(limit))
			}
			if offset > 0 {
				opts.Offset = new(int32(offset))
			}
			if name != "" {
				opts.Name = &name
			}
			if ip != "" {
				opts.Ip = &ip
			}
			if q != "" {
				opts.Q = &q
			}
			opts.Sort = sort

			servers, err := cc.client.ListServers(cc.ctx, opts)
			if err != nil {
				return err
			}
			return serverListDisplayer.print(cc, servers)
		},
	}
	serverListDisplayer.addFlags(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	cmd.Flags().StringVar(&name, "name", "", "filter by name")
	cmd.Flags().StringVar(&ip, "ip", "", "filter by IP address")
	cmd.Flags().StringVar(&q, "q", "", "search query (name, nickname, IPv4)")
	cmd.Flags().StringSliceVar(&sort, "sort", nil, "sort by field(s): name, nickname; prefix with '-' for descending")
	_ = cmd.RegisterFlagCompletionFunc(
		"sort",
		cobra.FixedCompletions([]string{"name", "nickname", "-name", "-nickname"}, cobra.ShellCompDirectiveNoFileComp),
	)
	return cmd
}

func newServersGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get <server>",
		Short:             "Get server details",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			server, err := cc.client.GetServer(cc.ctx, id, &scp.GetServerOptions{LoadServerLiveInfo: new(true)})
			if err != nil {
				return err
			}
			return printResult(cc, server, func() {
				site := ""
				if server.Site != nil {
					site = server.Site.City
				}
				tmpl := ""
				if server.Template != nil {
					tmpl = server.Template.Name
				}
				printKV(
					"ID", derefInt32(server.Id),
					"Name", derefStr(server.Name),
					"Hostname", derefStr(server.Hostname),
					"Nickname", derefStr(server.Nickname),
					"Site", site,
					"Architecture", derefStr((*string)(server.Architecture)),
					"Template", tmpl,
					"Disabled", deref(server.Disabled),
					"Rescue Active", deref(server.RescueSystemActive),
					"Snapshot Allowed", deref(server.SnapshotAllowed),
					"Snapshot Count", derefInt32(server.SnapshotCount),
					"Disk Free (MiB)", deref(server.DisksAvailableSpaceInMiB),
					"GPU Driver Available", deref(server.GpuDriverAvailable),
				)

				if v4 := server.Ipv4Addresses; v4 != nil && len(*v4) > 0 {
					fmt.Println("\nIPv4 Addresses:")
					t := newTable("ID", "IP", "NETMASK", "BROADCAST", "GATEWAY")
					for _, ip := range *v4 {
						t.AppendRow(table.Row{
							derefInt32(ip.Id),
							derefStr(ip.Ip),
							derefStr(ip.Netmask),
							derefStr(ip.Broadcast),
							derefStr(ip.Gateway),
						})
					}
					t.Render()
				}

				if v6 := server.Ipv6Addresses; v6 != nil && len(*v6) > 0 {
					fmt.Println("\nIPv6 Addresses:")
					t := newTable("ID", "PREFIX", "PREFIX LENGTH", "GATEWAY")
					for _, ip := range *v6 {
						t.AppendRow(table.Row{
							derefInt32(ip.Id),
							derefStr(ip.NetworkPrefix),
							derefInt32(ip.NetworkPrefixLength),
							derefStr(ip.Gateway),
						})
					}
					t.Render()
				}

				if li := server.ServerLiveInfo; li != nil {
					bootorder := ""
					if li.Bootorder != nil {
						parts := make([]string, len(*li.Bootorder))
						for i, b := range *li.Bootorder {
							parts[i] = string(b)
						}
						bootorder = strings.Join(parts, ", ")
					}
					uptime := ""
					if li.UptimeInSeconds != nil {
						s := int(*li.UptimeInSeconds)
						uptime = fmt.Sprintf("%dd %dh %dm", s/86400, (s%86400)/3600, (s%3600)/60)
					}
					fmt.Println("\nLive Info:")
					printKV(
						"State", derefStr((*string)(li.State)),
						"Uptime", uptime,
						"Machine Type", derefStr(li.MachineType),
						"UEFI", deref(li.Uefi),
						"Latest QEMU", deref(li.LatestQemu),
						"Config Changed", deref(li.ConfigChanged),
						"Autostart", deref(li.Autostart),
						"Boot Order", bootorder,
						"CPU Count", derefInt32(li.CpuCount),
						"CPU Max", derefInt32(li.CpuMaxCount),
						"Sockets", derefInt32(li.Sockets),
						"Cores/Socket", derefInt32(li.CoresPerSocket),
						"Memory (MiB)", deref(li.CurrentServerMemoryInMiB),
						"Memory Max (MiB)", deref(li.MaxServerMemoryInMiB),
						"OS Optimization", derefStr((*string)(li.OsOptimization)),
						"Keyboard Layout", derefStr(li.KeyboardLayout),
						"Nested Guest", deref(li.NestedGuest),
						"Cloudinit", deref(li.CloudinitAttached),
					)

					if li.Disks != nil && len(*li.Disks) > 0 {
						fmt.Println("\nDisks:")
						t := newTable("DEV", "DRIVER", "CAPACITY (MiB)", "ALLOCATION (MiB)")
						for _, d := range *li.Disks {
							t.AppendRow(table.Row{derefStr(d.Dev), derefStr(d.Driver), deref(d.CapacityInMiB), deref(d.AllocationInMiB)})
						}
						t.Render()
					}

					if li.Interfaces != nil && len(*li.Interfaces) > 0 {
						fmt.Println("\nInterfaces:")
						t := newTable("MAC", "DRIVER", "SPEED (Mbit/s)", "MTU", "RX/month (MiB)", "TX/month (MiB)", "THROTTLED", "VLAN")
						for _, iface := range *li.Interfaces {
							vlan := ""
							if deref(iface.VlanInterface) && iface.VlanId != nil {
								vlan = fmt.Sprintf("%d", *iface.VlanId)
							}
							t.AppendRow(table.Row{
								derefStr(iface.Mac),
								derefStr(iface.Driver),
								derefInt32(iface.SpeedInMBits),
								derefInt32(iface.Mtu),
								derefInt32(iface.RxMonthlyInMiB),
								derefInt32(iface.TxMonthlyInMiB),
								deref(iface.TrafficThrottled),
								vlan,
							})
						}
						t.Render()
					}
				}
			})
		},
	}
	return cmd
}

func newServersStartCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "start <server>",
		Short:             "Power on a server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			task, err := cc.client.StartServer(cc.ctx, id)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newServersStopCmd() *cobra.Command {
	var powerOff, wait bool
	cmd := &cobra.Command{
		Use:               "stop <server>",
		Short:             "Shut down a server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			task, err := cc.client.StopServer(cc.ctx, id, powerOff)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&powerOff, "power-off", false, "hard power off instead of graceful shutdown")
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newServersRestartCmd() *cobra.Command {
	var reset, wait bool
	cmd := &cobra.Command{
		Use:               "restart <server>",
		Short:             "Reboot a server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			task, err := cc.client.RestartServer(cc.ctx, id, reset)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&reset, "reset", false, "hard reset instead of power cycle")
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newServersAutostartCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "autostart <server> <on|off>",
		Short:             "Configure autostart",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, static("on", "off")),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			enabled, err := parseBool(args[1])
			if err != nil {
				return err
			}
			task, err := cc.client.SetAutostart(cc.ctx, id, enabled)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newServersUEFICmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "uefi <server> <on|off>",
		Short:             "Configure UEFI boot",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, static("on", "off")),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			enabled, err := parseBool(args[1])
			if err != nil {
				return err
			}
			task, err := cc.client.SetUEFI(cc.ctx, id, enabled)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newServersNicknameCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "nickname <server> <nickname>",
		Short:             "Set server nickname",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			if err := cc.client.UpdateNickname(cc.ctx, id, args[1]); err != nil {
				return err
			}
			cc.invalidateCompletionCache("servers")
			printOK(cc)
			return nil
		},
	}
}

func newServersLogsCmd() *cobra.Command {
	var limit, offset int
	cmd := &cobra.Command{
		Use:               "logs <server>",
		Short:             "Get server event log",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			opts := &scp.GetServerLogsOptions{}
			if limit > 0 {
				opts.Limit = new(int32(limit))
			}
			if offset > 0 {
				opts.Offset = new(int32(offset))
			}
			logs, err := cc.client.GetServerLogs(cc.ctx, id, opts)
			if err != nil {
				return err
			}
			return logDisplayer.print(cc, logs)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	logDisplayer.addFlags(cmd)
	return cmd
}

// logDisplayer renders event-log entries; shared by 'servers logs' and
// 'users logs'.
var logDisplayer = newDisplayer(
	column("date", "DATE (UTC)", func(l scp.Log) any { return fmtTime(l.Date) }),
	column("type", "TYPE", func(l scp.Log) any { return derefStr((*string)(l.Type)) }),
	column("key", "KEY", func(l scp.Log) any { return derefStr(l.LogKey) }),
	column("message", "MESSAGE", func(l scp.Log) any { return derefStr(l.Message) }),
)

func newServersCPUTopologyCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "cpu-topology <server> <sockets> <cores>",
		Short:             "Set CPU topology",
		Args:              cobra.ExactArgs(3),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			sockets, err := parseID(args[1], "sockets")
			if err != nil {
				return err
			}
			cores, err := parseID(args[2], "cores")
			if err != nil {
				return err
			}
			task, err := cc.client.SetCPUTopology(cc.ctx, id, sockets, cores)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newServersGuestAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "guest-agent <server>",
		Short:             "Get guest agent info",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			data, err := cc.client.GetGuestAgent(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, data, func() {
				printKV("Available", deref(data.GuestAgentAvailable))
				if data.GuestAgentData == nil {
					return
				}
				d := *data.GuestAgentData
				gstr := func(key string) string {
					if v, ok := d[key].(string); ok {
						return v
					}
					return ""
				}
				gfloat := func(key string) int {
					if v, ok := d[key].(float64); ok {
						return int(v)
					}
					return 0
				}
				fmt.Println()
				printKV(
					"Hostname", gstr("hostname"),
					"Kernel", gstr("os.kernel-release"),
					"Kernel Version", gstr("os.kernel-version"),
					"Architecture", gstr("os.machine"),
					"Timezone", fmt.Sprintf("%s (offset %d)", gstr("timezone.name"), gfloat("timezone.offset")),
					"Users", gfloat("user.count"),
				)
				ifCount := gfloat("if.count")
				if ifCount > 0 {
					fmt.Println("\nInterfaces:")
					t := newTable("NAME", "MAC", "ADDRESSES")
					for i := 0; i < ifCount; i++ {
						name := gstr(fmt.Sprintf("if.%d.name", i))
						mac := gstr(fmt.Sprintf("if.%d.hwaddr", i))
						addrCount := gfloat(fmt.Sprintf("if.%d.addr.count", i))
						addrs := make([]string, 0, addrCount)
						for j := 0; j < addrCount; j++ {
							addr := gstr(fmt.Sprintf("if.%d.addr.%d.addr", i, j))
							prefix := gfloat(fmt.Sprintf("if.%d.addr.%d.prefix", i, j))
							addrs = append(addrs, fmt.Sprintf("%s/%d", addr, prefix))
						}
						t.AppendRow(table.Row{name, mac, strings.Join(addrs, ", ")})
					}
					t.Render()
				}
			})
		},
	}
}

var imageFlavourDisplayer = newDisplayer(
	column("id", "FLAVOUR ID", func(f scp.ImageFlavour) any { return derefInt32(f.Id) }),
	column("image-id", "IMAGE ID", func(f scp.ImageFlavour) any {
		if f.Image != nil {
			return fmt.Sprintf("%d", derefInt32(f.Image.Id))
		}
		return ""
	}),
	column("image", "IMAGE", func(f scp.ImageFlavour) any {
		if f.Image != nil {
			return f.Image.Name
		}
		return ""
	}),
	column("description", "DESCRIPTION", func(f scp.ImageFlavour) any { return f.Text }),
)

func newServersImageFlavoursCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "image-flavours <server>",
		Short:             "List available OS image flavours",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			flavours, err := cc.client.ListImageFlavours(cc.ctx, id)
			if err != nil {
				return err
			}
			return imageFlavourDisplayer.print(cc, flavours)
		},
	}
	imageFlavourDisplayer.addFlags(cmd)
	return cmd
}

func newServersInstallImageCmd() *cobra.Command {
	var (
		hostname       string
		diskName       string
		locale         string
		timezone       string
		rootFullDisk   bool
		sshPassword    bool
		emailNotify    bool
		additionalUser string
		additionalPass string
		customScript   string
		sshKeyIDs      []int
		wait           bool
	)
	cmd := &cobra.Command{
		Use:               "install-image <server> <flavour-id>",
		Short:             "Install OS image on a server",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, imageFlavourIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			flavourID, err := parseID(args[1], "flavour-id")
			if err != nil {
				return err
			}
			setup := scp.ServerImageSetup{ImageFlavourId: new(flavourID)}
			if cmd.Flags().Changed("hostname") {
				setup.Hostname = &hostname
			}
			if cmd.Flags().Changed("disk") {
				setup.DiskName = &diskName
			}
			if cmd.Flags().Changed("locale") {
				setup.Locale = &locale
			}
			if cmd.Flags().Changed("timezone") {
				setup.Timezone = &timezone
			}
			if cmd.Flags().Changed("root-full-disk") {
				setup.RootPartitionFullDiskSize = &rootFullDisk
			}
			if cmd.Flags().Changed("ssh-password") {
				setup.SshPasswordAuthentication = &sshPassword
			}
			if cmd.Flags().Changed("email-notify") {
				setup.EmailToExecutingUser = &emailNotify
			}
			if cmd.Flags().Changed("additional-user") {
				setup.AdditionalUserUsername = &additionalUser
			}
			if cmd.Flags().Changed("additional-pass") {
				setup.AdditionalUserPassword = &additionalPass
			}
			if cmd.Flags().Changed("custom-script") {
				setup.CustomScript = &customScript
			}
			if len(sshKeyIDs) > 0 {
				ids := make([]int32, len(sshKeyIDs))
				for i, k := range sshKeyIDs {
					ids[i] = int32(k)
				}
				setup.SshKeyIds = &ids
			}
			if err := confirmRetype(cc, id, "install a fresh OS image, erasing all data on the server"); err != nil {
				return err
			}
			task, err := cc.client.InstallImage(cc.ctx, id, setup)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().StringVar(&hostname, "hostname", "", "hostname for the new installation")
	cmd.Flags().StringVar(&diskName, "disk", "", "disk name to install to")
	cmd.Flags().StringVar(&locale, "locale", "", "locale (e.g. en_US.UTF-8)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "timezone (e.g. UTC)")
	cmd.Flags().BoolVar(&rootFullDisk, "root-full-disk", false, "use full disk for root partition")
	cmd.Flags().BoolVar(&sshPassword, "ssh-password", false, "enable SSH password authentication")
	cmd.Flags().BoolVar(&emailNotify, "email-notify", false, "send email notification on completion")
	cmd.Flags().StringVar(&additionalUser, "additional-user", "", "additional user to create")
	cmd.Flags().StringVar(&additionalPass, "additional-pass", "", "password for additional user")
	cmd.Flags().StringVar(&customScript, "custom-script", "", "custom post-install script")
	cmd.Flags().IntSliceVar(&sshKeyIDs, "ssh-key-ids", nil, "SSH key IDs to install (comma-separated)")
	registerWaitFlags(cmd, &wait)

	registerFlagCompleter(cmd, "disk", diskNameCompletions)
	registerFlagCompleter(cmd, "ssh-key-ids", sshKeyIDCompletions)

	return cmd
}

func newServersInstallUserImageCmd() *cobra.Command {
	var (
		diskName    string
		emailNotify bool
		wait        bool
	)
	cmd := &cobra.Command{
		Use:               "install-user-image <server> <image-name>",
		Short:             "Install a user-uploaded image on a server",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, imageKeyCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			setup := scp.ServerUserImageSetup{UserImageName: args[1]}
			if cmd.Flags().Changed("disk") {
				setup.DiskName = &diskName
			}
			if cmd.Flags().Changed("email-notify") {
				setup.EmailNotification = &emailNotify
			}
			action := fmt.Sprintf(
				"install user image %q on server %s, overwriting its disk",
				args[1],
				serverLabelByID(cc, id),
			)
			if err := confirm(action); err != nil {
				return err
			}
			task, err := cc.client.InstallUserImage(cc.ctx, id, setup)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().StringVar(&diskName, "disk", "", "disk name to install to")
	cmd.Flags().BoolVar(&emailNotify, "email-notify", false, "send email notification on completion")
	registerWaitFlags(cmd, &wait)
	registerFlagCompleter(cmd, "disk", diskNameCompletions)
	return cmd
}

func newServersOptimizeStorageCmd() *cobra.Command {
	var disksFlag string
	var startAfter, wait bool
	cmd := &cobra.Command{
		Use:               "optimize-storage <server>",
		Short:             "Optimize disk storage",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			opts := &scp.OptimizeStorageOptions{}
			if disksFlag != "" {
				parts := strings.Split(disksFlag, ",")
				opts.Disks = parts
			}
			if startAfter {
				opts.StartAfterOptimization = new(true)
			}
			task, err := cc.client.OptimizeStorage(cc.ctx, id, opts)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().StringVar(&disksFlag, "disks", "", "comma-separated disk names to optimize (empty = all)")
	cmd.Flags().BoolVar(&startAfter, "start-after", false, "start server after optimization")
	registerWaitFlags(cmd, &wait)
	return cmd
}

type serverQemuStatus struct {
	ID            int32  `json:"id"`
	Nickname      string `json:"nickname"`
	State         string `json:"state"`
	LatestQemu    bool   `json:"latestQemu"`
	ConfigChanged bool   `json:"configChanged"`
}

var qemuStatusDisplayer = newDisplayer(
	column("id", "ID", func(r serverQemuStatus) any { return r.ID }),
	column("nickname", "NICKNAME", func(r serverQemuStatus) any { return r.Nickname }),
	column("state", "STATE", func(r serverQemuStatus) any { return r.State }),
	column("latest-qemu", "LATEST QEMU", func(r serverQemuStatus) any { return r.LatestQemu }),
	column("config-changed", "CONFIG CHANGED", func(r serverQemuStatus) any { return r.ConfigChanged }),
)

func newServersQemuStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qemu-status",
		Short: "Show QEMU version status for all servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			servers, err := cc.client.ListServers(cc.ctx, nil)
			if err != nil {
				return err
			}
			rows := []serverQemuStatus{}
			for _, s := range servers {
				detail, err := cc.client.GetServer(cc.ctx, derefInt32(s.Id), nil)
				if err != nil {
					return fmt.Errorf("get server %d: %w", derefInt32(s.Id), err)
				}
				row := serverQemuStatus{
					ID:       derefInt32(detail.Id),
					Nickname: derefStr(detail.Nickname),
				}
				if li := detail.ServerLiveInfo; li != nil {
					row.State = derefStr((*string)(li.State))
					row.LatestQemu = deref(li.LatestQemu)
					row.ConfigChanged = deref(li.ConfigChanged)
				}
				rows = append(rows, row)
			}
			return qemuStatusDisplayer.print(cc, rows)
		},
	}
	qemuStatusDisplayer.addFlags(cmd)
	return cmd
}

func newServersGPUDriverCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "gpu-driver <server>",
		Short:             "Get GPU driver download info for a server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			info, err := cc.client.GetGPUDriver(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, info, func() {
				printKV(
					"Filename", derefStr(info.Filename),
					"Download URL", derefStr(info.PresignedUrl),
					"URL Valid (hours)", deref(info.PresignedUrlValidityDurationInHours),
				)
			})
		},
	}
}
