# Secureconn Transport Context

This file owns context for UDP transport, enrolled handshake, encrypted
ping/pong, peer routing, reconnect behavior, and transport roadmap.

## Architecture

- Transport package: `secureconn/transport`.
- The transport accepts a `LocalNode` with node ID, role, cluster ID, public
  identity, and private identity key.
- The consuming app supplies trust validation through a `TrustValidator`.
- The consuming app receives peer observations through a `PeerObserver`.
- Transport does not import the root Tailedbox app module.
- Direct UDP is the MVP transport. Relay fallback and production NAT traversal
  are future work.

## Implemented

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

## Tests

- Loopback UDP worker-to-master ping/pong.
- In-memory identity generation.
- In-memory trust validation.
- Peer observation callback verification.

## Current Limitations

- No durable multi-peer session lifecycle yet.
- No rekey loop for active sessions yet.
- No retry or reconnect backoff loop yet.
- Master-to-worker ping depends on a consuming app having an observed endpoint
  for that worker.
- No production NAT traversal.
- No relay fallback.
- Reconnect lease enforcement is available through consuming-app trust
  validation for new handshakes, but active-session closure on lease expiry is
  not implemented.
- Network enrollment over `--master-endpoint` is designed but not implemented.

## Roadmap

1. Add durable multi-peer session lifecycle management.
2. Add rekey handling.
3. Close active sessions on trust or lease expiry.
4. Add broader peer routing that does not depend only on observed endpoints.
5. Add reconnect/backoff behavior for live sessions.
6. Implement network enrollment over the secure connection channel.
7. Revisit relay fallback and NAT traversal after the direct UDP MVP is stable.
