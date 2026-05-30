# Tailedbox

Tailedbox is a lightweight, CLI-first control system for securely connecting,
provisioning, and managing services on Linux VPSs.

Part 1 is the project bootstrap and CLI skeleton. It intentionally does not
implement PostgreSQL, enrollment, the agent daemon, or the mesh transport yet.

## Build

This module is pinned to the current stable Go toolchain with:

```txt
go 1.26
toolchain go1.26.3
```

Build the CLI:

```bash
go build ./cmd/tailedbox
```

## Initial Commands

```bash
tailedbox version
tailedbox status
tailedbox init --role master
tailedbox init --role worker
tailedbox master status
tailedbox worker status
tailedbox logs
tailedbox logs --follow
tailedbox debug logs enable
tailedbox debug logs disable
```

PostgreSQL and mesh command namespaces are present as planned stubs so future
parts can fill them in without reshaping the CLI.
