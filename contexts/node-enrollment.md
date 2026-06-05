# Node And Enrollment Context

This file owns context for role initialization, node identity, local state,
join-code enrollment, joined-cluster metadata, trusted-node records, and audit
events.

## Architecture

- The same `tailedbox` binary runs on every node.
- Local role is selected by initialization and persisted in local config/state.
- Node identity uses Ed25519 keys generated locally.
- Master nodes store joining nodes' public identity metadata only.
- Raw join codes are displayed once and never persisted.
- Local enrollment currently uses `--master-state-dir` as a temporary transport
  stand-in until network enrollment is implemented in the secure connection
  module.

## Implemented

- `tailedbox init --role master`
- `tailedbox init --role worker`
- `tailedbox master init`
- `tailedbox worker init`
- Durable local node ID.
- Durable Ed25519 node identity.
- Public identity metadata.
- Private identity key generation.
- Local node metadata.
- Agent config scaffold.
- Role-specific state directory.
- Idempotent init for the same role.
- Refusal to change role after initialization.
- Status output includes identity readiness and agent config readiness.
- Master-only join-code creation.
- Worker and master join commands.
- One-time join-code lifecycle.
- Role-scoped join codes.
- TTL expiry.
- Used-code state.
- Hashed join-code secret storage.
- Trusted-node records on the issuing master.
- Joined-cluster metadata on the joining node.
- Master-controlled reconnect lease metadata.
- Audit JSONL events for join-code creation, join attempts, join success, and
  join failure.
- Master status includes trusted nodes.
- Worker status shows joined-cluster state.

## Commands

Initialization:

```bash
tailedbox init --role master
tailedbox init --role worker
tailedbox master init
tailedbox worker init
```

Status:

```bash
tailedbox master status
tailedbox master status --json
tailedbox worker status
tailedbox worker status --json
```

Enrollment:

```bash
tailedbox master join-code create --role worker --ttl 15m
tailedbox master join-code create --role master --ttl 15m
tailedbox worker join --code <join-code> --master-state-dir <path>
tailedbox master join --code <join-code> --master-state-dir <path>
```

## Local State Layout

After initialization, local state includes:

```txt
<state-dir>/
  agent/
    config.json
    status.json
  audit/events.jsonl
  enrollment/
    join-codes/
    trusted-nodes/
  master/ or worker/
  node.json
  node_identity_public.json
  secrets/node_identity_ed25519.pem
```

Permissions:

- state and secret directories: `0700`
- config, metadata, identity, audit, and secret files: `0600`

## Security Decisions

- The Ed25519 private identity key is generated locally.
- The private key is never logged.
- Master nodes store only public identity information for joined nodes.
- Join-code secrets are hashed before storage.
- Raw join codes are not persisted.

## Limitations And Next Work

- Join-code enrollment is local-state backed.
- `--master-state-dir` is temporary.
- Network enrollment is tracked by `secureconn/CONTEXT.md`.
- Master status knows trusted nodes from local files only.
- Worker status intentionally does not expose full cluster inventory.
