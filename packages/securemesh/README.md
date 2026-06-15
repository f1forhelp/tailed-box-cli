# securemesh

Secure mesh foundation and transport primitives module.

## Packages

- `identity`: Node IDs, network IDs, roles, identity key metadata, identity/network generation, and restart-safe save/load.
- `crypto`: Secure random helpers, hashing/verifier helpers, constant-time equality, and safe encodings.
- `config`: Config paths, restrictive directory/file permissions, atomic JSON writes, and local locks.
- `join`: High-entropy join-code generation, verifier records, single-use local consumption, wrong-role/wrong-network rejection, and no plaintext persistence.
- `peer`: Peer metadata, active/revoked state, and local peer store.
- `revocation`: Local revocation records, revocation creation, and revoked-node checks.
- `network`: Transport/session metadata and interfaces.
- `network/tlsidentity`: Runtime TLS certificates derived from persistent node identity and custom mesh peer verification.
- `network/pairing`: Online join-code pairing over TLS/TCP with explicit master node ID pinning.
- `network/tlstcp`: Minimal TLS/TCP ping listener/dialer for current CLI real-server testing.
- `network/quictransport`: Minimal QUIC/TLS ping listener/dialer over reliable streams.

## Boundaries

This module does not implement NAT traversal, service management, remote command execution, or application secret transmission. The transport packages are limited connectivity/authentication primitives, not service-management protocols.

Cryptographic helpers use reviewed Go standard-library primitives. The module does not implement custom crypto math.

## Testing

Run:

```sh
go test ./...
```
