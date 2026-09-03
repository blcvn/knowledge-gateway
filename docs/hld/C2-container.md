# C2 — Container Diagram

> **C4 Level 2:** Các runtime process và data stores trong VNP Memory.
> Container = một đơn vị có thể deploy độc lập (process, DB, message broker...).

---

## Diagram — Monolith Mode (Development)

```mermaid
C4Container
    title Container Diagram — VNP Memory (Monolith Mode)

    Person(developer, "AI Agent Developer / IDE User")
    Person(ops, "Platform Engineer / Admin")

    System_Boundary(vnp, "VNP Memory — Single Binary (backend/apps/memory)") {

        Container(gateway, "API Gateway", "Go — net/http",
            "Entry point: REST :8080, MCP :8082 (embedded), Health :8083\nAuth middleware, rate limiting, routing\nMCP Server (22 tools, SSE + HTTP Streamable)\nInProcessRegistry lookup")

        Container(search_hub, "vnp-search-hub", "Go — gRPC service",
            "Cross-engine parallel search fan-out\nBM25 + Vector + RRF fusion\nMerge + rerank results")

        Container(memory_engines, "Memory Engines (6)", "Go — gRPC in-process",
            "cognee-ingestion/search/cognify\ngraphiti-ingestion/search/knowledge\nmemobase-ingestion/engine/context\nov-fs/search/session\nsm-memory/search/document\nzep-memory/graph/search")

        Container(agent_memory, "AgentMemory Layer (3)", "Go — gRPC in-process",
            "observe-service (12 HookTypes, 13-step pipeline)\norchestration-service (leases, signals, DAG)\npipeline-service (4-tier consolidation)")

        Container(platform, "Platform Services", "Go — gRPC in-process",
            "vnp-platform (admin, tenants, API keys)\nmemory-service (unified memory adapter)\nvnp-event (timeline, UserEvent)\nobs-service (metrics, traces)")

        Container(nats, "NATS JetStream", "Embedded NATS",
            "Message broker\nStreams: memory.*, agent.*, consolidation.*\nGuaranteed delivery (at-least-once)")
    }

    ContainerDb(postgres, "PostgreSQL 17 + pgvector", "RDBMS + Vector",
        "Primary data store\nAll domain entities with TenantID\nVector embeddings via pgvector\nPort: 5432")

    ContainerDb(neo4j, "Neo4j 5+", "Graph Database",
        "Knowledge graphs:\nCognee, Graphiti, Zep graph entities\nPorts: 7474 (HTTP), 7687 (Bolt)")

    ContainerDb(redis, "Redis 7+", "Cache / Rate Limit",
        "Session cache, rate limiting\nDedupMap (30s TTL)\nPort: 6379")

    ContainerDb(minio, "MinIO", "Object Storage",
        "OpenViking file storage (VikingFS)\nS3-compatible API\nPort: 9000")

    ContainerDb(qdrant, "Qdrant (optional)", "Vector Database",
        "High-scale vector search\nOptional: pgvector used by default\nPort: 6333")

    System_Ext(llm, "LLM Providers", "via Bifrost router")

    Rel(developer, gateway, "REST API calls\nPOST /v1/memory/store", "HTTPS :8080")
    Rel(developer, gateway, "MCP tool calls\nmemory_store, ov_grep", "SSE/HTTP :8082")
    Rel(ops, gateway, "Admin APIs\nPOST /v1/admin/tenants", "HTTPS :8080")

    Rel(gateway, memory_engines, "Route to engine\nbased on memory type", "bufconn gRPC")
    Rel(gateway, search_hub, "Cross-engine recall\nPOST /v1/memory/recall", "bufconn gRPC")
    Rel(gateway, agent_memory, "Observe sessions\nOrchestration calls", "bufconn gRPC")
    Rel(gateway, platform, "Admin, auth, events", "bufconn gRPC")

    Rel(memory_engines, postgres, "R/W domain data\nEntities, embeddings", "SQL + pgvector")
    Rel(memory_engines, neo4j, "R/W knowledge graphs\nCognee, Graphiti, Zep", "Bolt")
    Rel(memory_engines, minio, "R/W files\nVikingFS blobs", "S3 API")

    Rel(agent_memory, postgres, "R/W observations\nsession_summaries, procedures", "SQL")
    Rel(agent_memory, redis, "Dedup cache\nDedupMap TTL 30s", "Redis commands")

    Rel(search_hub, postgres, "Vector similarity\npgvector cosine", "SQL")
    Rel(search_hub, neo4j, "Graph traversal\nCypher queries", "Bolt")

    Rel(nats, memory_engines, "Events: memory.blob.inserted\nConsolidation triggers", "NATS JetStream")
    Rel(nats, agent_memory, "Hook events\nSession lifecycle", "NATS JetStream")

    Rel(memory_engines, llm, "LLM calls:\nextract, cognify, classify", "HTTPS Bifrost")
    Rel(agent_memory, llm, "Consolidation LLM calls\nCompress, summarize", "HTTPS Bifrost")
```

---

## Diagram — Distributed Mode (Production)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Production Deployment                             │
│                                                                     │
│  ┌──────────────┐                                                   │
│  │  API Gateway │ REST :8080 / MCP :8082 (embedded) / Health :8083 │
│  │  (standalone)│ → JWT/API Key auth → rate limit → route          │
│  └──────┬───────┘                                                   │
│         │ gRPC (network)                                            │
│  ┌──────▼────────────────────────────────────────────────┐         │
│  │              Engine Service Pods (Kubernetes)          │         │
│  │  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌────────┐  │         │
│  │  │  Cognee  │ │ Graphiti │ │  Memobase │ │  Zep   │  │         │
│  │  │  cluster │ │  cluster │ │  cluster  │ │cluster │  │         │
│  │  └──────────┘ └──────────┘ └───────────┘ └────────┘  │         │
│  │  ┌──────────────────┐ ┌────────────────────────────┐  │         │
│  │  │ OpenViking       │ │ Supermemory                │  │         │
│  │  │ cluster          │ │ cluster                    │  │         │
│  │  └──────────────────┘ └────────────────────────────┘  │         │
│  └───────────────────────────────────────────────────────┘         │
│         │                                                           │
│  ┌──────▼────────────────────┐                                      │
│  │  Shared Infrastructure    │                                      │
│  │  PostgreSQL (HA)          │ Neo4j (cluster)                      │
│  │  Redis (sentinel)         │ NATS (external cluster)              │
│  │  MinIO (distributed)      │ (pgvector — no Qdrant)              │
│  └───────────────────────────┘                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Switch:** `VNP_MEMORY_NATS_MODE=external` → dùng external NATS cluster.

---

## Container Summary

| Container | Technology | Responsibility | Ports |
|---|---|---|---|
| API Gateway | Go net/http | Auth, routing, rate limiting, MCP (embedded) | :8080, :8082, :8083 |
| vnp-search-hub | Go gRPC | Cross-engine search, RRF fusion | in-process |
| Memory Engines (×6) | Go gRPC | Domain-specific memory operations | in-process / network |
| AgentMemory Layer (×3) | Go gRPC | Observe, orchestration, consolidation | in-process / network |
| Platform Services | Go gRPC | Admin, events, memory adapter, observability | in-process / network |
| NATS JetStream | NATS server | Async event bus, guaranteed delivery | embedded / 4222 |
| PostgreSQL 17 | RDBMS + pgvector | Primary data + vector embeddings | 5432 |
| Neo4j 5+ | Graph DB | Knowledge graphs (Cognee, Graphiti, Zep) | 7474/7687 |
| Redis 7+ | Cache | Session cache, rate limiting, dedup TTL | 6379 |
| MinIO | Object storage | VikingFS files (OpenViking) | 9000 |

> **Note**: Qdrant has been removed from the stack. pgvector (PostgreSQL extension) is the production vector store. See ADR-001.

---

*[← C1 Context](./C1-context.md) | [→ C3 Component](./C3-component.md)*
