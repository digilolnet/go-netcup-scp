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
	"strconv"
	"strings"
)

// serverRef is one row of the shared server-list cache, used by both server
// name resolution and shell completion.
type serverRef struct {
	ID       int32
	Name     string
	Nickname string
	Hostname string
}

// label returns the friendliest identifier for a server, for error messages.
func (r serverRef) label() string {
	if r.Nickname != "" {
		return fmt.Sprintf("%s (id %d)", r.Nickname, r.ID)
	}
	if r.Name != "" {
		return fmt.Sprintf("%s (id %d)", r.Name, r.ID)
	}
	return fmt.Sprintf("id %d", r.ID)
}

// encodeServerRef serializes a serverRef for the completion cache.
func encodeServerRef(r serverRef) string {
	return fmt.Sprintf("%d\t%s\t%s\t%s", r.ID, r.Name, r.Nickname, r.Hostname)
}

// decodeServerRef parses a cache entry written by encodeServerRef.
func decodeServerRef(s string) (serverRef, bool) {
	parts := strings.SplitN(s, "\t", 4)
	if len(parts) != 4 {
		return serverRef{}, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return serverRef{}, false
	}
	return serverRef{ID: int32(id), Name: parts[1], Nickname: parts[2], Hostname: parts[3]}, true
}

// fetchServerRefs fetches the server list from the API and refreshes the
// shared "servers" completion-cache entry.
func (cc *cmdContext) fetchServerRefs() ([]serverRef, error) {
	servers, err := cc.client.ListServers(cc.ctx, nil)
	if err != nil {
		return nil, err
	}
	refs := make([]serverRef, 0, len(servers))
	entries := make([]string, 0, len(servers))
	for _, s := range servers {
		r := serverRef{
			ID:       derefInt32(s.Id),
			Name:     derefStr(s.Name),
			Nickname: derefStr(s.Nickname),
			Hostname: derefStr(s.Hostname),
		}
		refs = append(refs, r)
		entries = append(entries, encodeServerRef(r))
	}
	newCompletionCache(cc.tokenFile).set("servers", entries)
	return refs, nil
}

// serverRefs returns the server list, served from the completion cache when
// fresh. The second return reports whether the data came from the cache (and
// may therefore be stale).
func (cc *cmdContext) serverRefs() ([]serverRef, bool, error) {
	if entries, ok := newCompletionCache(cc.tokenFile).get("servers"); ok {
		refs := make([]serverRef, 0, len(entries))
		for _, e := range entries {
			r, ok := decodeServerRef(e)
			if !ok {
				refs = nil
				break
			}
			refs = append(refs, r)
		}
		if refs != nil {
			return refs, true, nil
		}
	}
	refs, err := cc.fetchServerRefs()
	return refs, false, err
}

// matchServers returns the servers whose nickname, name, or hostname matches
// arg (case-insensitive). Exact matches win; only when there is no exact
// match do prefix matches count.
func matchServers(servers []serverRef, arg string) []serverRef {
	want := strings.ToLower(arg)
	exact := []serverRef{}
	prefix := []serverRef{}
	for _, s := range servers {
		fields := []string{
			strings.ToLower(s.Nickname),
			strings.ToLower(s.Name),
			strings.ToLower(s.Hostname),
		}
		matched := false
		for _, f := range fields {
			if f != "" && f == want {
				exact = append(exact, s)
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		for _, f := range fields {
			if strings.HasPrefix(f, want) {
				prefix = append(prefix, s)
				break
			}
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return prefix
}

// resolveServerArg resolves a server argument to a server id. An integer
// passes through unchanged; anything else is matched against the nickname,
// name, and hostname of the account's servers (unique prefixes work too).
func resolveServerArg(cc *cmdContext, arg string) (int32, error) {
	if id, err := strconv.ParseInt(arg, 10, 32); err == nil {
		return int32(id), nil
	}

	servers, cached, err := cc.serverRefs()
	if err != nil {
		return 0, fmt.Errorf("resolve server %q: %w", arg, err)
	}
	matches := matchServers(servers, arg)
	if len(matches) == 0 && cached {
		// The cached list may be stale; retry against the live API.
		if servers, err = cc.fetchServerRefs(); err != nil {
			return 0, fmt.Errorf("resolve server %q: %w", arg, err)
		}
		matches = matchServers(servers, arg)
	}

	switch len(matches) {
	case 1:
		return matches[0].ID, nil
	case 0:
		return 0, fmt.Errorf("no server named %q (tried nickname, name, and hostname)", arg)
	default:
		labels := make([]string, len(matches))
		for i, m := range matches {
			labels[i] = m.label()
		}
		last := labels[len(labels)-1]
		rest := strings.Join(labels[:len(labels)-1], ", ")
		return 0, fmt.Errorf("server %q is ambiguous: matched %s and %s", arg, rest, last)
	}
}
