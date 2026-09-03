# VNP Memory — Unified Enterprise Architecture

> **Version**: 1.2 | **Date**: 2026-09-03 | **Status**: Active (synced from code)  
> **Stack**: Go 1.25+ · gRPC · NATS JetStream · PostgreSQL+pgvector · Neo4j · Redis · MinIO  
> **Architecture**: Gateway + Domain Services, Clean Architecture per service
> **Module prefix**: `vnp-memory/` (services) | `github.com/vnp-community/vnp-memory/` (gateway)

---

## 1. Executive Summary

Hợp nhất **Cognee** (semantic KG, RAG), **Graphiti** (temporal episodic KG), **Memobase** (user profile memory), **OpenViking** (agent-native context DB), **Supermemory** (adaptive KG memory), **Zep** (context engineering) vào **single Go monorepo** — `vnp-memory` — với:

- **1 Unified API Gateway** — single entry point, REST/gRPC/MCP/WebSocket/WebDAV
- **35 Domain Services** — mỗi service 1 bounded context, gRPC internal
- **4-layer Clean Architecture** per service — domain → usecase → adapter → infra
- **Production-grade** — observability, resilience, multi-tenancy, envelope encryption, horizontal scaling

### Design Principles

| # | Principle | Rationale |
|---|-----------|-----------|
| 1 | **Single Monorepo** | Shared proto, shared `pkg/`, single `go.mod`, unified CI |
| 2 | **Unified Gateway** | One entry point for ALL memory APIs + MCP |
| 3 | **Clean Architecture** | 4 layers per service, strict dependency rule |
| 4 | **gRPC internal, REST external** | Type-safe inter-service, developer-friendly external |
| 5 | **NATS JetStream async** | Pipeline orchestration, event-driven processing |
| 6 | **Shared `pkg/` — NO business logic** | Only types, interfaces, middleware, adapters |
| 7 | **Multi-Tenant by Design** | tenant_id/project_id + namespace isolation |
| 8 | **Cold-path LLM** | Buffer batching, async pipeline, avoid hot-path LLM |

---

## 2. System Context

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                           External Consumers                                  │
│  Web UI · CLI · SDKs(Go/Py/TS) · MCP Clients · AI Agents · Chat Apps        │
└───────────────────────────────┬────────────────────────────────────────────────┘
                                │ REST / gRPC-Web / MCP(SSE) / WebSocket
                                ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                     UNIFIED API GATEWAY  (vnp-gateway)                         │
│  Auth(JWT/APIKey) · RateLimit · CORS · Protocol Translation · Routing        │
│  Circuit Breaker · Request Validation · Tenant Resolution · MCP Server       │
└──┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬───┬──────────────────────┘
   │   │   │   │   │   │   │   │   │   │   │   │   │   │
   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼   ▼
┌─── COGNEE ────┐ ┌── GRAPHITI ──┐ ┌── MEMOBASE ───┐ ┌── PLATFORM ──┐
│Ingst│Cgnf│Srch│ │Ingst│Srch│Knw│ │Ingst│Engn│Ctxt│ │Evnt│Admn│Srch│
│ ion │ ify │    │ │ ion │    │ldg│ │ ion │ ine│    │ │    │    │ Hub│
│ Svc │ Svc│Svc │ │ Svc │Svc │Svc│ │ Svc │ Svc│Svc │ │ Svc│ Svc│ Svc│
└─────┴────┴────┘ └─────┴────┴───┘ └─────┴────┴────┘ └────┴────┴────┘
                                │
                ┌───────────────▼───────────────────┐
                │     SHARED INFRASTRUCTURE          │
                │  PostgreSQL+pgvector · Neo4j       │
                │  Redis · NATS JetStream · MinIO    │
                │  Bifrost(LLM router) · OTel        │
                └───────────────────────────────────┘
```

---

## 3. Service Inventory

### 3.1 Complete Service Map (35 services + Gateway)

| # | Service | gRPC Port | Health | Domain | Origin |
|---|---------|----------|--------|--------|--------|
| 0 | `vnp-gateway` | 8081 | 8083 | Routing, Auth, MCP, WebDAV | Unified |
| | **Cognee (Semantic KG)** | | | | |
| 1 | `cognee-ingestion` | 9011 | 9091 | Data pipeline, file extract | Cognee |
| 2 | `cognee-cognify` | 9012 | 9092 | KG build, chunking, ontology | Cognee |
| 3 | `cognee-search` | 9013 | 9093 | 15 retriever strategies, RAG | Cognee |
| | **Graphiti (Episodic KG)** | | | | |
| 4 | `graphiti-ingestion` | 9021 | 9094 | Episode pipeline, saga | Graphiti |
| 5 | `graphiti-search` | 9022 | 9095 | Hybrid search, reranking | Graphiti |
| 6 | `graphiti-knowledge` | 9023 | 9096 | LLM extraction, resolution | Graphiti |
| 7 | `graphiti-store` | 9024 | 9097 | Graph DB abstraction, CRUD | Graphiti |
| | **Memobase (Profile Memory)** | | | | |
| 8 | `memobase-ingestion` | 9031 | 9098 | Blob insert, Buffer Zone, Flush | Memobase |
| 9 | `memobase-engine` | 9032 | 9099 | Profile extraction, YOLO merge | Memobase |
| 10 | `memobase-context` | 9033 | 9100 | Context assembly < 100ms | Memobase |
| | **OpenViking (Procedural Context)** | | | | |
| 11 | `ov-fs` | 9051 | 9104 | File CRUD, Tree, Grep, Glob | OpenViking |
| 12 | `ov-search` | 9052 | 9105 | Hierarchical retrieval, Hotness | OpenViking |
| 13 | `ov-session` | 9053 | 9106 | 2-phase commit, WM v2, Memory | OpenViking |
| 14 | `ov-resource` | 9054 | 9107 | Ingestion pipeline, Parse, Watch | OpenViking |
| 15 | `ov-crypto` | 9055 | 9108 | Envelope encryption, KMS, Rotate | OpenViking |
| 16 | `ov-admin` | 9056 | 9109 | Account/User/Key CRUD | OpenViking |
| | **Zep (Context Engineering)** | | | | |
| 17 | `zep-user` | 9061 | 9110 | User CRUD, metadata, project isolation | Zep |
| 18 | `zep-thread` | 9062 | 9111 | Thread/session lifecycle, ended_at | Zep |
| 19 | `zep-memory` | 9063 | 9112 | Message ingestion, context assembly | Zep |
| 20 | `zep-graph` | 9064 | 9113 | KG extraction, temporal reasoning | Zep |
| 21 | `zep-search` | 9065 | 9114 | Semantic search, 5 reranking strategies | Zep |
| 22 | `zep-admin` | 9066 | 9115 | Health aggregation, project mgmt | Zep |
| | **Supermemory (Adaptive KG)** | | | | |
| 23 | `sm-document` | 9071 | 9116 | Document CRUD, ingestion pipeline | Supermemory |
| 24 | `sm-memory` | 9072 | 9117 | Memory engine, KG, forgetting curve | Supermemory |
| 25 | `sm-search` | 9073 | 9118 | Hybrid search, RAG, reranking | Supermemory |
| 26 | `sm-profile` | 9074 | 9119 | User profiles (static + dynamic) | Supermemory |
| 27 | `sm-connector` | 9075 | 9120 | External sync (GDrive, Notion) | Supermemory |
| 28 | `sm-mcp` | 9076 | 9121 | MCP server (SSE/JSON-RPC) | Supermemory |
| 29 | `sm-auth` | 9077 | 9122 | Auth, API keys, RBAC, orgs | Supermemory |
| 30 | `sm-analytics` | 9078 | 9123 | Usage tracking, token economics | Supermemory |
| 31 | `sm-project` | 9079 | 9124 | Spaces, container tags, membership | Supermemory |
| | **Platform (Cross-Engine)** | | | | |
| 32 | `vnp-event` | 9041 | 9101 | Event timeline, semantic search | Cross-domain |
| 33 | `vnp-search-hub` | 9042 | 9102 | Cross-engine search orchestration | Unified |
| 34 | `vnp-admin` | 9050 | 9103 | Users, tenants, billing, health | Shared |
| | **AgentMemory Layer** | | | | |
| 35 | `observe-service` | 9081 | 9131 | Hook capture, 14-step pipeline, SSE | AgentMemory |
| 36 | `orchestration-service` | 9082 | 9132 | Distributed leases, agent signals | AgentMemory |
| 37 | `pipeline-service` | 9083 | 9133 | 4-tier consolidation pipeline | AgentMemory |
| 38 | `obs-service` | 9084 | 9134 | Observability metrics, infra topology | Platform |
| 39 | `search-service` | 9085 | 9135 | Cross-engine search hub (unified) | Platform |
| 40 | `storage-service` | 9086 | 9136 | Object storage + OV crypto adapter | Platform |
| 41 | `memory-service` | 9087 | 9137 | Unified memory gateway adapter | Platform |
| 42 | `kg-service` | 9088 | 9138 | Knowledge graph orchestrator | Platform |

### 3.2 Service Grouping

| Group | Services | Count | Shared Concerns |
|-------|----------|-------|----------------|
| **Cognee (Semantic)** | cognee-ingestion, cognee-cognify, cognee-search | 3 | Knowledge graph, RAG, multi-modal |
| **Graphiti (Episodic)** | graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store | 4 | Temporal graph, bi-temporal model |
| **Memobase (Profile)** | memobase-ingestion, memobase-engine, memobase-context | 3 | User profiles, buffer zone, events |
| **OpenViking (Procedural)** | ov-fs, ov-search, ov-session, ov-resource, ov-crypto, ov-admin | 6 | VikingFS, tiered context, L0/L1/L2 tiers, encryption |
| **Zep (Context)** | zep-user, zep-thread, zep-memory, zep-graph, zep-search, zep-admin | 6 | Context engineering, temporal KG, sub-200ms |
| **Supermemory (Adaptive)** | sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-mcp, sm-auth, sm-analytics, sm-project | 9 | Adaptive KG, forgetting curve, connectors |
| **Platform** | vnp-event, vnp-search-hub, vnp-admin, kg-service | 4 | Cross-engine orchestration |
| **AgentMemory** | observe-service, memory-service, orchestration-service, pipeline-service | 4 | Hook capture, memory lifecycle, multi-agent leases, consolidation |
| **Infrastructure** | obs-service, search-service, storage-service | 3 | Observability, cross-engine search, unified storage adapter |

### 3.3 What Is Shared vs Separate

| Component | Strategy | Rationale |
|-----------|----------|-----------|
| **Gateway** | Unified | Single entry point, single auth/rate-limit |
| **Admin** | Unified | Users, tenants, API keys, health — same for all |
| **Search Hub** | Unified | Cross-engine recall, result merging, reranking |
| **Event** | Unified | Temporal events shared across engines |
| **`pkg/`** | Unified | Proto, middleware, resilience, observability |
| **Domain services** | Separate | Each engine has own business logic |
| **Infrastructure** | Shared | Same DB instances, separate schemas/namespaces |

---

## 4. Technology Stack

| Layer | Technology | Used By |
|-------|-----------|---------|
| **Language** | Go 1.25+ | All |
| **External API** | net/http (stdlib) + OpenAPI 3.1 | Gateway |
| **MCP Server** | JSON-RPC 2.0 (SSE + HTTP Streamable) | Gateway (MCP adapter, 37+ tools target) |
| **Internal RPC** | gRPC + Protobuf v3 | All services |
| **Async events** | NATS JetStream (embedded dev / external prod) | All services |
| **Relational DB** | PostgreSQL 17 | Admin, Memobase, observe-service |
| **Vector DB** | pgvector (PostgreSQL extension) | Cognee search, Memobase events, RRF hybrid |
| **Graph DB** | Neo4j 5.x (primary) | Cognee KG + Graphiti episodic |
| **Object Storage** | MinIO / S3-compatible | Cognee ingestion, storage-service |
| **Cache** | Redis | Rate limiting, session cache |
| **LLM Router** | Bifrost (self-hosted proxy) | All LLM calls (OpenAI, Anthropic, Ollama) |
| **Observability** | OpenTelemetry + Prometheus | All services |
| **Filesystem** | Go-native VikingFS | OpenViking file operations |
| **LLM Gateway** | Bifrost | All LLM-dependent services |
| **Encryption** | AES-256-GCM envelope | OpenViking per-file encryption |
| **KMS** | Local / Vault / Cloud KMS | OpenViking key management |
| **DI** | Google Wire | All services |
| **Observability** | OTel + Prometheus + Jaeger + slog | All services |
| **Config** | Viper + ENV | All services |
| **Tokenizer** | tiktoken-go | Memobase engine |

---

## 5. Monorepo Structure

```
vnp-memory/
├── api/proto/                          # ALL Protobuf definitions
│   ├── common/v1/                      #   Shared: pagination, temporal, errors, health
│   ├── graph/v1/                       #   Shared: node.proto, edge.proto
│   ├── gateway/v1/                     #   Gateway-specific
│   ├── cognee/                         #   Cognee domain protos
│   │   ├── ingestion/v1/
│   │   ├── cognify/v1/
│   │   └── search/v1/
│   ├── graphiti/                       #   Graphiti domain protos
│   │   ├── ingestion/v1/
│   │   ├── search/v1/
│   │   ├── knowledge/v1/
│   │   └── store/v1/
│   ├── memobase/                       #   Memobase domain protos
│   │   ├── ingestion/v1/
│   │   ├── engine/v1/
│   │   └── context/v1/
│   ├── openviking/                     #   OpenViking domain protos
│   │   ├── fs/v1/
│   │   ├── search/v1/
│   │   ├── session/v1/
│   │   ├── resource/v1/
│   │   └── crypto/v1/
│   ├── event/v1/                       #   Cross-domain events
│   ├── searchhub/v1/                   #   Cross-engine search
│   └── admin/v1/                       #   Shared admin
│
├── services/                           # Service binaries
│   ├── vnp-gateway/                    #   Unified API Gateway
│   ├── cognee-ingestion/               #   Cognee: data pipeline
│   ├── cognee-cognify/                 #   Cognee: KG construction
│   ├── cognee-search/                  #   Cognee: 15 retrievers
│   ├── graphiti-ingestion/             #   Graphiti: episode pipeline
│   ├── graphiti-search/                #   Graphiti: hybrid search
│   ├── graphiti-knowledge/             #   Graphiti: LLM processing
│   ├── graphiti-store/                 #   Graphiti: graph DB abstraction
│   ├── memobase-ingestion/             #   Memobase: blob + buffer zone
│   ├── memobase-engine/                #   Memobase: profile extraction
│   ├── memobase-context/               #   Memobase: context assembly
│   ├── ov-fs/                          #   OpenViking: file CRUD, tree
│   ├── ov-search/                      #   OpenViking: hierarchical retrieval
│   ├── ov-session/                     #   OpenViking: session, WM v2
│   ├── ov-resource/                    #   OpenViking: ingestion pipeline
│   ├── ov-crypto/                      #   OpenViking: envelope encryption
│   ├── ov-admin/                       #   OpenViking: account/user CRUD
│   ├── vnp-event/                      #   Cross-domain: event timeline
│   ├── vnp-search-hub/                 #   Cross-domain: unified search
│   └── vnp-admin/                      #   Shared: users, tenants
│
├── pkg/                                # Shared packages (NO business logic)
│   ├── graph/                          #   Shared graph domain types
│   │   ├── node.go                     #     EntityNode, EpisodicNode, CommunityNode
│   │   ├── edge.go                     #     EntityEdge, EpisodicEdge
│   │   ├── temporal.go                 #     BiTemporal model
│   │   ├── group.go                    #     Multi-tenancy primitives
│   │   └── embedding.go               #     EmbeddingVector type
│   ├── profile/                        #   Memobase profile types
│   │   ├── profile.go                  #     Topic/SubTopic/Content model
│   │   ├── event.go                    #     UserEvent, EventGist
│   │   └── blob.go                     #     ChatBlob, DocBlob, SummaryBlob
│   ├── adapters/                       #   Infrastructure adapter interfaces
│   │   ├── graphdb/                    #     GraphDB interface + Neo4j/FalkorDB/Kuzu/SurrealDB
│   │   ├── vectordb/                   #     VectorDB interface + Qdrant/pgvector/SurrealDB
│   │   ├── reldb/                      #     RelationalDB interface + PostgreSQL/SurrealDB
│   │   ├── llm/                        #     LLMClient interface + Bifrost
│   │   ├── embedder/                   #     EmbedderClient interface + providers
│   │   ├── reranker/                   #     CrossEncoder interface
│   │   ├── vlm/                        #     VLM interface (vision models)
│   │   ├── kms/                        #     KMS interface (Local/Vault/Cloud)
│   │   └── storage/                    #     ObjectStorage interface + S3/MinIO
│   ├── surrealdb/                      #   SurrealDB unified adapter
│   │   ├── client.go                   #     Connection pool, SurrealQL builder
│   │   ├── graph_adapter.go            #     GraphDB interface → RELATE/graph queries
│   │   ├── vector_adapter.go           #     VectorDB interface → HNSW/MTREE queries
│   │   ├── relational_adapter.go       #     RelationalDB interface → SurrealQL CRUD
│   │   ├── multi_tenant.go             #     Namespace-based tenant isolation
│   │   └── migration.go                #     DEFINE TABLE/INDEX schema helpers
│   ├── viking/                         #   OpenViking shared domain types
│   │   ├── context.go                  #     Context, ContextType, ContextLevel
│   │   ├── namespace.go                #     URI resolution, ownership
│   │   ├── uri.go                      #     viking:// URI validation
│   │   ├── identity.go                 #     UserIdentifier, Role
│   │   └── tiered.go                   #     L0/L1/L2 context levels
│   ├── vikingfs/                       #   Go-native filesystem engine
│   │   ├── fs.go                       #     Core FS operations
│   │   ├── tree.go                     #     Directory tree operations
│   │   └── lock.go                     #     PathLock (point/subtree/mv)
│   ├── parse/                          #   File parser registry
│   │   ├── registry.go                 #     Extension → parser routing
│   │   ├── treesitter.go               #     tree-sitter Go bindings
│   │   ├── markdown.go                 #     Markdown parser
│   │   └── document.go                 #     PDF/DOCX parser
│   ├── middleware/                      #   Shared gRPC/HTTP interceptors
│   │   ├── auth/                       #     JWT/APIKey/DEV/TRUSTED extraction
│   │   ├── logging/                    #     Structured access logging
│   │   ├── tracing/                    #     OTel trace propagation
│   │   ├── recovery/                   #     Panic recovery
│   │   ├── ratelimit/                  #     Redis sliding window
│   │   └── validation/                 #     Request validation
│   ├── resilience/                     #   Circuit breaker, retry, bulkhead
│   ├── observability/                  #   Tracer, metrics, logger, health
│   ├── config/                         #   Viper loader + validator
│   ├── errors/                         #   Domain error → gRPC status mapping
│   ├── nats/                           #   NATS client helpers
│   ├── auth/                           #   JWT provider, API key validator
│   ├── tenant/                         #   Tenant context extraction
│   ├── tokenizer/                      #   tiktoken-go wrapper
│   ├── prompt/                         #   Prompt template engine (EN/ZH)
│   ├── pagination/                     #   Cursor/offset pagination
│   └── testutil/                       #   Fixtures, mocks, testcontainers
│
├── migrations/                         #   SQL + Cypher migration files
├── deploy/
│   ├── docker-compose/                 #   Dev environment
│   └── kubernetes/                     #   Kustomize base + overlays
├── go.mod
├── buf.yaml
├── Makefile
└── README.md
```

---

## 6. Clean Architecture — Standard Per Service

Mỗi service trong `services/<name>/` tuân theo 4-layer chuẩn hóa:

```
services/<service-name>/
├── cmd/server/main.go                  # Entry point, wire injection
├── internal/
│   ├── domain/                         # Layer 1: ZERO external imports
│   │   ├── entity.go                   #   Domain models (pure Go structs)
│   │   ├── value_object.go             #   Value objects (immutable)
│   │   ├── event.go                    #   Domain events
│   │   └── errors.go                   #   Domain-specific errors
│   ├── usecase/                        # Layer 2: imports domain only
│   │   ├── <usecase_name>.go           #   One file per use case
│   │   ├── port/
│   │   │   ├── input.go               #   Input ports (use case interfaces)
│   │   │   └── output.go              #   Output ports (repository, external)
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                        # Layer 3: implements ports
│   │   ├── grpc/                       #   gRPC handlers (controllers)
│   │   │   ├── handler.go
│   │   │   └── mapper.go              #   Proto ↔ Domain mapping
│   │   ├── repository/                 #   Output adapter implementations
│   │   │   ├── postgres/
│   │   │   ├── neo4j/
│   │   │   └── redis/
│   │   ├── client/                     #   External service gRPC clients
│   │   └── event/                      #   NATS publisher/subscriber
│   └── infra/                          # Layer 4: Frameworks & Drivers
│       ├── config/config.go
│       ├── server/grpc.go
│       ├── telemetry/
│       └── wire/wire.go
├── Dockerfile
└── README.md
```

### Dependency Rule (STRICT)

```
domain ← usecase ← adapter ← infra
 (inner)                     (outer)

✅ domain: ZERO external imports (no gRPC, no DB, no framework)
✅ usecase: imports domain only; defines port interfaces
✅ adapter: imports usecase(ports) + domain; implements interfaces
✅ infra: imports everything; wires via Google Wire
```

---

## 7. Inter-Service Communication

### 7.1 Synchronous (gRPC)

```
Gateway → All services (fan-out by route)

Cognee:      ingestion → cognify → (search for reindex)
Graphiti:    ingestion → knowledge → store; search → knowledge + store
Memobase:    ingestion → engine; context → vnp-event
OpenViking:  fs ↔ crypto (encrypt/decrypt); session → fs + search
             resource → fs (write) + search (embed + index)
Zep:         memory → thread (upsert); memory → graph (async NATS)
             search → graph (traversal); user → search (context)
Supermemory: document → memory (extract facts); document → search (index)
             connector → document (sync batch); memory → profile (update)
Platform:    search-hub → cognee-search + graphiti-search + memobase-context
                         + ov-search + zep-search + sm-search
Admin:       health checks to all services
```

### 7.2 Async Events (NATS JetStream)

| Stream | Subject | Publisher | Subscriber |
|--------|---------|-----------|------------|
| `cognee` | `cognee.data.ingested` | cognee-ingestion | cognee-cognify |
| `cognee` | `cognee.pipeline.completed` | cognee-cognify | cognee-search |
| `graphiti` | `graphiti.episode.ingested` | graphiti-ingestion | graphiti-search |
| `graphiti` | `graphiti.entity.resolved` | graphiti-knowledge | graphiti-search |
| `graphiti` | `graphiti.community.rebuilt` | graphiti-knowledge | graphiti-search |
| `memobase` | `memobase.buffer.ready` | memobase-ingestion | memobase-engine |
| `memobase` | `memobase.engine.completed` | memobase-engine | memobase-context |
| `memobase` | `memobase.profile.changed` | memobase-engine | memobase-context |
| `memobase` | `memobase.event.created` | memobase-engine | vnp-event |
| `openviking` | `ov.resource.ingested` | ov-resource | ov-search (reindex) |
| `openviking` | `ov.session.committed` | ov-session | ov-search (hotness) |
| `openviking` | `ov.session.memory.extracted` | ov-session | ov-fs (write memories) |
| `openviking` | `ov.content.written` | ov-fs | ov-search (embed + upsert) |
| `openviking` | `ov.content.deleted` | ov-fs | ov-search (remove index) |
| `openviking` | `ov.crypto.key.rotated` | ov-crypto | ov-fs (re-wrap files) |
| `zep` | `zep.memory.messages.ingested` | zep-memory | zep-graph |
| `zep` | `zep.graph.extraction.completed` | zep-graph | zep-search |
| `zep` | `zep.graph.fact.created` | zep-graph | zep-search |
| `zep` | `zep.graph.fact.invalidated` | zep-graph | zep-search |
| `zep` | `zep.thread.session.ended` | zep-thread | zep-memory |
| `zep` | `zep.user.deleted` | zep-user | zep-thread, zep-graph |
| `supermemory` | `sm.document.created` | sm-document | sm-memory, sm-search |
| `supermemory` | `sm.document.deleted` | sm-document | sm-memory, sm-search |
| `supermemory` | `sm.memory.created` | sm-memory | sm-search, sm-profile |
| `supermemory` | `sm.memory.forgotten` | sm-memory | sm-search, sm-profile |
| `supermemory` | `sm.connection.synced` | sm-connector | sm-document |
| `supermemory` | `sm.auth.api_key.used` | sm-auth | sm-analytics |
| `admin` | `admin.tenant.created` | vnp-admin | All |
| `admin` | `admin.tenant.deleted` | vnp-admin | All (cascade) |

---

## 8. Cross-Engine Search Hub

`vnp-search-hub` is the cross-engine recall orchestrator:

```
Client: memory.recall(query)
  │
  ▼
Gateway → vnp-search-hub.RecallContext(query, tenant_id, scope)
  │
  ├── Parallel fan-out:
  │   ├── cognee-search.Search(query)          → semantic KG results
  │   ├── graphiti-search.HybridSearch(query)   → temporal graph results
  │   ├── memobase-context.GetContext(query)     → user profile + events
  │   ├── ov-search.HierarchicalSearch(query)   → tiered context (L0/L1/L2)
  │   ├── zep-search.GraphSearch(query)         → temporal facts + context
  │   ├── sm-search.HybridSearch(query)          → adaptive KG results
  │   └── vnp-event.SearchEvents(query)          → cross-domain events
  │
  ▼
  Merge + Rerank (RRF/MMR/Cross-Encoder) + Dedup
  │
  ▼
Response: UnifiedRecallResult {
  profiles:   []ProfileSection,
  facts:      []GraphFact,
  events:     []TemporalEvent,
  documents:  []DocumentChunk,
  metadata:   RecallMetadata,
}
```

---

## 9. Data Flow Diagrams

### 9.1 Memory Store (Auto-routing)

```
Client: memory.store(data, type)
  │
  ▼
Gateway routes by type:
  ├── "semantic"       → cognee-ingestion.Ingest()
  ├── "episodic"       → graphiti-ingestion.IngestEpisode()
  ├── "conversational" → memobase-ingestion.InsertBlob(ChatBlob)
  ├── "profile"        → memobase-ingestion.InsertBlob(SummaryBlob)
  ├── "procedural"     → ov-resource.Ingest()
  ├── "auto"           → Gateway classifies → route
```

### 9.2 OpenViking Session Pipeline

```
[ov-session]: 2-Phase Commit
  Phase 1 (Archive): compress conversation → write to ov-fs
  Phase 2 (Extract): LLM extract memories → write to ov-fs
  Emit: ov.session.memory.extracted
  [ov-search] updates hotness scores
```

### 9.3 Memobase Buffer Flush Pipeline

```
[Insert ChatBlob] → ingestion stores blob + buffer entry
        │
        buffer full? (token_sum > 1024)
        ▼
NATS: memobase.buffer.ready {user_id, project_id, buffer_ids[]}
        │
        ▼
[memobase-engine]:
  1. Fetch blobs
  2. LLM #1: extract_topics
  3. LLM #2: merge_yolo (3 fixed calls total)
  4. Persist profiles + events + embeddings
  5. Emit: memobase.engine.completed
        │
        ▼
[memobase-context] invalidates Redis cache
[vnp-event] indexes new event embeddings
```

### 9.3 Cognee Cognify Pipeline

```
NATS: cognee.data.ingested {dataset_id, tenant_id}
        │
        ▼
[cognee-cognify]:
  1. Classify content
  2. Chunk (recursive/AST/paragraph)
  3. Extract entities + relationships (LLM)
  4. Build knowledge graph (Neo4j)
  5. Generate embeddings (Qdrant)
  6. Emit: cognee.pipeline.completed
```

### 9.4 Graphiti Episode Pipeline

```
[graphiti-ingestion]:
  1. Validate + enqueue
  2. → knowledge.ExtractEntities()
  3. → knowledge.ResolveEntities()
  4. → knowledge.ExtractEdges()
  5. → knowledge.ResolveEdges()
  6. → store.SaveBulk()
  7. → knowledge.UpdateCommunity()
  8. Emit: graphiti.episode.ingested
```

---

## 10. Multi-Tenancy Strategy

| Engine | Isolation Key | Mechanism |
|--------|--------------|-----------|
| **Cognee** | `tenant_id` | PostgreSQL RLS, Neo4j namespace labels |
| **Graphiti** | `group_id` | Property filter on all graph queries |
| **Memobase** | `project_id` | Composite PK (id, project_id), DB partition |
| **OpenViking** | `account_id` | Account/User/Agent namespace isolation, RBAC |
| **Zep** | `project_uuid` | Advisory locks, schema-based isolation |
| **Supermemory** | `org_id` | Org isolation at DB/cache/queue, RBAC |
| **Platform** | `tenant_id` | Propagated via gRPC metadata `x-tenant-id` |

```
Request flow:
  Client → JWT/APIKey (tenant claim) → Gateway extracts tenant_id
    → Propagates as gRPC metadata "x-tenant-id"
    → Each service maps to appropriate isolation key
    → All queries filtered by tenant scope
```

---

## 11. Cross-Cutting Concerns

| Concern | Package | Implementation |
|---------|---------|----------------|
| Auth (JWT + API Key) | `pkg/middleware/auth/` | Gateway validates; propagates via gRPC metadata |
| Multi-Tenancy | `pkg/tenant/` | tenant_id in gRPC metadata, scoped queries |
| Rate Limiting | `pkg/middleware/ratelimit/` | Redis sliding window, per-tenant per-endpoint |
| Circuit Breaker | `pkg/resilience/` | sony/gobreaker, per-downstream-service |
| Retry | `pkg/resilience/` | Exponential backoff + jitter |
| Bulkhead | `pkg/resilience/` | Channel semaphore for LLM calls |
| Observability | `pkg/observability/` | OTel traces + Prometheus + slog JSON |
| Health | `pkg/observability/` | gRPC Health v1 + HTTP /healthz /readyz /livez |
| Error Mapping | `pkg/errors/` | Domain errors → gRPC status → HTTP status |
| Encryption | `pkg/adapters/kms/` | Envelope encryption transparent to services |
| Token Counting | `pkg/tokenizer/` | tiktoken-go (gpt-4o encoder) |
| Prompt Templates | `pkg/prompt/` | EN/ZH template registry |
| File Parsing | `pkg/parse/` | tree-sitter, markdown, PDF/DOCX |

---

## 12. Deployment

### 12.1 Development (Docker Compose)

```yaml
services:
  vnp-gateway:          # 8080 (REST) + 8081 (gRPC) + 8082 (MCP)
  cognee-ingestion:     # 9011
  cognee-cognify:       # 9012
  cognee-search:        # 9013
  graphiti-ingestion:   # 9021
  graphiti-search:      # 9022
  graphiti-knowledge:   # 9023
  graphiti-store:       # 9024
  memobase-ingestion:   # 9031
  memobase-engine:      # 9032
  memobase-context:     # 9033
  ov-fs:                # 9051
  ov-search:            # 9052
  ov-session:           # 9053
  ov-resource:          # 9054
  ov-crypto:            # 9055
  ov-admin:             # 9056
  vnp-event:            # 9041
  vnp-search-hub:       # 9042
  vnp-admin:            # 9050
  # Infrastructure
  postgresql:           # 5432
  neo4j:                # 7474/7687
  qdrant:               # 6333
  redis:                # 6379
  nats:                 # 4222
  bifrost:              # 8443
  otel-collector:       # 4317
```

### 12.2 Production (Kubernetes)

```
Namespace: vnp-memory
├── Deployments (HPA enabled)
│   ├── vnp-gateway           (3-10 replicas)
│   ├── cognee-ingestion      (2-8)
│   ├── cognee-cognify        (2-6)
│   ├── cognee-search         (3-12)
│   ├── graphiti-ingestion    (2-8)
│   ├── graphiti-search       (3-12)
│   ├── graphiti-knowledge    (2-6)
│   ├── graphiti-store        (2-6)
│   ├── memobase-ingestion    (2-8)
│   ├── memobase-engine       (2-6)
│   ├── memobase-context      (3-10)
│   ├── ov-fs                 (2-6)
│   ├── ov-search             (3-10)
│   ├── ov-session            (2-6)
│   ├── ov-resource           (2-6)
│   ├── ov-crypto             (1-3)
│   ├── ov-admin              (1-2)
│   ├── vnp-event             (2-6)
│   ├── vnp-search-hub        (3-10)
│   └── vnp-admin             (1-2)
├── StatefulSets
│   ├── postgresql            (3, patroni)
│   ├── neo4j-core            (3, cluster)
│   ├── redis-cluster         (6)
│   ├── qdrant                (3)
│   └── nats-cluster          (3)
└── Ingress
    └── vnp-gateway (HTTPS, path-based)
```

---

## 13. Document Index

| Document | Description |
|----------|-------------|
| [01-gateway.md](./01-gateway.md) | Unified API Gateway — REST, gRPC, MCP, WebDAV, Auth |
| [02-cognee-services.md](./02-cognee-services.md) | Cognee: Ingestion + Cognify + Search (3 services) |
| [03-graphiti-services.md](./03-graphiti-services.md) | Graphiti: Ingestion + Search + Knowledge + Store (4 services) |
| [04-memobase-services.md](./04-memobase-services.md) | Memobase: Ingestion + Engine + Context (3 services) |
| [05-openviking-services.md](./05-openviking-services.md) | OpenViking: FS + Search + Session + Resource + Crypto + Admin (6 services) |
| [06-zep-services.md](./06-zep-services.md) | Zep: User + Thread + Memory + Graph + Search + Admin (6 services) |
| [07-supermemory-services.md](./07-supermemory-services.md) | Supermemory: Document + Memory + Search + Profile + Connector + MCP + Auth + Analytics + Project (9 services) |
| [08-platform-services.md](./08-platform-services.md) | Platform: Event + Search Hub + Admin (3 services) |
| [09-shared-packages.md](./09-shared-packages.md) | Shared `pkg/` — interfaces, middleware, resilience, SurrealDB adapter |
| [10-data-models-deployment.md](./10-data-models-deployment.md) | Unified domain models + Docker Compose + Kubernetes |
| [11-surrealdb-integration.md](./11-surrealdb-integration.md) | SurrealDB multi-model backend — replaces Neo4j + Qdrant + PostgreSQL |
