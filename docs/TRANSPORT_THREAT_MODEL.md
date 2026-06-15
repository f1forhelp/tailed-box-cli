# Transport Threat Model And Requirements

Milestone 10 defines the exact requirements for the first real server-to-server secure connection milestone. It does not implement transport code, add dependencies, transmit secrets, execute commands, or manage services.

## Scope

The first real-server transport milestone should connect one master node and one worker node over a real network using an encrypted, authenticated, restart-safe channel.

The first transport is for reliable control messages only. It is not a low-level VPN, not a kernel tunnel, not a remote shell, and not a service manager.

## First Transport Choice

Use QUIC/TLS 1.3 for the first real server-to-server control transport.

Reasons:

- Mature TLS 1.3 security model.
- UDP-based and generally fast enough for control traffic.
- Reliable streams and multiplexing are built in.
- Avoids writing custom reliability, congestion, retransmission, stream framing, and handshake machinery early.
- Reduces risk of bad custom cryptographic protocol implementation.

Long-term low-overhead data-plane work can still use a reviewed Noise-style UDP design later.

## Assets

Protect these assets:

- Node private identity keys.
- Future session keys.
- Join codes.
- Join-code verifier state.
- Peer membership state.
- Revocation state.
- Network ID.
- Future control messages.
- Future service config and secret-bearing messages.

## Threat Actors

Assume an attacker may:

- Observe network traffic.
- Modify network traffic.
- Replay old packets or handshake messages.
- Delay or drop packets.
- Attempt MITM during first contact.
- Attempt to connect as an unknown peer.
- Attempt to use a join code for the wrong network or role.
- Race a legitimate node with a copied join code.
- Attempt to reconnect after local revocation.

Do not assume protection if:

- A master private key is compromised.
- A worker private key is compromised.
- Local persistent state is compromised.
- A user discloses a join code to an attacker.
- The operating system or process memory is compromised.

Those cases require revocation, rotation, recovery procedures, and future operational controls.

## Security Goals

The transport must provide:

- Confidentiality for all payloads.
- Integrity for all payloads.
- Mutual authentication for already-paired peers.
- Network ID binding.
- Node ID binding.
- Role validation.
- Peer allowlist enforcement.
- Revocation enforcement before session acceptance.
- Replay resistance through TLS 1.3/QUIC plus application-level replay handling where needed.
- Restart-safe reconnect without new join codes for already-paired peers.
- Secret-safe logs.

## Non-Goals For First Transport Milestone

- No service management.
- No remote shell.
- No root/admin command execution.
- No Postgres, Redis, Valkey, Docker, reverse proxy, app deployment, logs, monitoring, backup, or secrets product features.
- No website/dashboard.
- No MCP server.
- No NAT traversal.
- No external VPN integration.
- No kernel VPN dependency.
- No shelling out to networking/VPN tools.
- No multi-master consensus.
- No optimized Noise/UDP data plane yet.

## Deployment Shape V1

The first real-server transport assumes reachable endpoints.

Supported shape:

- Master listens on a UDP address.
- Worker dials the master's reachable UDP address.
- Worker does not need inbound reachability for v1.
- Firewalls must allow inbound UDP to the master's configured port.

Not supported in v1:

- Symmetric NAT traversal.
- STUN/TURN.
- Relays.
- Automatic public endpoint discovery.
- Multi-hop mesh routing.

## Bind And Port Behavior

CLI commands should require explicit intent before exposing a public listener.

Recommended behavior:

- Development default listen address: `127.0.0.1:9443`.
- Real server listen example: `0.0.0.0:9443` or a specific public/private interface address.
- If a dial endpoint omits a port, default to `9443`.
- The CLI must display the chosen bind address before listening.
- The listener should fail if the local identity is missing.
- The listener should fail if network identity is missing.

Future command shape:

```sh
infra mesh listen --bind 0.0.0.0:9443
infra mesh dial --peer <node-id> --endpoint worker.example.com:9443
```

Pairing commands are separate from normal dial commands and are covered by `docs/PAIRING.md` and `docs/REAL_SERVER_CONNECTION_PLAN.md`.

## TLS Identity Model

Do not use public Web PKI or system trust roots for mesh peer authorization.

Recommended model:

- Generate an in-memory TLS certificate from the persistent node identity.
- Use Ed25519 node identity key material for TLS authentication if feasible.
- If the TLS stack requires certificate structures, generate them at runtime and avoid separate persisted TLS keys.
- Use custom certificate verification.
- Derive or verify the peer `NodeID` from expected public identity material.
- Reject unknown peers.
- Reject wrong-network peers.
- Reject wrong-role peers where role is expected.
- Reject revoked peers.

Verification must not rely on:

- `InsecureSkipVerify` without replacement verification.
- System CA trust.
- DNS name alone.
- IP address alone.
- Hostname, MAC address, or mutable machine attributes.

## QUIC Session Requirements

The QUIC transport implementation should:

- Use TLS 1.3.
- Use a fixed ALPN value such as `tailed-box-mesh/1`.
- Expose a `network.Transport` implementation.
- Wrap QUIC connections as `network.Session`.
- Support context cancellation.
- Support graceful close.
- Apply read/write deadlines or context-driven cancellation.
- Bound message sizes.
- Surface remote `NodeID`, `Role`, `NetworkID`, and endpoint metadata after authentication.

Initial message mode:

- Use reliable QUIC streams for control messages.
- Prefer simple length-prefixed binary framing over JSON in the hot path.
- Start with conservative message size limits.
- Add QUIC datagrams only after a clear need appears.

## Session Acceptance Rules

For already-paired peers, a listener may accept a session only if all checks pass:

- Local identity exists and validates.
- Local network identity exists and validates.
- Remote TLS identity validates cryptographically.
- Remote node ID is in the local peer store.
- Remote peer network ID matches local network ID.
- Remote peer role is allowed for the local operation.
- Remote peer is not revoked.
- Remote public identity material matches persisted peer material.
- Transport protocol version and ALPN match expected values.

If any check fails, the listener must reject the session and log only non-secret metadata.

## Pairing Session Rules

Unknown peers must not be admitted through the normal session path.

Future pairing should use a separate pairing path with stronger admission rules:

- Require a single-use join-code proof.
- Bind network ID, expected role, issuing master ID, joining node ID, and public keys into the pairing transcript.
- Persist the peer record and consume the join code atomically.
- Reject wrong-network, wrong-role, already-used, replayed, or tampered pairing attempts.

The preferred future pairing direction is documented in `docs/PAIRING.md`.

## Reconnect Behavior

After successful pairing, reconnect must not require a join code.

Reconnect requirements:

- Peer authenticates with persistent identity.
- Local peer allowlist is checked.
- Revocation state is checked.
- Network ID is checked.
- Role is checked.
- Session keys are fresh for each connection.
- Connection failure uses bounded exponential backoff.

Recommended backoff defaults:

- Initial retry delay: 500 ms.
- Maximum retry delay: 30 s.
- Add jitter.
- Stop retrying when context is cancelled or peer is revoked locally.

## Revocation Enforcement

Revocation must be enforced in three places:

- Before accepting an inbound session.
- Before dialing an outbound session.
- During active-session lifecycle when local revocation changes.

If a peer is revoked while connected, the local transport should close the active session and prevent reconnect with old credentials.

## Logging Requirements

Logs may include:

- Local node ID.
- Remote node ID.
- Remote role.
- Network ID.
- Endpoint address.
- Session state transitions.
- Error categories.

Logs must not include:

- Private keys.
- Join codes.
- Join-code verifier bytes.
- Session keys.
- PAKE/OPAQUE secrets.
- TLS private key material.
- Raw decrypted payloads.
- Credentials, API keys, database passwords, or deployment secrets.

## Failure Behavior

Failure should be explicit and safe.

Requirements:

- Unknown peer: reject.
- Revoked peer: reject.
- Wrong network: reject.
- Wrong role: reject.
- Invalid certificate: reject.
- Unsupported protocol version: reject.
- Oversized message: reject and close session.
- Malformed frame: reject and close session.
- Context cancellation: close cleanly.
- Temporary network failure: retry with backoff only when configured to maintain connection.

## Dependency Gate For QUIC

Before adding a QUIC dependency, evaluate:

- License compatibility.
- Maintenance activity.
- Security history.
- TLS verification hooks.
- QUIC stream and context APIs.
- Datagram support for future use.
- Testability on localhost.
- Cross-platform support.
- Benchmarking support.

Current likely candidate:

- `github.com/quic-go/quic-go`

This document does not approve adding the dependency by itself; the implementation milestone should add it deliberately with tests.

## Required Tests Before Real-Server Use

The first transport implementation must include tests for:

- TLS identity generation from persistent node identity.
- Valid peer certificate accepted.
- Unknown peer rejected.
- Revoked peer rejected.
- Wrong network rejected.
- Wrong role rejected.
- Public key mismatch rejected.
- Local QUIC listener/dialer session succeeds.
- Restart-safe reconnect succeeds without a join code.
- Context cancellation closes sessions.
- Oversized messages are rejected.
- Malformed frames are rejected.
- No secret material appears in logs.

Integration tests should cover:

- Two local processes or two isolated config roots.
- Master listen and worker dial.
- Pairing path once pairing is implemented.
- Reconnect after process restart.
- Revoke then reconnect rejection.

## Performance Requirements

The first real transport should optimize safety and correctness first, but it should be measurable from the beginning.

Track:

- Handshake latency.
- Small message round-trip latency.
- Allocations per send/receive.
- Throughput for bounded control messages.
- Reconnect latency after restart.

Avoid claiming zero overhead. Use precise language such as low-overhead encrypted control transport.

## Definition Of Done For Milestone 10

Milestone 10 is complete when:

- This threat model exists.
- The first transport choice is recorded.
- V1 deployment assumptions are explicit.
- Session acceptance rules are explicit.
- Logging and failure rules are explicit.
- Required tests for the implementation milestone are listed.
- Existing root, `packages/securemesh`, and `packages/control` tests pass.

## Next Milestone

Milestone 11 should implement TLS identity binding without adding QUIC transport yet:

- Generate runtime TLS certificates from persistent node identity.
- Verify peer certificates against local peer identity material.
- Reject unknown, wrong-network, wrong-role, and revoked peers.
- Add tests for all verification outcomes.
