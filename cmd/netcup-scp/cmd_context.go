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

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage named account contexts (multi-account support)",
	}
	cmd.AddCommand(
		newContextListCmd(),
		newContextCurrentCmd(),
		newContextUseCmd(),
		newContextAddCmd(),
		newContextDeleteCmd(),
	)
	return cmd
}

func newContextListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(rootFlags.configFile)
			if err != nil {
				return err
			}
			if rootFlags.jsonOut {
				return printJSON(cfg.Contexts)
			}
			if len(cfg.Contexts) == 0 {
				fmt.Println("no contexts configured")
				return nil
			}
			fmt.Printf("  %-30s  %s\n", "NAME", "TOKEN FILE")
			for _, c := range cfg.Contexts {
				marker := " "
				if c.Name == cfg.CurrentContext {
					marker = "*"
				}
				fmt.Printf("%s %-30s  %s\n", marker, c.Name, c.TokenFile)
			}
			return nil
		},
	}
}

func newContextCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Show the currently active context",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(rootFlags.configFile)
			if err != nil {
				return err
			}
			if cfg.CurrentContext == "" && !rootFlags.jsonOut {
				fmt.Println("no current context set")
				return nil
			}
			if rootFlags.jsonOut {
				return printJSON(map[string]string{"current_context": cfg.CurrentContext})
			}
			fmt.Println(cfg.CurrentContext)
			return nil
		},
	}
}

func newContextUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "use <name>",
		Short:             "Set the current context",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContextNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(rootFlags.configFile)
			if err != nil {
				return err
			}
			if cfg.findContext(args[0]) == nil {
				return fmt.Errorf("context %q not found; use 'context add' to create it", args[0])
			}
			cfg.CurrentContext = args[0]
			if err := saveConfig(rootFlags.configFile, cfg); err != nil {
				return err
			}
			if rootFlags.jsonOut {
				return printJSON(map[string]string{"current_context": cfg.CurrentContext})
			}
			fmt.Printf("Switched to context %q\n", args[0])
			return nil
		},
	}
}

func newContextAddCmd() *cobra.Command {
	var tokenFile string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := loadConfig(rootFlags.configFile)
			if err != nil {
				return err
			}
			if cfg.findContext(name) != nil {
				return fmt.Errorf("context %q already exists", name)
			}
			if tokenFile == "" {
				tokenFile = defaultContextTokenFile(name)
			}
			entry := ContextEntry{Name: name, TokenFile: tokenFile}
			cfg.Contexts = append(cfg.Contexts, entry)
			if cfg.CurrentContext == "" {
				cfg.CurrentContext = name
			}
			if err := saveConfig(rootFlags.configFile, cfg); err != nil {
				return err
			}
			if rootFlags.jsonOut {
				return printJSON(entry)
			}
			fmt.Printf("Added context %q (token file: %s)\n", name, tokenFile)
			if cfg.CurrentContext == name {
				fmt.Println("Set as current context.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(
		&tokenFile,
		"token-file",
		"",
		"token file path (default: ~/.config/netcup-scp/contexts/<name>.json)",
	)
	return cmd
}

func newContextDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "delete <name>",
		Short:             "Delete a context",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContextNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := loadConfig(rootFlags.configFile)
			if err != nil {
				return err
			}
			found := false
			var kept []ContextEntry
			for _, c := range cfg.Contexts {
				if c.Name == name {
					found = true
				} else {
					kept = append(kept, c)
				}
			}
			if !found {
				return fmt.Errorf("context %q not found", name)
			}
			cfg.Contexts = kept
			if cfg.CurrentContext == name {
				cfg.CurrentContext = ""
				if len(kept) > 0 {
					cfg.CurrentContext = kept[0].Name
				}
			}
			if err := saveConfig(rootFlags.configFile, cfg); err != nil {
				return err
			}
			if rootFlags.jsonOut {
				return printJSON(map[string]string{"deleted": name})
			}
			fmt.Printf("Deleted context %q\n", name)
			return nil
		},
	}
}
