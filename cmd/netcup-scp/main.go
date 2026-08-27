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

// Command netcup-scp is a CLI for the netcup Server Control Panel API.
// All API operations are available. Use --json for machine-readable output.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
	"github.com/digilolnet/go-netcup-scp/pkg/scp/auth"
)

// cmdContext carries per-invocation state to every command handler.
type cmdContext struct {
	ctx       context.Context
	client    *scp.Client   // nil for noAuth commands
	authMgr   *auth.Manager // always set
	tokenFile string
	jsonOut   bool
	userID    int32 // extracted from JWT; 0 for noAuth commands
}

// rootFlags holds persistent flag values set on the root command.
var rootFlags struct {
	tokenFile   string
	jsonOut     bool
	contextName string
	configFile  string
}

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "netcup-scp",
		Short:         "CLI for the netcup Server Control Panel API",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("json") {
				if v, err := strconv.ParseBool(os.Getenv("NETCUP_SCP_JSON")); err == nil {
					rootFlags.jsonOut = v
				}
			}
			return resolveTokenFile()
		},
	}

	cmd.PersistentFlags().StringVar(&rootFlags.tokenFile, "token-file", "", "token file path (overrides context)")
	cmd.PersistentFlags().BoolVarP(&rootFlags.jsonOut, "json", "j", false, "output in JSON format (also: NETCUP_SCP_JSON=1)")
	cmd.PersistentFlags().StringVar(&rootFlags.contextName, "context", "", "named context to use for this invocation")
	cmd.PersistentFlags().StringVar(&rootFlags.configFile, "config", defaultConfigFilePath(), "path to config file")

	cmd.AddCommand(
		newAuthCmd(),
		newContextCmd(),
		newServersCmd(),
		newDisksCmd(),
		newSnapshotsCmd(),
		newInterfacesCmd(),
		newRDNSv4Cmd(),
		newRDNSv6Cmd(),
		newTasksCmd(),
		newUsersCmd(),
		newSSHKeysCmd(),
		newMetricsCmd(),
		newFailoverV4Cmd(),
		newFailoverV6Cmd(),
		newImagesCmd(),
		newRescueCmd(),
		newSystemCmd(),
		newVLansCmd(),
		newIsosCmd(),
		newUserIsosCmd(),
		newFirewallCmd(),
		newFirewallPoliciesCmd(),
	)

	return cmd
}

// resolveTokenFile determines the effective token file path from flags/env/config.
// Priority: --token-file > --context flag > NETCUP_SCP_CONTEXT env > current context in config > default path.
func resolveTokenFile() error {
	// Explicitly provided via flag — use as-is.
	if rootFlags.tokenFile != "" {
		return nil
	}

	contextName := rootFlags.contextName
	if contextName == "" {
		contextName = os.Getenv("NETCUP_SCP_CONTEXT")
	}

	if contextName == "" {
		// No explicit context: try the current context from config, then fall back.
		cfg, err := loadConfig(rootFlags.configFile)
		if err != nil {
			rootFlags.tokenFile = defaultTokenFilePath()
			return nil
		}
		if cfg.CurrentContext != "" {
			if entry := cfg.findContext(cfg.CurrentContext); entry != nil {
				rootFlags.tokenFile = entry.TokenFile
				return nil
			}
		}
		rootFlags.tokenFile = defaultTokenFilePath()
		return nil
	}

	// Explicit context name: must exist in config.
	cfg, err := loadConfig(rootFlags.configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	entry := cfg.findContext(contextName)
	if entry == nil {
		return fmt.Errorf("context %q not found; use 'context add' to create it", contextName)
	}
	rootFlags.tokenFile = entry.TokenFile
	return nil
}

// makeCmdContext creates a cmdContext for use in a command's RunE.
// If noAuth is true, the token is not loaded and client is nil (used by auth commands).
// Returns the context and a cleanup function that must be deferred.
func makeCmdContext(noAuth bool) (*cmdContext, func(), error) {
	ctx := context.Background()

	authMgr := auth.NewManager(
		auth.WithAutoRefresh(true),
		auth.WithTokenRefreshCallback(func(tok *auth.TokenResponse) {
			_ = saveToken(rootFlags.tokenFile, tok)
		}),
	)

	cc := &cmdContext{
		ctx:       ctx,
		authMgr:   authMgr,
		tokenFile: rootFlags.tokenFile,
		jsonOut:   rootFlags.jsonOut,
	}

	cleanup := func() {
		authMgr.Close()
		if cc.client != nil {
			cc.client.Close()
		}
	}

	if !noAuth {
		tok, err := loadToken(rootFlags.tokenFile)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("not authenticated: run 'auth login' first")
		}
		authMgr.LoadToken(tok)

		userID, err := jwtUserID(tok.AccessToken)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("resolve user ID from token: %w", err)
		}
		cc.userID = userID

		var clientOpts []scp.ClientOption
		if dir := os.Getenv("NETCUP_SCP_TRACE_DIR"); dir != "" {
			clientOpts = append(clientOpts, scp.WithTraceDir(dir))
		}
		client, err := scp.NewClient(authMgr, clientOpts...)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("create client: %w", err)
		}
		cc.client = client
	}

	return cc, cleanup, nil
}
