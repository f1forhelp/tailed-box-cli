# Project Context

## Project Goal

Build a CLI-first infrastructure management application with a future secure machine-to-machine mesh between master and worker nodes. The current focus is only the local foundation for secure mesh identity, pairing, revocation, persistence, transport abstractions, and shared CLI/TUI control flow. Future service management, websites, MCP, secret transmission, and remote command execution are explicitly out of scope for the current milestone.

## Current Milestone

Secure mesh foundation milestone. This milestone should establish local, restart-safe foundations for machine identity, network identity, master/worker roles, join-code pairing metadata, single-use join-code state, peer metadata, revocation metadata, secure config layout, transport interfaces, thin CLI/TUI skeletons, and shared control/actions.

## Current Status

Step 2 architecture and security planning is complete. No implementation files were created in Step 2. The approved direction is to implement only local foundations and interfaces in this milestone, while designing toward a future low-overhead encrypted mesh that uses reviewed cryptographic building blocks rather than custom crypto primitives.

## Completed Steps

- [x] Step 1: Repository inspection
- [x] Step 2: Architecture and security plan
- [ ] Step 3: File plan
- [ ] Step 4: Foundation types
- [ ] Step 5: Crypto and persistence
- [ ] Step 6: Join-code and revocation logic
- [ ] Step 7: Thin control/CLI/TUI skeleton
- [ ] Step 8: Tests
- [ ] Step 9: Documentation

## Pending Steps

- Await user validation of Step 2 architecture and security plan.
- After approval, proceed to Step 3: exact file plan.
- In Step 3, list every file to create or modify, its package, key types/functions, and planned tests.

## Repository Structure

Current repository structure:

```text
repo-root/
  .git/
  AGENTS.md
  context.md
  go.mod
```

No `go.work` file exists yet. No `cmd/` directory exists yet. No `packages/` directory exists yet. No Go source files exist yet.

Recommended future structure, pending Step 3 approval:

```text
repo-root/
  go.work
  context.md
  cmd/
    infra/
      main.go
    infra-tui/
      main.go
  packages/
    securemesh/
      go.mod
      identity/
      join/
      config/
      peer/
      crypto/
      revocation/
      network/
      README.md
    control/
      go.mod
      actions/
      README.md
```

## Files Created or Modified

- `context.md`: Created in Step 1 and updated in Step 2 with architecture, security model, technology evaluation, assumptions, and next steps. It intentionally contains no secrets, join codes, credentials, private keys, tokens, or sensitive derived material.

## Important Design Decisions

- Preserve module path `github.com/f1forhelp/tailed-box-cli` from `go.mod`.
- Preserve Go version `1.25.1`; do not downgrade or change it unless explicitly requested.
- Use the user-requested pause workflow: update `context.md` after each logical step, then wait for validation.
- Keep business logic out of future CLI/TUI entrypoints and place it behind a shared control/action layer.
- Implement local foundations before any production network transport.
- Do not implement custom cryptographic primitives or custom crypto math.
- Prefer reviewed primitives and protocols: Go standard library crypto for foundational pieces, and a reviewed Noise-style implementation or reviewed protocol design for the future transport.
- Safe default for future transport direction: hybrid architecture with a minimal encrypted UDP data plane using a Noise-style authenticated handshake, plus optional QUIC/TLS 1.3 reliable control streams later if justified.
- Do not choose any design that requires external system VPN tooling, kernel VPN features, OS-managed VPN configuration, or shelling out to networking/VPN commands.

## Step 2 Architecture And Security Plan

### Secure Mesh Architecture

The planned architecture separates responsibilities so future interfaces can share behavior without duplicating security logic:

```text
Direct CLI
Thin TUI
Future Web Dashboard
Future MCP Server
        ↓
Shared Control / Action Layer
        ↓
Secure Mesh Foundation
        ↓
Future Secure Transport
        ↓
Future Service Managers
```

The current milestone should implement the secure mesh foundation and shared control/action layer only. Transport package code should define interfaces, metadata, and future packet/session concepts, but not a production mesh implementation.

### Identity Model

- A node has a persistent `NodeID`, `Role`, `NetworkID`, public key material, private key material, creation timestamp, and key algorithm metadata.
- Valid roles for this milestone are `master` and `worker`.
- A `NetworkID` identifies the local mesh network and is generated with cryptographically secure randomness when a network is initialized.
- A `NodeID` should be derived from public identity material using a stable hash/encoding scheme, not from hostnames, IP addresses, MAC addresses, or mutable local machine attributes.
- Recommended identity key plan: use an Ed25519 signing identity key for stable node identity and future signed membership/revocation records, plus an X25519 static transport public key for future Noise-style handshakes.
- The Ed25519 and X25519 public keys should be persisted together in the identity record so future transport credentials are explicitly bound to the node identity.
- Private identity material must be persisted with restrictive permissions and never logged.
- Restart-safe reconnects should rely on persistent node identity and peer membership state, not old join codes.

### Join-Code Model

- Join codes are only for initial pairing, never for routine reconnects.
- Join codes are generated only by an authorized local master node.
- A join code is high entropy, generated with cryptographically secure randomness, and practically unguessable.
- Safe default entropy target: at least 256 bits of random secret material, encoded with a human-transferable encoding such as base32 without padding and optional display grouping.
- No plaintext join code should be stored persistently.
- Persisted join-code state should store only verifier/hash material plus non-secret metadata: network ID, expected joining role, issuing master node ID, creation timestamp, consumed state, and optional consumed timestamp/node ID.
- Because codes are high entropy, a fast verifier such as SHA-256 or HMAC-SHA-256 with a per-code random salt is acceptable for local verifier storage. Constant-time comparison should be used when comparing verifier bytes.
- Validation must reject invalid codes, already-consumed codes, wrong-network codes, wrong-role codes, and codes created by unauthorized nodes.
- Consuming a join code must be atomic in the local persistence model so concurrent local operations cannot use the same code twice.
- No mandatory expiry is required in this milestone. This is a deliberate product decision; the compensating controls are high entropy, no plaintext persistence, explicit unused/consumed state, and single-use enforcement.

### Join-Code Pairing Ambiguity And Safe Default

There is one important future transport ambiguity: verifier-only storage is ideal for local secrecy, but a future online authenticated pairing handshake must also resist MITM without requiring the master to store plaintext join codes.

Options considered:

- Verifier-only storage plus a PAKE/OPAQUE-style pairing protocol later. This best preserves the no-plaintext-storage goal and can resist MITM, but adds protocol/dependency complexity.
- Store encrypted join-code secret material locally and use it as a Noise PSK later. This is simpler for a Noise-PSK bootstrap, but the encrypted secret becomes sensitive persisted material and is weaker than verifier-only storage if local state is compromised.
- Send the join code over an unauthenticated channel. This is rejected because it exposes the pairing secret to interception and MITM risk.

Safe default for this milestone: implement verifier-only local join-code foundations and do not implement online pairing transport yet. Future pairing should use a reviewed PAKE/OPAQUE-style approach or a Noise-style flow with explicit out-of-band master identity/fingerprint verification and transcript binding. The future handshake must bind network ID, expected role, master identity, worker identity, and join authorization into the authenticated transcript.

### Revocation Model

- Revocation is modeled locally in this milestone; no multi-master consensus is implemented.
- A master can create a local revocation record for a worker or another master.
- A revocation record includes node ID, role, revoked timestamp, revoked-by master node ID, and optional reason.
- A revoked node must not be treated as an active peer.
- A revoked node must not reconnect with old credentials.
- Safe default: a revoked `NodeID` remains blocked. Rejoining requires a new join code and fresh node identity material unless a future explicit authorized unrevocation or membership-epoch design is added.
- Future open questions include revocation propagation, master-removal quorum, split-brain handling, revocation signatures, and how to safely recover from compromised or lost master nodes.

### Local Persistence Model

- Use an application config root based on `os.UserConfigDir()` by default, with an injectable root for tests.
- Safe default app directory name: `tailed-box-cli`, while public command names can remain `infra` and `infra-tui` unless changed by user decision.
- Create directories with restrictive permissions, preferably `0700` on Unix-like systems.
- Store private identity files and local security state with restrictive permissions, preferably `0600` on Unix-like systems.
- Use atomic write patterns: write to a temporary file in the same directory, set restrictive permissions, flush where practical, then rename.
- For atomic join-code consumption, use a local lock around read-check-update-write. A cross-platform lock-directory approach using `os.Mkdir` is a possible low-dependency default for this milestone.
- Local state should be structured so tests can run in temporary directories without touching real user config.
- Do not store secrets or private key material in `context.md`, logs, test output, or error messages.

Recommended local state layout for later implementation:

```text
config-root/
  identity.json
  network.json
  join-codes.json
  peers.json
  revocations.json
  locks/
```

### MITM Prevention Strategy

- Routine reconnects should use mutually authenticated peer identities, not join codes.
- Future handshakes must authenticate long-term node identities and bind the transcript to the network ID and role expectations.
- Unknown or revoked public keys must be rejected by the peer allowlist/revocation checks before a session becomes usable.
- For initial pairing, join authorization must be bound to the authenticated handshake. The system must not accept a peer solely because it can open a UDP socket or present an arbitrary public key.
- Future pairing must avoid sending join codes in plaintext over unauthenticated channels.
- The future transport should include replay protection, key separation, session IDs, monotonically checked counters or sliding windows, and key rotation.

### Low-Overhead Transport Strategy

- Do not implement the full production encrypted mesh in this milestone.
- Define transport interfaces and metadata that can support a future UDP data plane.
- Future data plane should use compact binary framing, authenticated encryption, replay protection, explicit key separation, and bounded per-packet work.
- Future session establishment should use a Noise-style authenticated handshake with long-term identity keys and ephemeral session keys.
- Future payload encryption should use reviewed AEAD primitives such as ChaCha20-Poly1305 or AES-GCM as selected by the reviewed transport implementation and benchmarks.
- Future key rotation should be designed by packet count, byte count, or time interval.
- Reliable control streams may be added later if needed; QUIC/TLS 1.3 is a candidate, but should not be added until a concrete need exists.
- NAT traversal is left for a later milestone. This milestone may define endpoint metadata but should not implement traversal.

### Technology Evaluation

| Option | Security | Speed/Overhead | Complexity | Cross-Platform | NAT Traversal | Local Testability | Go Fit | Multi-Master Fit | Revocation Fit | Bad Crypto Risk | Operational Simplicity | Maintainability |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Custom minimal encrypted UDP mesh | Potentially strong only if built on reviewed protocols/primitives; unsafe if ad hoc | Best potential per-packet overhead | High once replay, rotation, handshake, and reliability are included | Good in user space | Must be added separately | Good | Good | Good if membership is separate | Good if allowlists are separate | High if crypto protocol is invented | Good after built, hard during development | Medium to high burden |
| Noise-style transport | Strong practical security when using reviewed Noise patterns and primitives | Low overhead; good fit for UDP | Medium | Good | Must be added separately | Good | Good | Good with allowlist/revocation layer | Lower if using reviewed implementation | Good | Good if implementation is maintained |
| QUIC + TLS 1.3 | Strong and mature | Higher overhead than minimal UDP but efficient for reliable streams | Medium to high dependency/protocol complexity | Good | Better NAT behavior than raw TCP, still not traversal by itself | Good | Strong Go ecosystem | Good for control channels | Good with certificate/key authorization | Low if using mature library | Good, but heavier | Good if dependency is stable |
| SSH-based control plane | Mature security for remote administration | Fine for commands, poor fit for low-overhead mesh data plane | Medium operational complexity | Good where SSH exists | No mesh traversal story | Good | Good libraries, but semantics are admin-shell oriented | Weak fit for mesh membership | Key revocation possible but operationally awkward | Low protocol risk | Depends on existing SSH model/daemon unless embedded | Less aligned with product goals |
| Hybrid: Noise-style UDP data plane plus optional QUIC/TLS control later | Strong if boundaries are clear and both use reviewed primitives | Low overhead data path, reliable control available only when needed | Medium, staged over milestones | Good | Traversal still later | Good | Good | Good | Good with shared membership/revocation | Low to medium, depending on implementation choices | Good because pieces are added only when needed | Best long-term balance |

Recommended direction: use the hybrid design as the long-term target, but implement only foundations and interfaces in this milestone. The first real transport milestone should prefer a reviewed Noise-style handshake over UDP for low-overhead peer sessions. QUIC/TLS 1.3 should remain an optional future reliable control transport, not the initial data-plane dependency. SSH should not be used as the core control or mesh design.

### Technology Recommendations For This Milestone

- Use Go standard library cryptography where possible: `crypto/rand`, `crypto/sha256`, `crypto/hmac`, `crypto/subtle`, `crypto/ed25519`, and `crypto/ecdh`.
- Use stable text encodings such as base32/base64url for IDs and code display, with decoding validation.
- Keep package APIs algorithm-aware so key formats can evolve without silent migrations.
- Do not add a Noise, QUIC, or TUI dependency until the specific implementation step requires it and the user approves the file plan.
- Keep all state managers testable with injected filesystem roots and clocks.

## Assumptions

- The app can use `infra` and `infra-tui` as initial binary names unless the user chooses different names.
- The app config directory can default to `tailed-box-cli` under `os.UserConfigDir()` unless the user chooses a different product/config name.
- One local config root represents one mesh network for this milestone.
- Rejoining after revocation should use a new join code and fresh node identity by default; old revoked node IDs remain blocked.
- Multi-master authorization, revocation quorum, consensus, and propagation are future design topics, not milestone-one features.
- The first implementation should optimize correctness and security boundaries before hot-path performance.
- Transport abstractions should be designed for future UDP/Noise but should not require transport dependencies yet.

## Security Notes

- `context.md` must never store secrets, private keys, join codes, passwords, tokens, credentials, or sensitive derived material.
- Join codes must eventually be stored only as verifier/hash material, not plaintext.
- Join-code verifier comparison should use constant-time comparison.
- Join codes have no mandatory expiry in this milestone by explicit requirement.
- Private identity material must be persisted with restrictive permissions and never logged.
- Revoked nodes must not be active peers and must not reconnect with old credentials.
- Peer allowlists, network ID checks, role checks, and revocation checks are required before future sessions are accepted.
- Do not claim the system is unhackable or non-hackable. Use accurate language such as cryptographically strong, high entropy, single-use, practically unguessable, designed to resist MITM, and low-overhead encrypted mesh.

## Commands Run

- Repository directory inspection using file-read tooling.
- File glob inspection for `**/*`, `context.md`, `go.work`, and `**/*.go`.
- Read `go.mod`.
- `git status --short` returned no output before creating `context.md` in Step 1.
- Read `context.md` at the start of Step 2.
- `git status --short` in Step 2 showed `?? context.md`, expected because the context file was newly created and not committed.

## Test Results

- No tests were run in Step 1 or Step 2.
- No Go packages currently exist, so `go test ./...` would not yet exercise project code.

## Known Issues

- No source tree exists yet.
- No workspace file exists yet.
- No CLI or TUI code exists yet.
- No tests exist yet.
- No README or security documentation exists yet.
- Future pairing handshake needs a deliberate choice between verifier-only PAKE/OPAQUE-style pairing and another reviewed MITM-resistant bootstrap design.

## Open Questions

- Validate whether to proceed with the suggested multi-module layout using root `go.work`, `packages/securemesh`, and `packages/control`.
- Validate whether the CLI binary name should remain `infra` or use another public command name.
- Validate whether the default config directory name should be `tailed-box-cli`.
- Validate the recommended future transport direction: Noise-style UDP data plane, optional QUIC/TLS reliable control later if needed, and no SSH/VPN-based mesh.
- Validate the safe default that revoked nodes must rejoin with fresh node identity material, not old node IDs.

## Hard Boundaries

- Do not implement Postgres, Redis, Valkey, Docker, service installation, log streaming, worker command execution, website/dashboard, MCP server, secret transmission, admin/root remote command execution, full production mesh transport, external system VPN integration, kernel-level VPN dependency, shelling out to networking/VPN tools, or multi-master consensus in this milestone.

## Next Recommended Action

Wait for user validation of Step 2. After approval, proceed to Step 3: exact file plan.

## Resume Instructions

Future sessions must first read `context.md`, then inspect the repository for changes made after this file was last updated. Continue only from the next pending step and preserve the pause-after-each-logical-step workflow.
