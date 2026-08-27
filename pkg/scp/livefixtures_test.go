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

package scp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/digilolnet/go-netcup-scp/internal/generated"
)

// liveFixture mirrors the schema written by traceTransport (see trace.go):
// real request/response pairs captured against the production netcup SCP API
// (sanitized), so these tests exercise the wrappers against actual API shapes,
// status codes and content types rather than hand-written guesses.
type liveFixture struct {
	Method       string          `json:"method"`
	URL          string          `json:"url"`
	Status       int             `json:"status"`
	ContentType  string          `json:"contentType"`
	ResponseBody json.RawMessage `json:"responseBody"`
}

// fixtureHandler replays the named fixture from testdata/live for any request.
func fixtureHandler(t *testing.T, name string) http.HandlerFunc {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "live", name))
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	var fx liveFixture
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if fx.ContentType != "" {
			w.Header().Set("Content-Type", fx.ContentType)
		}
		w.WriteHeader(fx.Status)
		if len(fx.ResponseBody) > 0 {
			_, _ = w.Write(fx.ResponseBody)
		}
	}
}

func TestLiveFixtures(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		fixture string
		wantErr bool
		call    func(c *Client) (any, error)
		check   func(t *testing.T, got any)
	}{
		{
			fixture: "GET_servers.json",
			call:    func(c *Client) (any, error) { return c.ListServers(ctx, nil) },
			check: func(t *testing.T, got any) {
				servers := got.([]generated.ServerListMinimal)
				if len(servers) != 8 {
					t.Errorf("want 8 servers, got %d", len(servers))
				}
			},
		},
		{
			fixture: "GET_servers_111007.json",
			call:    func(c *Client) (any, error) { return c.GetServer(ctx, 111007, nil) },
			check: func(t *testing.T, got any) {
				srv := got.(*generated.Server)
				if deref(srv.Nickname) != "demo-k8s-wrk-02" {
					t.Errorf("unexpected nickname %q", deref(srv.Nickname))
				}
			},
		},
		{
			fixture: "GET_servers_111007_logs.json",
			call:    func(c *Client) (any, error) { return c.GetServerLogs(ctx, 111007, nil) },
		},
		{
			fixture: "GET_servers_111007_guest-agent.json",
			call:    func(c *Client) (any, error) { return c.GetGuestAgent(ctx, 111007) },
		},
		{
			fixture: "GET_servers_111007_imageflavours.json",
			call:    func(c *Client) (any, error) { return c.ListImageFlavours(ctx, 111007) },
		},
		{
			fixture: "GET_servers_111007_disks.json",
			call:    func(c *Client) (any, error) { return c.ListDisks(ctx, 111007) },
			check: func(t *testing.T, got any) {
				disks := got.([]generated.Disk)
				if len(disks) != 1 || deref(disks[0].Name) != "vda" {
					t.Errorf("want single disk vda, got %+v", disks)
				}
			},
		},
		{
			fixture: "GET_servers_111007_disks_vda.json",
			call:    func(c *Client) (any, error) { return c.GetDisk(ctx, 111007, "vda") },
		},
		{
			fixture: "GET_servers_111007_disks_supported-drivers.json",
			call:    func(c *Client) (any, error) { return c.GetSupportedDiskDrivers(ctx, 111007) },
		},
		{
			fixture: "GET_servers_111007_interfaces.json",
			call:    func(c *Client) (any, error) { return c.ListInterfaces(ctx, 111007, nil) },
			check: func(t *testing.T, got any) {
				ifaces := got.([]generated.Interface)
				if len(ifaces) != 2 {
					t.Errorf("want 2 interfaces, got %d", len(ifaces))
				}
			},
		},
		{
			fixture: "GET_servers_111007_interfaces_02_00_00_97_47_9f.json",
			call: func(c *Client) (any, error) {
				return c.GetInterface(ctx, 111007, "02:00:00:97:47:9f", nil)
			},
		},
		{
			fixture: "GET_servers_111007_interfaces_02_00_00_97_47_9f_firewall.json",
			call: func(c *Client) (any, error) {
				return c.GetFirewall(ctx, 111007, "02:00:00:97:47:9f", nil)
			},
		},
		{
			fixture: "GET_servers_111007_metrics_cpu.json",
			call: func(c *Client) (any, error) {
				return c.GetCPUMetrics(ctx, 111007, nil)
			},
		},
		{
			fixture: "GET_servers_111007_metrics_disk.json",
			call: func(c *Client) (any, error) {
				return c.GetDiskMetrics(ctx, 111007, nil)
			},
		},
		{
			fixture: "GET_servers_111007_metrics_network.json",
			call: func(c *Client) (any, error) {
				return c.GetNetworkMetrics(ctx, 111007, nil)
			},
		},
		{
			fixture: "GET_servers_111007_snapshots.json",
			call:    func(c *Client) (any, error) { return c.ListSnapshots(ctx, 111007) },
		},
		{
			// Live API refuses snapshots on UEFI servers with a 400.
			fixture: "POST_servers_111007_snapshots.json",
			wantErr: true,
			call: func(c *Client) (any, error) {
				return c.CreateSnapshot(ctx, 111007, "audit-test", "")
			},
		},
		{
			fixture: "GET_servers_111007_rescuesystem.json",
			call:    func(c *Client) (any, error) { return c.GetRescueSystem(ctx, 111007) },
		},
		{
			// Real 202 with TaskInfo from the fixed Content-Type path.
			fixture: "POST_servers_111007_rescuesystem.json",
			call:    func(c *Client) (any, error) { return c.ActivateRescueSystem(ctx, 111007) },
			check: func(t *testing.T, got any) {
				task := got.(*generated.TaskInfo)
				if task == nil || task.Uuid == nil {
					t.Error("want TaskInfo with UUID")
				}
			},
		},
		{
			fixture: "POST_servers_111007_storageoptimization.json",
			call:    func(c *Client) (any, error) { return c.OptimizeStorage(ctx, 111007, nil) },
			check: func(t *testing.T, got any) {
				if got.(*TaskInfo) == nil {
					t.Error("want TaskInfo")
				}
			},
		},
		{
			// No ISO attached: 200 with {"iso": null, "isoAttached": false} —
			// the wrapper returns (nil, nil) per its documented contract.
			fixture: "GET_servers_111007_iso_none.json",
			call:    func(c *Client) (any, error) { return c.GetAttachedISO(ctx, 111007) },
			check: func(t *testing.T, got any) {
				if got.(*generated.Iso) != nil {
					t.Error("want nil for no attached ISO")
				}
			},
		},
		{
			fixture: "GET_servers_111007_iso.json",
			call:    func(c *Client) (any, error) { return c.GetAttachedISO(ctx, 111007) },
		},
		{
			fixture: "GET_servers_111007_isoimages.json",
			call:    func(c *Client) (any, error) { return c.ListAvailableISOs(ctx, 111007) },
		},
		{
			fixture: "GET_tasks.json",
			call:    func(c *Client) (any, error) { return c.ListTasks(ctx, nil) },
		},
		{
			fixture: "GET_users_555001.json",
			call:    func(c *Client) (any, error) { return c.GetUser(ctx, 555001) },
			check: func(t *testing.T, got any) {
				u := got.(*generated.User)
				if deref(u.Email) != "user@example.com" {
					t.Errorf("unexpected email %q (sanitization drift?)", deref(u.Email))
				}
			},
		},
		{
			fixture: "GET_users_555001_ssh-keys.json",
			call:    func(c *Client) (any, error) { return c.ListSSHKeys(ctx, 555001) },
		},
		{
			fixture: "GET_users_555001_vlans.json",
			call:    func(c *Client) (any, error) { return c.ListVLans(ctx, 555001, nil) },
		},
		{
			fixture: "GET_vlans_222001.json",
			call:    func(c *Client) (any, error) { return c.GetVLanByID(ctx, 222001) },
		},
		{
			// Live PUT of a VLAN name returns 204 No Content.
			fixture: "PUT_users_555001_vlans_222001.json",
			call: func(c *Client) (any, error) {
				return nil, c.UpdateVLan(ctx, 555001, 222001, "lan-2500-1")
			},
		},
		{
			fixture: "GET_users_555001_failoverips_v4.json",
			call:    func(c *Client) (any, error) { return c.ListFailoverIPv4(ctx, 555001, nil) },
		},
		{
			fixture: "GET_users_555001_failoverips_v6.json",
			call:    func(c *Client) (any, error) { return c.ListFailoverIPv6(ctx, 555001, nil) },
		},
		{
			fixture: "GET_users_555001_firewall-policies.json",
			call:    func(c *Client) (any, error) { return c.ListFirewallPolicies(ctx, 555001, nil) },
		},
		{
			fixture: "GET_users_555001_isos.json",
			call:    func(c *Client) (any, error) { return c.ListUserISOs(ctx, 555001) },
		},
		{
			fixture: "GET_users_555001_images.json",
			call:    func(c *Client) (any, error) { return c.ListUserImages(ctx, 555001) },
		},
		{
			fixture: "GET_rdns_ipv4_198_18_28_137.json",
			call:    func(c *Client) (any, error) { return c.GetRDNSv4(ctx, "198.18.28.137") },
		},
		{
			// rDNS lookup for an address without an entry: live API 404s.
			fixture: "GET_rdns_ipv6_2001_db8_c1_5265__1.json",
			wantErr: true,
			call:    func(c *Client) (any, error) { return c.GetRDNSv6(ctx, "2001:db8:c1:5265::1") },
		},
		{
			// Live no-op driver update: 204 No Content, no task.
			fixture: "PUT_servers_111007_interfaces_02_00_00_97_47_9f.json",
			call: func(c *Client) (any, error) {
				return c.UpdateInterfaceDriver(ctx, 111007, "02:00:00:97:47:9f", NetworkDriver("VIRTIO"))
			},
			check: func(t *testing.T, got any) {
				if got.(*TaskInfo) != nil {
					t.Error("204 no-op should return nil TaskInfo")
				}
			},
		},
		{
			// BIOS mode: online snapshot creation succeeds with a 202 task.
			fixture: "POST_servers_111007_snapshots_bios-online.json",
			call: func(c *Client) (any, error) {
				return c.CreateSnapshot(ctx, 111007, "audit-s1", "")
			},
		},
		{
			fixture: "GET_servers_111007_snapshots_audit-s1.json",
			call:    func(c *Client) (any, error) { return c.GetSnapshot(ctx, 111007, "audit-s1") },
		},
		{
			fixture: "POST_servers_111007_snapshots_audit-s1_revert.json",
			call:    func(c *Client) (any, error) { return c.RevertSnapshot(ctx, 111007, "audit-s1") },
		},
		{
			// Offline snapshot export: 202 task.
			fixture: "POST_servers_111007_snapshots_audit-off_export.json",
			call:    func(c *Client) (any, error) { return c.ExportSnapshot(ctx, 111007, "audit-off") },
		},
		{
			// Online snapshots cannot be exported: live 400.
			fixture: "POST_servers_111007_snapshots_audit-s1_export.json",
			wantErr: true,
			call:    func(c *Client) (any, error) { return c.ExportSnapshot(ctx, 111007, "audit-s1") },
		},
		{
			fixture: "DELETE_servers_111007_snapshots_audit-s1.json",
			call:    func(c *Client) (any, error) { return c.DeleteSnapshot(ctx, 111007, "audit-s1") },
		},
		{
			fixture: "POST_servers_111007_disks_vda_format.json",
			call:    func(c *Client) (any, error) { return c.FormatDisk(ctx, 111007, "vda") },
		},
		{
			fixture: "PATCH_servers_111007_disks.json",
			call: func(c *Client) (any, error) {
				return c.SetDiskDriver(ctx, 111007, StorageDriver("VIRTIO"))
			},
		},
		{
			// Covers both PATCH /servers/{id} mutations (uefi, cpu topology).
			fixture: "PATCH_servers_111007.json",
			call:    func(c *Client) (any, error) { return c.SetUEFI(ctx, 111007, true) },
		},
		{
			fixture: "POST_servers_111007_image.json",
			call: func(c *Client) (any, error) {
				return c.InstallImage(ctx, 111007, ServerImageSetup{})
			},
		},
		{
			fixture: "POST_servers_111007_user-image.json",
			call: func(c *Client) (any, error) {
				return c.InstallUserImage(ctx, 111007, ServerUserImageSetup{})
			},
		},
		{
			fixture: "PUT_servers_111007_interfaces_02_00_00_97_47_9f_firewall.json",
			call: func(c *Client) (any, error) {
				return c.UpdateFirewall(ctx, 111007, "02:00:00:97:47:9f", ServerFirewallSave{})
			},
		},
		{
			fixture: "POST_servers_111007_interfaces_02_00_00_97_47_9f_firewall_reapply.json",
			call: func(c *Client) (any, error) {
				return c.ReapplyFirewall(ctx, 111007, "02:00:00:97:47:9f")
			},
		},
		{
			fixture: "DELETE_servers_111007_interfaces_02_00_00_97_47_a0.json",
			call: func(c *Client) (any, error) {
				return c.DeleteInterface(ctx, 111007, "02:00:00:97:47:a0")
			},
		},
		{
			// One NIC per VLAN per server: live 400 with message.
			fixture: "POST_servers_111007_interfaces.json",
			wantErr: true,
			call: func(c *Client) (any, error) {
				return c.CreateVLanInterface(ctx, 111007, 222001, NetworkDriver("VIRTIO"))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			client, cleanup := newTestClient(t, fixtureHandler(t, tc.fixture))
			defer cleanup()
			got, err := tc.call(client)
			if tc.wantErr {
				if err == nil {
					t.Fatal("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// TestHALContentTypeNegotiation replays a real fixture with the content type
// flipped to application/hal+json. The generated parser then populates
// HALJSON200 instead of JSON200 — before the pickBody/taskBody helpers, ~50
// wrappers only checked JSON200 and would fail with "empty response" (or
// silently drop task handles) if the server's content negotiation ever chose
// hal+json, which the spec declares for every body-bearing response.
func TestHALContentTypeNegotiation(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "live", "GET_servers.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fx liveFixture
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatal(err)
	}
	client, cleanup := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/hal+json")
		w.WriteHeader(fx.Status)
		_, _ = w.Write(fx.ResponseBody)
	})
	defer cleanup()

	servers, err := client.ListServers(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListServers with hal+json response: %v", err)
	}
	if len(servers) != 8 {
		t.Errorf("want 8 servers, got %d", len(servers))
	}
}
