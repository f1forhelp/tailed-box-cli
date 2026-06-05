# Secureconn Standalone Lab

This guide tests `secureconn` without the root Tailedbox CLI.

## Local Invite Join

Run from `secureconn/`:

```bash
go run ./cmd/secureconn lab init --role master --node-id lab_master --state-dir .tmp/manual/master
go run ./cmd/secureconn lab init --role worker --node-id lab_worker --state-dir .tmp/manual/worker

go run ./cmd/secureconn lab invite create \
  --state-dir .tmp/manual/master \
  --public-endpoint 127.0.0.1:24177 \
  --vpc-endpoint 10.0.1.5:24177
```

Keep the master running in one terminal:

```bash
go run ./cmd/secureconn lab run \
  --state-dir .tmp/manual/master \
  --host 127.0.0.1 \
  --port 24177
```

In a second terminal, join with the code printed by `invite create`:

```bash
go run ./cmd/secureconn lab join \
  --state-dir .tmp/manual/worker \
  --code <invite-code> \
  --master-endpoint 127.0.0.1:24177
```

Then test encrypted ping/pong:

```bash
go run ./cmd/secureconn lab ping \
  --state-dir .tmp/manual/worker \
  --peer lab_master \
  --endpoint 127.0.0.1:24177
```

## VPS Public IP Test

On the VPS, initialize the master and create an invite:

```bash
go run ./cmd/secureconn lab init --role master --node-id vps_master --state-dir /tmp/secureconn-master

go run ./cmd/secureconn lab invite create \
  --state-dir /tmp/secureconn-master \
  --public-endpoint <vps-public-ip>:24177
```

Allow inbound UDP `24177` in the VPS firewall or provider security group.

On the VPS, run the listener:

```bash
go run ./cmd/secureconn lab run \
  --state-dir /tmp/secureconn-master \
  --host 0.0.0.0 \
  --port 24177
```

On the worker machine:

```bash
go run ./cmd/secureconn lab init --role worker --node-id worker_1 --state-dir .tmp/vps/worker

go run ./cmd/secureconn lab join \
  --state-dir .tmp/vps/worker \
  --code <invite-code> \
  --master-endpoint <vps-public-ip>:24177

go run ./cmd/secureconn lab ping \
  --state-dir .tmp/vps/worker \
  --peer vps_master \
  --endpoint <vps-public-ip>:24177
```

## VPS VPC/Private IP Test

If the worker can reach the VPS over a private VPC network, create the invite
with both endpoints:

```bash
go run ./cmd/secureconn lab invite create \
  --state-dir /tmp/secureconn-master \
  --public-endpoint <vps-public-ip>:24177 \
  --vpc-endpoint <vps-vpc-ip>:24177
```

Join through the private/VPC address:

```bash
go run ./cmd/secureconn lab join \
  --state-dir .tmp/vps/worker \
  --code <invite-code> \
  --master-endpoint <vps-vpc-ip>:24177

go run ./cmd/secureconn lab ping \
  --state-dir .tmp/vps/worker \
  --peer vps_master \
  --endpoint <vps-vpc-ip>:24177
```

The public and VPC endpoints are persisted as peer metadata. The endpoint used
for the actual join is stored as the peer's last endpoint.

## Explicit Trust Revocation

After a worker joins, future restarts use persisted trust and the normal signed
encrypted session handshake. The original invite code is not reused.

To explicitly remove trust for a node:

```bash
go run ./cmd/secureconn lab trust revoke \
  --state-dir /tmp/secureconn-master \
  --peer worker_1
```

After revocation, that peer must enroll again with a new invite before normal
encrypted sessions are accepted.

## Notes

- Invite codes are one-time use and expire by default after 15 minutes.
- The raw invite secret is printed once and is not persisted in state.
- The invite code pins the expected master node ID, cluster ID, role, identity
  fingerprint, and expiry. A worker rejects enrollment responses signed by any
  other master identity.
- Enrollment challenges and accepts are bound to fresh nonces and the worker
  identity, so recorded enrollment packets cannot be replayed into a later join.
- After a worker joins, future restarts use persisted trust and the normal
  signed encrypted session handshake. The original invite code is not reused.
- Explicit revocation uses `secureconn lab trust revoke --state-dir <dir>
  --peer <node-id>`.
- Protect the invite code delivery channel. The invite code is the trust anchor
  for first enrollment; if an operator is tricked into using a different code,
  the software cannot infer the intended master.
- This lab flow tests network enrollment plus encrypted UDP ping/pong. Durable
  multi-peer sessions, reconnect leases, and production NAT traversal are still
  future work.
