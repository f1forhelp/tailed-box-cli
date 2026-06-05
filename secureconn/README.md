# Secureconn

`secureconn` owns Tailedbox secure connection primitives and runtime pieces that
can be developed independently from the CLI application.

It includes:

- `protocol`: `TBXM` packet envelope and JSON control messages.
- `crypto`: Ed25519 transcript signing, X25519/HKDF session keys, nonce helpers,
  and AES-GCM construction.
- `session`: replay-window tracking and AEAD packet seal/open helpers.
- `transport`: direct UDP listener/client handshake, encrypted auth, and
  encrypted ping/pong.
- `control`: private local JSON request/response control socket.
- `store`: private runtime status and peer observation files.

The Tailedbox CLI integrates this module through `internal/mesh/service`, which
supplies local config, node identity, trusted-node validation, joined-cluster
validation, and runtime peer observation.

Module context starts at [`CONTEXT.md`](CONTEXT.md). Feature-specific secure
connection context lives under [`contexts/`](contexts/).

## Development

```bash
go test ./...
```

The protocol design lives in
[`docs/mesh-protocol-design.md`](docs/mesh-protocol-design.md).
