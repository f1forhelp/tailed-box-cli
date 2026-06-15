# Real Server Secure Connection Plan

This plan describes how to move from the current local secure mesh foundation to actual real server-to-server secure connections. It is a plan only; it does not implement networking, remote command execution, service management, or secret transmission.

## Target Outcome

Enable a master node and worker node on real servers to establish a cryptographically strong, authenticated, restart-safe, encrypted machine-to-machine connection.

The first production-capable connection should support:

- Persistent node identities.
- Network ID checks.
- Role checks.
- Join-code based first pairing.
- Single-use join-code enforcement.
- Peer allowlists.
- Revocation checks before session acceptance.
- Restart-safe reconnect without a new join code.
- Encrypted authenticated transport.
- Replay-resistant session establishment.
- Operationally safe logging that never logs private keys, join codes, session keys, credentials, or derived secret material.

## Explicit Ambiguities

### NAT Traversal

Options:

- Require reachable server endpoints for the first real-server milestone.
- Add STUN/TURN/relay or hole-punching immediately.

Tradeoff:

- Reachable endpoints are simpler, testable, and safer for the first production-capable milestone.
- NAT traversal adds substantial complexity and operational edge cases.

Safe default:

- First real-server milestone assumes both servers have routable addresses or explicit firewall/port forwarding. NAT traversal is a later milestone.

### Transport Choice

Options:

- QUIC/TLS 1.3 first, with custom node identity verification.
- Noise over custom UDP first.
- Implement both immediately.

Tradeoff:

- QUIC/TLS 1.3 is mature, reliable, multiplexed, UDP-based, and safer for sensitive real server control traffic.
- Noise over custom UDP can be lower overhead, but requires correctly implementing packet framing, replay protection, retransmission or reliability strategy, key rotation, congestion behavior, and operational hardening.
- Building both immediately increases surface area.

Safe default:

- Implement QUIC/TLS 1.3 as the first real server-to-server reliable control transport.
- Preserve the long-term Noise/UDP data-plane plan for a later low-overhead milestone.

### Online Pairing Protocol

Options:

- Verifier-only PAKE/OPAQUE-style pairing.
- Noise PSK pairing with recoverable join-code secret material.
- Out-of-band fingerprint verification only.

Tradeoff:

- PAKE/OPAQUE-style pairing best preserves no-plaintext join-code persistence and can resist MITM when implemented with a reviewed protocol/library.
- Noise PSK is simpler but conflicts with verifier-only persistence unless secret material becomes recoverable or reintroduced.
- Fingerprint verification helps but is not enough by itself.

Safe default:

- Evaluate a reviewed Go PAKE/OPAQUE-style implementation and use it for online pairing if it passes review.
- Require master identity fingerprint display/verification as supporting UX.

## Architecture

```text
CLI / TUI / Future Web / Future MCP
        v
Shared Control Actions
        v
Membership + Identity + Revocation
        v
Pairing Protocol
        v
Reliable Control Transport: QUIC/TLS 1.3 first
        v
Future Optimized Data Plane: Noise-style UDP
```

## Milestone 10: Transport Requirements And Threat Model

Purpose:

- Convert the current design into exact real-server requirements before implementation.

Deliverables:

- `docs/TRANSPORT_THREAT_MODEL.md`.
- Exact supported deployment shape for v1: public IP or configured reachable endpoint.
- Port and bind behavior.
- Session establishment requirements.
- Session acceptance requirements.
- Logging redaction rules.
- Failure and retry behavior.

Key decisions:

- First real transport is QUIC/TLS 1.3 control transport.
- Noise/UDP data plane remains future work.
- No NAT traversal in v1.
- No remote command execution in this milestone.

Tests:

- Documentation consistency tests can remain lightweight.
- Existing `go test ./...` must stay green.

Commit:

- `docs: define transport threat model`

Status:

- Completed as `docs/TRANSPORT_THREAT_MODEL.md`.

## Milestone 11: Node Certificate And TLS Identity Binding

Purpose:

- Bind existing persistent node identity to TLS authentication without relying on public PKI.

Plan:

- Generate an in-memory self-signed TLS certificate from the persistent Ed25519 node identity.
- Do not persist separate TLS private keys unless a concrete need appears.
- Use custom certificate verification to map peer certificate public key to expected node ID.
- Reject certificates whose derived node ID is unknown, wrong-network, wrong-role, or revoked.

Implementation areas:

- `packages/securemesh/network/tlsidentity` or equivalent package.
- Certificate generation from existing identity.
- Certificate verification callback.
- Peer allowlist lookup interface.

Security requirements:

- Do not use system trust roots for mesh peer identity.
- Do not skip certificate verification.
- Do not log certificate private material.
- Bind network ID and node ID into verification logic.

Tests:

- Valid node certificate accepted.
- Unknown node rejected.
- Wrong node ID rejected.
- Revoked node rejected.
- Wrong role rejected.
- Restart-safe identity produces stable authentication identity.

Commit:

- `feat: bind node identity to tls auth`

Status:

- Completed as `packages/securemesh/network/tlsidentity`.

## Milestone 12: QUIC Control Transport MVP

Purpose:

- Establish real encrypted server-to-server sessions over UDP using QUIC/TLS 1.3.

Recommended dependency:

- Evaluate `github.com/quic-go/quic-go` before adding it.
- Add it only after confirming license, maintenance, API stability, test support, and security posture.

Implementation areas:

- `packages/securemesh/network/quictransport` or similar.
- `Transport` implementation for listen/dial.
- Session wrapper implementing existing `network.Session` interface.
- Message send/receive using QUIC streams or datagrams.
- Context cancellation and graceful close.

Safe default behavior:

- Use reliable QUIC streams for control messages.
- Use one short-lived stream per message initially for simplicity, or one bidirectional stream with framed messages if easier to test.
- Keep message payload size bounded.
- Do not add service-management message types yet.

Tests:

- Local listener/dialer integration test.
- Valid master/worker session succeeds.
- Unknown peer rejected.
- Revoked peer rejected.
- Wrong network rejected.
- Restart-safe reconnect succeeds without join code.
- Connection close and context cancellation behavior.

Benchmarks:

- Handshake latency on localhost.
- Small message round-trip latency.
- Allocation count for send/receive paths.

Commit:

- `feat: add quic control transport`

Status:

- Completed at the package level as `packages/securemesh/network/quictransport` using `github.com/quic-go/quic-go` v0.60.0.
- It provides authenticated QUIC/TLS listen and ping over reliable streams with runtime mesh TLS certificates, peer allowlist checks, network checks, role checks, and revocation checks.
- It is not wired into the CLI yet; the current real-server CLI procedure still uses TLS/TCP.

## Milestone 13: Online Pairing Prototype

Purpose:

- Allow a worker on a real server to pair with a master over the network using a single-use join code without storing plaintext join codes.

Plan:

- Use the direction in `docs/PAIRING.md`.
- Evaluate a reviewed Go PAKE/OPAQUE-style library before implementation.
- Implement a local network pairing endpoint over the QUIC control transport.
- Bind network ID, expected role, issuing master ID, joining node ID, public keys, and transcript version.
- Mark the join code consumed only after peer admission is persisted.

CLI commands to add:

- `infra mesh listen --bind host:port`
- `infra join-code create --role worker`
- `infra mesh join --master host:port --code <code> --role worker`
- `infra peer list`
- `infra peer revoke --node <node-id> --role worker`

Security requirements:

- Never send join code in plaintext over an unauthenticated channel.
- Never persist plaintext join code.
- Do not log join code or pairing transcript secrets.
- Reject replayed pairing attempts.
- Reject wrong network and wrong role.
- Reject already-consumed join codes.

Tests:

- Successful master/worker network pairing.
- MITM transcript tampering rejection.
- Wrong network rejection.
- Wrong role rejection.
- Already-used code rejection.
- No plaintext join-code persistence.
- Restart-safe reconnect after pairing.

Commit:

- `feat: prototype online pairing`

## Milestone 14: Revocation Enforcement In Live Sessions

Purpose:

- Ensure revoked nodes cannot connect or stay connected with old credentials.

Plan:

- Check revocation during handshake/session acceptance.
- Check revocation before every new logical session.
- Add a local revocation generation or state version.
- Close active sessions when a peer is revoked locally.

Tests:

- Revoked node cannot dial.
- Revoked node cannot be accepted.
- Active session closes after local revoke.
- Rejoining requires new join code and fresh identity by default.

Commit:

- `feat: enforce revocation in sessions`

## Milestone 15: Real Server Integration Harness

Purpose:

- Prove the connection works outside unit tests before using real infrastructure workflows.

Plan:

- Add two-process localhost integration tests.
- Add optional Docker-based or VM-based integration test instructions if approved later.
- Add packet loss, delay, and reconnect scenarios where feasible without shelling out to VPN tools.
- Add CLI smoke tests with temporary config roots.

Tests:

- Master process starts listener.
- Worker process joins.
- Worker reconnects after restart without join code.
- Master revokes worker.
- Worker reconnect is rejected after revocation.

Commit:

- `test: add real transport integration coverage`

## Milestone 16: Hardening And Operational Readiness

Purpose:

- Make the first real-server transport safe enough for controlled non-production deployments.

Plan:

- Add structured logs with secret redaction.
- Add explicit config file permission checks.
- Add rate limits for pairing attempts.
- Add handshake/message size limits.
- Add peer endpoint update rules.
- Add backoff and reconnect behavior.
- Add benchmarks in CI if CI is introduced.
- Add a security review checklist.

Tests:

- Fuzz message framing.
- Race tests where practical.
- Replay tests.
- Rate-limit tests.
- Permission tests.
- Benchmark baselines.

Commit:

- `chore: harden transport operations`

## Milestone 17: Optimized Noise/UDP Data Plane

Purpose:

- Add a lower-overhead encrypted UDP data plane after the reliable QUIC control transport is working.

Plan:

- Select a reviewed Noise implementation or protocol package.
- Use long-term node identity plus ephemeral session keys.
- Add compact binary framing.
- Add replay window.
- Add key separation.
- Add key rotation by packet count, byte count, or time.
- Keep QUIC as reliable control path if useful.

Tests:

- Handshake authentication.
- Replay rejection.
- Key rotation.
- Packet tampering rejection.
- Benchmarks versus QUIC control path.

Commit:

- `feat: add noise udp data plane prototype`

## Definition Of Done For First Real Server MVP

The first real server-to-server MVP is complete when:

- A master can listen on a configured UDP address.
- A worker can join the master over the network using a single-use join code.
- Pairing is MITM-resistant under the selected reviewed protocol.
- Both nodes persist peer state and reconnect after restart without a join code.
- Unknown peers are rejected.
- Wrong-network peers are rejected.
- Wrong-role peers are rejected during pairing.
- Revoked peers are rejected.
- No plaintext join code is persisted.
- No private keys, join codes, session keys, or derived secrets are logged.
- Root, `packages/securemesh`, and `packages/control` tests all pass.
- Integration tests cover two local processes.
- Documentation clearly states remaining production limitations.

## What Must Still Not Be Built During These Transport Milestones

- Postgres, Redis, Valkey, Docker, reverse proxy, app deployment, log streaming, monitoring, backups, or service managers.
- Website/dashboard.
- MCP server.
- Secret management or secret transmission beyond handshake internals.
- Remote admin/root command execution.
- External system VPN integration.
- Kernel-level VPN dependency.
- Shelling out to VPN/networking tools.
- Multi-master consensus until separately designed.
