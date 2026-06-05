# Secureconn Tailedbox Integration Context

This file owns context for how the root Tailedbox app consumes `secureconn`.

## Architecture

- Root app adapter: `internal/mesh/service`.
- CLI surfaces live in the root app under `internal/cli`.
- The Tailedbox app loads local config, node identity, trusted-node records, and
  joined-cluster records.
- The app converts Tailedbox identity metadata to `secureconn/identity`.
- The app supplies `secureconn/transport.TrustValidator`.
- The app supplies `secureconn/store.PeerWriter` as the peer observer.
- The app starts `secureconn/control` and `secureconn/transport` from
  `tailedbox agent run`.

## Implemented

- Root `go.work` includes the root CLI module and `./secureconn`.
- Root `go.mod` requires `github.com/tailedbox/secureconn v0.0.0`.
- Root `go.mod` uses a local `replace` so root-only commands work offline.
- `tailedbox agent run` starts the app adapter and local control socket.
- Mesh runtime status is written through `secureconn/store`.
- CLI commands use `secureconn/control` and `secureconn/store`:
  - `tailedbox mesh enable`
  - `tailedbox mesh disable`
  - `tailedbox mesh status`
  - `tailedbox mesh peers`
  - `tailedbox mesh ping <node-id>`
  - `tailedbox mesh diagnose`

## Integration Tests

Root app tests cover CLI status surfaces, mesh config toggles, control socket
presence during `agent run`, state-file fallback, and peer observation reads.

Module transport tests stay self-contained and do not import the root app.

## Limitations And Next Work

- Network enrollment still has a local-state stand-in in the root app.
- Primary app status fields need to consume richer secure connection runtime
  state once durable sessions exist.
- Master-to-worker routing in the root app depends on observed peer endpoints
  until secureconn transport routing improves.
