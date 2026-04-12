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
| `images.go`            | List, get, delete, upload user images                                                                                                  |
| `rescue.go`            | Enable/disable rescue system                                                                                                           |
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
  list                          list servers
  get <id>                      get server details (live info included)
  start/stop/restart <id>       power management
  autostart <id> <on|off>       configure autostart
  uefi <id> <on|off>            configure UEFI boot
  nickname <id> <name>          set nickname
  cpu-topology <id> <s> <c>     set CPU sockets and cores
  logs <id>                     event log
  guest-agent <id>              guest agent info and network interfaces
  image-flavours <id>           list installable OS images
  install-image <id> <flavour>  install OS image
  install-user-image <id> <img> install user-uploaded image
  optimize-storage <id>         compact disk allocation
  qemu-status                   QEMU version status across all servers
  gpu-driver <server-id>        get GPU driver download URL

disks
  list/get <server-id>          list or inspect disks
  format <server-id> <name>     format disk (destructive)
  set-driver <server-id> <drv>  change storage driver
  supported-drivers <server-id> list supported drivers

network
  list/get <server-id>          list or inspect interfaces
  create/delete <server-id>     add or remove an interface
  set-driver <server-id> <mac>  change network driver
  rdns-v4/v6 get/set/delete     manage reverse DNS

snapshots
  list/get/create/restore/delete <server-id>

metrics
  cpu/disk/network/network-packet <server-id>
                                time-series charts (ASCII, last 6h by default)

firewall
  policies list/get/create/update/delete
  rules list/get/create/update/delete/reorder

failover
  v4/v6 list/route/unroute

vlans
  list/get/update

users
  get/update                    view or modify your account
  logs                          account activity log
  ssh-keys list/get/create/delete
  isos list
  images list/get/upload/delete

tasks
  list/get/cancel               manage async tasks

upload <file>                   upload a file
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
