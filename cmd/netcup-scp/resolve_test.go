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
	"strings"
	"testing"
)

var testServers = []serverRef{
	{ID: 1, Name: "v9001", Nickname: "web-prod", Hostname: "web.example.com"},
	{ID: 2, Name: "v9002", Nickname: "web-staging", Hostname: "staging.example.com"},
	{ID: 3, Name: "v9003", Nickname: "", Hostname: "db.example.com"},
}

func TestMatchServers(t *testing.T) {
	tests := []struct {
		arg  string
		want []int32
	}{
		{"web-prod", []int32{1}},       // exact nickname
		{"WEB-PROD", []int32{1}},       // case-insensitive
		{"v9003", []int32{3}},          // exact name
		{"db.example.com", []int32{3}}, // exact hostname
		{"web", []int32{1, 2}},         // ambiguous prefix
		{"web-s", []int32{2}},          // unique prefix
		{"staging", []int32{2}},        // hostname prefix
		{"nope", nil},                  // no match
		{"web-staging", []int32{2}},    // exact beats prefix rivals
	}
	for _, tc := range tests {
		got := matchServers(testServers, tc.arg)
		ids := make([]int32, len(got))
		for i, s := range got {
			ids[i] = s.ID
		}
		if len(ids) != len(tc.want) {
			t.Errorf("matchServers(%q) = %v, want %v", tc.arg, ids, tc.want)
			continue
		}
		for i := range ids {
			if ids[i] != tc.want[i] {
				t.Errorf("matchServers(%q) = %v, want %v", tc.arg, ids, tc.want)
				break
			}
		}
	}
}

func TestServerRefCodec(t *testing.T) {
	for _, r := range testServers {
		got, ok := decodeServerRef(encodeServerRef(r))
		if !ok || got != r {
			t.Errorf("roundtrip %+v = %+v (ok=%v)", r, got, ok)
		}
	}
	if _, ok := decodeServerRef("111007\tdemo-c1 (v9001)"); ok {
		t.Error("legacy two-field cache entry must not decode")
	}
	if _, ok := decodeServerRef("notanid\ta\tb\tc"); ok {
		t.Error("non-integer id must not decode")
	}
}

func TestResolveServerArgIntegerPassthrough(t *testing.T) {
	// Integers must pass through without touching the API (cc is unused).
	id, err := resolveServerArg(nil, "111007")
	if err != nil || id != 111007 {
		t.Fatalf("resolveServerArg(111007) = %d, %v", id, err)
	}
}

func TestServerRefLabel(t *testing.T) {
	if got := testServers[0].label(); !strings.Contains(got, "web-prod") {
		t.Errorf("label should prefer nickname, got %q", got)
	}
	if got := testServers[2].label(); !strings.Contains(got, "v9003") {
		t.Errorf("label should fall back to name, got %q", got)
	}
}
