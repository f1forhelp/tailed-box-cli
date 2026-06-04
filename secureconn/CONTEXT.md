# Secureconn Context

This file tracks module-specific context for `github.com/tailedbox/secureconn`.
When secure connection behavior, architecture, commands, tests, limitations, or
roadmap items change, update this file in the same edit.

Root Tailedbox context lives in `../CONTEXT.md` and should only point here for
module-specific progress.

## Purpose

`secureconn` owns the secure connection layer used by Tailedbox for node-to-node
and local control communication. It is developed as a standalone Go workspace
module so protocol, crypto, transport, control, and runtime-state code can be
tested independently from the CLI application.

## Module Layout

```txt
secureconn/
  protocol/    TBXM packet envelope and JSON control messages
  crypto/      Ed25519 transcript signing, X25519/HKDF, AES-GCM helpers
  session/     replay windows and AEAD packet seal/open helpers
  transport/   UDP listen/send/receive, handshake, encrypted ping/pong
  control/     private local JSON request/response socket
  store/       private runtime status and peer observation files
  identity/    public identity shape and fingerprint validation helpers
  docs/        protocol design documents
```

The Tailedbox CLI integrates this module through `internal/mesh/service`, which
supplies app config, local node identity, trusted-node validation,
joined-cluster validation, and runtime peer observation.

## Documentation

- Protocol design: `docs/mesh-protocol-design.md`.

## Implemented

- Versioned `TBXM` UDP packet envelope with packet types for hello, auth, data,
  rekey, and close.
- JSON control-message types for ping, pong, peer updates, status, diagnostics,
  and future network enrollment messages.
- Private mesh runtime status and peer observation files with strict
  permissions, sorted peer listing, and path-traversal rejection for peer IDs.
- Ed25519 transcript signing and verification.
- Ephemeral X25519 key generation and HKDF-SHA256 session key derivation.
- AES-256-GCM construction and nonce construction from directional nonce prefix
  plus packet sequence.
- Replay-window tracking and directional packet sender/receiver helpers.
- AEAD packet seal/open bound to the `TBXM` envelope header as associated data.
- Local JSON request/response control socket for status, peers, ping dispatch,
  and diagnostics.
- Direct UDP listen/send/receive transport.
- Enrolled client/server handshake payloads with transcript signatures,
  X25519/HKDF session keys, encrypted client auth, replay protection, and
  app-supplied trust validation.
- Encrypted worker-to-master ping/pong through the transport.
- Peer observation callback support so the consuming app owns persistence
  policy.
- Self-contained loopback transport tests that generate in-memory identities and
  validate encrypted ping/pong without importing the Tailedbox app module.

## Tests

Run module tests from either the workspace root or this module:

```bash
go test ./secureconn/...
```

From inside `secureconn/`:

```bash
go test ./...
```

Current focused coverage includes packet encode/decode, malformed envelope
rejection, control-message shape, strict store permissions, peer listing,
transcript tamper detection, matching X25519/HKDF derivation on both sides,
AEAD associated-data enforcement, replay rejection, duplicate packet rejection,
stale packet rejection, packet-header tamper rejection, and loopback UDP
ping/pong.

## Current Limitations

- No durable multi-peer session lifecycle yet.
- No rekey loop for active sessions yet.
- No retry or reconnect backoff loop yet.
- Master-to-worker ping depends on a consuming app having an observed endpoint
  for that worker.
- No production NAT traversal or relay fallback.
- Reconnect lease enforcement is available through consuming-app trust
  validation for new handshakes, but active-session closure on lease expiry is
  not implemented.
- Network enrollment over `--master-endpoint` is designed but not implemented.

## Roadmap

1. Add durable multi-peer session lifecycle management.
2. Add rekey handling and active-session closure on trust or lease expiry.
3. Add broader peer routing that does not depend only on observed endpoints.
4. Add reconnect/backoff behavior for live sessions.
5. Implement network enrollment over the secure connection channel.
6. Revisit relay fallback and NAT traversal after the direct UDP MVP is stable.
