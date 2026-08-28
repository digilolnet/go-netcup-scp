# CLAUDE.md

Go client library and CLI for the netcup SCP REST API.

## Build

```bash
task build       # build netcup-scp binary
task test        # run tests with race detection
task generate    # regenerate client from openapi.json
task check       # build + test
```

## Architecture

Two-layer design:

- **`internal/generated/client.gen.go`** — auto-generated from OpenAPI spec via oapi-codegen. **Never edit manually.** Regenerate with `task generate`.
- **`pkg/scp/*.go`** — handwritten high-level wrappers with simplified signatures. One file per domain (`servers.go`, `disks.go`, `network.go`, etc.).
- **`cmd/netcup-scp/`** — cobra CLI, single `main` package, one file per command group.

## Wrapper pattern

```go
func (c *Client) ListServers(ctx context.Context, opts *ListServersOptions) ([]ServerListMinimal, error) {
    params := &generated.GetApiV1ServersParams{}
    if opts != nil { /* map fields */ }

    resp, err := c.api.GetApiV1ServersWithResponse(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("list servers: %w", err)
    }
    if err := checkResponse(resp, 200); err != nil {
        return nil, fmt.Errorf("list servers: %w", err)
    }
    if resp.JSON200 == nil {
        return nil, fmt.Errorf("list servers: empty response")
    }
    return *resp.JSON200, nil
}
```

Some endpoints return `application/hal+json` — check `HALJSON200` before `JSON200`.

Generated types returned by wrappers are aliased in `pkg/scp`
(`type ServerListMinimal = generated.ServerListMinimal`, …) so callers never
need `internal/generated`.

## Authentication

`pkg/scp/auth/auth.go` implements OAuth2 device flow. `auth.Manager` handles token storage, auto-refresh (30s before expiry), and injects Bearer tokens into every request. See `Authentication.md`.

The CLI stores tokens as JSON files and extracts the user ID from the JWT `id` claim directly, falling back to the documented `/userinfo` endpoint if the claim is missing.

## CLI conventions

- `makeCompleter(fn...)` — positional arg autocomplete with `makeCmdContext` built in
- `registerFlagCompleter(cmd, flag, fn)` — same for flag completions
- `printResult(cc, data, textFn)` — prints JSON (`-j`) or calls `textFn` for human output (detail views)
- `newDisplayer(column(name, header, extract), ...)` — per-resource list renderer; `d.print(cc, items)`
  honors `-j`, `--format`, `--no-header`, `-q`; `d.addFlags(cmd)` registers those flags with
  per-resource column help
- `resolveServerArg(cc, arg)` — turns a server id/nickname/name/hostname (or unique prefix) into an id
- `confirm(action)` / `confirmRetype(cc, id, action)` — destructive-op guards, skipped by `--force`
- `fmtTime(*time.Time)` — formats UTC timestamps as `"2006-01-02 15:04:05"`; unit goes in column header, not value
- `deref[T]`, `derefStr`, `derefInt32` — nil-safe pointer helpers

## Tests

Two layers:

- `pkg/scp/livefixtures_test.go` + `pkg/scp/testdata/live/` — recorded real API
  responses (sanitized) replayed through the wrappers. Prefer extending this
  when touching wrappers; capture new fixtures by running the CLI with
  `NETCUP_SCP_TRACE_DIR=<dir>` and sanitizing before committing.
- Hand-written `httptest.Server` tests for error paths and transport logic.
  See `pkg/scp/servers_test.go`, `client_test.go`.

## Updating the API

```bash
curl -o openapi.json https://www.servercontrolpanel.de/scp-core/api/v1/openapi
task generate
task check
```
