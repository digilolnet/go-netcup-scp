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
	"github.com/spf13/cobra"
)

func newRescueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rescue",
		Short: "Manage rescue system",
	}
	cmd.AddCommand(
		newRescueGetCmd(),
		newRescueActivateCmd(),
		newRescueDeactivateCmd(),
	)
	return cmd
}

func newRescueGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <server-id>",
		Short:             "Get rescue system status",
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
			status, err := cc.client.GetRescueSystem(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, status, func() {
				printKV(
					"Active", deref(status.Active),
					"Password", derefStr(status.Password),
				)
			})
		},
	}
}

func newRescueActivateCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "activate <server-id>",
		Short:             "Boot server into rescue mode",
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
			task, err := cc.client.ActivateRescueSystem(cc.ctx, id)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	return cmd
}

func newRescueDeactivateCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "deactivate <server-id>",
		Short:             "Exit rescue mode and boot normally",
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
			task, err := cc.client.DeactivateRescueSystem(cc.ctx, id)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to complete")
	return cmd
}
