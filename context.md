# Project Context

## Goal

Build a CLI-first infrastructure management app with a secure master/worker machine mesh. Current scope is identity, membership, revocation, persistence, authenticated connection primitives, and shared CLI/TUI control flow.

## Boundaries

Do not build service management, remote command execution, websites, MCP, application secret transmission beyond pairing, Docker/Postgres/Redis management, log streaming, external VPN integrations, kernel VPN dependencies, OS-managed VPN config, shell-out networking tools, or multi-master consensus unless explicitly requested.

## Current Status

- TLS/TCP mesh ping is implemented and wired into `infra mesh listen` / `infra mesh ping` for manual real-server testing.
- Online join-code pairing is implemented as `infra pair listen` / `infra pair join` over TLS/TCP with explicit master node ID pinning.
- QUIC/TLS mesh ping is implemented at package level in `packages/securemesh/network/quictransport` using `github.com/quic-go/quic-go v0.60.0`.
- CLI mesh commands still use TLS/TCP; QUIC is not wired through control actions or CLI yet.
- Runtime TLS certificates are derived from persistent node identity; no separate TLS private key is persisted.
- Mesh peer verification uses local peer state, network ID, role, public key matching, ALPN, and revocation state instead of public Web PKI.

## Key Files

- `docs/REAL_SERVER_TESTING.md`: Current two-server TLS/TCP CLI test procedure.
- `docs/PAIRING.md`: Current online pairing model and future PAKE/OPAQUE direction.
- `docs/REAL_SERVER_CONNECTION_PLAN.md`: Real-server transport roadmap.
- `docs/TRANSPORT_THREAT_MODEL.md`: Transport threat model and requirements.
- `packages/control/actions`: Shared action layer.
- `packages/securemesh/network/tlsidentity`: Runtime TLS identity and peer verification.
- `packages/securemesh/network/pairing`: Online join-code pairing transport.
- `packages/securemesh/network/tlstcp`: Current CLI-backed authenticated ping transport.
- `packages/securemesh/network/quictransport`: Package-level QUIC/TLS authenticated ping transport.

## Durable Decisions

- Preserve module path `github.com/f1forhelp/tailed-box-cli` and Go version `1.25.1` unless explicitly asked to change them.
- Keep business logic behind `packages/control/actions` and `packages/securemesh`; CLI/TUI entrypoints should stay thin.
- Use Go standard-library crypto where possible and reviewed libraries/protocols for transport dependencies.
- Do not implement custom cryptographic primitives or custom crypto math.
- Join codes are only for initial pairing, high entropy, single-use, and not persisted in plaintext.
- Online pairing currently uses TLS with explicit master node ID pinning; future pairing may use PAKE/OPAQUE to avoid sending the code itself.
- Revoked nodes must not reconnect with old credentials; default rejoin path requires fresh node identity and new join authorization.
- Manual `network import --id <network_id>` remains only a testing/manual setup helper; prefer `pair join` for new nodes.
- QUIC uses reliable streams first. QUIC datagrams, NAT traversal, reconnect supervisors, and service-management messages remain future work.
- `context.md` should stay a durable state snapshot only. Do not record routine command transcripts or repeated test-pass history here.

## Dependencies

- Root module and `packages/control` currently use local module replacements only.
- `packages/securemesh` depends on `github.com/quic-go/quic-go v0.60.0` for package-level QUIC/TLS transport.

## Known Gaps

- QUIC transport is not wired into `packages/control/actions` or `cmd/infra` yet.
- No daemon/supervisor, reconnect loop, NAT traversal, service-management protocol, remote command execution, or multi-master consensus exists.

## Verification Commands

```sh
go test ./...
(cd packages/securemesh && go test ./...)
(cd packages/control && go test ./...)
```

Run `go mod tidy` in any module whose imports/dependencies changed.

## Next Action

Next useful step is wiring QUIC through `packages/control/actions` and `cmd/infra` behind an explicit transport choice, or adding daemon/reconnect behavior. The current real-server CLI path remains TLS/TCP until QUIC wiring is added.

## Resume Instructions

Read `context.md`, inspect the worktree, then continue without repeated approval prompts when the next step is clear. Ask if a security/product decision is ambiguous.
