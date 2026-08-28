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

func newFirewallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firewall",
		Short: "Manage per-interface firewall configuration",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newFirewallGetCmd(),
		newFirewallUpdateCmd(),
		newFirewallReapplyCmd(),
		newFirewallRestoreCopiedCmd(),
		newFirewallClearCmd(),
		newFirewallActiveCmd(),
	)
	return cmd
}

func newFirewallGetCmd() *cobra.Command {
	var checkConsistency bool
	cmd := &cobra.Command{
		Use:               "get <server> <mac>",
		Short:             "Get firewall config for an interface",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			serverID, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			opts := &scp.GetFirewallOptions{}
			if checkConsistency {
				opts.ConsistencyCheck = new(true)
			}
			fw, err := cc.client.GetFirewall(cc.ctx, serverID, args[1], opts)
			if err != nil {
				return err
			}
			return printResult(cc, fw, func() {
				printKV(
					"Active", deref(fw.Active),
					"Consistent", deref(fw.Consistent),
				)
				if fw.UserPolicies != nil && len(*fw.UserPolicies) > 0 {
					fmt.Printf("\nUser policies:\n")
					t := newTable("ID", "NAME")
					for _, p := range *fw.UserPolicies {
						t.AppendRow(table.Row{derefInt32(p.Id), derefStr(p.Name)})
					}
					t.Render()
				}
				if fw.CopiedPolicies != nil && len(*fw.CopiedPolicies) > 0 {
					fmt.Printf("\nCopied policies:\n")
					t := newTable("ID", "NAME")
					for _, p := range *fw.CopiedPolicies {
						t.AppendRow(table.Row{derefInt32(p.Id), derefStr(p.Name)})
					}
					t.Render()
				}
			})
		},
	}
	cmd.Flags().BoolVar(&checkConsistency, "check-consistency", false, "verify rules are applied correctly")
	return cmd
}

func newFirewallUpdateCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "update <server> <mac>",
		Short:             "Replace firewall config for an interface (JSON body from stdin)",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			serverID, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			var body scp.ServerFirewallSave
			if err := readJSONStdin(&body); err != nil {
				return err
			}
			task, err := cc.client.UpdateFirewall(cc.ctx, serverID, args[1], body)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newFirewallReapplyCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "reapply <server> <mac>",
		Short:             "Re-apply firewall rules without changing configuration",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			serverID, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			task, err := cc.client.ReapplyFirewall(cc.ctx, serverID, args[1])
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newFirewallRestoreCopiedCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "restore-copied-policies <server> <mac>",
		Short:             "Restore copied firewall policies for an interface",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			serverID, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			task, err := cc.client.RestoreCopiedFirewallPolicies(cc.ctx, serverID, args[1])
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newFirewallClearCmd() *cobra.Command {
	var keepCopied, wait bool
	cmd := &cobra.Command{
		Use:               "clear <server> <mac>",
		Short:             "Remove all policies from an interface",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			serverID, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			mac := args[1]

			action := fmt.Sprintf("remove all firewall policies from %s on server %s", mac, serverLabelByID(cc, serverID))
			if err := confirm(action); err != nil {
				return err
			}
			tasks, err := cc.client.ClearFirewall(cc.ctx, serverID, mac, keepCopied)
			if err != nil {
				return err
			}
			for _, task := range tasks {
				if err := printTaskAndWait(cc, task, wait); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&keepCopied, "keep-copied", false, "restore netcup copied policies after clearing")
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newFirewallActiveCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "active <server> <mac> <on|off>",
		Short:             "Enable or disable the firewall for an interface",
		Args:              cobra.ExactArgs(3),
		ValidArgsFunction: makeCompleter(serverIDCompletions, macCompletions, static("on", "off")),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			serverID, err := resolveServerArg(cc, args[0])
			if err != nil {
				return err
			}
			active, err := parseBool(args[2])
			if err != nil {
				return err
			}
			mac := args[1]

			task, err := cc.client.SetFirewallActive(cc.ctx, serverID, mac, active)
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

// --- firewall-policies ---

func newFirewallPoliciesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firewall-policies",
		Short: "Manage user-defined firewall policies",
		// A parent without RunE never runs Args validation, so an unknown
		// subcommand would silently print help with exit 0.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newFWPoliciesListCmd(),
		newFWPoliciesGetCmd(),
		newFWPoliciesCreateCmd(),
		newFWPoliciesAddRuleCmd(),
		newFWPoliciesUpdateCmd(),
		newFWPoliciesDeleteCmd(),
	)
	return cmd
}

func newFWPoliciesListCmd() *cobra.Command {
	var limit, offset int
	var q string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your firewall policies",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			opts := &scp.ListFirewallPoliciesOptions{}
			if q != "" {
				opts.Q = &q
			}
			if limit > 0 {
				opts.Limit = new(int32(limit))
			}
			if offset > 0 {
				opts.Offset = new(int32(offset))
			}
			policies, err := cc.client.ListFirewallPolicies(cc.ctx, cc.userID, opts)
			if err != nil {
				return err
			}
			return fwPolicyDisplayer.print(cc, policies)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	cmd.Flags().StringVar(&q, "q", "", "search query")
	fwPolicyDisplayer.addFlags(cmd)
	return cmd
}

var fwPolicyDisplayer = newDisplayer(
	column("id", "ID", func(p scp.FirewallPolicy) any { return derefInt32(p.Id) }),
	column("name", "NAME", func(p scp.FirewallPolicy) any { return derefStr(p.Name) }),
)

func newFWPoliciesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get <policy-id>",
		Short:             "Get firewall policy details",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(fwPolicyIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			policyID, err := parseID(args[0], "policy-id")
			if err != nil {
				return err
			}
			policy, err := cc.client.GetFirewallPolicy(cc.ctx, cc.userID, policyID)
			if err != nil {
				return err
			}
			return printResult(cc, policy, func() {
				printKV(
					"ID", derefInt32(policy.Id),
					"Name", derefStr(policy.Name),
					"Description", derefStr(policy.Description),
				)
				if policy.Rules != nil {
					fmt.Printf("\nRules (%d):\n", len(*policy.Rules))
					for _, r := range *policy.Rules {
						fmt.Printf("  [%s] %s  sources:%v -> destinations:%v  src-ports:%s dst-ports:%s\n",
							string(r.Direction),
							string(r.Protocol),
							r.Sources,
							r.Destinations,
							derefStr(r.SourcePorts),
							derefStr(r.DestinationPorts),
						)
					}
				}
			})
		},
	}
	return cmd
}

func newFWPoliciesCreateCmd() *cobra.Command {
	var description string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a firewall policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			body := scp.FirewallPolicySave{Name: args[0]}
			if description != "" {
				body.Description = &description
			}
			policy, err := cc.client.CreateFirewallPolicy(cc.ctx, cc.userID, body)
			if err != nil {
				return err
			}
			cc.invalidateCompletionCache("fw-policies")
			return printResult(cc, policy, func() {
				printKV("ID", derefInt32(policy.Id), "Name", derefStr(policy.Name))
			})
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "optional description")
	return cmd
}

func newFWPoliciesAddRuleCmd() *cobra.Command {
	var direction, protocol, action, description string
	var srcPorts, dstPorts, sources, destinations string
	var wait bool
	cmd := &cobra.Command{
		Use:               "add-rule <policy-id>",
		Short:             "Add a rule to a firewall policy",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(fwPolicyIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			policyID, err := parseID(args[0], "policy-id")
			if err != nil {
				return err
			}
			rule := scp.FirewallRule{
				Direction: scp.FirewallRuleDirection(direction),
				Protocol:  scp.FirewallProtocol(protocol),
				Action:    scp.FirewallAction(action),
			}
			if description != "" {
				rule.Description = &description
			}
			if srcPorts != "" {
				rule.SourcePorts = &srcPorts
			}
			if dstPorts != "" {
				rule.DestinationPorts = &dstPorts
			}
			if sources != "" {
				parts := splitCSV(sources)
				rule.Sources = &parts
			}
			if destinations != "" {
				parts := splitCSV(destinations)
				rule.Destinations = &parts
			}

			result, err := cc.client.AddFirewallRule(cc.ctx, cc.userID, policyID, rule)
			if err != nil {
				return err
			}
			if err := printResult(cc, result, func() {
				if result.FirewallPolicy != nil {
					p := result.FirewallPolicy
					fmt.Printf("Policy %d %q now has %d rule(s)\n", derefInt32(p.Id), derefStr(p.Name), len(deref(p.Rules)))
				}
				if result.TaskInfo != nil {
					printKV("Task UUID", derefStr(result.TaskInfo.Uuid))
				}
			}); err != nil {
				return err
			}
			if wait && result.TaskInfo != nil && result.TaskInfo.Uuid != nil {
				return waitTask(cc, *result.TaskInfo.Uuid)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&direction, "direction", "", "INGRESS or EGRESS (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "TCP, UDP, ICMP, or ICMPv6 (required)")
	cmd.Flags().StringVar(&action, "action", "", "ACCEPT or DROP (required)")
	for _, f := range []string{"direction", "protocol", "action"} {
		_ = cmd.MarkFlagRequired(f)
	}
	cmd.Flags().StringVar(&description, "description", "", "rule description")
	cmd.Flags().StringVar(&srcPorts, "src-ports", "", "source ports (e.g. 22 or 1024-65535)")
	cmd.Flags().StringVar(&dstPorts, "dst-ports", "", "destination ports (e.g. 443 or 8080-8090)")
	cmd.Flags().StringVar(&sources, "sources", "", "comma-separated source IPs/CIDRs (empty = any)")
	cmd.Flags().StringVar(&destinations, "destinations", "", "comma-separated destination IPs/CIDRs (empty = any)")
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newFWPoliciesUpdateCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "update <policy-id>",
		Short:             "Update a firewall policy (JSON body from stdin)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(fwPolicyIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			policyID, err := parseID(args[0], "policy-id")
			if err != nil {
				return err
			}
			var body scp.FirewallPolicySave
			if err := readJSONStdin(&body); err != nil {
				return err
			}
			result, err := cc.client.UpdateFirewallPolicy(cc.ctx, cc.userID, policyID, body)
			if err != nil {
				return err
			}
			cc.invalidateCompletionCache("fw-policies")
			if err := printResult(cc, result, func() {
				if result.FirewallPolicy != nil {
					printKV("ID", derefInt32(result.FirewallPolicy.Id), "Name", result.FirewallPolicy.Name)
				}
				if result.TaskInfo != nil {
					printKV("Task UUID", derefStr(result.TaskInfo.Uuid))
				}
			}); err != nil {
				return err
			}
			if wait && result.TaskInfo != nil && result.TaskInfo.Uuid != nil {
				return waitTask(cc, *result.TaskInfo.Uuid)
			}
			return nil
		},
	}
	registerWaitFlags(cmd, &wait)
	return cmd
}

func newFWPoliciesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "delete <policy-id>",
		Short:             "Delete a firewall policy",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(fwPolicyIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			policyID, err := parseID(args[0], "policy-id")
			if err != nil {
				return err
			}
			if err := confirm(fmt.Sprintf("delete firewall policy %d", policyID)); err != nil {
				return err
			}
			if err := cc.client.DeleteFirewallPolicy(cc.ctx, cc.userID, policyID); err != nil {
				return err
			}
			cc.invalidateCompletionCache("fw-policies")
			printOK(cc)
			return nil
		},
	}
}
