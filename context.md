# Project Context

## Project Goal

Build a CLI-first infrastructure management application with a future secure machine-to-machine mesh between master and worker nodes. The current focus is only the local foundation for secure mesh identity, pairing, revocation, persistence, transport abstractions, and shared CLI/TUI control flow. Future service management, websites, MCP, secret transmission, and remote command execution are explicitly out of scope for the current milestone.

## Current Milestone

Secure mesh foundation milestone. This milestone should establish local, restart-safe foundations for machine identity, network identity, master/worker roles, join-code pairing metadata, single-use join-code state, peer metadata, revocation metadata, secure config layout, transport interfaces, thin CLI/TUI skeletons, and shared control/actions.

## Current Status

Step 6 join-code and revocation logic is complete. The repository now has local join-code generation/verifier logic, single-use join-code consumption under a local lock, peer state storage, and revocation record/check storage. Per the latest user instruction, continue without pausing for approval between todo steps, but still update `context.md` after every logical step and create a brief commit after each completed todo step.

## Completed Steps

- [x] Step 1: Repository inspection
- [x] Step 2: Architecture and security plan
- [x] Step 3: File plan
- [x] Step 4: Foundation types
- [x] Step 5: Crypto and persistence
- [x] Step 6: Join-code and revocation logic
- [ ] Step 7: Thin control/CLI/TUI skeleton
- [ ] Step 8: Tests
- [ ] Step 9: Documentation

## Pending Steps

- Proceed directly to Step 7: implement the thin shared control layer plus CLI/TUI skeletons.
- Step 7 should add `packages/control`, shared actions, equivalent CLI command strings, root command-module wiring, and thin `cmd/infra` plus `cmd/infra-tui` entrypoints.
- Step 7 must keep business logic out of CLI/TUI, must not add full production transport, online pairing transport, secret transmission, service management, website, MCP, tests, or final documentation.

## Repository Structure

Current repository structure:

```text
repo-root/
  .git/
  AGENTS.md
  context.md
  go.mod
  go.work
  packages/
    securemesh/
      go.mod
      config/
        file.go
        lock.go
        paths.go
      crypto/
        encoding.go
        hash.go
        random.go
      identity/
        generate.go
        store.go
        types.go
      join/
        code.go
        store.go
        types.go
      network/
        types.go
        transport.go
      peer/
        store.go
        types.go
      revocation/
        store.go
        types.go
```

No `cmd/` directory exists yet. No `packages/control` module exists yet. No tests, README, or security documentation exist yet.

Planned milestone structure after implementation and documentation:

```text
repo-root/
  go.mod
  go.work
  context.md
  context_test.go
  README.md
  SECURITY.md
  cmd/
    infra/
      main.go
      main_test.go
    infra-tui/
      main.go
      main_test.go
  packages/
    securemesh/
      go.mod
      README.md
      config/
        paths.go
        file.go
        lock.go
        config_test.go
      crypto/
        random.go
        hash.go
        encoding.go
        crypto_test.go
      identity/
        types.go
        generate.go
        store.go
        identity_test.go
      join/
        types.go
        code.go
        store.go
        join_test.go
      network/
        types.go
        transport.go
        network_test.go
      peer/
        types.go
        store.go
        peer_test.go
      revocation/
        types.go
        store.go
        revocation_test.go
    control/
      go.mod
      README.md
      actions/
        result.go
        options.go
        identity.go
        join.go
        peer.go
        revocation.go
        actions_test.go
```

`go.sum` is not planned because the milestone should use only the Go standard library unless a later approved step introduces an external dependency. If `go mod tidy` creates `go.sum`, it should be kept only if needed.

## Files Created or Modified

- `context.md`: Created in Step 1, updated in Step 2 with architecture and security planning, updated in Step 3 with the exact file plan, and updated in Step 4 with implementation status. It intentionally contains no secrets, join codes, credentials, private keys, tokens, or sensitive derived material.
- `go.work`: Added in Step 4 to include the root module and `packages/securemesh` in the local workspace.
- `packages/securemesh/go.mod`: Added in Step 4 as the securemesh foundation module with Go `1.25.1` and no external dependencies.
- `packages/securemesh/identity/types.go`: Added in Step 4 with `NodeID`, `NetworkID`, `Role`, key metadata, identity, network, and validation types.
- `packages/securemesh/join/types.go`: Added in Step 4 with join-code metadata types, verifier-backed record shape, status values, and request/result types. It does not persist plaintext join codes.
- `packages/securemesh/peer/types.go`: Added in Step 4 with peer records, endpoint metadata, active/revoked status, and validation.
- `packages/securemesh/revocation/types.go`: Added in Step 4 with local revocation record metadata and validation.
- `packages/securemesh/network/types.go`: Added in Step 4 with future transport/session metadata constants and validation.
- `packages/securemesh/network/transport.go`: Added in Step 4 with future transport, session, endpoint, message, dial, listen, and peer-authenticator interfaces/types.
- `packages/securemesh/crypto/random.go`: Added in Step 5 with cryptographically secure random byte/base32 helpers and injectable-reader support for future tests.
- `packages/securemesh/crypto/hash.go`: Added in Step 5 with SHA-256, HMAC-SHA-256, and constant-time equality helpers.
- `packages/securemesh/crypto/encoding.go`: Added in Step 5 with base32 no-padding, base64url, and join-code-friendly base32 normalization helpers.
- `packages/securemesh/config/paths.go`: Added in Step 5 with default `tailed-box-cli` config root, local state file paths, lock paths, and restrictive directory creation.
- `packages/securemesh/config/file.go`: Added in Step 5 with atomic file writes, JSON save/load, restrictive file permissions, and best-effort directory sync.
- `packages/securemesh/config/lock.go`: Added in Step 5 with a low-dependency directory lock helper for local atomic operations.
- `packages/securemesh/identity/generate.go`: Added in Step 5 with Ed25519 identity generation, X25519 transport key generation, stable public-key-derived node IDs, and random network IDs.
- `packages/securemesh/identity/store.go`: Added in Step 5 with restart-safe identity/network save/load helpers using restrictive local persistence.
- `packages/securemesh/join/code.go`: Added in Step 6 with 256-bit join-code generation, HMAC-SHA-256 verifier creation, code ID derivation, create/consume request validation, wrong-network/wrong-role errors, and no expiry field.
- `packages/securemesh/join/store.go`: Added in Step 6 with local join-code state, lock-protected creation, lock-protected validate-and-consume behavior, single-use consumed state, and no plaintext join-code persistence.
- `packages/securemesh/peer/store.go`: Added in Step 6 with local peer add/get/list/active-peer behavior and revocation marking.
- `packages/securemesh/revocation/store.go`: Added in Step 6 with revocation record creation, idempotent local revoke behavior, list/get, and revoked-node checks.

## Important Design Decisions

- Preserve module path `github.com/f1forhelp/tailed-box-cli` from root `go.mod`.
- Preserve Go version `1.25.1`; do not downgrade or change it unless explicitly requested.
- Original workflow required pausing after each logical step. The latest user instruction supersedes that: continue without repeated approvals, still update `context.md` after each todo step, and commit after every completed todo step.
- Keep business logic out of future CLI/TUI entrypoints and place it behind a shared control/action layer.
- Implement local foundations before any production network transport.
- Do not implement custom cryptographic primitives or custom crypto math.
- Prefer reviewed primitives and protocols: Go standard library crypto for foundational pieces, and a reviewed Noise-style implementation or reviewed protocol design for the future transport.
- Long-term transport direction: hybrid architecture with a minimal encrypted UDP data plane using a Noise-style authenticated handshake, plus optional QUIC/TLS 1.3 reliable control streams later if justified.
- Do not choose any design that requires external system VPN tooling, kernel VPN features, OS-managed VPN configuration, or shelling out to networking/VPN commands.
- Use `infra` and `infra-tui` as initial binary names.
- Use `tailed-box-cli` as the default application config directory name.
- Use a multi-module workspace with root `go.work`, root `go.mod` for commands, `packages/securemesh/go.mod`, and `packages/control/go.mod`.
- A revoked node must rejoin with a new join code and fresh node identity material by default.
- Step 4 intentionally created only foundation types and interfaces; generation, persistence, join-code logic, stores, control, CLI/TUI, tests, and docs remain for later steps.
- Step 5 uses only Go standard-library cryptography and persistence APIs. No external dependencies were added.
- Step 5 stores private identity material only through restrictive local JSON persistence helpers; it does not log or write generated secrets to `context.md`.
- Step 5 derives `NodeID` from Ed25519 and X25519 public key material with domain-separated SHA-256 and base32 encoding, not from hostnames, IPs, MACs, or mutable machine attributes.
- Step 5 changed the identity struct field name from `PrivateKey` to `PrivateKeys` while preserving the `json:"private_keys"` field name; no compatibility migration is needed because no persisted release exists yet.
- Step 6 join codes use 32 random bytes encoded as base32 without padding. The generated plaintext code is returned to the caller exactly once and is not stored in persistent state.
- Step 6 join-code records store verifier material, salt, network ID, expected role, issuing master node ID, creation timestamp, and consumed metadata only.
- Step 6 validates join codes by recomputing per-record verifiers and comparing with constant-time equality.
- Step 6 consumes join codes under a local directory lock so local concurrent callers cannot successfully use the same code twice.
- Step 6 implements local revocation state only. No multi-master consensus or propagation is implemented.

## Step 2 Architecture And Security Plan

### Secure Mesh Architecture

The planned architecture separates responsibilities so future interfaces can share behavior without duplicating security logic:

```text
Direct CLI
Thin TUI
Future Web Dashboard
Future MCP Server
        v
Shared Control / Action Layer
        v
Secure Mesh Foundation
        v
Future Secure Transport
        v
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
- Use `tailed-box-cli` as the default application config directory name.
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

## Step 3 File Plan

### Workspace And Root Files

- `go.work`: Workspace file. Purpose: include root command module plus `packages/securemesh` and, once created, `packages/control`. Key content: `go 1.25.1`, `use .`, `use ./packages/securemesh`, and later `use ./packages/control`. Tests: none directly.
- `go.mod`: Root module `github.com/f1forhelp/tailed-box-cli`. Purpose: keep command packages under the root module. Key changes: preserve `go 1.25.1`; in Step 7 add local `require`/`replace` for `github.com/f1forhelp/tailed-box-cli/packages/control` if the command packages need it to build outside workspace mode. Tests: root command tests compile against this module.
- `context_test.go`: Root test package. Purpose: verify `context.md` exists and records the current milestone status. Key test: context file is present and mentions secure mesh foundation/current milestone. Tests added in Step 8.
- `README.md`: Root documentation. Purpose: milestone overview, implemented/not implemented list, CLI/TUI usage, and future direction. Added in Step 9.
- `SECURITY.md`: Root security documentation. Purpose: security model, join-code properties, revocation model, MITM prevention plan, transport plan, no external VPN dependency. Added in Step 9.

### Securemesh Module Files

- `packages/securemesh/go.mod`: Module `github.com/f1forhelp/tailed-box-cli/packages/securemesh`. Purpose: secure mesh foundation module. Key content: `go 1.25.1`; no external dependencies planned. Tests: all securemesh package tests compile under this module.
- `packages/securemesh/identity/types.go`: Package `identity`. Purpose: foundation identity types. Key types/functions: `NodeID`, `NetworkID`, `Role`, role constants, `Role.Valid`, `ParseRole`, `PublicKeySet`, `PrivateKeySet`, `Identity`, `Network`, validation errors. Tests: role validation, network identity type validation, identity validation.
- `packages/securemesh/identity/generate.go`: Package `identity`. Purpose: identity and network generation after crypto helpers exist. Key functions: `GenerateNetwork`, `GenerateIdentity`, `DeriveNodeID`. Tests: identity generation, network ID creation, stable node ID derivation, key algorithm metadata.
- `packages/securemesh/identity/store.go`: Package `identity`. Purpose: restart-safe identity/network save/load after config persistence exists. Key functions: `SaveIdentity`, `LoadIdentity`, `SaveNetwork`, `LoadNetwork`. Tests: identity save/load, restart-safe persistence, private file permissions where possible.
- `packages/securemesh/crypto/random.go`: Package `crypto`. Purpose: secure random helpers. Key functions: `RandomBytes`, `RandomBase32`, future join-code random secret generation helper. Tests: expected byte lengths, non-empty randomness, error behavior using injectable reader if added.
- `packages/securemesh/crypto/hash.go`: Package `crypto`. Purpose: hash/verifier helpers. Key functions: `SHA256`, `HMACSHA256`, `ConstantTimeEqual`. Tests: constant-time helper behavior, verifier mismatch behavior.
- `packages/securemesh/crypto/encoding.go`: Package `crypto`. Purpose: safe encoding helpers. Key functions: base32 no-padding encode/decode, base64url encode/decode if needed. Tests: round trip, invalid input rejection, expected join-code display length support.
- `packages/securemesh/config/paths.go`: Package `config`. Purpose: config root and local state paths. Key types/functions: `Paths`, `DefaultRoot`, `NewPaths`, `Ensure`, `IdentityPath`, `NetworkPath`, `JoinCodesPath`, `PeersPath`, `RevocationsPath`, `LocksDir`. Tests: expected filenames, injectable temp root, directory permissions where possible.
- `packages/securemesh/config/file.go`: Package `config`. Purpose: safe local reads/writes. Key functions: `AtomicWriteFile`, `ReadFile`, `SaveJSON`, `LoadJSON`. Tests: write/read round trip, restrictive file permissions where possible, no partial plaintext assumptions.
- `packages/securemesh/config/lock.go`: Package `config`. Purpose: local atomic operation lock helper. Key type/functions: `DirLock`, `AcquireLock`, `Release`. Tests: prevents double acquire, releases cleanly, works in temp directory.
- `packages/securemesh/join/types.go`: Package `join`. Purpose: join-code metadata types only in Step 4. Key types/functions: `CodeID`, `Record`, `Status`, `CreateRequest`, `ConsumeRequest`, `ConsumeResult`, validation methods. Tests: metadata validation, status behavior.
- `packages/securemesh/join/code.go`: Package `join`. Purpose: join-code generation/verifier logic in Step 6. Key functions: `GenerateCode`, `NewRecord`, `VerifierForCode`. Tests: high entropy/expected length, no mandatory expiry, no plaintext stored in record.
- `packages/securemesh/join/store.go`: Package `join`. Purpose: local join-code state and single-use consumption in Step 6. Key type/functions: `Store`, `Create`, `ValidateAndConsume`, `List`, consumed-state update under lock. Tests: invalid rejection, already-used rejection, wrong role/network rejection, single-use behavior, consumed atomically.
- `packages/securemesh/peer/types.go`: Package `peer`. Purpose: peer metadata types. Key types/functions: `Status`, `Endpoint`, `Record`, `Record.Active`, role/public-key metadata. Tests: active versus revoked status behavior.
- `packages/securemesh/peer/store.go`: Package `peer`. Purpose: local peer state after persistence exists. Key type/functions: `Store`, `Add`, `Get`, `List`, `MarkRevoked`, `ActivePeers`. Tests: revoked node not active in local model.
- `packages/securemesh/revocation/types.go`: Package `revocation`. Purpose: revocation metadata types only in Step 4. Key types/functions: `Record`, `Reason`, validation methods. Tests: revocation record validation.
- `packages/securemesh/revocation/store.go`: Package `revocation`. Purpose: local revocation state and checks in Step 6. Key type/functions: `Store`, `Revoke`, `IsRevoked`, `List`. Tests: record creation, revoked-node check, revoked metadata fields.
- `packages/securemesh/network/types.go`: Package `network`. Purpose: future transport/session metadata types. Key types/functions: `Protocol`, `SessionID`, `SessionState`, `SessionMetadata`, `PacketType`, `MessageType`. Tests: type validation and constants compile.
- `packages/securemesh/network/transport.go`: Package `network`. Purpose: future transport interfaces only. Key interfaces/types: `Transport`, `Session`, `DialOptions`, `ListenOptions`, `PeerAuthenticator`. Tests: compile-time fake implementation can satisfy interfaces.
- `packages/securemesh/README.md`: Securemesh documentation. Purpose: package responsibilities, security boundaries, future transport notes. Added in Step 9.

### Control Module Files

- `packages/control/go.mod`: Module `github.com/f1forhelp/tailed-box-cli/packages/control`. Purpose: shared control/action layer module. Key content: `go 1.25.1`; local dependency on `github.com/f1forhelp/tailed-box-cli/packages/securemesh`. Tests: action tests compile under this module.
- `packages/control/actions/result.go`: Package `actions`. Purpose: common action result model. Key types/functions: `Result`, `EquivalentCLI`, `Message`, `SecretValue` if needed for explicit display handling, `Command` helper. Tests: equivalent CLI strings returned by supported actions.
- `packages/control/actions/options.go`: Package `actions`. Purpose: injected dependencies. Key types/functions: `Options`, `WithConfigRoot`, `WithClock`, internal `env` construction. Tests: temp config root use, deterministic clock behavior.
- `packages/control/actions/identity.go`: Package `actions`. Purpose: shared identity/network actions. Key functions: `InitNetwork`, `InitIdentity`, `ShowIdentity`. Tests: actions call securemesh packages and return equivalent CLI command.
- `packages/control/actions/join.go`: Package `actions`. Purpose: shared join-code actions. Key functions: `CreateJoinCode`, `ConsumeJoinCode`. Tests: create/consume call join package and return equivalent CLI command.
- `packages/control/actions/peer.go`: Package `actions`. Purpose: shared peer listing/status action. Key functions: `ListPeers`. Tests: action reports active/revoked model without duplicating peer logic.
- `packages/control/actions/revocation.go`: Package `actions`. Purpose: shared revocation action. Key functions: `RevokePeer`. Tests: action creates revocation and returns equivalent CLI command.
- `packages/control/actions/actions_test.go`: Package `actions`. Purpose: control action tests. Key tests: equivalent CLI command strings, no business logic in CLI/TUI needed for tested behavior, actions use temp config root.
- `packages/control/README.md`: Control documentation. Purpose: explain shared action layer for CLI/TUI/future Web/future MCP. Added in Step 9.

### Command Files

- `cmd/infra/main.go`: Package `main`. Purpose: thin CLI skeleton. Key functions: `main`, `run`, command parsing, usage output, role parsing adapter, calls into `packages/control/actions`. Supported local-only commands planned for Step 7: `network init`, `identity init --role master|worker`, `identity show`, `join-code create --role master|worker`, `join-code consume --code <code> --role master|worker`, `peer list`, `peer revoke --node <node-id> --role master|worker [--reason <reason>]`. Tests: CLI invokes control runner rather than duplicating business logic.
- `cmd/infra/main_test.go`: Package `main`. Purpose: CLI skeleton tests. Key tests: parser maps a command to the expected control action, equivalent CLI output is surfaced, invalid arguments fail before action invocation.
- `cmd/infra-tui/main.go`: Package `main`. Purpose: thin text-menu TUI skeleton using only standard library. Key functions: `main`, `run`, menu item definitions, action dispatch through control actions, equivalent CLI command display for each menu item. Tests: menu items carry equivalent CLI command and dispatch through injected control functions.
- `cmd/infra-tui/main_test.go`: Package `main`. Purpose: TUI skeleton tests. Key tests: TUI calls control layer, displays equivalent CLI command for supported action, no duplicated business logic.

### Test Execution Plan

There is one important ambiguity from the approved multi-module layout: `go test ./...` from the repository root may not cover nested modules in all Go workspace/module modes. Safe default for Step 8 is to run the required root command and also run the same command inside each module:

```text
go test ./...
go test ./...    # with workdir packages/securemesh
go test ./...    # with workdir packages/control
```

After adding or changing imports/dependencies, run `go mod tidy` in the affected module directories. With the planned standard-library-only implementation, no external dependency downloads are expected.

## Assumptions

- The user approval after Step 2 validated use of `infra` and `infra-tui` as command names.
- The user approval after Step 2 validated `tailed-box-cli` as the default config directory name.
- The user approval after Step 2 validated the multi-module workspace layout using root `go.work`, `packages/securemesh`, and `packages/control`.
- The user approval after Step 2 validated the hybrid future transport direction: Noise-style UDP data plane, optional QUIC/TLS reliable control later, no SSH/VPN-based mesh.
- The user approval after Step 2 validated the safe default that revoked nodes must rejoin with fresh node identity material, not old node IDs.
- One local config root represents one mesh network for this milestone.
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
- CLI output may need to display a newly generated join code exactly once to the authorized user. That value must not be written to persistent state, logs, docs, or `context.md`.
- Do not claim the system is unhackable or non-hackable. Use accurate language such as cryptographically strong, high entropy, single-use, practically unguessable, designed to resist MITM, and low-overhead encrypted mesh.

## Commands Run

- Repository directory inspection using file-read tooling.
- File glob inspection for `**/*`, `context.md`, `go.work`, and `**/*.go`.
- Read `go.mod`.
- `git status --short` returned no output before creating `context.md` in Step 1.
- Read `context.md` at the start of Step 2.
- `git status --short` in Step 2 showed `?? context.md`, expected because the context file was newly created and not committed at that point.
- Read `context.md` at the start of Step 3.
- `git status --short` at the start of Step 3 returned no output in this environment.
- Inspected `git status --short`, `git diff -- context.md`, and `git log --oneline -10` before committing Step 3.
- `git add context.md && git commit -m "docs: record file plan"` created Step 3 commit `36acba7`.
- Added Step 4 files using `apply_patch`.
- `gofmt -w` formatted Step 4 Go source files.
- `go test ./...` from the repository root returned `no packages to test`, expected before command packages exist.
- `go test ./...` from `packages/securemesh` passed for all foundation packages with no test files yet.
- `go mod tidy` from `packages/securemesh` completed with no output.
- Inspected `git status --short`, `git diff`, and `git log --oneline -10` before committing Step 4.
- `git add ... && git commit -m "feat: add securemesh foundation types"` created Step 4 commit `6f2c520`.
- Added Step 5 files using `apply_patch`.
- `gofmt -w` formatted Step 5 Go source files.
- `go test ./...` from the repository root returned `no packages to test`, expected before command packages exist.
- `go test ./...` from `packages/securemesh` passed for `config`, `crypto`, `identity`, `join`, `network`, `peer`, and `revocation`; no test files yet.
- `go mod tidy` from `packages/securemesh` completed with no output.
- Inspected `git status --short`, `git diff`, and `git log --oneline -10` before committing Step 5.
- `git add ... && git commit -m "feat: add crypto and persistence helpers"` created Step 5 commit `6579122`.
- Added Step 6 files using `apply_patch`.
- `gofmt -w` formatted Step 6 Go source files.
- `go test ./...` from the repository root returned `no packages to test`, expected before command packages exist.
- `go test ./...` from `packages/securemesh` passed for `config`, `crypto`, `identity`, `join`, `network`, `peer`, and `revocation`; no test files yet.
- `go mod tidy` from `packages/securemesh` completed with no output.

## Test Results

- No tests were run in Step 1, Step 2, or Step 3.
- Step 4 verification: root `go test ./...` returned `no packages to test`.
- Step 4 verification: `packages/securemesh` `go test ./...` passed for `identity`, `join`, `network`, `peer`, and `revocation`; no test files exist yet.
- Step 5 verification: root `go test ./...` returned `no packages to test`.
- Step 5 verification: `packages/securemesh` `go test ./...` passed for `config`, `crypto`, `identity`, `join`, `network`, `peer`, and `revocation`; no test files exist yet.
- Step 6 verification: root `go test ./...` returned `no packages to test`.
- Step 6 verification: `packages/securemesh` `go test ./...` passed for `config`, `crypto`, `identity`, `join`, `network`, `peer`, and `revocation`; no test files exist yet.

## Known Issues

- No CLI or TUI code exists yet.
- `packages/control` does not exist yet.
- Tests do not exist yet.
- No tests exist yet.
- No README or security documentation exists yet.
- Future pairing handshake needs a deliberate choice between verifier-only PAKE/OPAQUE-style pairing and another reviewed MITM-resistant bootstrap design.
- Multi-module `go test ./...` coverage from the repository root may be insufficient by itself; Step 8 should also run `go test ./...` in each module directory.

## Open Questions

- No blocking open questions for Step 7.
- Future open question: choose the specific reviewed pairing protocol/handshake design for online pairing without plaintext join-code storage.
- Future open question: choose the reviewed Noise implementation or protocol package for the production encrypted UDP transport.
- Future open question: define multi-master revocation propagation, quorum, and master-removal safety.

## Hard Boundaries

- Do not implement Postgres, Redis, Valkey, Docker, service installation, log streaming, worker command execution, website/dashboard, MCP server, secret transmission, admin/root remote command execution, full production mesh transport, external system VPN integration, kernel-level VPN dependency, shelling out to networking/VPN tools, or multi-master consensus in this milestone.

## Next Recommended Action

Proceed to Step 7: implement thin control/CLI/TUI skeleton, then update `context.md` and commit the completed step.

## Resume Instructions

Future sessions must first read `context.md`, then inspect the repository for changes made after this file was last updated. Continue from the next pending step. Per the latest user instruction, do not pause for repeated approvals; update `context.md` after each todo step and create a brief commit after every completed todo step.
