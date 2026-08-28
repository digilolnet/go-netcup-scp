# go-netcup-scp

[![Go Report Card](https://goreportcard.com/badge/github.com/digilolnet/go-netcup-scp)](https://goreportcard.com/report/github.com/digilolnet/go-netcup-scp)
[![GoDoc](https://pkg.go.dev/badge/github.com/digilolnet/go-netcup-scp.svg)](https://pkg.go.dev/github.com/digilolnet/go-netcup-scp)
[![License](https://img.shields.io/github/license/digilolnet/go-netcup-scp.svg)](https://github.com/digilolnet/go-netcup-scp/blob/master/LICENSE.txt)
[![Code with hearth by Stnby](https://img.shields.io/badge/%3C%2F%3E%20with%20%E2%99%A5%20by-Stnby-ff1414.svg)](https://github.com/stnby)

Go client library and CLI for the [netcup SCP](https://www.servercontrolpanel.de) (Server Control Panel) REST API.

## Contents

- [`pkg/scp`](#library) — high-level Go client library
- [`cmd/netcup-scp`](#cli) — full-featured command-line tool

## Library

### Installation

```bash
go get github.com/digilolnet/go-netcup-scp
```

### Authentication

The library uses OAuth2 device flow. See [Authentication.md](./Authentication.md) for the full protocol description.

```go
import (
    "github.com/digilolnet/go-netcup-scp/pkg/scp/auth"
)

mgr := auth.NewManager(
    auth.WithAutoRefresh(true),
    auth.WithTokenRefreshCallback(func(tok *auth.TokenResponse) {
        // persist the refreshed token
    }),
)
defer mgr.Close()

// First-time login
deviceAuth, err := mgr.InitiateDeviceAuth(ctx)
fmt.Printf("Open: %s\n", deviceAuth.VerificationURIComplete)

token, err := mgr.PollForToken(ctx, deviceAuth.DeviceCode,
    time.Duration(deviceAuth.Interval)*time.Second)
// save token.RefreshToken for future sessions

// Subsequent sessions — load the saved token
mgr.LoadToken(savedToken)
```

### Client

```go
import "github.com/digilolnet/go-netcup-scp/pkg/scp"

client, err := scp.NewClient(mgr)
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### API coverage

`pkg/scp` provides high-level wrappers for all meaningful endpoints:

| Package file           | Operations                                                                                                                             |
| ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `servers.go`           | List, get, start/stop/restart, autostart, UEFI, nickname, CPU topology, logs, guest agent, image install, storage optimize, GPU driver |
| `disks.go`             | List, get, format, set driver                                                                                                          |
| `network.go`           | List/get/create/delete interfaces, rDNS (IPv4/IPv6)                                                                                    |
| `snapshots.go`         | Create, list, get, restore, delete                                                                                                     |
| `metrics.go`           | CPU, disk I/O, network throughput, network packets                                                                                     |
| `firewall_policies.go` | List, get, create, update, delete policies and rules                                                                                   |
| `failover.go`          | List, route, unroute IPv4/IPv6 failover addresses                                                                                      |
| `vlans.go`             | List, get, update VLANs                                                                                                                |
| `users.go`             | Get/update user, logs                                                                                                                  |
| `ssh_keys.go`          | List, create, delete SSH keys                                                                                                          |
| `isos.go`              | List user ISOs                                                                                                                         |
| `images.go`            | List, download-url, delete, upload user images                                                                                                  |
| `rescue.go`            | `servers rescue <on|off>` toggle                                                                                                           |
| `tasks.go`             | List, get, cancel async tasks                                                                                                          |
| `upload.go`            | Upload files                                                                                                                           |

For operations not covered by wrappers, the full generated client is accessible via `client.API()`.

### Architecture

The library uses a two-layer design:

- **`internal/generated/client.gen.go`** — auto-generated from the OpenAPI spec via [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). Never edit manually; regenerate with `task generate`.
- **`pkg/scp/*.go`** — handwritten wrappers with simplified signatures and sensible defaults.

### Example

```go
// List servers
servers, err := client.ListServers(ctx, nil)

// Get server with live info
server, err := client.GetServer(ctx, serverID, &scp.GetServerOptions{
    LoadServerLiveInfo: ptr(true),
})

// Power operations
err = client.StartServer(ctx, serverID)
err = client.StopServer(ctx, serverID, false)    // graceful shutdown
err = client.RestartServer(ctx, serverID, true)  // hard reset

// Snapshots
err = client.CreateSnapshot(ctx, serverID, "before-upgrade", "")
err = client.RestoreSnapshot(ctx, serverID, "before-upgrade")

// Access the raw generated client for advanced use
resp, err := client.API().GetApiV1ServersServerIdWithResponse(ctx, serverID, params)
```

---

## CLI

`netcup-scp` is a full-featured CLI built on top of the library.

### Build

```bash
task build
# or
go build -o netcup-scp ./cmd/netcup-scp/
```

### Authentication

```bash
netcup-scp auth login          # device flow login, saves token
netcup-scp auth logout         # revoke and remove token
netcup-scp auth status         # show current auth state
```

Multiple accounts are supported via named contexts:

```bash
netcup-scp context add prod --token-file ~/.config/netcup/prod.json
netcup-scp context use prod
netcup-scp context list
```

### Commands

```
servers
  list                            list servers (--sort, filters)
  get <server-id>                 server details (live info included)
  start/stop/restart <server-id>  power management (--wait)
  autostart <server-id> <on|off>  configure autostart
  uefi <server-id> <on|off>       configure UEFI boot
  rescue <server-id> <on|off>     enable/disable the rescue system
  nickname <server-id> <name>     set nickname
  cpu-topology <server-id> <sockets> <cores>
  logs <server-id>                event log
  guest-agent <server-id>         guest agent info
  image-flavours <server-id>      list installable OS images
  install-image <server-id> <flavour-id>       install official OS image
  install-user-image <server-id> <image-name>  install user-uploaded image
  optimize-storage <server-id>    compact disk allocation
  qemu-status                     QEMU version status across all servers
  gpu-driver <server-id>          GPU driver download URL
  vnc <server-id>                 bridge the VNC console (native port + noVNC)

disks
  list/get <server-id> [name]     list or inspect disks
  format <server-id> <name>       format disk (destructive)
  set-driver <server-id> <driver> change storage driver
  supported-drivers <server-id>

snapshots                         (BIOS-mode servers only; UEFI is refused by the API)
  list/get/create/revert/delete/export <server-id> [name]
  dry-run <server-id>             check whether a snapshot is possible

interfaces
  list/get <server-id> [mac]      list or inspect NICs
  create-vlan <server-id> <vlan-id>
  delete <server-id> <mac>        (primary NICs are refused)
  update-driver <server-id> <mac> <driver>

rdns-v4 / rdns-v6
  get/set/delete <ip>             manage reverse DNS

firewall
  get/update/reapply/clear/restore-copied-policies <server-id> <mac>
  active <server-id> <mac> <on|off>

firewall-policies
  list/get/create/update/delete/add-rule

failover-v4 / failover-v6
  list / route <failover-id> <server-id> / unroute <failover-id>

isos
  list/attached <server-id>       available and currently attached ISO
  attach/detach <server-id>       (--iso-id, --user-iso, --boot-cdrom)

user-isos / user-images
  list/upload/delete/download-url <key>
  upload-url <file>               presigned URL for out-of-band upload (isos only)

vlans
  list / get <vlan-id> / update <vlan-id> <name>

metrics
  cpu/disk/network/network-packet <server-id>   ASCII time-series charts

users
  get/update, logs
ssh-keys
  list/create/delete
tasks
  list/get/cancel <uuid>
system
  ping, maintenance
```

All commands support `--json` / `-j` for raw JSON output and shell completion (`bash`, `zsh`, `fish`, `powershell`).

---

## Development

```bash
task build       # build all packages
task test        # run tests with race detection
task lint        # run golangci-lint
task generate    # regenerate client from openapi.json
task check       # build + test + lint
```

Regenerate the client after updating `openapi.json`:

```bash
oapi-codegen -config oapi-codegen.yaml openapi.json
```

---

## License

This project is licensed under the Apache License, Version 2.0 - see the [LICENSE.txt](./LICENSE.txt) file for details.
