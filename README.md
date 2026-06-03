# Tailedbox

Tailedbox is a lightweight, CLI-first control system for securely connecting,
provisioning, and managing services on Linux VPSs.

The current POC includes the project bootstrap, CLI skeleton, local
master/worker role initialization, local-state enrollment foundation, and a
lightweight local agent/systemd lifecycle. The mesh protocol design is captured
in `docs/mesh-protocol-design.md`, but the encrypted mesh transport is not
implemented yet. PostgreSQL is intentionally still reserved for after the mesh
foundation.

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

Running `tailedbox` with no arguments opens an interactive terminal menu when
stdin/stdout are connected to a real terminal. In scripts, pipes, tests, or other
non-interactive contexts it prints normal help and exits.

The interactive menu is a launcher for normal CLI workflows. Every selectable
action shows and runs the equivalent `tailedbox ...` command, so anything
available in the UI remains available from scripts and direct CLI usage.

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
tailedbox logs --follow
tailedbox debug logs enable
tailedbox debug logs disable
```

PostgreSQL and mesh command namespaces are present as planned stubs. The mesh
commands will become active during the Part 7 mesh MVP implementation.

## Mesh Protocol Design

Part 6 is documented in
[`docs/mesh-protocol-design.md`](docs/mesh-protocol-design.md). The design
covers the mesh threat model, node trust model, Ed25519/X25519 handshake,
session key lifecycle, UDP packet envelope, network enrollment flow, peer
discovery, direct UDP MVP, future relay fallback, firewall posture, and the
implementation boundaries for Part 7.

## Enrollment POC

`tailedbox master join-code create` creates one-time, short-lived enrollment
codes for workers or additional masters. The raw code is printed once for the
operator and is not persisted; the master stores only a hash and minimal
metadata.

Until the mesh transport exists, `tailedbox worker join` and `tailedbox master
join` use a local-state transport stand-in:

```bash
tailedbox worker join --code <join-code> --master-state-dir <master-state-dir>
```

The `--master-state-dir` flag will be replaced by network enrollment over the
Tailedbox mesh/control channel in the mesh MVP.

## Local Agent

`tailedbox agent run` starts the lightweight foreground agent loop. It writes a
local heartbeat file with node role, node ID, uptime, memory usage, goroutine
count, and health state.

```bash
tailedbox agent run
tailedbox agent status
tailedbox agent status --json
```

After `tailedbox init --role master|worker`, Linux systemd integration is
explicit:

```bash
tailedbox agent install --dry-run
sudo tailedbox agent install --binary /usr/local/bin/tailedbox --start
sudo tailedbox agent start
sudo tailedbox agent stop
sudo tailedbox agent restart
tailedbox agent logs
```

`--dry-run` prints the generated unit without writing to `/etc/systemd/system`.
Non-Linux development machines can build and preview the unit, but service
installation/control is refused outside Linux.

## Local State

`tailedbox init --role master|worker` creates a local node identity and agent
configuration scaffold:

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

State directories are written with `0700` permissions. Config, metadata, and
identity files are written with `0600` permissions. The Ed25519 private identity
key is generated locally and is not logged.
