# Project Context

## Project Goal

Build a CLI-first infrastructure management application with a secure machine-to-machine mesh between master and worker nodes. Current work is limited to mesh identity, membership, revocation, persistence, authenticated connection primitives, and shared CLI/TUI control flow.

Do not build service management, remote command execution, websites, MCP, secret transmission, Docker/Postgres/Redis management, log streaming, external VPN integrations, kernel VPN dependencies, OS-managed VPN config, shell-out networking tools, or multi-master consensus unless explicitly requested.

## Current Milestone

Secure mesh foundation milestone with first real connection primitives.

## Current Status

- TLS/TCP mesh ping is implemented and wired into the CLI for manual real-server testing.
- Package-level QUIC/TLS mesh ping is implemented in `packages/securemesh/network/quictransport` using `github.com/quic-go/quic-go v0.60.0`.
- Current CLI mesh commands still use TLS/TCP, not QUIC.
- Manual public peer metadata exchange is still required; online join-code pairing is not implemented.
- Runtime TLS certificates are derived from persistent node identity; no separate TLS private key is persisted.
- Mesh TLS verification uses local peer state, network ID, role, public key matching, ALPN, and revocation state instead of public Web PKI.

## Completed Steps

- [x] Step 1: Repository inspection
- [x] Step 2: Architecture and security plan
- [x] Step 3: File plan
- [x] Step 4: Foundation types
- [x] Step 5: Crypto and persistence
- [x] Step 6: Join-code and revocation logic
- [x] Step 7: Thin control/CLI/TUI skeleton
- [x] Step 8: Tests
- [x] Step 9: Documentation
- [x] Milestone 10: Transport threat model
- [x] Milestone 11: TLS identity binding
- [x] TLS/TCP mesh ping package
- [x] TLS/TCP mesh CLI wiring
- [x] Milestone 12: Package-level QUIC/TLS mesh ping

## Important Files

- `README.md`: Project overview, current scope, and transport direction.
- `SECURITY.md`: Current security model and explicit non-goals.
- `docs/PAIRING.md`: Future online pairing design direction.
- `docs/REAL_SERVER_CONNECTION_PLAN.md`: Staged real-server transport plan.
- `docs/REAL_SERVER_TESTING.md`: Current two-server TLS/TCP CLI test procedure.
- `docs/TRANSPORT_THREAT_MODEL.md`: Transport threat model and QUIC dependency gate.
- `cmd/infra/main.go`: Thin CLI parser/dispatcher.
- `cmd/infra-tui/main.go`: Thin TUI skeleton.
- `packages/control/actions`: Shared action layer for CLI/TUI/future interfaces.
- `packages/securemesh/identity`: Network and node identity generation/persistence.
- `packages/securemesh/join`: Local verifier-backed single-use join-code state.
- `packages/securemesh/peer`: Local peer allowlist state.
- `packages/securemesh/revocation`: Local revocation state.
- `packages/securemesh/network/tlsidentity`: Runtime TLS certificate generation and mesh peer verification.
- `packages/securemesh/network/tlstcp`: Minimal TLS/TCP authenticated ping transport used by current CLI mesh commands.
- `packages/securemesh/network/quictransport`: Minimal QUIC/TLS authenticated ping transport over reliable streams.

## Current CLI Flow

Master:

```sh
infra --config-root ./state-master network init
infra --config-root ./state-master identity init --role master
infra --config-root ./state-master peer export > master.peer.json
```

Worker:

```sh
infra --config-root ./state-worker network import --id <network_id>
infra --config-root ./state-worker identity init --role worker
infra --config-root ./state-worker peer export > worker.peer.json
```

Exchange public peer files, then add the opposite peer on each side:

```sh
infra --config-root ./state-master peer add --file worker.peer.json
infra --config-root ./state-worker peer add --file master.peer.json
```

Start listener and ping:

```sh
infra --config-root ./state-master mesh listen --bind 0.0.0.0:9443
infra --config-root ./state-worker mesh ping --endpoint <master-host-or-ip>:9443
```

## Durable Decisions

- Preserve module path `github.com/f1forhelp/tailed-box-cli`.
- Preserve Go version `1.25.1` unless explicitly asked to change toolchain/versioning.
- Keep business logic behind `packages/control/actions` and `packages/securemesh`; CLI/TUI entrypoints should stay thin.
- Use Go standard-library cryptography where possible and reviewed libraries/protocols for transport dependencies.
- Do not implement custom cryptographic primitives or custom crypto math.
- Join codes are only for initial pairing, high entropy, single-use, and not persisted in plaintext.
- Online pairing should use a reviewed PAKE/OPAQUE-style approach or another reviewed MITM-resistant bootstrap design that preserves no-plaintext join-code persistence.
- Revoked nodes must not reconnect with old credentials; default rejoin path requires fresh node identity and a new join authorization.
- Current manual `network import --id <network_id>` is only a testing helper, not a secure online pairing protocol.
- QUIC transport uses reliable streams first. QUIC datagrams, NAT traversal, reconnect supervisors, and service-management messages remain future work.
- `context.md` should be a durable state snapshot only. Do not record routine command transcripts or repeated test-pass history here.

## Dependencies

- Root module and `packages/control` currently use local module replacements only.
- `packages/securemesh` directly depends on `github.com/quic-go/quic-go v0.60.0` for package-level QUIC/TLS transport.
- `packages/securemesh/go.sum` is expected because of the QUIC dependency graph.

## Known Gaps

- QUIC transport is not wired into `packages/control/actions` or `cmd/infra` yet.
- Online join-code pairing is not implemented.
- No daemon/supervisor or reconnect loop exists.
- No NAT traversal exists.
- No service-management protocol/messages exist.
- No remote command execution exists.
- No multi-master consensus, revocation quorum, or propagation exists.

## Verification Commands

Run these before committing code changes:

```sh
go test ./...
(cd packages/securemesh && go test ./...)
(cd packages/control && go test ./...)
```

Run `go mod tidy` in any module whose imports/dependencies changed.

## Next Recommended Action

Wire the QUIC transport through `packages/control/actions` and `cmd/infra` behind an explicit transport choice, or implement online pairing next. The current real-server CLI test path remains TLS/TCP until that wiring is added.

## Resume Instructions

Future sessions must first read `context.md`, then inspect the repository for changes made after this file was last updated. Continue without repeated approval prompts when the next step is clear, but ask if a security/product decision is ambiguous.
