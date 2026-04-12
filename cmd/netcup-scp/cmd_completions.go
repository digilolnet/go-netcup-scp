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

// posCompleter is a completion function for one positional argument position.
// cc is already authenticated. args holds the positional args typed before this position.
type posCompleter func(cc *cmdContext, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// registerFlagCompleter wires a posCompleter to a named flag on cmd.
// It handles makeCmdContext setup/teardown, mirroring how makeCompleter works
// for positional arguments.
func registerFlagCompleter(cmd *cobra.Command, flag string, fn posCompleter) {
	_ = cmd.RegisterFlagCompletionFunc(flag, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cc, cleanup, err := makeCmdContext(false)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cleanup()
		return fn(cc, args, toComplete)
	})
}

// makeCompleter returns a ValidArgsFunction that routes to the right per-position
// completer based on how many positional args have already been typed.
// A nil entry means "no completion for this position" (disables file completion but
// shows nothing, so the user can still type freely).
func makeCompleter(fns ...posCompleter) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= len(fns) || fns[len(args)] == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cc, cleanup, err := makeCmdContext(false)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		defer cleanup()
		return fns[len(args)](cc, args, toComplete)
	}
}

// static returns a posCompleter that always offers the given fixed values.
func static(values ...string) posCompleter {
	return func(_ *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

// --- per-position completion functions ---

func serverIDCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	servers, err := cc.client.ListServers(cc.ctx, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(servers))
	for _, s := range servers {
		name := derefStr(s.Name)
		nickname := derefStr(s.Nickname)
		var desc string
		if nickname != "" {
			desc = nickname + " (" + name + ")"
		} else {
			desc = name
		}
		out = append(out, fmt.Sprintf("%d\t%s", derefInt32(s.Id), desc))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func macCompletions(cc *cmdContext, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	serverID, err := parseID(args[0], "server-id")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ifaces, err := cc.client.ListInterfaces(cc.ctx, serverID, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		out = append(out, derefStr(iface.Mac))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func deletableInterfaceMACCompletions(cc *cmdContext, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	serverID, err := parseID(args[0], "server-id")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ifaces, err := cc.client.ListInterfaces(cc.ctx, serverID, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		if !isPrimaryInterface(&iface) {
			out = append(out, derefStr(iface.Mac))
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func snapshotNameCompletions(cc *cmdContext, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	serverID, err := parseID(args[0], "server-id")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	snaps, err := cc.client.ListSnapshots(cc.ctx, serverID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(snaps))
	for _, s := range snaps {
		entry := derefStr(s.Name)
		if desc := derefStr(s.Description); desc != "" {
			entry += "\t" + desc
		}
		out = append(out, entry)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func diskNameCompletions(cc *cmdContext, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	serverID, err := parseID(args[0], "server-id")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	disks, err := cc.client.ListDisks(cc.ctx, serverID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(disks))
	for _, d := range disks {
		out = append(out, fmt.Sprintf("%s\t%d MiB", derefStr(d.Name), deref(d.CapacityInMiB)))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func diskDriverCompletions(cc *cmdContext, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	serverID, err := parseID(args[0], "server-id")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	drivers, err := cc.client.GetSupportedDiskDrivers(cc.ctx, serverID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, len(drivers))
	for i, d := range drivers {
		out[i] = string(d)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func taskUUIDCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	tasks, err := cc.client.ListTasks(cc.ctx, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, fmt.Sprintf("%s\t%s %s",
			derefStr(t.Uuid),
			derefStr((*string)(t.State)),
			derefStr(t.Name),
		))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func failoverV4IDCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	ips, err := cc.client.ListFailoverIPv4(cc.ctx, cc.userID, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(ips))
	for _, f := range ips {
		out = append(out, fmt.Sprintf("%d\t%s", derefInt32(f.Id), derefStr(f.Ip)))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func failoverV6IDCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	ips, err := cc.client.ListFailoverIPv6(cc.ctx, cc.userID, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(ips))
	for _, f := range ips {
		out = append(out, fmt.Sprintf("%d\t%s/%d", derefInt32(f.Id), derefStr(f.NetworkPrefix), derefInt32(f.NetworkPrefixLength)))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func vlanIDCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	vlans, err := cc.client.ListVLans(cc.ctx, cc.userID, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(vlans))
	for _, v := range vlans {
		out = append(out, fmt.Sprintf("%d\t%s", derefInt32(v.VlanId), derefStr(v.Name)))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func fwPolicyIDCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	policies, err := cc.client.ListFirewallPolicies(cc.ctx, cc.userID, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(policies))
	for _, p := range policies {
		out = append(out, fmt.Sprintf("%d\t%s", derefInt32(p.Id), derefStr(p.Name)))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func imageFlavourIDCompletions(cc *cmdContext, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	serverID, err := parseID(args[0], "server-id")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	flavours, err := cc.client.ListImageFlavours(cc.ctx, serverID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(flavours))
	for _, f := range flavours {
		img := ""
		if f.Image != nil {
			img = f.Image.Name
		}
		out = append(out, fmt.Sprintf("%d\t%s — %s", derefInt32(f.Id), img, f.Text))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func sshKeyIDCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	keys, err := cc.client.ListSSHKeys(cc.ctx, cc.userID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%d\t%s", derefInt32(k.Id), k.Name))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func userISOKeyCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	isos, err := cc.client.ListUserISOs(cc.ctx, cc.userID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(isos))
	for _, iso := range isos {
		out = append(out, derefStr(iso.Key))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func imageKeyCompletions(cc *cmdContext, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	images, err := cc.client.ListUserImages(cc.ctx, cc.userID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(images))
	for _, img := range images {
		out = append(out, derefStr(img.Key))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeContextNames completes context names from the config file (no auth needed).
func completeContextNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := loadConfig(rootFlags.configFile)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	out := make([]string, 0, len(cfg.Contexts))
	for _, c := range cfg.Contexts {
		desc := c.TokenFile
		if c.Name == cfg.CurrentContext {
			desc = "current  " + c.TokenFile
		}
		out = append(out, c.Name+"\t"+desc)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}
