# Complete Multi-Store Runtime Matrix

## Why

The repository has partial building blocks for deployment automation, graph/vector adapters, and async replica sync, but it still does not run a true end-to-end production-shaped flow across the storage planes the project wants to support.

The current gaps are concrete:

- `deploy/compose/docker-compose.yml`, `deploy/k8s/deployment.yaml`, and `scripts/deploy-vm.sh` start `kg-service` with `VECTOR_ADAPTER=memory`, `GRAPH_ADAPTER=memory`, and `FTS_ADAPTER=memory`, so the documented deployment paths do not exercise graph/vector replicas at all.
- `internal/bootstrap/wiring.go` supports `pgvector`, but `GRAPH_ADAPTER=neo4j` still returns `graph adapter neo4j is not wired in bootstrap yet`, and there is no runtime support for `memgraph`, `nebula`, `qdrant`, or `milvus`.
- `internal/platform/graphstore/neo4j.go` exposes `ListNodes` and `ListRelationships` as `nil`, which means reconciliation cannot verify real graph replica state.
- `internal/workers/runtime.go` still keeps a legacy in-memory projection copy and falls back to that state during reconciliation, which hides divergence between PostgreSQL and external replicas.
- The source record carries `domain_version`, but there is no durable projection-version model that records which outbox version has been applied to PostgreSQL, graph, and vector stores for the same entity.

Without closing those gaps, the project cannot confidently claim that Compose, Kubernetes, or VM deployments support a full relationship-db + graph-db + vector-db workflow, and operators cannot verify whether sync convergence or version alignment is actually correct.

## What Changes

- Add a repo-owned deployment matrix for full-flow runtime profiles spanning PostgreSQL plus one graph adapter and one vector adapter.
- Extend the graph adapter surface to support `memgraph` and `nebula`, and wire `neo4j`/`memgraph` into bootstrap as real selectable runtime backends.
- Extend the vector adapter surface to support `qdrant` and `milvus` alongside `pgvector`.
- Replace deployment defaults that silently use in-memory adapters with explicit runtime profiles and validation scripts that prove projections, queries, and reconciliation against real backends.
- Add durable projection version metadata and reconciliation rules so PostgreSQL, graph, and vector stores can be compared by applied sync version, not only by best-effort payload equality.

## Capabilities

- Compose, Kubernetes, and VM entrypoints can launch at least one full end-to-end runtime using external graph and vector stores.
- Operators can choose supported backend profiles through documented configuration rather than patching manifests by hand.
- Adapter conformance and post-deploy validation cover `postgres + graph + vector` flows, not just memory-backed smoke tests.
- Sync workers persist and expose a replica version record that can prove whether graph/vector projections are ahead, aligned, or behind the authoritative PostgreSQL state.

## Impact

- Makes the deployment documentation truthful about what is actually exercised.
- Closes the mismatch between the earlier OpenSpec design and the current bootstrap/runtime wiring.
- Gives the team a safe path to add `Memgraph`, `NebulaGraph`, `Qdrant`, and `Milvus` without rewriting read/search/write services.
