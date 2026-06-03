# Tailedbox Mesh Protocol Design

This document defines the Part 6 protocol design for the first Tailedbox mesh
MVP. It is the implementation guide for Part 7.

The mesh is the secure node-to-node control transport for initialized and
enrolled Tailedbox nodes. It is not a full IP tunnel yet, and it does not
implement PostgreSQL, service orchestration, master HA, or a web UI.

Current implementation status:

- Implemented: protocol envelope/control-message types, private mesh runtime
  store, transcript signing, X25519/HKDF key derivation, nonce construction, and
  AES-GCM helpers, replay-window and encrypted packet seal/open helpers, local
  agent control socket, agent mesh service scaffold, and mesh CLI commands for
  enable/disable, status, peers, ping dispatch, and diagnostics.
- Not implemented yet: UDP transport, enrolled handshake/session wiring, rekey
  loops, live mesh ping/pong, and network enrollment.

## Goals

- Use the existing local node identity, enrollment, trusted-node, joined-cluster,
  and agent state as the foundation.
- Provide encrypted and authenticated node-to-node control messages.
- Keep the same `tailedbox` binary for master, worker, CLI, and agent behavior.
- Make the long-running agent own persistent mesh sockets and sessions.
- Preserve CLI-first workflows and JSON output behavior.
- Avoid raw join-code persistence and avoid logging secrets, private keys,
  session keys, decrypted payloads, or raw join-code material.
- Keep the direct UDP MVP lightweight and leave room for future relay fallback.
- Avoid Kubernetes, external etcd, Consul, or an external VPN dependency.

## Non-Goals

- No PostgreSQL module work.
- No full overlay IP routing, TUN device, or service proxying in the first mesh
  MVP.
- No production NAT traversal or relay network in the first mesh MVP.
- No master HA consensus, leader election, or replicated state.
- No automatic firewall mutation in the first mesh MVP.

## Existing State Inputs

Part 7 should build on the state that already exists:

```txt
<state-dir>/
  agent/
    config.json
    status.json
  audit/
    events.jsonl
  enrollment/
    join-codes/
    trusted-nodes/
  joined_cluster.json
  node.json
  node_identity_public.json
  secrets/node_identity_ed25519.pem
```

Important existing records:

- `node_identity_public.json` contains the node ID, Ed25519 public key, and
  fingerprint.
- `secrets/node_identity_ed25519.pem` contains the local Ed25519 private key.
- `enrollment/trusted-nodes/*.json` lets a master authenticate joined nodes.
- `joined_cluster.json` lets a worker or additional master pin the issuing
  master identity fingerprint and reconnect lease.
- `agent/config.json` already has a `mesh` section with `enabled`, `provider`,
  and `listen_udp_port`.

Part 7 should add mesh runtime state under:

```txt
<state-dir>/
  mesh/
    status.json
    peers/
      <node-id>.json
```

Mesh runtime state files should be `0600`; the `mesh` and `peers` directories
should be `0700`. Session keys must never be persisted.

## Threat Model

Assets to protect:

- Node private identity keys.
- Raw join-code secrets.
- Session keys.
- Control-plane messages.
- Trusted-node and joined-cluster state.
- Future service-management commands carried over the mesh.

Attackers considered:

- Passive network observers.
- Active network attackers who can spoof, replay, drop, reorder, or modify UDP
  packets.
- Untrusted nodes that have not completed enrollment.
- Previously trusted nodes whose reconnect lease has expired.
- Operators accidentally exposing database or control-plane ports.

Security properties expected:

- A node only accepts a mesh session from a peer whose identity is trusted for
  the local cluster and role.
- A worker pins the master's identity fingerprint from enrollment before
  accepting control traffic.
- Join-code secrets are sent only inside an encrypted enrollment channel and are
  never persisted after creation.
- Session payloads are encrypted and authenticated with per-session keys.
- Replayed encrypted packets are rejected.
- Session keys rotate periodically and are discarded when sessions end.
- Logs and audit records never include raw join codes, private keys, session
  keys, or decrypted payloads.

Out of scope for the MVP:

- A compromised root user on a node.
- A stolen node private key.
- A malicious but still-trusted node sending authorized traffic.
- Large-scale denial of service.
- Production-grade NAT traversal.
- Formal cryptographic proof of the custom protocol shape.

## Roles and Trust

Tailedbox keeps one binary and role-specific behavior:

- `master`: issues join codes, stores trusted-node records, accepts enrolled
  peers, and will later coordinate cluster control.
- `worker`: joins a master cluster and accepts authenticated master control
  traffic.
- additional `master`: can enroll with a master-scoped join code, but HA
  consensus is future work.

Long-term identity:

- Every node has one Ed25519 identity generated locally during initialization.
- The identity key signs handshake transcripts and key-update messages.
- The identity key is never used directly for payload encryption.
- Identity fingerprints use the existing `tbx1_...` fingerprint format.

Trust sources:

- A master trusts nodes listed under `enrollment/trusted-nodes`.
- A worker trusts the master fingerprint stored in `joined_cluster.json`.
- Additional masters follow the same joined-cluster pinning until HA is designed.
- No unauthenticated peer discovery is trusted.

## Cryptographic Suite

The initial protocol suite is `TBX-MESH-V1`:

- Long-term identity signatures: Ed25519.
- Ephemeral key agreement: X25519.
- Key derivation: HKDF-SHA256.
- Payload AEAD: AES-256-GCM.
- Randomness: OS CSPRNG only.

Implementation guidance:

- Use Go standard-library primitives where available.
- Do not implement custom ciphers, MACs, or hash functions.
- ChaCha20-Poly1305 can be considered later if the project accepts a Go crypto
  dependency for better software-only performance on small VPSs.
- The handshake transcript must include protocol version, roles, cluster ID,
  node IDs, identity fingerprints, both ephemeral public keys, both nonces, and
  negotiated cipher suite.

Key lifecycle:

- Each session uses fresh X25519 ephemeral keys.
- HKDF derives separate client-to-server and server-to-client AEAD keys.
- Nonces are 96-bit values built from a per-direction session nonce prefix plus
  a monotonically increasing sequence number.
- Session keys rotate after 30 minutes, 1 GiB sent in either direction, or before
  the reconnect lease expires, whichever comes first.
- A session must close if rekeying fails.
- Session keys and plaintext buffers should be released as soon as practical.

## Packet Envelope

All UDP mesh packets use one versioned envelope:

```txt
magic      4 bytes   "TBXM"
version    1 byte    protocol version, currently 1
type       1 byte    packet type
flags      2 bytes   reserved, zero for MVP
session    16 bytes  session ID, zero during initial hello
sequence   8 bytes   packet sequence, zero for unauthenticated handshake packets
length     4 bytes   payload length
payload    N bytes   clear handshake payload or AEAD ciphertext plus tag
```

All multi-byte integers are network byte order.

Packet types:

```txt
1  client_hello
2  server_hello
3  client_auth
4  encrypted_data
5  rekey
6  close
```

For `encrypted_data`, `rekey`, and `close`, the envelope header is AEAD
associated data. The encrypted payload is a typed control message.

The MVP control message set:

```txt
ping
pong
peer_update
status_request
status_response
diagnose_request
diagnose_response
enroll_request
enroll_accept
enroll_reject
```

JSON encoding is acceptable for MVP control payloads because the traffic volume
is small and the standard library is sufficient. The fixed envelope keeps room
for a future binary payload format without changing socket routing.

## Enrolled Session Handshake

This handshake is for nodes that already have local trust state.

1. Initiator sends `client_hello`.
   - Includes protocol version, cluster ID, role, node ID, identity fingerprint,
     X25519 ephemeral public key, random nonce, timestamp, and supported cipher
     suites.
   - Does not include secrets.

2. Responder checks local trust state.
   - Master responders require a matching trusted-node record.
   - Worker responders require the peer fingerprint to match the pinned master.
   - Cluster ID and role must match expected local state.
   - Handshake timestamp skew greater than 5 minutes is rejected.

3. Responder sends `server_hello`.
   - Includes responder public identity, X25519 ephemeral public key, random
     nonce, chosen cipher suite, session ID, and an Ed25519 signature over the
     transcript so far.

4. Initiator verifies `server_hello`.
   - The responder fingerprint must match the local trust source.
   - The signature must validate against the responder public identity.

5. Both sides derive handshake keys.
   - Shared secret: X25519 initiator ephemeral x responder ephemeral.
   - HKDF salt: SHA256 of the handshake transcript.
   - HKDF info: `tailedbox mesh v1 session`.

6. Initiator sends encrypted `client_auth`.
   - Includes initiator public identity, current role, current cluster ID, and an
     Ed25519 signature over the full transcript.

7. Responder verifies `client_auth`.
   - Signature must validate.
   - Master checks trust state and reconnect lease.
   - Worker checks pinned master identity.
   - If accepted, both sides mark the session established.

Reconnect lease enforcement:

- A master refuses a new session when the trusted node's
  `reconnect_lease_expires_at` is in the past.
- Session lifetime cannot extend beyond the remaining reconnect lease.
- If a lease expires during an active session, the next required rekey fails and
  the session closes.
- An expired node must use a new join code to regain trust.

Replay protection:

- Handshake nonces must be random.
- Recent handshake IDs should be retained briefly and rejected on reuse.
- Encrypted packets use per-direction monotonic sequence numbers.
- Receivers keep a replay window and reject duplicate or stale sequence numbers.

## Network Enrollment Handshake

This replaces the current local `--master-state-dir` enrollment transport in a
future implementation slice. The join-code lifecycle stays the same: one-time,
short-lived, role-scoped, hash persisted, raw code printed once.

Target CLI shape:

```bash
tailedbox worker join --code <join-code> --master-endpoint <host:port>
tailedbox master join --code <join-code> --master-endpoint <host:port>
```

Enrollment flow:

1. Joiner parses the join code locally.
   - The code payload gives code ID, allowed role, cluster ID, issuer node ID,
     issuer fingerprint, and expiry.
   - The raw code secret is kept only in memory.

2. Joiner sends `client_hello` in enrollment mode.
   - Includes code ID, expected role, cluster ID, local node ID, local identity
     fingerprint, X25519 ephemeral public key, nonce, and timestamp.
   - Does not include the raw code secret.

3. Master sends `server_hello`.
   - Includes master public identity and signature.
   - Joiner verifies the master public identity fingerprint matches the issuer
     fingerprint embedded in the join code.

4. Joiner sends encrypted `enroll_request`.
   - Includes raw join-code secret, full local public identity, expected role,
     and a signature over the enrollment transcript.

5. Master validates enrollment.
   - Code ID exists.
   - Secret hash matches.
   - Code is active, not expired, not already used, and role-scoped correctly.
   - Joining node identity is not already trusted.
   - The request signature validates.

6. Master commits enrollment state.
   - Writes trusted-node record.
   - Marks the join code used.
   - Records reconnect lease metadata.
   - Appends audit events.

7. Master sends encrypted `enroll_accept`.
   - Includes cluster ID, cluster name, master node ID, master public identity
     fingerprint, reconnect lease expiry, and known master endpoints.

8. Joiner writes `joined_cluster.json`, updates cluster config endpoints, and
   can start a normal enrolled mesh session.

Enrollment failure handling:

- The master returns `enroll_reject` with a sanitized reason.
- The raw join-code secret is never logged.
- Repeated failures should be rate limited by source address and code ID.
- Audit events should record join attempts, success, and sanitized failures.

## Peer Discovery

The MVP uses explicit and learned peer endpoints:

- Workers and additional masters get master endpoints from config and from
  successful network enrollment.
- Masters learn a node's observed UDP source address during authenticated
  sessions.
- Authenticated peers may send `peer_update` messages containing candidate
  endpoints.
- The runtime may persist observed peer endpoint metadata under
  `<state-dir>/mesh/peers/<node-id>.json`.
- Endpoint metadata is not a trust source. It is only a routing hint.

Peer metadata should include:

```json
{
  "version": 1,
  "node_id": "node_...",
  "role": "worker",
  "identity_fingerprint": "tbx1_...",
  "last_endpoint": "203.0.113.10:41677",
  "last_seen_at": "2026-06-03T00:00:00Z",
  "session_state": "connected"
}
```

No unauthenticated LAN broadcast, mDNS, gossip, or public discovery service is
part of the MVP.

## Agent and CLI Model

The agent owns mesh sockets and long-lived sessions:

- `tailedbox agent run` loads local config, identity, trust state, and mesh
  runtime state.
- If mesh is enabled, the agent binds UDP and maintains sessions.
- The agent writes mesh health to `<state-dir>/mesh/status.json`.
- The agent writes peer observations to `<state-dir>/mesh/peers`.
- The agent appends sanitized logs and audit events.

CLI commands remain launchers over the same backend behavior:

```bash
tailedbox mesh status [--json]
tailedbox mesh peers [--json]
tailedbox mesh ping <node-id>
tailedbox mesh diagnose [--json]
```

Part 7 should add a minimal local agent control socket at:

```txt
<state-dir>/agent/control.sock
```

The socket is local-only, protected by the `0700` agent directory, and should use
small JSON request/response messages. The first supported operations should be
mesh status, peer listing, ping, and diagnostics. If the agent is not running,
CLI commands may fall back to reading mesh status files but should report that
live ping requires the agent. On platforms with short Unix socket path limits,
the implementation may use a deterministic short private temp path for the
socket while reporting the actual path in diagnostics.

## Direct Path MVP

The first implementation uses direct UDP:

- Masters listen on a configured UDP port.
- Workers can use an ephemeral UDP port for outbound sessions.
- The default master mesh port should be stable and configurable.
- Direct path assumes the operator can route and open the master's UDP port.
- Workers do not need public inbound ports for the MVP.

Recommended default:

```txt
provider: tailedbox-mesh
master UDP port: 41677
worker UDP port: 0
```

Retries:

- Initial handshake retry uses exponential backoff with jitter.
- Keepalive interval is 15 seconds.
- A peer is degraded after 45 seconds without authenticated traffic.
- A session is considered disconnected after 120 seconds without authenticated
  traffic.

## Future Relay Fallback

Relay fallback is not part of the MVP, but the direct path design should not
block it.

Future relay rules:

- Relay packets remain end-to-end encrypted.
- Relays never receive node private keys, join-code secrets, session keys, or
  decrypted payloads.
- Relay authorization is separate from node trust. A relay can route packets but
  cannot make an untrusted node trusted.
- Nodes prefer direct UDP when available and fall back to relay only after direct
  path failure.
- Relay routing should key on opaque session or node routing metadata, not on
  decrypted control payloads.

## Firewall Model

Mesh security should not depend on the firewall alone, but firewall posture
still matters:

- Do not expose PostgreSQL ports publicly.
- Do not expose future control APIs publicly.
- The local agent control socket is local-only and must never bind a public TCP
  address.
- Masters need the configured UDP mesh port reachable from workers.
- Workers can default to outbound-only UDP for the MVP.
- Part 7 should not mutate firewall rules automatically.
- `tailedbox mesh diagnose` should print actionable firewall guidance for
  common Linux setups, with future provider abstractions for UFW, firewalld, and
  nftables.

## Audit and Logging

Add audit actions in Part 7:

```txt
mesh.session_started
mesh.session_failed
mesh.session_closed
mesh.rekey_succeeded
mesh.rekey_failed
mesh.ping_succeeded
mesh.ping_failed
enrollment.network_attempt
enrollment.network_succeeded
enrollment.network_failed
```

Logging rules:

- Log node IDs, roles, fingerprints, session IDs, endpoints, state transitions,
  timing, and sanitized reasons.
- Do not log raw join codes, join-code secrets, private keys, session keys,
  plaintext control payloads, or decrypted service data.
- Preserve existing redaction behavior at write and display boundaries.

## Part 7 Implementation Boundaries

Suggested package layout:

```txt
internal/mesh/
  protocol/    envelope constants, message structs, encoding
  crypto/      handshake, key derivation, AEAD helpers
  session/     session state, replay windows, rekeying
  transport/   UDP listener, peer send/receive loop
  store/       mesh status and peer observation files
  service/     agent-facing mesh service
```

Suggested first implementation slice:

1. Add pure protocol and crypto tests for envelope encoding, transcript signing,
   key derivation, and replay rejection.
2. Add mesh status and peer observation state files with strict permissions.
3. Add UDP transport with encrypted `ping` and `pong`.
4. Wire `agent run` to start the mesh service when mesh is enabled.
5. Implement `tailedbox mesh status`, `peers`, `ping`, and `diagnose`.
6. Add network enrollment as a follow-up slice that replaces
   `--master-state-dir` with `--master-endpoint`.

The MVP is complete when two initialized and enrolled nodes can authenticate,
establish an encrypted session, exchange `mesh ping`, report peer status, and
diagnose basic connectivity without exposing secrets.

## Open Decisions

These should be revisited during Part 7 implementation:

- Whether AES-256-GCM remains the only MVP AEAD or whether accepting
  `golang.org/x/crypto/chacha20poly1305` is worth the dependency.
- Whether the default UDP port should remain `41677`.
- Whether the local agent control socket should support only Unix sockets for
  the MVP or include a Windows named-pipe path later.
- Exact binary payload format if JSON control messages become too large.
- Relay protocol details after direct UDP is working.
