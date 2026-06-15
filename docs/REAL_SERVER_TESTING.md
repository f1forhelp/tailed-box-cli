# Real Server Testing Procedure

This procedure tests the current minimal secure server-to-server connection path.

Current transport:

- Standard-library TLS over TCP.
- Runtime TLS certificates generated from persistent node identity.
- Custom mesh certificate verification against local peer state and revocation state.
- One `ping`/`pong` control message for connectivity verification.

This is not service management, remote command execution, secret transfer, Docker/Postgres management, website, MCP, NAT traversal, or the future optimized QUIC/Noise transport.

## Network Requirements

- Master has a reachable TCP port.
- Worker can connect to the master's host/port.
- Default test port is `9443`.
- Open the firewall for the master's selected TCP port.
- Use `127.0.0.1` only for local testing. On a real server, bind the master to an explicit interface such as `0.0.0.0:9443` or a private network IP.

## Master Setup

On the master server:

```sh
go run ./cmd/infra --config-root ./state-master network init
go run ./cmd/infra --config-root ./state-master identity init --role master
go run ./cmd/infra --config-root ./state-master peer export > master.peer.json
```

Record the `network_id` printed by `network init`. It is not a secret.

## Worker Setup

On the worker server, use the master's `network_id`:

```sh
go run ./cmd/infra --config-root ./state-worker network import --id <network_id>
go run ./cmd/infra --config-root ./state-worker identity init --role worker
go run ./cmd/infra --config-root ./state-worker peer export > worker.peer.json
```

## Exchange Public Peer Files

Copy `master.peer.json` to the worker and `worker.peer.json` to the master using your normal secure file-transfer method.

These files contain public identity metadata only. They do not contain private keys, join codes, session keys, credentials, or secrets.

On the master:

```sh
go run ./cmd/infra --config-root ./state-master peer add --file worker.peer.json
```

On the worker:

```sh
go run ./cmd/infra --config-root ./state-worker peer add --file master.peer.json
```

## Start Listener

On the master:

```sh
go run ./cmd/infra --config-root ./state-master mesh listen --bind 0.0.0.0:9443
```

For same-machine testing, use:

```sh
go run ./cmd/infra --config-root ./state-master mesh listen --bind 127.0.0.1:9443
```

## Ping From Worker

On the worker:

```sh
go run ./cmd/infra --config-root ./state-worker mesh ping --endpoint <master-host-or-ip>:9443
```

Expected success output includes:

```text
mesh ping ok
remote_node_id: <master-node-id>
remote_role: master
network_id: <network-id>
```

## Revocation Test

On the master, revoke the worker:

```sh
go run ./cmd/infra --config-root ./state-master peer revoke --node <worker-node-id> --role worker --reason test
```

The worker's next `mesh ping` should fail because the master rejects revoked peers.

## Current Limitations

- Peer exchange is manual public metadata import/export, not online join-code pairing.
- Transport is TLS/TCP for immediate real-server testing, not QUIC yet.
- No NAT traversal.
- No background daemon/supervisor.
- No service-management messages.
- No remote command execution.
- No secret transmission beyond TLS handshake internals.

## Troubleshooting

- If ping fails with an unknown peer error, confirm both sides ran `peer add` with the other side's exported peer file.
- If ping fails with a wrong network error, confirm the worker imported the exact master `network_id` before identity initialization.
- If ping fails after revocation, that is expected.
- If the worker cannot connect, confirm firewall, bind address, host/IP, and port.
