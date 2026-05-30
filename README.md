# Tailedbox

Tailedbox is a lightweight, CLI-first control system for securely connecting,
provisioning, and managing services on Linux VPSs.

The current POC includes the project bootstrap, CLI skeleton, and local
master/worker role initialization. It intentionally does not implement
PostgreSQL, enrollment, the agent daemon, or the mesh transport yet.

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

## Local State

`tailedbox init --role master|worker` creates a local node identity and agent
configuration scaffold:

```txt
<state-dir>/
  agent/config.json
  master/ or worker/
  node.json
  node_identity_public.json
  secrets/node_identity_ed25519.pem
```

State directories are written with `0700` permissions. Config, metadata, and
identity files are written with `0600` permissions. The Ed25519 private identity
key is generated locally and is not logged.
