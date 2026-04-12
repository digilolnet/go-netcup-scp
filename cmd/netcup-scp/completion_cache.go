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
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

const completionCacheTTL = 5 * time.Minute

// serverKey builds a cache key for a server-scoped resource, e.g. serverKey("disks-", 42) → "disks-42".
func serverKey(prefix string, id int32) string {
	return prefix + strconv.Itoa(int(id))
}

type completionCache struct {
	dir string
}

type completionCacheEntry struct {
	ExpiresAt time.Time `json:"expires_at"`
	Entries   []string  `json:"entries"`
}

// newCompletionCache returns a cache scoped to the given token file (i.e. per context).
func newCompletionCache(tokenFile string) *completionCache {
	h := fnv.New32a()
	_, _ = h.Write([]byte(tokenFile))

	baseDir, err := os.UserCacheDir()
	if err != nil {
		baseDir = os.TempDir()
	}

	dir := filepath.Join(baseDir, "netcup-scp", "completions", fmt.Sprintf("%08x", h.Sum32()))
	return &completionCache{dir: dir}
}

func (c *completionCache) get(key string) ([]string, bool) {
	data, err := os.ReadFile(filepath.Join(c.dir, key+".json"))
	if err != nil {
		return nil, false
	}
	var entry completionCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Entries, true
}

func (c *completionCache) delete(key string) {
	_ = os.Remove(filepath.Join(c.dir, key+".json"))
}

func (c *completionCache) set(key string, entries []string) {
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return
	}
	data, err := json.Marshal(completionCacheEntry{
		ExpiresAt: time.Now().Add(completionCacheTTL),
		Entries:   entries,
	})
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(c.dir, key+".json"), data, 0o600)
}

// invalidateCompletionCache removes one or more cache entries so the next tab completion
// fetches fresh data. Errors are silently ignored — a stale entry just means slightly
// stale completions until the TTL expires.
func (cc *cmdContext) invalidateCompletionCache(keys ...string) {
	cache := newCompletionCache(cc.tokenFile)
	for _, key := range keys {
		cache.delete(key)
	}
}

// completionWithCache fetches completion entries via fetch, caching the result under key
// for completionCacheTTL. Cache misses and errors in fetch are transparent to callers.
func (cc *cmdContext) completionWithCache(key string, fetch func() ([]string, error)) ([]string, cobra.ShellCompDirective) {
	cache := newCompletionCache(cc.tokenFile)
	if entries, ok := cache.get(key); ok {
		return entries, cobra.ShellCompDirectiveNoFileComp
	}
	entries, err := fetch()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	cache.set(key, entries)
	return entries, cobra.ShellCompDirectiveNoFileComp
}
