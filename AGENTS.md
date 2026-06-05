# AGENTS.md

This file is intentionally small. It tells coding agents which context to read
before changing Tailedbox.

## Project Basics

Tailedbox is a lightweight Go CLI for securely connecting, provisioning, and
managing Linux VPS nodes. The same `tailedbox` binary runs on master/control
nodes and worker nodes; behavior is role-based through local state and config.

Do not implement PostgreSQL, a web UI, or heavyweight infrastructure before the
secure connection foundation is reliable.

## Context Map

Always start with:

- `CONTEXT.md` for base Tailedbox app context.
- `secureconn/CONTEXT.md` when touching secure connection module code or mesh
  runtime behavior.

## Context Updates

- Update `CONTEXT.md` whenever base app behavior, architecture, decisions,
  commands, limitations, or roadmap items change.
- Update `secureconn/CONTEXT.md` whenever secure connection behavior,
  architecture, tests, limitations, or roadmap items change.
- Keep this `AGENTS.md` as routing plus basic guardrails only. Do not add
  detailed progress here.

## Core Guardrails

- Keep the CLI as one binary named `tailedbox`.
- Keep master and worker behavior role-based, not separate binaries.
- Prefer small, purpose-driven packages.
- Preserve JSON output behavior when improving human-readable CLI output.
- Keep README files limited to basic project/module information. Do not use
  README files for context, progress tracking, roadmap, or detailed status.
- Do not leak secrets, tokens, private keys, join codes, or decrypted payloads
  in logs or errors.
- Do not persist raw join codes. Persist only hashes and minimal metadata.
- Keep filesystem permissions strict:
  - state and secret directories: `0700`
  - config, metadata, identity, and secret files: `0600`
- Do not introduce Kubernetes, external etcd, Consul, or external VPN
  dependencies for the MVP.

## Go Toolchain

Use the pinned module toolchain:

```bash
go version
go test ./...
go test ./secureconn/...
go build ./cmd/tailedbox
```

The module is configured for:

```txt
go 1.26
toolchain go1.26.3
```

## Commit Policy

Do not commit automatically. Show the user the result and ask before creating a
commit.
