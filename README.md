# tailed-box-cli

CLI-first infrastructure management foundation for a future master/worker platform. The current milestone implements only the local secure mesh foundation; it does not manage services or execute remote commands.

## Current Milestone

Implemented local foundations for:

- Persistent machine identity with master/worker roles.
- Network identity.
- Join-code metadata and single-use local state.
- Peer metadata and active/revoked local model.
- Revocation metadata and local revoked-node checks.
- Restrictive local persistence layout.
- Future transport/session interfaces.
- Shared control/action layer used by CLI and TUI.
- Thin `infra` CLI skeleton.
- Thin `infra-tui` text-menu skeleton.
- Root `context.md` continuity tracking.

## Not Implemented Yet

- Postgres, Redis, Valkey, Docker, service installation, deployments, logs, monitoring, backups, or secrets management.
- Website/dashboard or MCP server.
- Worker command execution or remote admin/root command execution.
- Full production encrypted mesh transport.
- NAT traversal.
- Multi-master consensus or revocation quorum.
- External VPN, kernel VPN, OS-managed VPN, or shell-out networking integrations.

## Commands

Run tests from all modules:

```sh
go test ./...
(cd packages/securemesh && go test ./...)
(cd packages/control && go test ./...)
```

Example local CLI flow:

```sh
infra --config-root ./local-state network init
infra --config-root ./local-state identity init --role master
infra --config-root ./local-state join-code create --role worker
infra --config-root ./local-state peer list
```

The TUI is intentionally small:

```sh
infra-tui --config-root ./local-state
```

Each TUI action displays its equivalent CLI command.

## Architecture

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

Business logic belongs in `packages/control/actions` and `packages/securemesh`, not in CLI/TUI entrypoints.

## Secure Transport Direction

This repository does not depend on external system VPN tooling. It does not shell out to VPN/networking commands and does not require kernel VPN features.

See `docs/PAIRING.md` for the design-only next-milestone direction for a future online pairing handshake.

See `docs/REAL_SERVER_CONNECTION_PLAN.md` for the staged plan to reach real server-to-server secure connections.

See `docs/TRANSPORT_THREAT_MODEL.md` for the first real transport threat model and requirements.

See `docs/REAL_SERVER_TESTING.md` for the current two-server TLS/TCP mesh ping test procedure.

The planned direction is a low-overhead encrypted mesh owned by this application:

- Future UDP data plane.
- Reviewed Noise-style authenticated handshake.
- Persistent node identity plus ephemeral session keys.
- Authenticated encryption for payloads.
- Replay protection and key separation.
- Session key rotation.
- Peer allowlists and revocation checks before accepting sessions.
- Optional QUIC/TLS reliable control streams later only if needed.

This milestone intentionally implements only local identity, membership, pairing, revocation, persistence, and transport interfaces.

## Join Codes

Join codes are only for initial pairing. They are generated with cryptographically secure randomness, are high entropy, and are single-use in the local model.

Persisted join-code records store verifier material and metadata only. Plaintext join codes are not persisted. There is no mandatory expiry in this milestone; the compensating controls are high entropy, verifier-only persistence, explicit consumed state, and single-use enforcement.

## Revocation

Revocation is local in this milestone. A master can create a revocation record for a worker or another master. Revoked peers are not considered active locally. Rejoining after revocation requires a new join code and fresh node identity material by default.

Multi-master propagation, quorum, split-brain handling, and master-removal safety are future work.

## Project Continuity

`context.md` is maintained after each logical step. It must not contain secrets, private keys, join codes, tokens, passwords, credentials, or sensitive derived material.
