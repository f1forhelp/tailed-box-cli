# AGENTS.md

This file gives coding agents project-specific context for working on Tailedbox.

## Project

Tailedbox is a lightweight Go CLI for securely connecting, provisioning, and
managing Linux VPS nodes. The same `tailedbox` binary is used for master/control
nodes and worker nodes.

The first POC milestone is the secure master/worker communication layer. Do not
implement PostgreSQL, a web UI, or heavyweight infrastructure before the mesh
foundation is in place.

## Current Status

Implemented so far:

- Part 1: Go project bootstrap and CLI skeleton.
- CLI namespaces for `master`, `worker`, `agent`, `mesh`, `network`, `node`,
  and `pg`.
- Structured logging with redaction and opt-in debug logging.
- Polished terminal output using Lip Gloss.
- Interactive no-args command menu using Bubble Tea for real terminals, with
  plain help fallback for non-interactive execution.
- Part 3: durable role initialization with local node identity, strict file
  permissions, metadata, and agent config scaffold.
- Part 4: local-state join-code enrollment MVP with one-time codes, hashed code
  secrets, trusted-node records, joined-cluster metadata, and audit events.
- Part 5: lightweight local agent lifecycle with foreground run, heartbeat
  status, memory diagnostics, logs alias, and Linux systemd install/control
  commands.

Not implemented yet:

- Mesh protocol and transport.
- PostgreSQL managed service.
- Release installer.
- Web UI.

## Go Toolchain

Use the pinned module toolchain:

```bash
go version
go test ./...
go build ./cmd/tailedbox
```

The module is configured for:

```txt
go 1.26
toolchain go1.26.3
```

## Important Commands

```bash
tailedbox version
tailedbox status
tailedbox init --role master
tailedbox init --role worker
tailedbox master status
tailedbox worker status
tailedbox master join-code create --role worker --ttl 15m
tailedbox worker join --code <join-code> --master-state-dir <master-state-dir>
tailedbox agent run
tailedbox agent status
tailedbox agent install --dry-run
tailedbox agent start
tailedbox agent stop
tailedbox agent restart
tailedbox agent logs
tailedbox logs
tailedbox debug logs enable
tailedbox debug logs disable
```

## Coding Rules

- Keep the CLI as one binary named `tailedbox`.
- Keep master and worker behavior role-based, not separate binaries.
- Keep packages small and purpose-driven under `internal/`.
- Automatically update `CONTEXT.md` whenever meaningful project behavior,
  architecture, decisions, commands, limitations, or roadmap items change. Do
  this as part of the same edit before reporting back; do not wait for a
  separate user reminder.
- Prefer standard library code unless a package clearly improves maintainability
  or terminal UX.
- Preserve JSON output behavior when improving human-readable CLI output.
- Treat interactive/TUI screens as launchers for the same CLI workflows. Every
  UI action must have an equivalent `tailedbox ...` command and should run
  through the same command dispatcher rather than separate UI-only logic.
- Keep interactive UI responsibilities separated: Bubble Tea model/update code
  should own input and selection state, while Lip Gloss rendering/layout should
  live behind dedicated renderer helpers.
- Preserve non-interactive behavior for scripts. The interactive menu should run
  only when stdin and stdout are real terminals.
- Do not leak secrets, tokens, private keys, join codes, or decrypted payloads in
  logs or errors.
- Do not persist raw join codes. Persist only short-lived hashes and minimal
  metadata.
- Keep filesystem permissions strict:
  - state and secret directories: `0700`
  - config, metadata, identity, and secret files: `0600`
- Do not implement PostgreSQL before mesh/enrollment foundations are ready.
- Do not introduce Kubernetes, external etcd, Consul, or external VPN
  dependencies for the MVP.

## Commit Policy

Do not commit automatically. Show the user the result and ask before creating a
commit. If committing is approved, keep each commit focused and use a clear
message such as:

```txt
feat: add durable node role initialization
```

## Local State Layout

After `tailedbox init --role master|worker`, local state should look like:

```txt
<state-dir>/
  agent/
    config.json
    status.json
  master/ or worker/
  node.json
  node_identity_public.json
  secrets/node_identity_ed25519.pem
```

The private identity key must be generated locally, stored with strict
permissions, and never logged.

## Recommended Roadmap

1. Part 6: Tailedbox Mesh Protocol Design.
2. Part 7: Tailedbox Mesh MVP Implementation.
3. Part 2: Versioned GitHub Release Installer, once the binary has meaningful
   node behavior to ship.
