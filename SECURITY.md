# Security Model

This project is designed for strong practical security, but it does not claim to be unhackable. The current code provides secure mesh foundations and minimal authenticated connection primitives, not a production service-management mesh.

## Current Security Scope

Implemented now:

- Persistent node identity with Ed25519 signing key metadata and X25519 transport key metadata.
- Public-key-derived node IDs.
- Master/worker role validation.
- Network identity.
- Restrictive local config directory and file permissions where supported by the OS.
- Atomic local writes for JSON state.
- Local directory locks for state transitions such as join-code consumption.
- High-entropy join-code generation.
- Join-code verifier storage instead of plaintext persistence.
- Constant-time verifier comparison.
- Single-use local join-code consumed state.
- Local revocation records and revoked-node checks.
- Peer active/revoked local model.
- Runtime TLS identity binding for mesh peers.
- Minimal authenticated TLS/TCP mesh ping for CLI testing.
- Package-level authenticated QUIC/TLS mesh ping over reliable streams.

Not implemented yet:

- Online pairing handshake.
- Secret transmission.
- Remote command execution.
- Service installation or service-management messages.
- Multi-master consensus, quorum, or revocation propagation.
- NAT traversal.

## Local State

Default config root is under the user config directory using `tailed-box-cli` as the application directory. Tests inject temporary roots.

Current local state files:

- `identity.json`
- `network.json`
- `join-codes.json`
- `peers.json`
- `revocations.json`
- `locks/`

Private identity material and local security state are written with restrictive file permissions where possible. Directory and file permissions are tested on supported platforms.

## Join-Code Security

Join codes are used only for initial pairing.

Properties in this milestone:

- Generated with `crypto/rand`.
- 32 random bytes before encoding.
- Base32 no-padding display encoding.
- Practically unguessable under normal assumptions.
- Persisted as verifier material, not plaintext.
- Per-code salt plus HMAC-SHA-256 verifier.
- Constant-time verifier comparison.
- Explicit unused/consumed state.
- Consumed under a local lock.
- No mandatory expiry by current product requirement.

The generated plaintext join code may be displayed once to an authorized user. It must not be logged, committed, persisted as plaintext, or written to `context.md`.

## MITM Prevention

Routine reconnects must use persistent node identity and peer membership state, not join codes.

Transport and pairing handshakes must:

- Authenticate long-term node identities.
- Bind the transcript to network ID and role expectations.
- Reject unknown or revoked public keys.
- Bind initial join authorization into the authenticated pairing transcript.
- Avoid sending join codes in plaintext over unauthenticated channels.
- Include replay protection, key separation, and key rotation.

The online pairing design is intentionally not implemented yet. A reviewed PAKE/OPAQUE-style pairing protocol or reviewed Noise-style flow with explicit master identity binding should be selected before network pairing is implemented.

## Revocation Model

Current revocation is local only.

A revocation record includes:

- Node ID.
- Role.
- Revoked timestamp.
- Revoked-by master node ID.
- Optional reason.

Revoked nodes are not active peers locally and should not reconnect with old credentials in future transport layers. Rejoining requires a new join code and fresh node identity material by default.

Future work must address propagation, quorum, split-brain handling, master-removal safety, and signed revocation records.

## No External System VPN Dependency

The intended secure mesh is application-owned. The project must not depend on external system VPN tools, kernel VPN features, OS-managed VPN configuration, or shelling out to VPN/networking commands.

## Transport Direction

Current and planned design:

- QUIC/TLS reliable control streams.
- Future optimized UDP data plane if needed.
- Reviewed Noise-style authenticated handshake for the future optimized data plane.
- Long-term identity keys plus ephemeral session keys.
- AEAD payload encryption.
- Replay protection.
- Key separation.
- Session key rotation.
- Peer allowlists.
