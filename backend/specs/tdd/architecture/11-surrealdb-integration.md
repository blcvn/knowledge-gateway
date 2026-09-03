# VNP Memory — SurrealDB Integration Solution

> **Version**: 1.0 | **Date**: 2026-05-09  
> **Status**: Proposed  
> **Impact**: All 35 services — infrastructure adapter layer only

---

## 1. Executive Summary

SurrealDB 2.x is a **multi-model database** that natively supports:
- **Relational** (SQL-like queries) → replaces PostgreSQL
- **Graph** (`RELATE` edges, graph traversal) → replaces Neo4j
- **Vector** (HNSW/MTREE indexes, `<|K,M|>` KNN) → replaces Qdrant/pgvector

This enables a **single-database deployment** that eliminates operational complexity of managing 3+ separate database systems, while maintaining the same Go interface contracts.

### Design Principle: **Backend-Agnostic via Interface Adapters**

```
Services → Interface (port) → Adapter (implementation)
                                    ├── PostgreSQL + Neo4j + Qdrant  (default)
                                    └── SurrealDB                    (optional)
```

No service business logic changes required. The switch is purely at the **infra/wire layer**.

---

## 2. SurrealDB Capability Mapping

### 2.1 What SurrealDB Replaces

| Current Component | SurrealDB Equivalent | SurrealQL Feature |
|-------------------|---------------------|-------------------|
| **PostgreSQL** (relational) | SurrealDB tables | `CREATE`, `SELECT`, `UPDATE`, `DELETE`, `DEFINE TABLE` |
| **PostgreSQL RLS** (tenancy) | SurrealDB namespaces + scopes | `USE NS {tenant} DB {service}` |
| **PostgreSQL advisory locks** | SurrealDB transactions | `BEGIN TRANSACTION ... COMMIT` |
| **pgvector** (vector search) | SurrealDB HNSW/MTREE | `DEFINE INDEX ... HNSW DIMENSION 1536 DIST COSINE` |
| **Neo4j** (graph) | SurrealDB graph edges | `RELATE node:a -> relationship -> node:b` |
| **Neo4j traversal** | SurrealDB graph queries | `SELECT ->edge->node FROM root` |
| **Qdrant** (vector DB) | SurrealDB vector index | `WHERE embedding <\|K,M\|> $query_vec` |
| **Redis** (cache) | ❌ NOT replaced | SurrealDB is NOT a cache replacement |
| **MinIO/S3** (object storage) | ❌ NOT replaced | SurrealDB is NOT an object store |

### 2.2 What SurrealDB Does NOT Replace

| Component | Reason | Keep As-Is |
|-----------|--------|-----------|
| **Redis** | Sub-ms cache, rate limiting, session state | Redis 7+ |
| **MinIO/S3** | Large binary file storage | S3-compatible |
| **NATS JetStream** | Async messaging, event streams | NATS |
| **VikingFS** | Go-native filesystem for OpenViking | VikingFS |

---

## 3. Architecture Solution

### 3.1 Backend Selection via Config

```yaml
# config/vnp-memory.yaml

# Profile 1: Default (multi-DB)
db:
  backend: "multi"        # PostgreSQL + Neo4j + Qdrant
  postgres:
    dsn: "postgres://..."
  neo4j:
    uri: "bolt://..."
  qdrant:
    addr: "qdrant:6334"
  redis:
    addr: "redis:6379"

# Profile 2: SurrealDB (unified)
db:
  backend: "surrealdb"    # Single SurrealDB instance
  surrealdb:
    addr: "ws://surrealdb:8000/rpc"
    ns: "vnp_memory"      # Cluster namespace
    db: "production"      # Database name
    user: "root"
    pass: "${SURREAL_PASSWORD}"
  redis:
    addr: "redis:6379"    # Redis ALWAYS required (cache/ratelimit)
```

### 3.2 Wire-Level Backend Switch

```go
// services/<any>/internal/infra/wire/wire.go

func ProvideGraphDB(cfg *config.Config) (graphdb.GraphDB, error) {
    switch cfg.DB.Backend {
    case "surrealdb":
        client := surrealdb.NewClient(cfg.DB.SurrealDB)
        return client.AsGraphDB(), nil       // SurrealDB graph adapter
    default:
        return neo4j.NewDriver(cfg.DB.Neo4j) // Neo4j driver
    }
}

func ProvideVectorDB(cfg *config.Config) (vectordb.VectorDB, error) {
    switch cfg.DB.Backend {
    case "surrealdb":
        client := surrealdb.NewClient(cfg.DB.SurrealDB)
        return client.AsVectorDB(), nil      // SurrealDB vector adapter
    default:
        return qdrant.NewClient(cfg.DB.Qdrant)
    }
}

func ProvideRelDB(cfg *config.Config) (reldb.RelationalDB, error) {
    switch cfg.DB.Backend {
    case "surrealdb":
        client := surrealdb.NewClient(cfg.DB.SurrealDB)
        return client.AsRelDB(), nil         // SurrealDB relational adapter
    default:
        return postgres.NewPool(cfg.DB.Postgres)
    }
}
```

### 3.3 SurrealDB Client Architecture

```go
// pkg/surrealdb/client.go
package surrealdb

import (
    "github.com/surrealdb/surrealdb.go/v2"
    "vnp-memory/pkg/adapters/graphdb"
    "vnp-memory/pkg/adapters/vectordb"
    "vnp-memory/pkg/adapters/reldb"
)

type Config struct {
    Addr     string
    NS       string // Namespace — cluster-level isolation
    DB       string // Database — environment isolation
    User     string
    Password string
}

type Client struct {
    conn *surrealdb.DB
    cfg  Config
}

func NewClient(cfg Config) (*Client, error) {
    conn, err := surrealdb.New(cfg.Addr)
    if err != nil { return nil, err }
    
    if _, err := conn.Signin(map[string]string{
        "user": cfg.User, "pass": cfg.Password,
    }); err != nil { return nil, err }
    
    if _, err := conn.Use(cfg.NS, cfg.DB); err != nil {
        return nil, err
    }
    
    return &Client{conn: conn, cfg: cfg}, nil
}

// AsGraphDB returns a GraphDB-compliant adapter
func (c *Client) AsGraphDB() graphdb.GraphDB {
    return &GraphAdapter{client: c}
}

// AsVectorDB returns a VectorDB-compliant adapter
func (c *Client) AsVectorDB() vectordb.VectorDB {
    return &VectorAdapter{client: c}
}

// AsRelDB returns a RelationalDB-compliant adapter
func (c *Client) AsRelDB() reldb.RelationalDB {
    return &RelationalAdapter{client: c}
}

func (c *Client) Close() error {
    return c.conn.Close()
}
```

---

## 4. SurrealDB Schema Design

### 4.1 Graph Nodes & Edges (replaces Neo4j)

```sql
-- Entity nodes (Cognee, Graphiti, Zep)
DEFINE TABLE entity_node SCHEMAFULL;
DEFINE FIELD id          ON entity_node TYPE string;
DEFINE FIELD name        ON entity_node TYPE string;
DEFINE FIELD type        ON entity_node TYPE string;
DEFINE FIELD summary     ON entity_node TYPE string;
DEFINE FIELD embedding   ON entity_node TYPE array<float>;
DEFINE FIELD group_id    ON entity_node TYPE string;
DEFINE FIELD created_at  ON entity_node TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at  ON entity_node TYPE datetime DEFAULT time::now();

DEFINE INDEX idx_entity_embedding ON entity_node FIELDS embedding
    HNSW DIMENSION 1536 DIST COSINE;
DEFINE INDEX idx_entity_name ON entity_node FIELDS name
    SEARCH ANALYZER autocomplete BM25;
DEFINE INDEX idx_entity_group ON entity_node FIELDS group_id;

-- Graph edges via RELATE
DEFINE TABLE relates_to SCHEMAFULL TYPE RELATION IN entity_node OUT entity_node;
DEFINE FIELD fact        ON relates_to TYPE string;
DEFINE FIELD weight      ON relates_to TYPE float DEFAULT 1.0;
DEFINE FIELD valid_at    ON relates_to TYPE datetime;
DEFINE FIELD invalid_at  ON relates_to TYPE option<datetime>;
DEFINE FIELD group_id    ON relates_to TYPE string;
```

### 4.2 Vector Collections (replaces Qdrant)

```sql
-- Chunk embeddings (Cognee, Supermemory)
DEFINE TABLE chunk SCHEMAFULL;
DEFINE FIELD id          ON chunk TYPE string;
DEFINE FIELD content     ON chunk TYPE string;
DEFINE FIELD embedding   ON chunk TYPE array<float>;
DEFINE FIELD source_id   ON chunk TYPE string;
DEFINE FIELD tenant_id   ON chunk TYPE string;
DEFINE FIELD metadata    ON chunk TYPE object;

DEFINE INDEX idx_chunk_embedding ON chunk FIELDS embedding
    HNSW DIMENSION 1536 DIST COSINE;
DEFINE INDEX idx_chunk_content ON chunk FIELDS content
    SEARCH ANALYZER autocomplete BM25;
```

### 4.3 Relational Tables (replaces PostgreSQL)

```sql
-- Users (Zep, Supermemory, OpenViking)
DEFINE TABLE users SCHEMAFULL;
DEFINE FIELD id          ON users TYPE string;
DEFINE FIELD email       ON users TYPE string;
DEFINE FIELD metadata    ON users TYPE object;
DEFINE FIELD tenant_id   ON users TYPE string;
DEFINE FIELD created_at  ON users TYPE datetime DEFAULT time::now();
DEFINE FIELD deleted_at  ON users TYPE option<datetime>;

DEFINE INDEX idx_users_tenant ON users FIELDS tenant_id;
DEFINE INDEX idx_users_email  ON users FIELDS email UNIQUE;

-- Profiles (Memobase)
DEFINE TABLE profiles SCHEMAFULL;
DEFINE FIELD id          ON profiles TYPE string;
DEFINE FIELD user_id     ON profiles TYPE string;
DEFINE FIELD topic       ON profiles TYPE string;
DEFINE FIELD sub_topic   ON profiles TYPE string;
DEFINE FIELD content     ON profiles TYPE string;
DEFINE FIELD project_id  ON profiles TYPE string;

-- Sessions/Threads (Zep, OpenViking)
DEFINE TABLE sessions SCHEMAFULL;
DEFINE FIELD id          ON sessions TYPE string;
DEFINE FIELD user_id     ON sessions TYPE string;
DEFINE FIELD ended_at    ON sessions TYPE option<datetime>;
DEFINE FIELD metadata    ON sessions TYPE object;
DEFINE FIELD tenant_id   ON sessions TYPE string;
```

---

## 5. Multi-Tenancy via SurrealDB Namespaces

### 5.1 Isolation Strategy

```
SurrealDB Hierarchy:
  Namespace (NS)  → Cluster-level isolation (e.g., "vnp_memory")
  Database (DB)   → Environment isolation (e.g., "prod", "staging")
  Table scope     → Tenant isolation via tenant_id field + PERMISSIONS
```

### 5.2 Per-Tenant Scoping

```sql
-- Option A: Field-based isolation (simpler, recommended)
-- All tables include tenant_id; queries always filtered
DEFINE TABLE entity_node PERMISSIONS
    FOR select WHERE tenant_id = $auth.tenant_id
    FOR create WHERE tenant_id = $auth.tenant_id
    FOR update WHERE tenant_id = $auth.tenant_id
    FOR delete WHERE tenant_id = $auth.tenant_id;

-- Option B: Separate databases per tenant (stronger isolation)
-- Each tenant gets its own DB within the namespace
USE NS vnp_memory DB tenant_{tenant_id};
```

### 5.3 Adapter Tenant Propagation

```go
// pkg/surrealdb/multi_tenant.go
func (c *Client) WithTenant(ctx context.Context) *Client {
    tenantID := tenant.FromContext(ctx)
    // Clone client with tenant scope in all queries
    return &Client{
        conn: c.conn,
        cfg:  c.cfg,
        tenantFilter: fmt.Sprintf("tenant_id = '%s'", tenantID),
    }
}
```

---

## 6. Hybrid Search with SurrealDB

### 6.1 Vector + Full-text + Graph in One Query

```sql
-- Hybrid search: vector similarity + BM25 full-text + graph traversal
-- This REPLACES separate Qdrant + Neo4j + PostgreSQL queries

LET $query_vec = [0.1, 0.2, ...];  -- from embedding service

-- Step 1: Vector search (replaces Qdrant)
LET $vector_results = (
    SELECT id, name, summary,
           vector::similarity::cosine(embedding, $query_vec) AS vec_score
    FROM entity_node
    WHERE group_id = $group_id
      AND embedding <|10,40|> $query_vec
);

-- Step 2: Full-text search (replaces PostgreSQL FTS)
LET $text_results = (
    SELECT id, name, summary,
           search::score(1) AS text_score
    FROM entity_node
    WHERE name @1@ $query_text
       OR summary @1@ $query_text
);

-- Step 3: RRF merge (replaces Search Hub reranking)
SELECT *,
       search::rrf(vec_score, text_score) AS final_score
FROM (
    SELECT * FROM $vector_results
    UNION
    SELECT * FROM $text_results
)
ORDER BY final_score DESC
LIMIT 20;

-- Step 4: Graph expansion (replaces Neo4j traversal)
SELECT ->relates_to->entity_node.* AS related
FROM entity_node:$top_result_id;
```

---

## 7. Migration Strategy

### 7.1 Deployment Profiles

| Profile | Databases | Use Case | Complexity |
|---------|-----------|----------|-----------|
| **Multi-DB** (default) | PostgreSQL + Neo4j + Qdrant + Redis | Production with specialized DBs | High (4 systems) |
| **SurrealDB Unified** | SurrealDB + Redis | Simplified deployment | Low (2 systems) |
| **Hybrid** | PostgreSQL + SurrealDB + Redis | Gradual migration | Medium (3 systems) |

### 7.2 Migration Path

```
Phase 1: Interface Extraction
  ├── Extract RelationalDB interface from all services
  ├── Move PostgreSQL-specific code to pkg/adapters/reldb/postgres/
  ├── Ensure all graph code uses pkg/adapters/graphdb/ interface
  └── Ensure all vector code uses pkg/adapters/vectordb/ interface

Phase 2: SurrealDB Adapters
  ├── Implement pkg/surrealdb/graph_adapter.go
  ├── Implement pkg/surrealdb/vector_adapter.go
  ├── Implement pkg/surrealdb/relational_adapter.go
  └── Integration tests (pkg/surrealdb/*_test.go)

Phase 3: Wire Integration
  ├── Update Wire providers with backend switch
  ├── Add SurrealDB config to Viper schema
  └── Docker Compose profile: docker-compose.surrealdb.yml

Phase 4: Validation
  ├── Run full test suite with SurrealDB backend
  ├── Performance benchmarks (latency, throughput)
  └── Data migration tool: pg2surreal, neo4j2surreal
```

### 7.3 Docker Compose — SurrealDB Profile

```yaml
# deploy/docker-compose/docker-compose.surrealdb.yml
version: "3.9"

services:
  surrealdb:
    image: surrealdb/surrealdb:v2.2
    command: start --log trace --user root --pass ${SURREAL_PASSWORD}
    ports:
      - "8000:8000"
    volumes:
      - surreal_data:/data
    environment:
      - SURREAL_CAPS_ALLOW_ALL=true
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
      interval: 10s

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  nats:
    image: nats:2.10-alpine
    command: --jetstream --store_dir=/data
    ports:
      - "4222:4222"

volumes:
  surreal_data:
```

---

## 8. Per-Engine Impact Analysis

| Engine | Current DBs | SurrealDB Replaces | Adapter Changes |
|--------|------------|-------------------|-----------------|
| **Cognee** | PostgreSQL + Neo4j + Qdrant | All 3 | `graphdb.GraphDB` + `vectordb.VectorDB` + `reldb.RelationalDB` |
| **Graphiti** | Neo4j + PostgreSQL | Both | `graphdb.GraphDB` (RELATE for edges) + `reldb.RelationalDB` |
| **Memobase** | PostgreSQL + pgvector | Both | `reldb.RelationalDB` + `vectordb.VectorDB` (event embeddings) |
| **OpenViking** | — (VikingFS) | N/A (no DB) | No changes needed |
| **Zep** | PostgreSQL + Neo4j + pgvector | All 3 | `graphdb.GraphDB` + `vectordb.VectorDB` + `reldb.RelationalDB` |
| **Supermemory** | PostgreSQL + pgvector | Both | `reldb.RelationalDB` + `vectordb.VectorDB` |
| **Platform** | PostgreSQL | PostgreSQL only | `reldb.RelationalDB` |
| **Redis** | Redis | ❌ NOT replaced | No changes |

---

## 9. Trade-offs & Risks

### 9.1 Advantages of SurrealDB Mode

| Advantage | Detail |
|-----------|--------|
| **Operational simplicity** | 1 database instead of 3 (PostgreSQL + Neo4j + Qdrant) |
| **Cost reduction** | Single instance, single backup, single monitoring |
| **Unified query language** | SurrealQL for all: relational, graph, vector, full-text |
| **Built-in multi-tenancy** | Namespace/Database isolation native to SurrealDB |
| **Hybrid search native** | Vector + BM25 + graph in single query via `search::rrf()` |
| **Real-time subscriptions** | `LIVE SELECT` for change data capture (CDC) |
| **Simplified deployment** | Docker single container, embedded for dev |

### 9.2 Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|-----------|
| **Maturity** | Medium | SurrealDB 2.x is newer than PostgreSQL/Neo4j; keep multi-DB as default |
| **Vector performance at scale** | Medium | Benchmark HNSW performance vs Qdrant before production switch |
| **Graph traversal depth** | Low | SurrealDB handles multi-hop well; benchmark for 5+ depth |
| **Ecosystem** | Medium | Fewer Go libraries; official Go SDK is stable |
| **Backup/restore** | Medium | SurrealDB export/import vs pg_dump/neo4j-admin; document procedures |
| **Migration effort** | Low | Interface-driven; only adapter layer changes, zero business logic |

### 9.3 Recommendation

> **Default**: Keep multi-DB (PostgreSQL + Neo4j + Qdrant) for production.  
> **SurrealDB**: Use for dev/staging, small deployments, or teams preferring operational simplicity.  
> **Evaluation criteria**: Run comparative benchmarks before production adoption.

---

## 10. File Index

| File | Changes |
|------|---------|
| `specs/00-architecture-overview.md` | Added SurrealDB to tech stack, adapter layer, monorepo |
| `specs/09-shared-packages.md` | Added `reldb/`, `surrealdb/` packages + interfaces |
| `specs/11-surrealdb-integration.md` | This document — complete integration solution |
