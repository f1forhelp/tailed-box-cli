# Secureconn Context

This is the secure connection module context for
`github.com/tailedbox/secureconn`. Root Tailedbox context lives in
`../CONTEXT.md`.

## Purpose

`secureconn` owns the secure connection layer used by Tailedbox for node-to-node
and local control communication. It is a standalone Go workspace module so
protocol, crypto, transport, control, and runtime-state code can be developed
and tested independently from the CLI application.

## Module Layout

```txt
secureconn/
  protocol/    packet envelope and JSON control messages
  crypto/      transcript signing, key derivation, AEAD helpers
  session/     replay windows and encrypted packet helpers
  transport/   UDP transport behavior
  control/     private local request/response socket
  store/       private runtime status and peer observation files
  identity/    public identity shape and validation helpers
  docs/        protocol design documents
```

## Documentation

- Protocol design: `docs/mesh-protocol-design.md`.

## Architecture

- Protocol package: `secureconn/protocol`.
- Crypto package: `secureconn/crypto`.
- Session package: `secureconn/session`.
- Identity helper package: `secureconn/identity`.
- Control package: `secureconn/control`.
- Store package: `secureconn/store`.
- Transport package: `secureconn/transport`.
- Packet envelope uses the `TBXM` magic and a versioned binary header.
- Control messages are JSON payloads carried inside encrypted packets after
  session establishment.
- Ed25519 signs canonical handshake transcripts.
- X25519 ephemeral keys derive shared secrets.
- HKDF-SHA256 derives directional session keys and nonce prefixes.
- AES-256-GCM seals encrypted packet payloads.
- AEAD associated data binds ciphertexts to the `TBXM` envelope header.
- Replay windows reject duplicate and stale packet sequences.
- Control uses a local JSON request/response socket.
- Runtime store persists private mesh status and peer observations under the
  consuming app's state directory.
- Transport accepts a `LocalNode` with node ID, role, cluster ID, public
  identity, and private identity key.
- The consuming app supplies trust validation through a `TrustValidator`.
- The consuming app receives peer observations through a `PeerObserver`.
- Transport does not import the root Tailedbox app module.
- Direct UDP is the MVP transport. Relay fallback and production NAT traversal
  are future work.

## Tailedbox Integration

- Root app adapter: `internal/mesh/service`.
- CLI surfaces live in the root app under `internal/cli`.
- The Tailedbox app loads local config, node identity, trusted-node records, and
  joined-cluster records.
- The app converts Tailedbox identity metadata to `secureconn/identity`.
- The app supplies `secureconn/transport.TrustValidator`.
- The app supplies `secureconn/store.PeerWriter` as the peer observer.
- The app starts `secureconn/control` and `secureconn/transport` from
  `tailedbox agent run`.
- Root `go.work` includes the root CLI module and `./secureconn`.
- Root `go.mod` requires `github.com/tailedbox/secureconn v0.0.0`.
- Root `go.mod` uses a local `replace` so root-only commands work offline.

## Implemented

- Versioned `TBXM` UDP packet envelope.
- Packet types for client hello, server hello, client auth, encrypted data,
  rekey, and close.
- Strict packet decode validation.
- Payload size limit.
- Reserved flag rejection.
- JSON control-message types for ping, pong, peer update, status
  request/response, diagnostics, and future network enrollment.
- Public identity shape and fingerprint validation helpers.
- Ed25519 transcript signing and verification.
- Canonical transcript serialization and transcript hash.
- Ephemeral X25519 key generation.
- HKDF-SHA256 session key derivation.
- AES-256-GCM construction.
- Directional nonce construction from nonce prefix plus packet sequence.
- Sender and receiver helpers for encrypted packets.
- Replay-window tracking.
- AEAD open/seal helpers bound to packet header associated data.
- Local control operations for mesh status, peer listing, ping dispatch, and
  diagnostics.
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
- Direct UDP listen/send/receive.
- Enrolled client/server handshake payloads.
- Handshake timestamp validation.
- Ed25519 transcript signature validation.
- X25519/HKDF session key derivation.
- Encrypted client auth.
- Replay protection through session receivers.
- App-supplied trust validation.
- Encrypted ping/pong control messages.
- Peer observation callbacks.
- Self-contained loopback test with generated in-memory identities and trust
  map.

## Commands

From the workspace root:

```bash
go test ./secureconn/...
```

From inside `secureconn/`:

```bash
go test ./...
```

## Tests

Current focused coverage includes:

- packet encode/decode
- malformed envelope rejection
- unsupported packet type rejection
- reserved flags rejection
- payload length mismatch rejection
- control-message shape tests
- strict store permissions
- peer observation writes
- sorted peer listing
- path-traversal rejection for peer node IDs
- transcript tamper detection
- matching X25519/HKDF derivation on both sides
- AES-GCM construction and nonce behavior
- AEAD associated-data enforcement
- replay rejection
- duplicate packet rejection
- stale packet rejection
- packet-header tamper rejection
- loopback UDP worker-to-master ping/pong
- in-memory identity generation and trust validation
- peer observation callback verification

Root app tests cover CLI status surfaces, mesh config toggles, control socket
presence during `agent run`, state-file fallback, and peer observation reads.
Module transport tests stay self-contained and do not import the root app.

## Current Limitations

- Rekey and close packet types exist, but full rekey/close behavior is not wired
  into durable transport sessions yet.
- Network enrollment message types exist, but the network enrollment flow is not
  implemented yet.
- No durable multi-peer session lifecycle yet.
- No rekey loop for active sessions yet.
- No retry or reconnect backoff loop yet.
- No active-session closure on trust or lease expiry yet.
- Master-to-worker ping depends on a consuming app having an observed endpoint
  for that worker.
- No production NAT traversal.
- No relay fallback.
- The control socket is local-only.
- Windows named-pipe support is not implemented.
- Runtime status currently reflects the consuming app's service lifecycle and
  transport callbacks; richer durable session state depends on transport
  lifecycle work.
- Network enrollment still has a local-state stand-in in the root app.
- Primary app status fields need to consume richer secure connection runtime
  state once durable sessions exist.

## Roadmap

1. Add durable multi-peer session lifecycle management.
2. Add rekey handling.
3. Close active sessions on trust or lease expiry.
4. Add broader peer routing that does not depend only on observed endpoints.
5. Add reconnect/backoff behavior for live sessions.
6. Implement network enrollment over the secure connection channel.
7. Revisit relay fallback and NAT traversal after the direct UDP MVP is stable.
