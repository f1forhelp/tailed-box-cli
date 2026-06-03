# Tailedbox

Tailedbox is a lightweight, CLI-first control system for securely connecting,
provisioning, and managing services on Linux VPSs.

The current POC includes the project bootstrap, CLI skeleton, local
master/worker role initialization, local-state enrollment foundation, and a
lightweight local agent/systemd lifecycle. The mesh protocol design is captured
in `docs/mesh-protocol-design.md`, and the first internal mesh
protocol/store/crypto/session foundation exists. The local agent now owns a mesh
control socket and the `tailedbox mesh enable`, `disable`, `status`, `peers`,
`ping`, and `diagnose` command surfaces are active. Direct enrolled
worker-to-master UDP ping/pong is encrypted and authenticated. PostgreSQL is
intentionally still reserved for after the mesh foundation.

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
tailedbox mesh enable
tailedbox mesh enable --master-endpoint <host:port>
tailedbox mesh disable
tailedbox mesh status
tailedbox mesh peers
tailedbox mesh ping <node-id>
tailedbox mesh diagnose
```

PostgreSQL remains a planned namespace. Mesh commands are active for local
runtime status, peer observations, diagnostics, local-agent ping dispatch, and
direct encrypted UDP worker-to-master ping/pong.

## Mesh Protocol Design

Part 6 is documented in
[`docs/mesh-protocol-design.md`](docs/mesh-protocol-design.md). The design
covers the mesh threat model, node trust model, Ed25519/X25519 handshake,
session key lifecycle, UDP packet envelope, network enrollment flow, peer
discovery, direct UDP MVP, future relay fallback, firewall posture, and the
implementation boundaries for Part 7.

The current Part 7 foundation includes:

- `internal/mesh/protocol` for packet envelopes and control-message types.
- `internal/mesh/store` for private mesh status and peer observation files.
- `internal/mesh/crypto` for transcript signatures, X25519/HKDF session keys,
  nonce construction, and AES-GCM helpers.
- `internal/mesh/session` for replay-window tracking and encrypted packet
  seal/open helpers bound to the `TBXM` envelope header.
- `internal/mesh/transport` for direct UDP listen/send/receive, enrolled
  handshake validation, and encrypted worker-to-master ping/pong.
- `internal/mesh/control` for the local agent control socket used by mesh CLI
  commands.
- `internal/mesh/service` for the agent-owned mesh service scaffold and runtime
  status refresh.

The next Part 7 slice is durable session lifecycle, rekey handling,
master-to-worker routing beyond observed endpoints, reconnect lease enforcement
over live sessions, and network enrollment over `--master-endpoint`.

## Enrollment POC

`tailedbox master join-code create` creates one-time, short-lived enrollment
codes for workers or additional masters. The raw code is printed once for the
operator and is not persisted; the master stores only a hash and minimal
metadata.

Until network enrollment exists, `tailedbox worker join` and `tailedbox master
join` still use a local-state enrollment stand-in:

```bash
tailedbox worker join --code <join-code> --master-state-dir <master-state-dir>
```

The `--master-state-dir` flag will be replaced by network enrollment over the
Tailedbox mesh/control channel in the mesh MVP.

## Local Agent

`tailedbox agent run` starts the lightweight foreground agent loop. It writes a
local heartbeat file with node role, node ID, uptime, memory usage, goroutine
count, and health state. It also starts the local mesh control socket and writes
mesh runtime status under `<state-dir>/mesh/status.json`.

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
