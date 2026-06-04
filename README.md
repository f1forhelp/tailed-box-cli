# Tailedbox

Tailedbox is a lightweight, CLI-first Go control system for securely connecting,
provisioning, and managing services on Linux VPSs.

The same `tailedbox` binary runs on master/control nodes and worker nodes.
Role-specific behavior comes from local initialization and configuration rather
than separate binaries.

## Repository Layout

- Root module: the Tailedbox CLI application.
- `secureconn/`: standalone workspace module for secure connection and control
  communication.
- `internal/`: app-only packages for CLI, local state, enrollment, agent
  lifecycle, status, and adapters.
- `cmd/tailedbox`: binary entrypoint.

Project context lives in [`CONTEXT.md`](CONTEXT.md). Secure connection module
context lives in [`secureconn/CONTEXT.md`](secureconn/CONTEXT.md).

## Build And Test

```bash
go version
go test ./...
go test ./secureconn/...
go build ./cmd/tailedbox
```

The module is pinned to:

```txt
go 1.26
toolchain go1.26.3
```

## Basic Usage

```bash
tailedbox version
tailedbox status
tailedbox init --role master
tailedbox init --role worker
tailedbox agent run
tailedbox mesh status
```

Running `tailedbox` with no arguments opens an interactive terminal menu when
stdin and stdout are real terminals. In scripts or non-interactive execution it
prints normal CLI help.

## Local State

Tailedbox stores local node state, config, logs, agent status, and identity
files under the configured state directories. State and secret directories use
strict private permissions, and the node identity private key is generated
locally and never logged.
