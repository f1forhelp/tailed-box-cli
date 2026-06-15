# Real Server Testing Procedure

This procedure tests current secure server-to-server pairing and connectivity.

Current CLI transport:

- TLS/TCP online pairing on port `9444` by default.
- TLS/TCP mesh ping on port `9443` by default.
- Runtime TLS certificates generated from persistent node identity.
- Mesh peer verification against local peer state and revocation state.

This is not service management, remote command execution, application secret transfer beyond pairing, Docker/Postgres management, website, MCP, NAT traversal, or QUIC CLI transport.

## Network Requirements

- Master has reachable TCP ports for pairing and mesh ping.
- Worker can connect to the master's host/ports.
- Open firewall ports for the selected master binds.
- Use `127.0.0.1` only for local testing. On a real server, bind to an explicit interface such as `0.0.0.0:9444` and `0.0.0.0:9443` or a private network IP.

## Master Setup

On the master server:

```sh
go run ./cmd/infra --config-root ./state-master network init
go run ./cmd/infra --config-root ./state-master identity init --role master
go run ./cmd/infra --config-root ./state-master identity show
go run ./cmd/infra --config-root ./state-master join-code create --role worker
```

Record:

- `node_id` from `identity show`. This is the master node ID pin. It is not secret, but the worker must receive it through an authentic channel.
- `join_code` from `join-code create`. Treat it as a secret and use it once.

## Start Pairing Listener

On the master:

```sh
go run ./cmd/infra --config-root ./state-master pair listen --bind 0.0.0.0:9444
```

For same-machine testing:

```sh
go run ./cmd/infra --config-root ./state-master pair listen --bind 127.0.0.1:9444
```

## Join From Worker

On the worker:

```sh
go run ./cmd/infra --config-root ./state-worker pair join --endpoint <master-host-or-ip>:9444 --code <join_code> --role worker --master-node <master-node-id>
```

Expected success output includes:

```text
pairing complete
master_node_id: <master-node-id>
network_id: <network-id>
role: worker
```

The worker creates its network and identity state during pairing, and both sides persist public peer metadata.

## Start Mesh Listener

On the master:

```sh
go run ./cmd/infra --config-root ./state-master mesh listen --bind 0.0.0.0:9443
```

For same-machine testing:

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

The worker's next `mesh ping` should fail because the master rejects revoked peers. Pairing with the same revoked node identity is also rejected.

## Manual Peer Exchange Fallback

Manual public peer export/import is still available for debugging or offline setup:

```sh
go run ./cmd/infra --config-root ./state-master peer export > master.peer.json
go run ./cmd/infra --config-root ./state-worker peer export > worker.peer.json
go run ./cmd/infra --config-root ./state-master peer add --file worker.peer.json
go run ./cmd/infra --config-root ./state-worker peer add --file master.peer.json
```

Peer export files contain public identity metadata only. They do not contain private keys, join codes, session keys, credentials, or secrets.

## Current Limitations

- Worker must pin the expected master node ID during pairing.
- Pairing uses TLS/TCP, not PAKE/OPAQUE yet.
- Mesh CLI ping uses TLS/TCP, not QUIC yet.
- No NAT traversal.
- No background daemon/supervisor.
- No service-management messages.
- No remote command execution.

## Troubleshooting

- If pairing fails with a master mismatch, confirm the worker used the exact master `node_id` from `identity show`.
- If pairing fails with an invalid or consumed code, create a fresh join code on the master.
- If ping fails with an unknown peer error, confirm pairing completed or both sides manually added peer metadata.
- If ping fails after revocation, that is expected.
- If the worker cannot connect, confirm firewall, bind address, host/IP, and port.
