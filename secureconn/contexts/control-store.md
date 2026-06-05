# Secureconn Control And Store Context

This file owns context for local control sockets, runtime status, peer
observations, and private runtime files.

## Architecture

- Control package: `secureconn/control`.
- Store package: `secureconn/store`.
- Control uses a local JSON request/response socket.
- Runtime store persists private mesh status and peer observations under the
  consuming app's state directory.
- The module accepts minimal path inputs rather than importing the Tailedbox app
  config package.

## Implemented

- Local control operations:
  - mesh status
  - peer listing
  - ping dispatch
  - diagnostics
- Request/response schema.
- Local Unix socket listener.
- Client round trip helper.
- Default control socket path under `<state-dir>/agent/control.sock`.
- Deterministic temp fallback for long Unix socket paths.
- Private runtime paths under `<state-dir>/mesh`.
- Mesh status JSON.
- Peer observation JSON files.
- Sorted peer listing.
- Path-traversal rejection for peer node IDs.
- Private file helpers with strict directory and file permissions.

## Tests

- Strict store permissions.
- Peer observation writes.
- Sorted peer listing.
- Path-traversal rejection for peer node IDs.

## Limitations And Next Work

- The control socket is local-only.
- Windows named-pipe support is not implemented.
- Runtime status currently reflects the consuming app's service lifecycle and
  transport callbacks; richer durable session state depends on transport
  lifecycle work.
