# Design

## Current Behavior

The repo already has three useful foundations:

1. Deployment entrypoints exist for Compose, Kubernetes, and VM.
2. The write path commits to PostgreSQL and emits outbox events.
3. Search/read code already depends on adapter interfaces instead of raw backend clients.

That foundation is incomplete for a real multi-store deployment:

- deployment scripts still boot memory adapters
- config validation rejects all graph/vector backends beyond `neo4j` and `pgvector`
- bootstrap cannot construct a real graph client
- reconciliation cannot read external graph state
- sync correctness is payload-based only and has no persisted per-replica version ledger

## Goals

- Support a true full flow: PostgreSQL source of truth, graph replica, vector replica, worker projection, search/read over real adapters.
- Keep the service code backend-neutral by extending adapter implementations rather than branching `read.Service`, `search.Service`, or `workers.Runtime`.
- Keep scope tight by defining supported backend profiles instead of promising every possible graph/vector combination at once.
- Make replica version alignment observable and repairable.

## Non-Goals

- Building a cloud-specific operator stack.
- Introducing distributed transactions across the three datastores.
- Solving every backend-specific scaling concern in one change.

## Key Decisions

### 1. Support named backend profiles across all deployment surfaces

Each deployment surface will accept a small catalog of runtime profiles, for example:

- `pgvector-memgraph`
- `pgvector-neo4j`
- `qdrant-memgraph`
- `qdrant-neo4j`
- `milvus-neo4j`
- `qdrant-nebula`

The profile selects:

- graph adapter kind
- vector adapter kind
- supporting service containers or external endpoints
- required environment variables
- post-deploy validation target

This keeps Compose/K8s/VM aligned without forcing one giant cartesian-product test matrix on every change.

### 2. Expand adapter contracts without leaking backend details upward

`GraphAdapter` remains the service boundary, but gains production-grade snapshot and health semantics:

```go
type GraphAdapter interface {
    UpsertNode(ctx context.Context, node GraphNode) error
    DeleteNode(ctx context.Context, nodeID string) error
    UpsertRelationship(ctx context.Context, rel GraphRelationship) error
    DeleteRelationship(ctx context.Context, relID string) error
    ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error)
    ListNodes(ctx context.Context) ([]GraphNode, error)
    ListRelationships(ctx context.Context) ([]GraphRelationship, error)
    ReadSyncVersion(ctx context.Context, entityID string) (ReplicaVersion, error)
}
```

Adapters to add:

- `Neo4jGraphAdapter`
- `MemgraphGraphAdapter`
- `NebulaGraphAdapter`

`Neo4j` and `Memgraph` both compile `GraphQuery` to Cypher. `NebulaGraph` gets its own translation layer, but the compiler still emits the same neutral `GraphQuery`.

`VectorAdapter` stays neutral and adds version visibility:

```go
type VectorAdapter interface {
    Upsert(ctx context.Context, doc VectorDocument) error
    Delete(ctx context.Context, nodeID string) error
    ANN(ctx context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error)
    Snapshot(ctx context.Context) ([]VectorDocument, error)
    ReadSyncVersion(ctx context.Context, entityID string) (ReplicaVersion, error)
}
```

Adapters to add:

- `PgVectorAdapter`
- `QdrantVectorAdapter`
- `MilvusVectorAdapter`

### 3. Persist sync version in PostgreSQL and mirror it into graph/vector payloads

The service should stop treating sync convergence as “latest payload looks similar enough” and instead track an explicit projection version.

Add a PostgreSQL table:

```sql
CREATE TABLE kg_projection_versions (
    entity_id UUID NOT NULL,
    entity_kind TEXT NOT NULL,
    source_version BIGINT NOT NULL,
    source_event_id UUID NOT NULL,
    source_updated_at TIMESTAMPTZ NOT NULL,
    graph_backend TEXT,
    graph_version BIGINT,
    graph_synced_at TIMESTAMPTZ,
    vector_backend TEXT,
    vector_version BIGINT,
    vector_synced_at TIMESTAMPTZ,
    PRIMARY KEY (entity_id, entity_kind)
);
```

The worker increments or derives `source_version` from committed outbox order and writes that ledger in PostgreSQL. Each graph/vector adapter writes the same version into its replica payload:

- graph node / relationship property: `_kg_sync_version`
- vector payload metadata: `_kg_sync_version`

That gives reconciliation a deterministic comparison point:

- source version in PostgreSQL
- applied graph version in graph store
- applied vector version in vector store

### 4. Reconciliation must compare external state first, never legacy in-memory mirrors

`workers.Runtime.Reconcile` should:

- load authoritative nodes/relationships from PostgreSQL
- load replica snapshots from the configured graph/vector adapters
- compare payload equality
- compare `_kg_sync_version` / projection ledger versions
- report `missing_projection`, `stale_projection_version`, `payload_mismatch`, and `orphan_projection`

The legacy `r.graph` / `r.vector` mirrors stay test-only helpers and must not be used as reconciliation fallback when a real adapter is configured.

### 5. Deployment scripts become profile-aware and validation-driven

Compose:

- add profile-specific service definitions for graph/vector backends
- support `KG_RUNTIME_PROFILE=<profile>`
- run migrations plus a backend-aware validation script

Kubernetes:

- render backend-specific `ConfigMap`/`Secret`/env snippets
- support an external dependency mode and a local dev profile mode

VM:

- support `.env`-driven profile selection
- run the same validation script against reachable remote services

Validation should prove:

1. node write commits to PostgreSQL
2. outbox worker projects to graph and vector stores
3. template read hits the configured graph adapter
4. semantic search hits the configured vector adapter
5. reconciliation shows matching sync version across the three stores

## Risks And Mitigations

- Backend matrix can grow too wide.
  - Mitigation: define a small supported profile catalog and separate “implemented” from “future-capable”.
- NebulaGraph translation can diverge from Cypher-based adapters.
  - Mitigation: keep translation isolated behind adapter conformance and profile-specific integration tests.
- Version metadata may drift if adapters update payload but not ledger.
  - Mitigation: worker writes ledger state only after adapter write succeeds; reconciliation checks both ledger and backend payload.

## Validation Strategy

- Add adapter conformance suites for `Memgraph`, `NebulaGraph`, `Qdrant`, and `Milvus`.
- Add profile-based deployment smoke tests for Compose.
- Add profile-aware integration validation runnable against Kubernetes and VM targets.
- Verify reconciliation reports version-aligned state after successful projection and version drift after induced replica rollback.
