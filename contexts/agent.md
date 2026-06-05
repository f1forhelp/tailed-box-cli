# Agent Context

This file owns context for the foreground agent, heartbeat status, logs alias,
and Linux systemd lifecycle.

## Architecture

- The same `tailedbox` binary runs interactive CLI commands and long-running
  agent mode.
- Systemd runs `tailedbox agent run`.
- Local health is persisted to `<state-dir>/agent/status.json`.
- Secure connection runtime details live in `secureconn/CONTEXT.md`; this file
  tracks the root app agent lifecycle and service-management behavior.

## Implemented

- `agent` command namespace.
- Foreground agent loop.
- Periodic local health heartbeat writes.
- Agent status reading and stale heartbeat degradation.
- Memory diagnostics in status output.
- `tailedbox agent logs` alias over the redacted JSONL log reader.
- `tailedbox logs` alias.
- Linux systemd unit generation.
- `agent install --dry-run` preview without writing files or invoking
  `systemctl`.
- Agent install/preview requires node initialization.
- Real systemd install/control is Linux-only and refused on non-Linux
  development machines.

## Commands

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
tailedbox logs
```

## Heartbeat Status

The foreground agent writes:

```txt
<state-dir>/agent/status.json
```

Agent status includes:

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

## Systemd Unit

Generated units include:

- `Restart=always`
- `RestartSec=5s`
- `After=network-online.target`
- `Wants=network-online.target`
- `NoNewPrivileges=true`
- `PrivateTmp=true`
- `ProtectSystem=full`
- explicit writable paths for config/state/log directories

## Limitations And Next Work

- Systemd install usually requires root.
- Service control commands call `systemctl` and work only on Linux systems with
  systemd.
- Secure connection runtime integration details are tracked in
  `secureconn/CONTEXT.md`.
