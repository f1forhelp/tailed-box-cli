# Secureconn Protocol Context

This file owns context for packet envelopes, JSON control messages, and protocol
design documentation.

## Architecture

- Protocol package: `secureconn/protocol`.
- Protocol design document: `secureconn/docs/mesh-protocol-design.md`.
- Packet envelope uses the `TBXM` magic and a versioned binary header.
- Control messages are JSON payloads carried inside encrypted packets after
  session establishment.
- Network enrollment message shapes are defined for the future flow even though
  network enrollment is not implemented yet.

## Implemented

- Versioned `TBXM` UDP packet envelope.
- Packet types:
  - client hello
  - server hello
  - client auth
  - encrypted data
  - rekey
  - close
- Strict packet decode validation.
- Payload size limit.
- Reserved flag rejection.
- JSON control-message types for:
  - ping
  - pong
  - peer update
  - status request/response
  - diagnostics
  - future network enrollment

## Tests

- Packet encode/decode.
- Malformed envelope rejection.
- Unsupported packet type rejection.
- Reserved flags rejection.
- Payload length mismatch rejection.
- Control-message shape tests.

## Limitations And Next Work

- Rekey and close packet types exist, but full rekey/close behavior is not
  wired into durable transport sessions yet.
- Network enrollment message types exist, but the network enrollment flow is not
  implemented yet.
