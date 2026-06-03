# Tailedbox Project Context

This document captures the current project context, decisions, implemented work,
known limitations, and recommended next steps for Tailedbox.

## Product Intent

Tailedbox is a lightweight, Go-based, CLI-first control system for securely
connecting, provisioning, and managing services on Linux VPSs.

The long-term product is a Dokploy-like infrastructure control plane, but with a
small-VPS-friendly footprint and a secure master/worker communication layer
built into the `tailedbox` binary.

The same binary is installed everywhere:

- master/control nodes
- worker nodes
- future additional HA master nodes

There are no separate master and worker binaries. Role-specific behavior is
selected through local initialization and configuration.

## Current POC Focus

The first real POC milestone is the secure Tailedbox mesh/VPN communication
layer. The project is being built in smaller reviewable parts:

1. Bootstrap the CLI and project structure.
2. Create durable local node identity and role initialization.
3. Add a join-code enrollment foundation.
4. Add local agent/systemd lifecycle.
5. Design the mesh protocol.
6. Implement the mesh MVP.

PostgreSQL is intentionally not implemented yet. It remains the first planned
managed service after the mesh/enrollment foundation is ready.

## Important Naming

The binary and command name is:

```bash
tailedbox
```

Future service commands should remain service-oriented:

```bash
tailedbox pg init
tailedbox pg deploy
tailedbox pg status
tailedbox pg failover
tailedbox pg backup
tailedbox pg restore
```

## Current Implemented Parts

### Part 1: Project Bootstrap and CLI Skeleton

Implemented:

- Go module setup.
- Single binary entrypoint at `cmd/tailedbox`.
- Go toolchain pinned to:

```txt
go 1.26
toolchain go1.26.3
```

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
  - `network`
  - `node`
  - `pg`
- Structured JSONL logging.
- Opt-in debug logging:

```bash
tailedbox debug logs enable
tailedbox debug logs disable
```

- Log redaction for:
  - tokens
  - secrets
  - passwords
  - private keys
  - join codes
  - credentials
  - decrypted payload markers
- Human-readable and JSON-capable command output.

### CLI Presentation

Implemented:

- Lip Gloss-based terminal styling.
- Grouped command help.
- Polished status/key-value output.
- ASCII cluster tables for master status.
- Bubble Tea-based interactive no-args menu.
- Boxed full-screen terminal menu for interactive sessions, with a bordered
  action list and selected-action details panel.
- The interactive menu is a launcher over the normal CLI command graph. Every
  selectable UI action displays and runs an equivalent `tailedbox ...` command.
- UI-only workflows are intentionally avoided so scripts, automation, future
  web workflows, and terminal users all share the same backend behavior.
- Interactive UI code is split by responsibility:
  - `internal/ui` owns the interactive no-args menu.
  - Bubble Tea model/update code owns keyboard input, cursor state, selected
    command args, and quit behavior.
  - Lip Gloss rendering/layout code lives behind a dedicated menu renderer.
  - `internal/cli` launches the menu and dispatches the selected CLI args.
- Project context updates are part of the normal implementation workflow:
  whenever behavior, architecture, decisions, commands, limitations, or roadmap
  items change, `CONTEXT.md` should be updated automatically before the work is
  handed back.

Current no-args behavior:

- If stdin/stdout are real terminals, `tailedbox` opens an interactive menu.
- If running in scripts, pipes, tests, or non-TTY contexts, `tailedbox` prints
  normal help and exits.
- The interactive menu uses Bubble Tea's alternate screen so the UI feels like a
  terminal app instead of printing below the shell prompt.

Interactive menu options currently include:

- Status
- Agent status
- Initialize as master
- Initialize as worker
- Master status
- Worker status
- Create worker join code
- Create master join code
- Recent logs
- Version
- Help
- Exit

### Part 3: Master/Worker Role Initialization

Implemented:

- `tailedbox init --role master`
- `tailedbox init --role worker`
- `tailedbox master init`
- `tailedbox worker init`
- Durable local node ID.
- Durable Ed25519 node identity.
- Public identity metadata.
- Private identity key generation.
- Local node metadata.
- Agent config scaffold.
- Role-specific state directory.
- Idempotent init for the same role.
- Refusal to change role after initialization.
- Status output includes identity readiness and agent config readiness.

Local state layout:

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

Permissions:

- state and secret directories: `0700`
- config, metadata, identity, audit, and secret files: `0600`

Important security decision:

- The Ed25519 private identity key is generated locally.
- The private key is not logged.
- Master nodes store only joining nodes' public identity information, not their
  private key material.

### Part 4: Join-Code Enrollment Foundation

Implemented:

- Master-only join-code creation:

```bash
tailedbox master join-code create --role worker --ttl 15m
tailedbox master join-code create --role master --ttl 15m
```

- Worker/master join commands:

```bash
tailedbox worker join --code <join-code> --master-state-dir <path>
tailedbox master join --code <join-code> --master-state-dir <path>
```

- One-time join-code lifecycle.
- Role-scoped join codes.
- TTL expiry.
- Used-code state.
- Hashed join-code secret storage.
- Raw join code is printed once and is not persisted.
- Trusted-node records on the issuing master.
- Joined-cluster metadata on the joining node.
- Master-controlled reconnect lease metadata.
- Audit JSONL events for:
  - join-code creation
  - join attempts
  - join success
  - join failure
- Master status includes trusted nodes.
- Worker status shows joined-cluster state.

Current enrollment limitation:

- The join flow is a local-state POC.
- `--master-state-dir` is a temporary transport stand-in.
- Real network enrollment must be implemented after the daemon and mesh/control
  transport exist.

This is intentional. It avoids pretending secure network transport exists before
the mesh parts are built.

### Part 5: Local Agent Daemon and Systemd Integration

Implemented:

- `agent` command namespace:

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

- `tailedbox agent run` runs the lightweight agent in the foreground.
- The foreground agent writes periodic local health heartbeats to:

```txt
<state-dir>/agent/status.json
```

- Agent status includes:
  - node ID
  - role
  - state
  - health
  - PID
  - started time
  - last heartbeat time
  - uptime
  - heartbeat age
  - Go runtime allocated memory
  - Go runtime system memory
  - goroutine count
  - log/config/status file paths
  - systemd service name and unit path
- Stale heartbeats are marked degraded.
- `tailedbox agent logs` reuses the existing redacted JSONL log reader.
- `tailedbox agent install --dry-run` renders a systemd unit without writing
  files or invoking `systemctl`.
- Agent install/preview requires the node to be initialized first.
- Real systemd install/control is Linux-only and is refused on non-Linux
  development machines.
- The generated systemd unit uses:
  - `Restart=always`
  - `RestartSec=5s`
  - `After=network-online.target`
  - `Wants=network-online.target`
  - `NoNewPrivileges=true`
  - `PrivateTmp=true`
  - `ProtectSystem=full`
  - explicit writable paths for config/state/log directories

Current agent limitations:

- The agent does not yet start the designed mesh service or open mesh sockets.
- The agent does not yet expose a local API.
- Systemd install requires normal OS permissions, usually root.
- Service control commands call `systemctl` and therefore work only on Linux
  systems with systemd.

### Part 6: Tailedbox Mesh Protocol Design

Implemented:

- Protocol design document at `docs/mesh-protocol-design.md`.
- Mesh threat model covering passive observers, active network attackers,
  untrusted nodes, expired reconnect leases, and accidental public exposure.
- Trust model based on existing Ed25519 node identity, master trusted-node
  records, worker joined-cluster pinning, and reconnect lease metadata.
- Handshake design using Ed25519 identity signatures, ephemeral X25519 key
  exchange, HKDF-SHA256 key derivation, and AES-256-GCM payload encryption.
- Versioned UDP packet envelope for handshake, encrypted data, rekey, and close
  packets.
- Network enrollment design that replaces the temporary local
  `--master-state-dir` transport with a future `--master-endpoint` flow while
  keeping raw join-code secrets encrypted on the wire and never persisted.
- Peer discovery design based on explicit master endpoints, authenticated
  learned peer endpoints, and private mesh runtime state.
- Direct UDP MVP model with future relay fallback constraints.
- Firewall posture for mesh/control traffic: masters expose only the configured
  mesh UDP port, workers can remain outbound-only for the MVP, and local agent
  control remains local-only.
- Part 7 implementation boundaries for `internal/mesh`, agent integration,
  `tailedbox mesh status`, `peers`, `ping`, and `diagnose`.

### Part 7: Tailedbox Mesh MVP Implementation Foundation

Implemented:

- `internal/mesh/protocol` with the versioned `TBXM` UDP packet envelope,
  packet types for hello/auth/data/rekey/close, strict decode validation, and
  JSON control-message types for ping, pong, peer updates, status, diagnostics,
  and network enrollment messages.
- `internal/mesh/store` with private mesh runtime paths under
  `<state-dir>/mesh`, mesh status JSON, peer observation JSON files, sorted peer
  listing, and path-traversal rejection for peer node IDs.
- `internal/mesh/crypto` with Ed25519 transcript signing/verification, X25519
  ephemeral key generation, HKDF-SHA256 session key derivation, AES-256-GCM
  construction, and 96-bit nonce construction from a direction-specific prefix
  plus packet sequence.
- `internal/mesh/session` with replay-window tracking, directional packet
  sender/receiver helpers, AEAD packet seal/open, sequence allocation, and
  associated-data binding to the `TBXM` envelope header.
- Focused tests for packet encode/decode, malformed envelope rejection, control
  message shape, strict mesh store permissions, peer listing, transcript
  tamper detection, matching X25519/HKDF derivation on both sides, and AEAD
  associated-data enforcement, replay rejection, duplicate packet rejection,
  stale packet rejection, and packet-header tamper rejection.
- `internal/mesh/control` with a local JSON request/response control socket for
  mesh status, peer listing, ping dispatch, and diagnostics. The default path is
  `<state-dir>/agent/control.sock`; long Unix socket paths fall back to a
  deterministic private temp path and diagnostics report the actual socket path.
- `internal/mesh/service` with an agent-owned mesh service scaffold that writes
  mesh runtime status, serves the local control socket, reports disabled mesh as
  healthy, and reports enabled mesh as degraded until UDP transport is wired.
- `tailedbox agent run` starts the mesh service scaffold alongside heartbeat
  status, refreshes `<state-dir>/mesh/status.json`, and preserves existing mesh
  agent config when ensuring `agent/config.json`.
- `tailedbox mesh status`, `tailedbox mesh peers`, and
  `tailedbox mesh diagnose` are active commands backed by the running agent when
  reachable, with private state-file fallback when the agent is offline.
- `tailedbox mesh ping <node-id>` is active as a local-agent dispatch command;
  it requires a running agent and reports that encrypted UDP ping/pong is still
  the next transport slice.
- `tailedbox mesh enable [--listen-udp-port <port>]` and
  `tailedbox mesh disable` persist the mesh agent config without changing node
  identity or enrollment state. Masters default to UDP port `41677`; workers
  default to an ephemeral port until configured otherwise.

Current Part 7 limitations:

- The mesh service is a local control/status scaffold only; it does not open UDP
  mesh sockets yet.
- No UDP listener, enrolled handshake wiring, rekey loop, encrypted live packet
  exchange, or authenticated mesh ping/pong is wired yet.
- Network enrollment still uses local `--master-state-dir`; the designed
  `--master-endpoint` flow is not implemented yet.

## Current Commands

Core:

```bash
tailedbox
tailedbox version
tailedbox status
tailedbox status --json
tailedbox init --role master
tailedbox init --role worker
```

Master:

```bash
tailedbox master init
tailedbox master status
tailedbox master status --json
tailedbox master join-code create --role worker --ttl 15m
tailedbox master join-code create --role master --ttl 15m
tailedbox master join --code <join-code> --master-state-dir <path>
```

Worker:

```bash
tailedbox worker init
tailedbox worker status
tailedbox worker status --json
tailedbox worker join --code <join-code> --master-state-dir <path>
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

Mesh:

```bash
tailedbox mesh enable
tailedbox mesh enable --listen-udp-port 41677
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

Reserved future namespaces:

```bash
tailedbox network create --driver tailedbox-mesh
tailedbox network status
tailedbox network peers

tailedbox node list
tailedbox node approve <node-id>

tailedbox pg init
tailedbox pg deploy
tailedbox pg status
tailedbox pg failover
tailedbox pg backup
tailedbox pg restore
```

## Architectural Decisions

### Single Binary

Decision:

- Keep one `tailedbox` binary for all roles.

Rationale:

- Simpler install story.
- Easier small-VPS operations.
- Role-specific behavior can be selected through config and local state.

### CLI-First

Decision:

- Initial POC is CLI-only.
- No web UI yet.

Rationale:

- Faster path to secure infrastructure primitives.
- Avoids building a UI before the backend workflows are real.

### Interactive Menu

Decision:

- `tailedbox` with no arguments opens a terminal menu only when stdin/stdout are
  real terminals.
- Non-interactive execution prints normal help.

Rationale:

- Better beginner UX.
- Keeps scripts safe and non-blocking.
- Keeps the CLI as the source of truth: UI actions must be command-backed, not
  separate workflows.

### Identity

Decision:

- Use Ed25519 node identity keys for the local identity foundation.

Rationale:

- Ed25519 is in the Go standard library.
- It is compact, fast, and appropriate for small VPSs.
- Future enrollment and mesh handshakes can build on the public identity.

### Join Codes

Decision:

- Join codes are one-time, short-lived, and role-scoped.
- Raw join codes are never persisted.
- Master state stores only a hash and minimal metadata.

Rationale:

- Reduces credential leakage risk.
- Models the intended secure enrollment behavior before network transport is
  implemented.

### Local Enrollment Transport for Part 4

Decision:

- Use `--master-state-dir` as a temporary local transport stand-in.

Rationale:

- Daemon, mesh, and encrypted transport are not implemented yet.
- This keeps the enrollment state machine testable without faking network
  security.

### Agent Lifecycle

Decision:

- Use the same `tailedbox` binary for interactive CLI usage and long-running
  agent mode.
- Systemd runs `tailedbox agent run`; there is no separate worker daemon binary.
- Local health is persisted to `<state-dir>/agent/status.json`.

Rationale:

- Keeps installation simple.
- Makes master/worker behavior role-specific through state and config.
- Gives future mesh/session code a long-running process without introducing a
  sidecar.

### Logging

Decision:

- Use structured JSONL logs from the start.
- Redact sensitive-looking values at write/display boundaries.

Rationale:

- Easier diagnostics.
- Safer debug mode.
- Later agent and mesh troubleshooting will need this.

### Dependencies

Current direct UI dependencies:

- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/bubbletea`

Rationale:

- Lip Gloss handles polished terminal styling.
- Bubble Tea handles interactive menu input and terminal redraw lifecycle.
- Both are idiomatic Go terminal UI packages.
- The menu renderer boundary keeps layout changes separate from command
  selection behavior and makes the UI layer easier to test.
- The current terminal UI package lives under `internal/ui`, keeping UI behavior
  out of command dispatch code while staying simple at this project size.

## Not Implemented Yet

### Part 2: Versioned GitHub Release Installer

Not done:

- `install.sh`
- exact version installation
- OS/architecture detection
- checksum verification
- GitHub Release artifact layout
- optional signature verification
- self-update design

### Part 7: Tailedbox Mesh MVP Implementation

Not done:

- UDP peer transport and encrypted live packet exchange
- encrypted mesh ping/pong over UDP
- enrolled handshake wiring
- session key rotation
- reconnect lease enforcement over network
- network enrollment over `--master-endpoint`

### Security Hardening

Not done:

- firewall provider abstraction
- default deny public DB/control-plane ports
- secure remove-node flow
- key/certificate rotation
- lost/expired worker reconnect enforcement over real transport

### Master HA

Not done:

- embedded consensus
- Raft store
- leader election
- replicated cluster state
- split-brain refusal

### PostgreSQL Module

Not done:

- PostgreSQL deployment
- Docker/native/NixOS runtime support
- replication
- backup/restore
- HA/failover
- quorum safety

### Future Web UI

Not done:

- HTTPS browser UI
- auth model
- dashboard workflows
- user/team/project model

## Known Current Limitations

- No mesh session daemon is running yet.
- No real network communication exists yet.
- Mesh protocol, store, crypto, session helpers, local agent control, and mesh
  CLI surfaces exist but are not yet connected to UDP transport or encrypted
  live sessions.
- `tailedbox mesh ping` dispatches through the local agent control socket but
  cannot exchange network ping/pong until UDP transport lands.
- Join-code enrollment is local-state backed.
- `connected_to_master_cluster`, `authenticated`, and `mesh_reachable` are still
  false because mesh sessions do not exist.
- Master status knows trusted nodes from local files only.
- Worker status intentionally does not expose full cluster inventory.
- No firewall changes are made yet.
- No release installer exists yet.
- No PostgreSQL service implementation exists yet.

## Test and Build Commands

Use:

```bash
go test ./...
go build ./cmd/tailedbox
```

Useful smoke flow:

```bash
tailedbox init --role master
tailedbox master join-code create --role worker --ttl 15m

tailedbox init --role worker
tailedbox worker join --code <join-code> --master-state-dir <master-state-dir>
tailedbox worker status
tailedbox master status
```

## Work Completed So Far

- feat: bootstrap tailedbox CLI skeleton
- chore: add VS Code launch configs
- chore: add no-args CLI launch target
- style: polish CLI presentation
- feat: add durable node role initialization
- feat: add join-code enrollment foundation
- feat: add systemd service management for Tailedbox agent
- feat: refactor interactive UI into internal/ui package
- docs: add mesh protocol design
- feat: add mesh protocol, store, and crypto foundation
- feat: add mesh agent control and CLI status surfaces
- feat: add mesh config toggles and session packet helpers

## Commit Policy

The user explicitly requested:

- do not commit automatically
- ask before committing

Always show the result and ask before running `git commit`, unless the user
explicitly asks to commit.

## Recommended Next Steps

Recommended next implementation order:

1. Part 7: Mesh MVP Implementation.
2. Part 2: Versioned GitHub Release Installer.

Why Part 7 next:

- Enrollment now exists as local state.
- A long-running local agent now exists.
- Part 6 now provides a written protocol, threat model, handshake design,
  packet flow, peer discovery model, and firewall posture.
- The initial Part 7 protocol, store, and crypto primitives now exist.
- The initial Part 7 agent control socket and mesh CLI status surfaces now
  exist.
- Mesh config toggles and session packet helpers now exist.
- The next useful milestone is UDP transport plus enrolled handshake wiring so
  `tailedbox mesh ping` can exchange authenticated ping/pong packets.

Why not PostgreSQL next:

- PostgreSQL should depend on secure node identity, enrollment, agent lifecycle,
  mesh networking, runtime abstraction, and eventually HA coordination.

## Design Guardrails

- Keep the product lightweight.
- Keep the CLI and agent memory-efficient.
- Prefer secure defaults.
- Do not claim the system is impossible to hack.
- Do not expose PostgreSQL or internal control APIs publicly.
- Do not require Kubernetes, etcd, Consul, or an external VPN for the MVP.
- Do not hand-roll cryptographic primitives.
- Use proven Go crypto/networking libraries where appropriate.
- Keep interfaces clean for future providers:
  - runtime providers
  - network providers
  - mesh providers
  - firewall providers
  - service managers
  - HA coordinators
  - stores
  - audit loggers
