# Secureconn Crypto And Session Context

This file owns context for transcript signing, key derivation, AEAD helpers,
replay windows, and encrypted session packet helpers.

## Architecture

- Crypto package: `secureconn/crypto`.
- Session package: `secureconn/session`.
- Identity helper package: `secureconn/identity`.
- Ed25519 signs canonical handshakes transcripts.
- X25519 ephemeral keys derive shared secrets.
- HKDF-SHA256 derives directional session keys and nonce prefixes.
- AES-256-GCM seals encrypted packet payloads.
- AEAD associated data binds ciphertexts to the `TBXM` envelope header.
- Replay windows reject duplicate and stale packet sequences.

## Implemented

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

## Tests

- Transcript tamper detection.
- Matching X25519/HKDF derivation on both sides.
- AES-GCM construction and nonce behavior.
- AEAD associated-data enforcement.
- Replay rejection.
- Duplicate packet rejection.
- Stale packet rejection.
- Packet-header tamper rejection.

## Limitations And Next Work

- No rekey loop for active sessions yet.
- No active-session closure on trust or lease expiry yet.
- Session helpers exist, but durable multi-peer lifecycle management is tracked
  in `transport.md`.
