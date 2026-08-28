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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newAuthLoginCmd(),
		newAuthLogoutCmd(),
		newAuthRefreshCmd(),
		newAuthStatusCmd(),
	)
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate via device flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(true)
			if err != nil {
				return err
			}
			defer cleanup()

			deviceAuth, err := cc.authMgr.InitiateDeviceAuth(cc.ctx)
			if err != nil {
				return fmt.Errorf("initiate device auth: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Open this URL to authenticate:\n  %s\n\n", deviceAuth.VerificationURIComplete)
			fmt.Fprintf(os.Stderr, "Or visit %s and enter code: %s\n\n", deviceAuth.VerificationURI, deviceAuth.UserCode)
			fmt.Fprintln(os.Stderr, "Waiting for authorization...")

			pollCtx, cancel := context.WithTimeout(cc.ctx, time.Duration(deviceAuth.ExpiresIn)*time.Second)
			defer cancel()

			tok, err := cc.authMgr.PollForToken(pollCtx, deviceAuth.DeviceCode, time.Duration(deviceAuth.Interval)*time.Second)
			if err != nil {
				return fmt.Errorf("poll for token: %w", err)
			}

			if err := saveToken(cc.tokenFile, tok); err != nil {
				return fmt.Errorf("save token: %w", err)
			}

			if cc.jsonOut {
				return printJSON(map[string]string{"status": "authenticated", "token_file": cc.tokenFile})
			}
			fmt.Printf("Authenticated. Token saved to %s\n", cc.tokenFile)
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke token and delete from disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(true)
			if err != nil {
				return err
			}
			defer cleanup()

			tok, err := loadToken(cc.tokenFile)
			if err != nil {
				return fmt.Errorf("load token: %w", err)
			}
			cc.authMgr.LoadToken(tok)
			if err := cc.authMgr.RevokeToken(cc.ctx, tok.RefreshToken); err != nil {
				return fmt.Errorf("revoke token: %w", err)
			}
			if err := os.Remove(cc.tokenFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove token file: %w", err)
			}
			if cc.jsonOut {
				return printJSON(map[string]string{"status": "logged out"})
			}
			fmt.Println("Logged out")
			return nil
		},
	}
}

func newAuthRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh access token using stored refresh token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(true)
			if err != nil {
				return err
			}
			defer cleanup()

			tok, err := loadToken(cc.tokenFile)
			if err != nil {
				return fmt.Errorf("load token: %w", err)
			}
			newTok, err := cc.authMgr.RefreshToken(cc.ctx, tok.RefreshToken)
			if err != nil {
				return fmt.Errorf("refresh token: %w", err)
			}
			if err := saveToken(cc.tokenFile, newTok); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			if cc.jsonOut {
				return printJSON(map[string]string{"status": "refreshed", "token_file": cc.tokenFile})
			}
			fmt.Printf("Token refreshed. Saved to %s\n", cc.tokenFile)
			return nil
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(true)
			if err != nil {
				return err
			}
			defer cleanup()

			tok, err := loadToken(cc.tokenFile)
			if err != nil {
				if cc.jsonOut {
					return printJSON(map[string]string{"status": "not authenticated"})
				}
				fmt.Println("not authenticated")
				return nil
			}
			cc.authMgr.LoadToken(tok)
			_, tokenErr := cc.authMgr.GetAccessToken()

			status := "authenticated"
			if tokenErr != nil {
				status = "token expired"
			}
			if cc.jsonOut {
				return printJSON(map[string]string{"status": status, "token_file": cc.tokenFile})
			}
			fmt.Printf("%s (token file: %s)\n", status, cc.tokenFile)
			return nil
		},
	}
}
