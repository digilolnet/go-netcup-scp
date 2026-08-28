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

func newSnapshotsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "Manage server snapshots",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newSnapshotsListCmd(),
		newSnapshotsGetCmd(),
		newSnapshotsCreateCmd(),
		newSnapshotsDeleteCmd(),
		newSnapshotsRevertCmd(),
		newSnapshotsDryRunCmd(),
		newSnapshotsExportCmd(),
	)
	return cmd
}

func newSnapshotsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "list <server>",
		Short:             "List snapshots",
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
			snaps, err := cc.client.ListSnapshots(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, snaps, func() {
				t := newTable("NAME", "STATE", "ONLINE", "DESCRIPTION", "DATE")
				for _, s := range snaps {
					t.AppendRow(table.Row{
						derefStr(s.Name),
						derefStr((*string)(s.State)),
						deref(s.Online),
						derefStr(s.Description),
						fmtTime(s.CreationTime),
					})
				}
				t.Render()
			})
		},
	}
}

func newSnapshotsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <server> <name>",
		Short:             "Get snapshot details",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, snapshotNameCompletions),
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
			snap, err := cc.client.GetSnapshot(cc.ctx, id, args[1])
			if err != nil {
				return err
			}
			return printResult(cc, snap, func() {
				printKV(
					"Name", derefStr(snap.Name),
					"Description", derefStr(snap.Description),
					"Created At (UTC)", fmtTime(snap.CreationTime),
					"Online", deref(snap.Online),
					"State", derefStr((*string)(snap.State)),
				)
			})
		},
	}
}

func newSnapshotsCreateCmd() *cobra.Command {
	var description string
	var wait bool
	cmd := &cobra.Command{
		Use:               "create <server> <name>",
		Short:             "Create a snapshot",
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
			task, err := cc.client.CreateSnapshot(cc.ctx, id, args[1], description)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "snapshot description")
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newSnapshotsDeleteCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "delete <server> <name>",
		Short:             "Delete a snapshot",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, snapshotNameCompletions),
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
			action := fmt.Sprintf("delete snapshot %q of server %s", args[1], serverLabelByID(cc, id))
			if err := confirm(action); err != nil {
				return err
			}
			task, err := cc.client.DeleteSnapshot(cc.ctx, id, args[1])
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newSnapshotsRevertCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "revert <server> <name>",
		Short:             "Revert server to a snapshot",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, snapshotNameCompletions),
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
			action := fmt.Sprintf(
				"revert server %s to snapshot %q, discarding its current disk state",
				serverLabelByID(cc, id),
				args[1],
			)
			if err := confirm(action); err != nil {
				return err
			}
			task, err := cc.client.RevertSnapshot(cc.ctx, id, args[1])
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newSnapshotsDryRunCmd() *cobra.Command {
	var online bool
	var disk string
	cmd := &cobra.Command{
		Use:               "dry-run <server>",
		Short:             "Check if snapshot can be created",
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
			var diskPtr *string
			if disk != "" {
				diskPtr = &disk
			}
			errs, err := cc.client.DryRunSnapshot(cc.ctx, id, online, diskPtr)
			if err != nil {
				return err
			}
			return printResult(cc, errs, func() {
				if len(errs) == 0 {
					fmt.Println("snapshot can be created")
					return
				}
				fmt.Println("blocking issues:")
				for _, e := range errs {
					fmt.Printf("  %s: %s\n", derefStr(e.Code), derefStr(e.Message))
				}
			})
		},
	}
	cmd.Flags().BoolVar(&online, "online", true, "check for online snapshot (default)")
	cmd.Flags().StringVar(&disk, "disk", "", "disk name (required for offline snapshots)")
	return cmd
}

func newSnapshotsExportCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "export <server> <name>",
		Short:             "Export a snapshot",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, snapshotNameCompletions),
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
			task, err := cc.client.ExportSnapshot(cc.ctx, id, args[1])
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}
