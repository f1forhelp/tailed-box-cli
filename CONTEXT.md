# Tailedbox Context

This is the base application context. The secure connection module has its own
context at `secureconn/CONTEXT.md`.

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
- `secureconn/`: standalone secure connection workspace module.

## Architecture

- Single binary entrypoint: `cmd/tailedbox`.
- Command dispatch lives under `internal/cli`.
- Interactive terminal UI lives under `internal/ui`.
- Local role is selected by initialization and persisted in local config/state.
- Node identity uses Ed25519 keys generated locally.
- The foreground agent is `tailedbox agent run`.
- Local health is persisted to `<state-dir>/agent/status.json`.
- Secure connection behavior and roadmap are tracked in `secureconn/CONTEXT.md`.

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
  - `network`
  - `node`
  - `pg`
- Structured JSONL logging with redaction.
- Human-readable and JSON-capable command output.
- Bubble Tea interactive no-args menu for real terminals.
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
- Mesh CLI surfaces that integrate with `secureconn` through the root app
  adapter.

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

Build and test:

```bash
go version
go test ./...
go test ./secureconn/...
go build ./cmd/tailedbox
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

## Current Limitations

- Join-code enrollment is local-state backed.
- `--master-state-dir` is temporary until network enrollment is implemented in
  the secure connection module.
- Master status knows trusted nodes from local files only.
- Worker status intentionally does not expose full cluster inventory.
- Systemd install usually requires root.
- Service control commands call `systemctl` and work only on Linux systems with
  systemd.
- Network and node management namespaces are reserved but not implemented.
- Release installer work has not started.
- PostgreSQL, web UI, master HA, firewall provider abstraction, and heavyweight
  infrastructure are not implemented.

## Roadmap

1. Continue secure connection work tracked in `secureconn/CONTEXT.md`.
2. Add a versioned GitHub release installer.
3. Add future managed services only after the secure connection foundation is
   reliable.

## Release Installer Scope

Not implemented yet:

- `install.sh`
- exact version installation
- OS/architecture detection
- checksum verification
- GitHub Release artifact layout
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
