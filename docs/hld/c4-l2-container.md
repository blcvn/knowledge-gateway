# C4 Level 2 — Container Diagram

> **C4 Level**: 2 — Container  
> **Câu hỏi**: KG Service gồm những process/store nào? Chạy trên technology gì? Communicate thế nào?  
> **Audience**: Technical team, architects, DevOps

---

## Container Diagram

```
╔══════════════════════════════════════════════════════════════════════════════════╗
║  SYSTEM: KG SERVICE                                                              ║
║                                                                                  ║
║  ┌──────────────────────────────────────────────────────────────────────────┐    ║
║  │  { KG API Server }                                                       │    ║
║  │  Go 1.25 · HTTP server · Port 8082 · Stateless (scale-out)               │    ║
║  │                                                                           │    ║
║  │  Exposes:                                                                  │    ║
║  │    REST API  → /v1/tenants, /v1/access, /v1/ontology, /v1/kg             │    ║
║  │    MCP/SSE  → /v1/mcp/connect, /v1/mcp/messages/{session_id}             │    ║
║  │    Health   → /healthz                                                    │    ║
║  └──────────────────────────────────────────────────────────────────────────┘    ║
║        │  pgx/v5              │  graph driver        │  vector SDK               ║
║        ▼                      ▼                       ▼                           ║
║  ┌──────────────┐  ┌──────────────────────┐  ┌───────────────────────────┐       ║
║  │ { PostgreSQL }│  │ { Graph DB }          │  │ { Vector DB }             │       ║
║  │ v15 · :5432  │  │ Memory (dev)          │  │ Memory (dev)              │       ║
║  │              │  │ Neo4j (:7687)         │  │ Qdrant (:6333)            │       ║
║  │ Source of    │  │ Memgraph (:7687)      │  │ Milvus (:19530)           │       ║
║  │ truth · RLS  │  │ Nebula (:9669)        │  │ pgvector (via PG)         │       ║
║  │              │  │                       │  │                           │       ║
║  │ Schema (DDL):│  │ Nodes + Rels          │  │ Collection: kg_vectors    │       ║
║  │  tenants     │  │ acl_visible_to[]      │  │ vector_size: 1024         │       ║
║  │  apps        │  │ _kg_sync_version      │  │ payload: acl + status     │       ║
║  │  access_grants│ │ (read replica)        │  │ (read replica)            │       ║
║  │  domains     │  └──────────────────────┘  └───────────────────────────┘       ║
║  │  kg_nodes    │                                                                  ║
║  │  kg_outbox   │        ┌──────────────────────────────┐                         ║
║  └──────────────┘        │ { Redis }                    │                         ║
║        │                 │ :6379 · In-memory cache       │                         ║
║        │  outbox poll    │                               │                         ║
║        ▼                 │ apikey:{hash} → TTL 30s       │                         ║
║  ┌──────────────┐        │ acl:{t}:{a}  → TTL 60s       │                         ║
║  │ { Sync       │        │ rate_limit:{t} → TTL 1m       │                         ║
║  │   Workers }  │◀──────▶│                               │                         ║
║  │ (embedded in │        └──────────────────────────────┘                         ║
║  │  API server) │                                                                  ║
║  │              │  ┌──────────────────────────────┐                               ║
║  │  GraphSync   │  │ { Embedding Provider }        │                               ║
║  │  VectorSync  │─▶│ HTTP endpoint (external)      │                               ║
║  │  AccessSync  │  │ EMBEDDING_PROVIDER=http        │                               ║
║  │  Recon       │  │ or =deterministic (dev/test)   │                               ║
║  └──────────────┘  └──────────────────────────────┘                               ║
║                                                                                  ║
╚══════════════════════════════════════════════════════════════════════════════════╝

 External consumers:
  [ Platform Admin ]  ─────────────────────▶  REST API (:8082)
  [ Tenant Admin ]    ─────────────────────▶  REST API (:8082)
  [ App Integrator ]  ─────────────────────▶  REST API (:8082)
  [ AI Agent ]        ──── MCP/SSE ─────────▶  REST API (:8082)
  [ Ingestion Pipeline] ───────────────────▶  REST API (:8082)
```

---

## Containers Detail

### { KG API Server }

| Attribute | Value |
|:---|:---|
| **Technology** | Go 1.25, `net/http` + Gorilla Mux |
| **Port** | 8082 |
| **Protocol** | HTTP/1.1 (REST JSON + MCP SSE) |
| **Scaling** | Stateless — horizontal scale-out |
| **Dockerfile** | `Dockerfile` (multi-stage build) |
| **Make targets** | `make run`, `make test`, `make build` |
| **Config** | Environment variables (xem [environment.md](../../deployment/environment.md)) |

**Responsibilities**:
- HTTP routing và request handling
- Authentication middleware (API key → tenant/app)
- Identity injection vào mọi downstream call
- Expose REST endpoints và MCP SSE transport
- Chạy embedded sync workers trong goroutines
- Health check endpoint (public)

---

### { PostgreSQL }

| Attribute | Value |
|:---|:---|
| **Technology** | PostgreSQL 15 |
| **Port** | 5432 |
| **Driver** | `jackc/pgx/v5` (connection pool) |
| **Role** | **Source of truth** — mọi write land here first |
| **Features** | Row Level Security (RLS), JSONB, partitioned tables |
| **Migrations** | 15 migration files trong `migrations/` |

**Schema groups**:
```
Identity & Access:   tenants, apps, access_grants, access_audit_log
Ontology:            domains, node_type_schemas, rel_type_schemas,
                     domain_query_templates, domain_status_field_configs,
                     cross_domain_rel_rules, ontology_versions
KG Data:             kg_nodes (RLS), kg_relationships (RLS),
                     kg_outbox_events, kg_projection_versions,
                     kg_vector_documents
```

---

### { Graph DB }

| Attribute | Value |
|:---|:---|
| **Technology options** | Memory (test) / Neo4j (:7687) / Memgraph (:7687) / Nebula (:9669) |
| **Selected via** | `GRAPH_ADAPTER` env + `KG_RUNTIME_PROFILE` |
| **Protocol** | Bolt (Neo4j/Memgraph) / Nebula protocol |
| **Drivers** | `neo4j-go-driver/v6`, `nebula-go/v5`, in-memory adapter |
| **Role** | **Read replica** — graph traversal, pattern queries |
| **Write mode** | Async sync từ outbox (NEVER direct write từ API handler) |

**Data stored**:
```
Nodes:
  - All properties from kg_nodes.properties
  - + acl_visible_to[]      (denormalized ACL tokens)
  - + owner_tenant_id, owner_app_id, domain_id
  - + _kg_sync_version      (consistency tracking)
  - + status_value          (from domain_status_field_configs)

Relationships:
  - rel_type, from_id, to_id
  - + _kg_sync_version
```

---

### { Vector DB }

| Attribute | Value |
|:---|:---|
| **Technology options** | Memory (test) / Qdrant (:6333) / Milvus (:19530) / pgvector |
| **Selected via** | `VECTOR_ADAPTER` env + `KG_RUNTIME_PROFILE` |
| **SDK** | Qdrant gRPC, `milvus-sdk-go/v2`, pgvector via pgx |
| **Role** | **Read replica** — semantic search, RAG retrieval |
| **Write mode** | Async sync từ outbox (NEVER direct write từ API handler) |

**Collection**: `kg_vectors`
```
vector_size: 1024
distance: Cosine
hnsw: { m: 16, ef_construct: 200 }
payload: { node_id, node_type, domain_id, owner_tenant_id, owner_app_id,
           acl_visible_to[], is_deleted, status_value, authority_score,
           _kg_sync_version, domain_props: {} }
```

---

### { Redis }

| Attribute | Value |
|:---|:---|
| **Technology** | Redis 7.x |
| **Port** | 6379 |
| **Role** | Cache + Rate Limiting |
| **Client** | `rediscache` package (internal) |

**Key space**:

| Key Pattern | Value | TTL | Invalidation |
|:---|:---|:---|:---|
| `apikey:{sha256}` | `{tenant_id, app_id}` | 30s | Instant on app revoke |
| `acl:{tenant_id}:{app_id}` | visible_owners set | 60s | Instant on grant change |
| `ontology:effective:{t}:{a}` | domain list | 300s | On domain create/share |
| `domain_schema:{id}:{ver}` | schema object | ∞ | Immutable per version |
| `rate:{tenant_id}:{window}` | request count | 60s | Auto-expire |

---

### { Sync Workers }

| Attribute | Value |
|:---|:---|
| **Technology** | Go goroutines — embedded in API server process |
| **Trigger** | Polling `kg_outbox_events` WHERE status = 'PENDING' |
| **Retry** | Up to 5 attempts → `DEAD_LETTER` status |
| **Idempotency** | MERGE (Cypher), upsert (Qdrant) by stable node_id |

**Workers**:

| Worker | Trigger events | Action |
|:---|:---|:---|
| **GraphSyncWorker** | `NODE_UPSERTED`, `NODE_DELETED`, `REL_UPSERTED`, `STATUS_VALUE_CHANGED` | Merge/delete node/rel in Graph DB + update `acl_visible_to` |
| **VectorSyncWorker** | `NODE_UPSERTED`, `NODE_DELETED` | Embed text → upsert in Vector DB với full payload |
| **AccessSyncWorker** | `ACCESS_GRANT_CHANGED` | Recompute `acl_visible_to` cho tất cả nodes affected by grant scope |
| **ReconciliationWorker** | Scheduled (hourly) | Compare counts + versions PG ↔ Graph ↔ Vector; report drift |

---

### { Embedding Provider }

| Attribute | Value |
|:---|:---|
| **Mode `deterministic`** | Fake embedding (hash-based) — dev/test, zero cost |
| **Mode `http`** | Real embedding via HTTP API (OpenAI-compatible) |
| **Config** | `EMBEDDING_URL`, `EMBEDDING_MODEL`, `EMBEDDING_API_KEY` |
| **Cache** | `EMBEDDING_CACHE_TTL_S` — avoid re-embedding same text |

---

## Runtime Profiles

KG Service chọn backend combination qua `KG_RUNTIME_PROFILE`:

| Profile | Graph Backend | Vector Backend | FTS Backend | Use case |
|:---|:---:|:---:|:---:|:---|
| (none / memory) | Memory | Memory | Memory | Local dev, unit tests |
| `pgvector-memgraph` | Memgraph | pgvector | PostgreSQL | Lightweight staging |
| `qdrant-memgraph` | Memgraph | Qdrant | PostgreSQL | CodeGraph production |
| `qdrant-nebula` | Nebula | Qdrant | PostgreSQL | High-scale graph |
| `neo4j-*` | Neo4j | any | PostgreSQL | Enterprise graph |

---

## Communication Protocols

| From → To | Protocol | Sync/Async | Notes |
|:---|:---|:---|:---|
| Client → API Server | HTTP/1.1 REST JSON | Synchronous | Bearer token auth |
| AI Agent → API Server | HTTP/1.1 + SSE (MCP) | Session-based | Connection-level auth |
| API Server → PostgreSQL | pgx wire protocol | Synchronous | pgx/v5 connection pool |
| API Server → Graph DB | Bolt / Nebula protocol | Synchronous (read), Async (write via worker) | Timeout 3000ms |
| API Server → Vector DB | gRPC / HTTP | Synchronous (read), Async (write via worker) | — |
| API Server → Redis | RESP protocol | Synchronous | pipelining for batch ops |
| Sync Workers → Graph DB | Bolt / Nebula | Async (background goroutine) | Outbox polling loop |
| Sync Workers → Vector DB | gRPC / HTTP | Async | Outbox polling loop |
| Sync Workers → Embedding | HTTP/1.1 | Synchronous | Called during VectorSync |

---

## Deployment Topology

### Phase 1 — Docker Compose (Bootstrap)

```
┌─────────────────────────────────────────────────────────────────┐
│  docker-compose                                                  │
│                                                                  │
│  ┌─────────────────┐    ┌──────────────┐    ┌────────────────┐  │
│  │  migrate (job)  │───▶│  kg-service  │───▶│  postgres:5432 │  │
│  │  (one-shot)     │    │  :8082       │    │                │  │
│  └─────────────────┘    └──────┬───────┘    └────────────────┘  │
│                                │                                 │
│                    ┌───────────┼───────────┐                    │
│                    ▼           ▼           ▼                    │
│              ┌──────────┐ ┌────────┐ ┌─────────┐               │
│              │ redis    │ │ graph  │ │ vector  │               │
│              │ :6379    │ │ db     │ │ db      │               │
│              └──────────┘ └────────┘ └─────────┘               │
└─────────────────────────────────────────────────────────────────┘
```

### Phase 2 — Kubernetes (Production)

```
┌─────────────────────────────────────────────────────────────────┐
│  Kubernetes namespace: kg-service                                │
│                                                                  │
│  [Ingress / LB]                                                  │
│       │                                                          │
│  ┌────▼───────────────────────┐                                 │
│  │  Deployment: kg-service    │                                 │
│  │  replicas: 3               │   ← stateless, rolling update   │
│  │  resources: 500m/256Mi     │                                 │
│  └────────────────────────────┘                                 │
│                                                                  │
│  StatefulSet: postgres-primary + 2 replicas                     │
│  StatefulSet: memgraph (or neo4j)                               │
│  StatefulSet: qdrant (2 shards, replication 2)                  │
│  StatefulSet: redis-sentinel (3 nodes)                          │
└─────────────────────────────────────────────────────────────────┘
```
