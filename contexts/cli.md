# CLI Context

This file owns context for the root Tailedbox CLI, output, logging, status, and
interactive terminal UI.

## Architecture

- Single binary entrypoint: `cmd/tailedbox`.
- Command dispatch lives under `internal/cli`.
- Interactive terminal UI lives under `internal/ui`.
- Bubble Tea model/update code owns keyboard input, cursor state, selected
  command args, and quit behavior.
- Lip Gloss rendering/layout code lives behind dedicated renderer helpers.
- The interactive menu is a launcher over the normal CLI command graph. UI
  actions must display and run equivalent `tailedbox ...` commands.
- UI-only workflows are intentionally avoided so scripts, automation, future web
  workflows, and terminal users share backend behavior.

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
- Structured JSONL logging.
- Opt-in debug logging:

```bash
tailedbox debug logs enable
tailedbox debug logs disable
```

- Log redaction for tokens, secrets, passwords, private keys, join codes,
  credentials, and decrypted payload markers.
- Human-readable and JSON-capable command output.
- Lip Gloss-based terminal styling.
- Grouped command help.
- Status/key-value output.
- ASCII cluster tables for master status.
- Bubble Tea-based interactive no-args menu for real terminals.
- Plain help fallback for non-interactive execution.

## Current Commands

Core:

```bash
tailedbox
tailedbox version
tailedbox status
tailedbox status --json
```

Logs and debug:

```bash
tailedbox logs
tailedbox logs --follow
tailedbox logs --lines 50
tailedbox debug logs enable
tailedbox debug logs disable
```

Reserved future namespaces:

```bash
tailedbox network create --driver tailedbox-mesh
tailedbox network status
tailedbox network peers

tailedbox node list
tailedbox node approve <node-id>

tailedbox pg init
tailedbox pg deploy
tailedbox pg status
tailedbox pg failover
tailedbox pg backup
tailedbox pg restore
```

## Current Behavior

- If stdin/stdout are real terminals, `tailedbox` opens an interactive menu.
- If running in scripts, pipes, tests, or non-TTY contexts, `tailedbox` prints
  normal help and exits.
- The interactive menu uses Bubble Tea's alternate screen.

Interactive menu options currently include:

- Status
- Agent status
- Initialize as master
- Initialize as worker
- Master status
- Worker status
- Create worker join code
- Create master join code
- Recent logs
- Version
- Help
- Exit

## Limitations And Next Work

- PostgreSQL commands are reserved but not implemented.
- Network and node management namespaces are reserved but not implemented.
- Future web UI should use the same command/backend workflows rather than
  separate UI-only behavior.
