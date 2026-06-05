# Tailedbox Context

This root context is intentionally minimal. Feature-specific progress,
architecture, commands, limitations, and roadmap items live in feature context
files.

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
- `contexts/`: feature-specific context files for the root application.

## Feature Contexts

Read and update the relevant context before changing a feature:

- `contexts/cli.md`: CLI, output, logging, status, and interactive terminal UI.
- `contexts/node-enrollment.md`: role initialization, node identity, local
  state, join-code enrollment, and audit records.
- `contexts/agent.md`: foreground agent, heartbeat status, logs alias, and
  systemd lifecycle.
- `secureconn/CONTEXT.md`: secure connection module context map. Detailed
  module feature context lives under `secureconn/contexts/`.
- `contexts/release.md`: installer, release, packaging, and distribution.
- `contexts/future-services.md`: PostgreSQL, future web UI, HA, firewall, and
  later infrastructure features.

## Global Commands

```bash
go version
go test ./...
go test ./secureconn/...
go build ./cmd/tailedbox
```

## Global Guardrails

- Keep one `tailedbox` binary for all roles.
- Keep CLI workflows scriptable and JSON output stable.
- Keep secrets out of logs and errors.
- Keep private filesystem permissions strict.
- Avoid PostgreSQL, web UI, Kubernetes, external etcd, Consul, and external VPN
  dependencies until the secure connection foundation is reliable.
- Do not commit automatically; ask first.
