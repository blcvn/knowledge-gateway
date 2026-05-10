# VNP Memory — System Architecture (Post-Consolidation)

> **Version**: 2.0 | **Date**: 2026-05-10  
> **Status**: Proposed  
> **Linked**: `docs/adr/ADR-0001-service-consolidation-35-to-18.md`

---

## 1. System Topology

### 18 Domain Services + 1 Gateway

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              VNP Memory v7.0                               │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      vnp-gateway (:8080-8082)                       │   │
│  │                   REST → gRPC · MCP · WebDAV · Auth                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│        │           │           │           │           │           │       │
│  ┌─────┴──┐  ┌─────┴──┐  ┌─────┴──┐  ┌─────┴──┐  ┌─────┴──┐  ┌────┴──┐ │
│  │ Cognee │  │Graphiti│  │Memobase│  │OpenVikng│ │  Zep   │  │  SM   │ │
│  │ (2 svc)│  │ (3 svc)│  │ (2 svc)│  │ (3 svc) │ │ (3 svc)│  │(3 svc)│ │
│  └────────┘  └────────┘  └────────┘  └─────────┘ └────────┘  └───────┘ │
│        │                                                           │       │
│  ┌─────┴───────────────────────────────────────────────────────────┴────┐ │
│  │                    Platform Services (2 svc)                         │ │
│  │                vnp-platform · vnp-search-hub                        │ │
│  └─────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │               Shared Infrastructure                                  │   │
│  │  PostgreSQL+pgvector · Neo4j · Redis · NATS · MinIO · VikingFS     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Service Inventory (18 Services)

| # | Service | gRPC Port | Engine | Domain |
|---|---------|-----------|--------|--------|
| 0 | vnp-gateway | 8080-8082 | — | API Gateway, Auth, MCP |
| 1 | cognee-pipeline | 9011 | Cognee | Ingestion + KG build |
| 2 | cognee-search | 9013 | Cognee | 15 retrieval strategies |
| 3 | graphiti-pipeline | 9021 | Graphiti | Episode pipeline + LLM |
| 4 | graphiti-store | 9024 | Graphiti | Graph DB abstraction |
| 5 | graphiti-search | 9022 | Graphiti | Hybrid search |
| 6 | memobase-pipeline | 9031 | Memobase | Buffer Zone + YOLO engine |
| 7 | memobase-context | 9033 | Memobase | Context assembly |
| 8 | ov-storage | 9051 | OpenViking | FS + Crypto + Resource |
| 9 | ov-search | 9052 | OpenViking | Hierarchical retrieval |
| 10 | ov-session | 9053 | OpenViking | Session lifecycle |
| 11 | zep-core | 9061 | Zep | User + Thread + Memory |
| 12 | zep-graph | 9064 | Zep | KG extraction |
| 13 | zep-search | 9065 | Zep | Semantic search |
| 14 | sm-engine | 9071 | Supermemory | Doc + Memory + Profile |
| 15 | sm-search | 9073 | Supermemory | Hybrid search + RAG |
| 16 | sm-connector | 9075 | Supermemory | External sync |
| 17 | vnp-platform | 9050 | Platform | Admin + Event + Auth |
| 18 | vnp-search-hub | 9042 | Platform | Cross-engine recall |

## 2. Data Flow

### Ingestion Flow (Example: Cognee)

```
Client → Gateway (:8080) → cognee-pipeline (:9011)
    → [LOCAL] Ingest → Cognify (7 stages)
    → PostgreSQL + Neo4j + pgvector
    → [NATS] cognee.pipeline.completed
    → cognee-search (:9013) reindex
```

### Search Flow (Cross-Engine)

```
Client → Gateway (:8080) → vnp-search-hub (:9042)
    → [parallel gRPC] cognee-search + graphiti-search + memobase-context
                     + ov-search + zep-search + sm-search
    → [merge + rerank] → Response
```

## 3. Inter-Service Communication

| Pattern | Usage | Technology |
|---------|-------|-----------|
| Synchronous RPC | CRUD, search, context assembly | gRPC (Protobuf v3) |
| Async Events | Pipeline completion, cascade deletes | NATS JetStream |
| MCP Tools | AI agent tool calls | MCP via Gateway (:8082) |
| WebDAV | File system interop | WebDAV via Gateway (:8083) |

### NATS Event Topology (17 Subjects)

| Stream | Subjects |
|--------|----------|
| cognee | `cognee.pipeline.completed` |
| graphiti | `graphiti.episode.completed` |
| memobase | `memobase.pipeline.completed`, `memobase.profile.changed` |
| openviking | `ov.content.written`, `ov.content.deleted`, `ov.resource.ingested`, `ov.session.*` |
| zep | `zep.memory.messages.ingested`, `zep.user.deleted`, `zep.graph.*`, `zep.search.*` |
| supermemory | `sm.engine.*`, `sm.connector.synced` |
| admin | `admin.tenant.created`, `admin.tenant.deleted` |

## 4. Technology Stack

| Category | Technology | Version |
|----------|-----------|---------|
| Language | Go | 1.23+ |
| RPC | gRPC + Protobuf | v3 |
| Database (relational) | PostgreSQL | 17 |
| Database (vector) | pgvector | 0.7+ |
| Database (graph) | Neo4j | 5+ |
| Cache | Redis | 7+ |
| Message Broker | NATS JetStream | 2.10+ |
| Object Storage | MinIO (S3 compat) | — |
| Custom Storage | VikingFS | (Go-native) |
| LLM Gateway | Bifrost | multi-provider |
| Observability | OpenTelemetry + Prometheus | — |
| Logging | slog (structured) | stdlib |

## 5. Infrastructure Dependencies (Post-Consolidation)

| Backend | Used By | Notes |
|---------|---------|-------|
| PostgreSQL + pgvector | All 18 services | Primary relational + vector |
| Neo4j | cognee-pipeline, graphiti-pipeline/store, zep-graph | Graph-native |
| Redis | All services | Cache + rate limit + state |
| NATS JetStream | All pipeline + search services | Async events |
| MinIO/S3 | cognee-pipeline, sm-connector | Object storage |
| VikingFS | ov-storage | Go-native FS (unique) |

> **Removed**: Qdrant (migrated to pgvector)

## 6. Deployment Architecture

### Development (Docker Compose)

- 18 service containers + 1 gateway
- 4 infrastructure containers (PostgreSQL, Neo4j, Redis, NATS)
- 1 optional (MinIO)
- **Total**: ~24 containers (down from ~41)

### Production (Kubernetes)

- 1 Deployment per service (18 + gateway)
- HPA on search services (auto-scale on CPU)
- StatefulSet for PostgreSQL, Neo4j
- PVC for VikingFS persistent storage
