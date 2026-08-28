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

// newServersRescueCmd toggles the rescue system, following the same
// `<property> <server> <on|off>` idiom as autostart and uefi. Whether
// rescue is currently active is part of `servers get`.
func newServersRescueCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:   "rescue <server> <on|off>",
		Short: "Enable or disable the rescue system",
		Long: "Enable or disable the rescue system. The server must be powered off;\n" +
			"enabling boots the rescue environment on the next start. With --wait, the\n" +
			"one-time rescue password is printed once activation completes.",
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

			var task *scp.TaskInfo
			if enabled {
				task, err = cc.client.ActivateRescueSystem(cc.ctx, id)
			} else {
				task, err = cc.client.DeactivateRescueSystem(cc.ctx, id)
			}
			if err != nil {
				return err
			}
			if err := printTaskAndWait(cc, task, wait); err != nil {
				return err
			}
			if enabled && wait && !cc.jsonOut {
				status, err := cc.client.GetRescueSystem(cc.ctx, id)
				if err != nil {
					return fmt.Errorf("rescue activated, but fetching the password failed: %w", err)
				}
				printKV("Rescue Password", derefStr(status.Password))
			}
			return nil
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}
