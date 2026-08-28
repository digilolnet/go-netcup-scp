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
)

func newSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "System-level operations",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newSystemPingCmd(),
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
