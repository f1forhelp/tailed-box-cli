# Secureconn Context

This module-level context is intentionally small. Feature-specific secure
connection progress, architecture, limitations, and roadmap items live in
`secureconn/contexts/`.

Root Tailedbox context lives in `../CONTEXT.md` and should only point here for
secure connection work.

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
  contexts/    feature-specific module context
```

## Context Map

Read and update the relevant secureconn context before changing a feature:

- `contexts/protocol.md` for packet envelopes, control messages, and protocol
  design docs.
- `contexts/crypto-session.md` for transcript signatures, key derivation,
  AEAD helpers, replay windows, and session packet helpers.
- `contexts/control-store.md` for local control sockets, runtime status, peer
  observations, and private file permissions.
- `contexts/transport.md` for UDP transport, enrolled handshake, peer routing,
  and transport roadmap.
- `contexts/tailedbox-integration.md` for how the root Tailedbox app consumes
  this module.

## Commands

From the workspace root:

```bash
go test ./secureconn/...
```

From inside `secureconn/`:

```bash
go test ./...
```

## Context Updates

- Update the feature-specific file in `secureconn/contexts/` when secureconn
  behavior, architecture, tests, limitations, or roadmap items change.
- Keep this file as routing plus stable module basics only.
