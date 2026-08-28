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

func newUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage user accounts",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newUsersGetCmd(),
		newUsersUpdateCmd(),
		newUsersLogsCmd(),
	)
	return cmd
}

func newUsersGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get your user account info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			user, err := cc.client.GetUser(cc.ctx, cc.userID)
			if err != nil {
				return err
			}
			return printResult(cc, user, func() {
				printKV(
					"ID", derefInt32(user.Id),
					"Username", derefStr(user.Username),
					"Name", derefStr(user.Firstname)+" "+derefStr(user.Lastname),
					"Company", derefStr(user.Company),
					"Email", derefStr(user.Email),
					"Language", derefStr(user.Language),
					"Timezone", derefStr(user.TimeZone),
					"Passwordless mode", deref(user.PasswordlessMode),
				)
			})
		},
	}
}

func newUsersUpdateCmd() *cobra.Command {
	var language, timezone string
	var passwordlessMode string // "true"/"false"/"" (unset)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update your user account settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			user, err := cc.client.GetUser(cc.ctx, cc.userID)
			if err != nil {
				return err
			}

			body := scp.UserSave{
				Language:         derefStr(user.Language),
				TimeZone:         derefStr(user.TimeZone),
				PasswordlessMode: user.PasswordlessMode,
			}

			if cmd.Flags().Changed("language") {
				body.Language = language
			}
			if cmd.Flags().Changed("timezone") {
				body.TimeZone = timezone
			}
			if cmd.Flags().Changed("passwordless-mode") {
				v, err := parseBool(passwordlessMode)
				if err != nil {
					return err
				}
				body.PasswordlessMode = &v
			}

			result, err := cc.client.UpdateUser(cc.ctx, cc.userID, body)
			if err != nil {
				return err
			}
			return printResult(cc, result, func() {
				printKV(
					"Language", result.Language,
					"Timezone", result.TimeZone,
					"Passwordless mode", deref(result.PasswordlessMode),
				)
			})
		},
	}
	cmd.Flags().StringVar(&language, "language", "", "language code (e.g. en, de)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "timezone (e.g. Europe/Berlin)")
	cmd.Flags().StringVar(&passwordlessMode, "passwordless-mode", "", "enable passwordless mode (true/false)")
	return cmd
}

func newUsersLogsCmd() *cobra.Command {
	var limit, offset int
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Get your user audit log",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			opts := &scp.GetUserLogsOptions{}
			if limit > 0 {
				opts.Limit = new(int32(limit))
			}
			if offset > 0 {
				opts.Offset = new(int32(offset))
			}
			logs, err := cc.client.GetUserLogs(cc.ctx, cc.userID, opts)
			if err != nil {
				return err
			}
			return printResult(cc, logs, func() {
				t := newTable("DATE (UTC)", "TYPE", "KEY", "MESSAGE")
				for _, l := range logs {
					t.AppendRow(table.Row{fmtTime(l.Date), derefStr((*string)(l.Type)), derefStr(l.LogKey), derefStr(l.Message)})
				}
				t.Render()
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	return cmd
}

// --- ssh-keys ---

func newSSHKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh-keys",
		Short: "Manage SSH keys",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newSSHKeysListCmd(),
		newSSHKeysCreateCmd(),
		newSSHKeysDeleteCmd(),
	)
	return cmd
}

func newSSHKeysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your SSH keys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			keys, err := cc.client.ListSSHKeys(cc.ctx, cc.userID)
			if err != nil {
				return err
			}
			return printResult(cc, keys, func() {
				t := newTable("ID", "NAME", "CREATED (UTC)")
				for _, k := range keys {
					t.AppendRow(table.Row{derefInt32(k.Id), k.Name, fmtTime(k.CreatedAt)})
				}
				t.Render()
			})
		},
	}
}

func newSSHKeysCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name> <public-key>",
		Short: "Add an SSH key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			key := scp.SSHKey{
				Name: args[0],
				Key:  args[1],
			}
			created, err := cc.client.CreateSSHKey(cc.ctx, cc.userID, key)
			if err != nil {
				return err
			}
			cc.invalidateCompletionCache("ssh-keys")
			return printResult(cc, created, func() {
				printKV("ID", derefInt32(created.Id), "Name", created.Name)
			})
		},
	}
}

func newSSHKeysDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "delete <key-id>",
		Short:             "Delete an SSH key",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(sshKeyIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			keyID, err := parseID(args[0], "key-id")
			if err != nil {
				return err
			}
			if err := confirm(fmt.Sprintf("delete SSH key %d", keyID)); err != nil {
				return err
			}
			if err := cc.client.DeleteSSHKey(cc.ctx, cc.userID, keyID); err != nil {
				return err
			}
			cc.invalidateCompletionCache("ssh-keys")
			printOK(cc)
			return nil
		},
	}
}
