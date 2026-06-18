# Deployment Guides

This section is for operators who need to deploy and verify `kg-service` in supported runtime targets.

## Choose A Path

- [Docker Compose](./compose.md) for the simplest self-contained local or single-host stack.
- [Kubernetes](./kubernetes.md) for cluster deployments that point at reachable Postgres and Redis services.
- [VM](./vm.md) for standalone host deployment with a built binary and your own process supervisor.
- [Integration Validation](./integration-test.md) for the repeatable post-deploy smoke and integration check.
- [User Deployment Guide](../guides/deployment.md) for task-oriented deployment and validation steps.

## What Is Shared

- The HTTP service listens on `KG_HTTP_HOST:KG_HTTP_PORT`, with `0.0.0.0:8082` as the bootstrap default.
- The service always needs reachable Postgres and Redis endpoints because bootstrap opens both on startup.
- The runtime profile is selected with `KG_RUNTIME_PROFILE`, with supported profiles such as `pgvector-memgraph`, `pgvector-neo4j`, `qdrant-memgraph`, `qdrant-neo4j`, `milvus-neo4j`, and `qdrant-nebula`.
- `scripts/validate-runtime-profile.sh` is the post-deploy smoke entrypoint for write, read, search, integrity, and reconcile checks.
- `GET /healthz` is public; protected routes still require `Authorization: Bearer <api_key>`.
- The repository includes a `Makefile` with repeatable build, run, deploy, and validation targets.
- The repository also includes `docs/requirements/` for PRD, URD, and SRS references.

## What To Read Next

- [API Reference](../api/README.md) for the current HTTP contract.
- [User Guides](../guides/README.md) for consumer-facing usage flows.
- [Operations Runbooks](../operations) for incidents and recovery work.
