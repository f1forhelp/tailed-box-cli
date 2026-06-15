# securemesh

Local secure mesh foundation module.

## Packages

- `identity`: Node IDs, network IDs, roles, identity key metadata, identity/network generation, and restart-safe save/load.
- `crypto`: Secure random helpers, hashing/verifier helpers, constant-time equality, and safe encodings.
- `config`: Config paths, restrictive directory/file permissions, atomic JSON writes, and local locks.
- `join`: High-entropy join-code generation, verifier records, single-use local consumption, wrong-role/wrong-network rejection, and no plaintext persistence.
- `peer`: Peer metadata, active/revoked state, and local peer store.
- `revocation`: Local revocation records, revocation creation, and revoked-node checks.
- `network`: Future transport/session metadata and interfaces only.

## Boundaries

This module does not implement a production encrypted mesh transport, online pairing handshake, NAT traversal, service management, remote command execution, or secret transmission.

Cryptographic helpers use reviewed Go standard-library primitives. The module does not implement custom crypto math.

## Testing

Run:

```sh
go test ./...
```
