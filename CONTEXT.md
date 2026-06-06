# Tailedbox Context

This is the base application context. The secure connection module has its own
context at `packages/link/CONTEXT.md`.

## Product Intent

Tailedbox is a lightweight, Go-based, CLI-first control system for securely
connecting, provisioning, and managing services on Linux VPSs.

The same `tailedbox` binary is installed everywhere:

- master/control nodes
- worker nodes
- future HA master nodes

Role-specific behavior is selected through local initialization and
configuration. There are no separate master and worker binaries.

## Current Focus

The first POC milestone is the secure master/worker communication foundation.
PostgreSQL, a web UI, and heavier infrastructure should wait until that
foundation is reliable.

## Repository Map

- Root module: Tailedbox CLI application.
- `cmd/tailedbox`: binary entrypoint.
- `internal/`: app-only packages.
- `packages/link/`: secure connection workspace module.

## Architecture

- Single binary entrypoint: `cmd/tailedbox`.
- Command dispatch lives under `internal/cli`.
- Interactive terminal UI lives under `internal/ui`.
- The interactive terminal UI is a launcher over the same command dispatcher:
  every UI action maps to normal CLI args, and every runnable CLI leaf command
  should have a reachable UI action.
- The UI should also be usable by non-expert operators: when commands need
  runtime values, the TUI may show guided forms, but those forms must still
  build normal CLI-equivalent arguments.
- The TUI should feel like a complete terminal control app, with a
  sections-first navigation flow: the primary screen lists sections, Enter
  opens that section's actions in the same panel, and selected actions show
  compact details, command previews, or guided forms.
- The TUI should stay compact: minimal padding, short helper text, tight panel
  gaps, and space-efficient section/action lists. Section and action rows use
  simple ordinal prefixes instead of count badges.
- TUI color should stay minimal and semantic: use it to distinguish focus,
  hints, commands, success, warnings, and errors, not as decoration.
- The no-args TUI runs as a persistent app loop. Quick/form commands execute
  through the normal CLI dispatcher, show an in-app result screen, and return to
  the menu. Streaming commands run in the foreground with a stop path and then
  return to the TUI.
- Guided forms include a visible `No / cancel` row in addition to Esc so
  non-expert users can explicitly back out without running the command.
- Destructive guided forms should default focus to `No / cancel`; the user must
  deliberately move to the confirmation input before running the command.
- Esc is treated as back-one-level in the TUI. On the primary menu, Esc or q
  opens a quit confirmation instead of exiting immediately. Result screens use
  Esc/b to return to the originating opened section/action list so Enter stays
  reserved for activate/submit behavior.
- This command/UI parity is intentional groundwork for future MCP control:
  machine clients should be able to discover and invoke the same command
  surfaces without relying on TUI-only behavior.
- Local role is selected by initialization and persisted in local config/state.
- Node identity uses Ed25519 keys generated locally.
- The foreground agent is `tailedbox agent run`.
- Local health is persisted to `<state-dir>/agent/status.json`.
- Provisioning-oriented workers may run the Tailedbox agent with root
  privileges for the MVP so the master can request package/service operations
  such as installing PostgreSQL, Redis, or MySQL. This root-capable path must
  stay structured and allowlisted: the master sends typed provisioning tasks,
  not arbitrary remote shell commands.
- Secure connection behavior and roadmap are tracked in
  `packages/link/CONTEXT.md`.

## Implemented

- CLI namespaces:
  - `version`
  - `status`
  - `init`
  - `master`
  - `worker`
  - `agent`
  - `logs`
  - `debug`
  - `mesh`
  - `uninstall`
  - `network`
  - `node`
  - `pg`
- Structured JSONL logging with redaction.
- Human-readable and JSON-capable command output.
- Bubble Tea interactive no-args menu for real terminals.
- Interactive menu coverage for each runnable CLI leaf command, including
  planned command surfaces.
- Terminal app shell with hierarchical sections-first navigation: the primary
  screen lists sections, Enter opens active-section actions in the same panel,
  Esc backs to sections, and selected actions show compact details plus command
  previews.
- Compact, width-safe TUI renderer with one-line header, tight panel padding,
  short helper text, and no side-by-side menu columns.
- Minimal semantic TUI color roles for hints, command blocks, completed,
  stopped, and failed result states.
- Persistent no-args TUI loop with in-app result screens for completed commands.
- Result screens remember their originating action and return to the same
  opened section with that action still selected.
- Quit confirmation dialog from the primary TUI screen.
- Streaming action handling for foreground commands such as `logs --follow` and
  `agent run`, with Ctrl+C cancelling only the active stream command and
  returning to the TUI without cancelling the parent UI loop.
- Guarded top-level uninstall command with dry-run, exact confirmation for
  local file removal, forced local identity removal, optional systemd service
  removal, and optional install artifact cleanup for Debian packages and known
  `tailedbox` command paths. After confirmed local uninstall, the system must
  be initialized again before Tailedbox can use it.
- Guided UI forms for runtime values such as join codes, master state
  directories, master endpoints, peer node IDs, and planned node approvals.
- Visible no/cancel path on guided forms, including destructive uninstall
  confirmation forms. Destructive uninstall forms default focus to no/cancel.
- Tests that enforce UI-to-CLI command validity and CLI-leaf-to-UI coverage.
- Plain help fallback for non-interactive execution.
- Durable master/worker role initialization.
- Durable local node ID.
- Durable Ed25519 node identity.
- Agent config scaffold.
- Local node metadata.
- Master-only join-code creation.
- One-time, role-scoped, short-lived join codes.
- Hashed join-code secret storage.
- Trusted-node records on the issuing master.
- Joined-cluster metadata on the joining node.
- Audit JSONL events for enrollment.
- Foreground agent loop.
- Agent heartbeat status with memory diagnostics.
- Logs aliases.
- Linux systemd unit generation and control commands.
- Mesh CLI surfaces that integrate with the `link` module through the root app
  adapter.
- Root `install.sh` release installer for supported GitHub Release assets. The
  first installer target is Debian `amd64`; unsupported operating systems or
  architectures fail with a clear message instead of downloading an unrelated
  build.

## Commands

Core:

```bash
tailedbox
tailedbox version
tailedbox status
tailedbox status --json
```

Initialization and enrollment:

```bash
tailedbox init --role master
tailedbox init --role worker
tailedbox master status
tailedbox master status --json
tailedbox worker status
tailedbox worker status --json
tailedbox master join-code create --role worker --ttl 15m
tailedbox master join-code create --role master --ttl 15m
tailedbox worker join --code <join-code> --master-state-dir <path>
tailedbox master join --code <join-code> --master-state-dir <path>
```

Agent:

```bash
tailedbox agent run
tailedbox agent status
tailedbox agent status --json
tailedbox agent install --dry-run
tailedbox agent install --binary /usr/local/bin/tailedbox --start
tailedbox agent uninstall
tailedbox agent start
tailedbox agent stop
tailedbox agent restart
tailedbox agent logs
```

Mesh app surfaces:

```bash
tailedbox mesh enable
tailedbox mesh enable --listen-udp-port 41677
tailedbox mesh enable --master-endpoint <host:port>
tailedbox mesh disable
tailedbox mesh status
tailedbox mesh status --json
tailedbox mesh peers
tailedbox mesh peers --json
tailedbox mesh ping <node-id>
tailedbox mesh diagnose
tailedbox mesh diagnose --json
```

Logs and debug:

```bash
tailedbox logs
tailedbox logs --follow
tailedbox logs --lines 50
tailedbox debug logs enable
tailedbox debug logs disable
```

Uninstall:

```bash
tailedbox uninstall --dry-run
tailedbox uninstall --dry-run --all
tailedbox uninstall --confirm-delete DELETE
tailedbox uninstall --confirm-delete DELETE --systemd
tailedbox uninstall --confirm-delete DELETE --install-artifacts
tailedbox uninstall --confirm-delete DELETE --all
```

Build and test:

```bash
go version
go test ./...
go test ./packages/link/...
go build ./cmd/tailedbox
```

Install:

```bash
./install.sh
TAILEDBOX_VERSION=v0.1.0 ./install.sh
curl -sSL https://raw.githubusercontent.com/f1forhelp/tailed-box-cli/main/install.sh | sh
```

## Local State

After initialization, local state includes:

```txt
<state-dir>/
  agent/
    config.json
    status.json
  audit/events.jsonl
  enrollment/
    join-codes/
    trusted-nodes/
  master/ or worker/
  node.json
  node_identity_public.json
  secrets/node_identity_ed25519.pem
```

`tailedbox uninstall --dry-run` previews the local config, state, log, socket,
identity, trust, enrollment, mesh, and agent files that would be removed.
`tailedbox uninstall --confirm-delete DELETE` removes those local files,
including the Ed25519 node identity and public identity metadata. After that,
the system is no longer a Tailedbox node until `tailedbox init` is run again.
Add `--systemd` when the installed systemd service should also be disabled and
removed. Add `--install-artifacts` to purge the Debian package and remove known
terminal command paths such as `/usr/bin/tailedbox` and
`/usr/local/bin/tailedbox`. Use `--all` as the full cleanup shortcut for local
files, systemd, the Debian package, and known command paths.

Permissions:

- state and secret directories: `0700`
- config, metadata, identity, audit, and secret files: `0600`

## Security Decisions

- Keep one `tailedbox` binary for all roles.
- Keep CLI workflows scriptable and JSON output stable.
- Generate Ed25519 private identity keys locally.
- Never log private keys, raw join codes, tokens, secrets, or decrypted
  payloads.
- Never persist raw join codes.
- Store only join-code hashes and minimal metadata.
- Master nodes store only joined nodes' public identity information.
- For the provisioning MVP, allow worker agents to run with root-level
  capability when needed for package installs, systemd service management,
  firewall changes, and service configuration.
- Remote provisioning must use signed, typed, allowlisted operations such as
  `install postgres`, `install redis`, `install mysql`, `restart service`, or
  `open firewall port`.
- Do not make arbitrary remote shell execution the default control model.
- Keep room for a future split between a restricted non-root network agent and
  a narrow root helper, but do not block the MVP on that split.

## Current Limitations

- Join-code enrollment is local-state backed.
- `--master-state-dir` is temporary until network enrollment is implemented in
  the secure connection module.
- Master status knows trusted nodes from local files only.
- Worker status intentionally does not expose full cluster inventory.
- Systemd install usually requires root.
- Service control commands call `systemctl` and work only on Linux systems with
  systemd.
- Full install artifact removal requires Linux with Debian package tools when
  the Debian package is installed.
- Network and node management namespaces are reserved but not implemented.
- Remote master-to-worker provisioning is a planned direction, not implemented
  yet. Workers will need root-level capability for host package/service
  operations when that lands.
- The release installer currently supports Debian `amd64` only.
- PostgreSQL, web UI, master HA, firewall provider abstraction, and heavyweight
  infrastructure are not implemented.

## Roadmap

1. Continue secure connection work tracked in `packages/link/CONTEXT.md`.
2. Expand the GitHub release installer to additional operating systems and
   architectures as release assets are published.
3. Add future managed services only after the secure connection foundation is
   reliable.

## Release Installer Scope

Implemented:

- `install.sh`
- exact version installation through `TAILEDBOX_VERSION`
- OS/architecture detection
- OS/architecture detection runs before release download attempts, and
  unsupported platforms fail with a clear message.
- default latest-release install when no version is supplied
- interactive release selection from the latest 10 GitHub Releases when a
  terminal is available
- custom version entry from the installer prompt
- checksum verification with GitHub Release `checksums.txt`
- Debian `amd64` package installation from GitHub Release assets
- apt-readable temporary package files to avoid noisy `_apt` sandbox warnings
  during local `.deb` installs

Not implemented yet:

- additional OS/architecture release assets
- optional signature verification
- self-update design

## Future Services Scope

Do not start these before secure connection is reliable:

- PostgreSQL deployment, replication, backup/restore, and failover.
- HTTPS web UI.
- Master HA and replicated cluster state.
- Firewall provider abstraction.
- Secure remove-node flow and key/certificate rotation.
- Kubernetes, external etcd, Consul, and external VPN dependencies are explicit
  non-goals for the MVP.

When service provisioning begins, prefer typed commands over arbitrary shell:

```bash
tailedbox node provision <node-id> postgres
tailedbox node provision <node-id> redis
tailedbox node provision <node-id> mysql
tailedbox node service <node-id> restart postgres
```

The worker side should validate the master identity, task type, requested
service, and arguments before running privileged local steps. Provisioning
results should stream or persist logs/status back to the master and audit log.
