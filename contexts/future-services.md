# Future Services Context

This file owns context for later infrastructure features that should not start
before the secure connection foundation is reliable.

## Guardrail

Do not implement PostgreSQL, web UI, master HA, or heavyweight infrastructure
before the secure connection foundation is ready.

## PostgreSQL

Status: not implemented.

Known future scope:

- PostgreSQL deployment.
- Docker/native/NixOS runtime support.
- replication.
- backup/restore.
- HA/failover.
- quorum safety.

PostgreSQL should depend on secure node identity, enrollment, agent lifecycle,
reliable secure connectivity, runtime abstraction, and eventually HA
coordination.

## Web UI

Status: not implemented.

Known future scope:

- HTTPS browser UI.
- auth model.
- dashboard workflows.
- user/team/project model.

The web UI should use the same command/backend workflows as the CLI rather than
introducing separate UI-only behavior.

## Master HA

Status: not implemented.

Known future scope:

- embedded consensus.
- Raft store.
- leader election.
- replicated cluster state.
- split-brain refusal.

## Security Hardening

Status: not implemented.

Known future scope:

- firewall provider abstraction.
- default deny public DB/control-plane ports.
- secure remove-node flow.
- key/certificate rotation.
- lost/expired worker reconnect enforcement over live transport.

## Explicit Non-Goals For MVP

- Kubernetes dependency.
- external etcd dependency.
- Consul dependency.
- external VPN dependency.
