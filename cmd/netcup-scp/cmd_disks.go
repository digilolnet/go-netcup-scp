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

func newDisksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disks",
		Short: "Manage server disks",
	}
	cmd.AddCommand(
		newDisksListCmd(),
		newDisksGetCmd(),
		newDisksFormatCmd(),
		newDisksSetDriverCmd(),
		newDisksSupportedDriversCmd(),
	)
	return cmd
}

func newDisksListCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "list <server-id>",
		Short:             "List disks attached to a server",
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
			disks, err := cc.client.ListDisks(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, disks, func() {
				t := newTable("NAME", "CAPACITY (MiB)", "ALLOCATION (MiB)", "DRIVER")
				for _, d := range disks {
					t.AppendRow(table.Row{derefStr(d.Name), deref(d.CapacityInMiB), deref(d.AllocationInMiB), derefStr((*string)(d.StorageDriver))})
				}
				t.Render()
			})
		},
	}
}

func newDisksGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <server-id> <name>",
		Short:             "Get disk details",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, diskNameCompletions),
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
			disk, err := cc.client.GetDisk(cc.ctx, id, args[1])
			if err != nil {
				return err
			}
			return printResult(cc, disk, func() {
				printKV(
					"Name", derefStr(disk.Name),
					"Capacity (MiB)", deref(disk.CapacityInMiB),
					"Allocation (MiB)", deref(disk.AllocationInMiB),
					"Driver", derefStr((*string)(disk.StorageDriver)),
				)
			})
		},
	}
}

func newDisksFormatCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "format <server-id> <name>",
		Short:             "Format a disk (destroys data)",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, diskNameCompletions),
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
			if err := cc.client.FormatDisk(cc.ctx, id, args[1]); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}

func newDisksSetDriverCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "set-driver <server-id> <driver>",
		Short:             "Change storage driver for all disks",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, diskDriverCompletions),
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
			if err := cc.client.SetDiskDriver(cc.ctx, id, scp.StorageDriver(args[1])); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}

func newDisksSupportedDriversCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "supported-drivers <server-id>",
		Short:             "List supported storage drivers",
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
			drivers, err := cc.client.GetSupportedDiskDrivers(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, drivers, func() {
				for _, d := range drivers {
					fmt.Println(string(d))
				}
			})
		},
	}
}
