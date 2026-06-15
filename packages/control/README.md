# control

Shared control/action layer for CLI, TUI, and future Web/MCP control surfaces.

## Purpose

Control actions keep business logic out of user interfaces. CLI and TUI entrypoints parse input, call this module, and display results.

Current actions include:

- Network initialization.
- Identity initialization and display.
- Master-authorized join-code creation.
- Local join-code consumption.
- Peer listing.
- Master-authorized peer revocation.

Each action returns an equivalent CLI command string so UIs can show users what command corresponds to an action.

## Boundaries

This module is intentionally thin. It coordinates securemesh packages but does not implement service managers, production transport, websites, MCP, or remote command execution.

## Testing

Run:

```sh
go test ./...
```
