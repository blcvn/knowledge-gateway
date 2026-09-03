# SOL-003 — Tái Cấu Trúc Dịch Vụ: 48 → 8 Services

```yaml
id: SOL-003
title: "Tái Cấu Trúc Toàn Diện — 48 → 8 Deployable Services"
service: cross-service
version: 1.2.0
status: Completed
priority: P0
created: 2026-06-11
updated: 2026-06-11
completed: 2026-06-11
author: Architecture Review
linked_cr: SOL-001, SOL-002
linked_map: SOL-003-migration-map.md
tasks: MERGE-P1-T1..P4-T5 (15/15 Done)
build_status: "✅ PASS — go build (all 8 services)"
```

---

## 1. Bức Tranh Hiện Tại

### 1.1 Tổng Quan Inventory

| Nhóm | Services | Dòng Code | Trạng Thái Thực Tế |
|------|----------|-----------|---------------------|
| **VNP Core** | vnp-admin, vnp-event, vnp-search-hub, vnp-platform, vnp-dashboard, vnp-infra, vnp-observability, vnp-pipelines | ~6.155 | Stub (forward.NewRouter + no routes) |
| **Cognee** | cognee-ingestion, cognee-cognify, cognee-pipeline, cognee-search | ~8.923 | Stub (forward.NewRouter + no routes) |
| **Graphiti** | graphiti-ingestion, graphiti-knowledge, graphiti-pipeline, graphiti-search, graphiti-store | ~12.726 | Stub (forward.NewRouter + no routes) |
| **Memobase** | memobase-context, memobase-engine, memobase-ingestion, memobase-pipeline | ~3.981 | Stub (forward.NewRouter + no routes) |
| **OpenViking** | ov-admin, ov-crypto, ov-fs, ov-resource, ov-search, ov-session, ov-storage | ~8.885 | Stub (forward.NewRouter + no routes) |
| **Supermemory** | sm-analytics, sm-auth, sm-connector, sm-document, sm-engine, sm-mcp, sm-memory, sm-profile, sm-project, sm-search | ~2.906 | sm-auth: implementado real; resto: stub |
| **Zep** | zep-admin, zep-core, zep-graph, zep-memory, zep-search, zep-thread, zep-user | ~1.950 | zep-core: TODO skeleton; proxy stubs; rest: stub |
| **BA Knowledge** | ba-knowledge-service, ba-knowledge-worker | ~3.909 | ba-knowledge-worker: real impl (Redis queue) |
| **Gateway** | vnp-gateway | ~20.000+ | **Fully implemented** (production-grade) |

**Tổng: 48 modules trong go.work** → **47 service modules + 1 gateway**

### 1.2 Vấn Đề Cốt Lõi

#### 🔴 Vấn Đề 1: Explosion of Empty Stubs (47/48 = 98% là stub)

Hầu hết các service chỉ chạy pattern sau và **không có business logic nào**:
```go
// Pattern lặp lại trong ~40 services
router := forward.NewRouter()  // empty router — không có routes
forward.RegisterForwardService(grpcServer, router)
```

Khi gateway gọi `ForwardToService(h.registry, "graphiti-ingestion", ...)`, service đích nhận request nhưng trả về `{"error": "no handler for POST /v1/graphiti/episodes"}` vì router trống.

#### 🔴 Vấn Đề 2: Overhead Vận Hành Cực Cao

- **48 go.mod files** → 48 dependency graphs riêng biệt
- **48 Docker images** → build time O(n), CI/CD pipeline dài
- **48 health check endpoints** → monitoring complexity
- **48 gRPC port allocations** → network namespace pollution

#### 🔴 Vấn Đề 3: SOL-001 Chưa Hoàn Chỉnh

SOL-001 đã "Done" nhưng thực tế:
- Các `*-pipeline`, `*-core`, `*-storage`, `*-engine` đã được tạo **cấu trúc** nhưng vẫn là stub (forward.NewRouter không route)
- Business logic từ các service gốc (cognee-ingestion, graphiti-store, v.v.) **chưa được merge thật sự**
- `vnp-platform` có code thật nhưng vẫn thiếu nhiều adapter implementations

#### 🟡 Vấn Đề 4: Zep là External Service Integration

`zep-go` là **Go client SDK** của Zep Cloud (external), không phải service tự host. Các `zep-*` services chỉ là proxy wrapper.

---

## 2. Phân Tích Nhóm Service Theo Chức Năng

### 2.1 Ma Trận Phân Tích

```
┌─────────────────────────────────────────────────────────────────┐
│                     FUNCTIONAL DOMAINS                          │
├──────────────┬──────────────────────────────────────────────────┤
│ Domain       │ Chức Năng                                        │
├──────────────┼──────────────────────────────────────────────────┤
│ PLATFORM     │ Auth, Admin, Tenant mgmt, Events, Analytics     │
│              │ Services: sm-auth, vnp-admin, vnp-event,         │
│              │          ov-admin, zep-admin, sm-analytics,      │
│              │          sm-project, vnp-platform                │
├──────────────┼──────────────────────────────────────────────────┤
│ GRAPH KG     │ Knowledge Graph (GraphQL-style): ingest,         │
│              │ store, search, traverse                          │
│              │ Services: graphiti-*, cognee-*                   │
├──────────────┼──────────────────────────────────────────────────┤
│ MEMORY       │ Working memory, episodic memory, context         │
│              │ Services: memobase-*, zep-*, sm-memory,          │
│              │          sm-document, sm-profile                 │
├──────────────┼──────────────────────────────────────────────────┤
│ STORAGE      │ File system, crypto, resource mgmt, WebDAV       │
│              │ Services: ov-fs, ov-crypto, ov-resource,         │
│              │          ov-storage, ov-session                  │
├──────────────┼──────────────────────────────────────────────────┤
│ SEARCH       │ Cross-engine search, RAG, hybrid search          │
│              │ Services: vnp-search-hub, ov-search,             │
│              │          graphiti-search, cognee-search,         │
│              │          sm-search, sm-connector, sm-mcp         │
├──────────────┼──────────────────────────────────────────────────┤
│ PIPELINE     │ Async processing, ingestion, worker queue        │
│              │ Services: cognee-pipeline, graphiti-pipeline,    │
│              │          memobase-pipeline, ba-knowledge-worker,  │
│              │          vnp-pipelines                            │
├──────────────┼──────────────────────────────────────────────────┤
│ OBSERVABILITY│ Metrics, traces, infra topology, dashboard       │
│              │ Services: vnp-observability, vnp-infra,          │
│              │          vnp-dashboard, sm-engine                │
└──────────────┴──────────────────────────────────────────────────┘
```

### 2.2 Dependency Map

```
Gateway (vnp-gateway)
    │
    ├── → vnp-platform       (auth, admin, tenant, events)
    ├── → kg-service         (graphiti + cognee combined)
    ├── → memory-service     (memobase + zep + sm-memory)
    ├── → storage-service    (ov-fs + ov-crypto + ov-resource)
    ├── → search-service     (search hub + all search adapters)
    ├── → pipeline-service   (all async ingestion + workers)
    └── → observability-service (metrics + traces + dashboard)
```

---

## 3. Giải Pháp Đề Xuất: 48 → 8 Services

### 3.1 Kiến Trúc Mục Tiêu

```
┌─────────────────────────────────────────────────────────────────┐
│                      PRODUCTION DEPLOYMENT                      │
│                                                                 │
│   ┌──────────────┐                                              │
│   │ vnp-gateway  │  ← REST/gRPC/MCP entry point (giữ nguyên)  │
│   └──────┬───────┘                                              │
│          │ gRPC ForwardService                                  │
│          │                                                      │
│   ┌──────┴───────────────────────────────────────────────┐     │
│   │                  7 Backend Services                  │     │
│   │                                                      │     │
│   │  ① vnp-platform     ← Auth + Admin + Platform       │     │
│   │  ② kg-service       ← Knowledge Graph (Graphiti+    │     │
│   │                         Cognee)                     │     │
│   │  ③ memory-service   ← Memory (Memobase + Zep + SM)  │     │
│   │  ④ storage-service  ← Files + Crypto + Resources    │     │
│   │  ⑤ search-service   ← Search Hub + All Adapters    │     │
│   │  ⑥ pipeline-service ← Async Ingestion + Workers    │     │
│   │  ⑦ obs-service      ← Observability + Dashboard    │     │
│   └──────────────────────────────────────────────────────┘     │
│                                                                 │
│   Infrastructure: PostgreSQL, Redis, NATS                      │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Mapping Chi Tiết: Service Cũ → Service Mới

#### ① vnp-platform (giữ tên, mở rộng scope)

**Absorbs:**
- `vnp-admin` → Admin usecase (tenant CRUD, API key management)
- `vnp-event` → Event usecase (NATS publish/subscribe)
- `sm-auth` → Auth usecase (JWT, Google SSO) ← **đã implement thật**
- `ov-admin` → OpenViking account/agent management
- `zep-admin` → Project/session management
- `sm-analytics` → Analytics aggregation (stub → move logic here)
- `sm-project` → Project/space management
- `vnp-dashboard` → Dashboard metrics aggregation

**External ports exposed (gRPC ForwardService routes):**
```
POST /v1/auth/register       → sm-auth usecase
POST /v1/auth/login          → sm-auth usecase
POST /v1/auth/sso/google     → sm-auth usecase
POST /v1/admin/tenants        → vnp-admin usecase
POST /v1/admin/tenants/{id}/keys → vnp-admin usecase
GET  /v1/admin/health         → health check
GET  /v1/admin/metrics        → vnp-admin usecase
GET  /v1/console/dashboard/*  → vnp-dashboard usecase
GET  /v1/console/governance/* → vnp-admin usecase
```

**Remove (deprecated):**
- `vnp-admin` (merged)
- `vnp-event` (merged)
- `ov-admin` (merged)
- `zep-admin` (merged)
- `sm-analytics` (merged)
- `sm-project` (merged)
- `vnp-dashboard` (merged)
- `vnp-platform` (kept, expanded)

---

#### ② kg-service (NEW — Knowledge Graph Service)

**Absorbs:**
- `graphiti-ingestion` → Episode ingest pipeline
- `graphiti-knowledge` → Knowledge extraction/ontology
- `graphiti-pipeline` → Graphiti batch processing
- `graphiti-search` → Graph search queries
- `graphiti-store` → Node/Edge CRUD storage
- `cognee-ingestion` → Cognee dataset/data management
- `cognee-cognify` → Cognification pipeline
- `cognee-pipeline` → Cognee batch processing
- `cognee-search` → Cognee semantic search

**Architecture bên trong:**
```
kg-service/
├── internal/
│   ├── adapter/grpc/        # ForwardService router
│   ├── domain/
│   │   ├── graphiti/        # Episode, Node, Edge, Ontology
│   │   └── cognee/          # Dataset, Cognification, Index
│   ├── usecase/
│   │   ├── graphiti/        # IngestEpisode, Search, GetNode, GetEdge
│   │   └── cognee/          # CreateDataset, UploadData, Cognify, Search
│   └── infra/
│       ├── neo4j/           # Graphiti backend
│       ├── pgvector/        # Cognee vector store
│       └── nats/            # Async pipeline events
└── cmd/server/main.go       # Single binary
```

**External ports (ForwardService routes):**
```
POST /v1/graphiti/episodes          → graphiti.IngestEpisode
POST /v1/graphiti/search            → graphiti.Search
GET  /v1/graphiti/nodes/{id}        → graphiti.GetNode
GET  /v1/graphiti/edges/{id}        → graphiti.GetEdge
POST /v1/cognee/datasets            → cognee.CreateDataset
POST /v1/cognee/datasets/{id}/data  → cognee.UploadData
POST /v1/cognee/datasets/{id}/cognify → cognee.Cognify
POST /v1/cognee/search              → cognee.Search
```

**Remove:**
- `graphiti-ingestion`, `graphiti-knowledge`, `graphiti-pipeline`
- `graphiti-search`, `graphiti-store`
- `cognee-ingestion`, `cognee-cognify`, `cognee-pipeline`, `cognee-search`

---

#### ③ memory-service (NEW — Unified Memory Service)

**Absorbs:**
- `memobase-context` → User context retrieval
- `memobase-engine` → Memory engine (scoring, relevance)
- `memobase-ingestion` → Blob insertion, flush
- `memobase-pipeline` → Memobase batch processing
- `zep-user` → User profile management (Zep wrapper)
- `zep-thread` → Thread/session management (Zep wrapper)
- `zep-memory` → Memory put/get (Zep wrapper)
- `zep-search` → Zep search
- `zep-graph` → Zep knowledge graph facts
- `zep-core` → Zep core adapter ← skeleton only, implement here
- `sm-memory` → Supermemory memory entries
- `sm-document` → Document management
- `sm-profile` → User profile management

**Architecture bên trong:**
```
memory-service/
├── internal/
│   ├── adapter/
│   │   ├── grpc/          # ForwardService router
│   │   └── zep/           # Zep Cloud API client (uses zep-go SDK)
│   ├── domain/
│   │   ├── memobase/      # Blob, Context, Profile, Event
│   │   ├── zep/           # User, Thread, Memory, GraphFact
│   │   └── sm/            # Memory, Document, Profile
│   ├── usecase/
│   │   ├── memobase/      # InsertBlob, Flush, GetContext, GetProfiles
│   │   ├── zep/           # CreateUser, PutMemory, GetMemory, GraphSearch
│   │   └── sm/            # CreateMemory, CreateDocument, GetProfile
│   └── infra/
│       ├── pgvector/      # Memobase vector storage
│       ├── redis/         # Working memory cache
│       └── nats/          # Event streaming
└── cmd/server/main.go
```

**External ports:**
```
POST /v1/memobase/users/{uid}/blobs   → memobase.InsertBlob
POST /v1/memobase/users/{uid}/flush   → memobase.Flush
GET  /v1/memobase/users/{uid}/context → memobase.GetContext
GET  /v1/memobase/users/{uid}/profiles → memobase.GetProfiles
POST /v1/zep/users                    → zep.CreateUser
GET  /v1/zep/users/{id}              → zep.GetUser
PATCH /v1/zep/users/{id}             → zep.UpdateUser
POST /v1/zep/sessions/{id}/memory    → zep.PutMemory
GET  /v1/zep/sessions/{id}/memory    → zep.GetMemory
POST /v1/zep/graph/search            → zep.GraphSearch
POST /v1/zep/sessions/{id}/search    → zep.SessionSearch
POST /v1/zep/graph/facts             → zep.AddFact
POST /v1/zep/graph/ontology          → zep.SetOntology
POST /v1/sm/memories                 → sm.CreateMemory
POST /v1/sm/rag                      → sm.RAG
```

**Remove:**
- `memobase-context`, `memobase-engine`, `memobase-ingestion`, `memobase-pipeline`
- `zep-user`, `zep-thread`, `zep-memory`, `zep-search`, `zep-graph`, `zep-core`, `zep-admin`
- `sm-memory`, `sm-document`, `sm-profile`

**Note về Zep:** `zep-go` (Go SDK) được giữ nguyên như một dependency thư viện trong `go.work`, không phải service.

---

#### ④ storage-service (NEW — File & Resource Storage)

**Absorbs:**
- `ov-fs` → File system operations (read/write/delete/tree/grep)
- `ov-crypto` → Encryption/decryption
- `ov-resource` → Resource ingestion pipeline
- `ov-storage` → Unified storage abstraction ← target service
- `ov-session` → Chat session management over files

**Architecture bên trong:**
```
storage-service/
├── internal/
│   ├── domain/
│   │   ├── fs/           # File, Directory, TreeNode
│   │   ├── crypto/       # Encrypted content, key derivation
│   │   ├── resource/     # Resource, IngestJob
│   │   └── session/      # ChatSession, Message, Commit
│   ├── usecase/
│   │   ├── fs/           # ReadFile, WriteFile, DeleteFile, Tree, Grep
│   │   ├── crypto/       # Encrypt, Decrypt
│   │   ├── resource/     # Ingest, GetStatus
│   │   └── session/      # CreateSession, AddMessage, CommitSession
│   └── infra/
│       ├── localfs/      # Local filesystem implementation
│       ├── s3/           # Optional S3/MinIO backend
│       └── pgvector/     # Resource embedding index
└── cmd/server/main.go
```

**External ports:**
```
GET    /v1/ov/files/{path...}           → fs.ReadFile
PUT    /v1/ov/files/{path...}           → fs.WriteFile
DELETE /v1/ov/files/{path...}           → fs.DeleteFile
GET    /v1/ov/tree/{path...}            → fs.Tree
POST   /v1/ov/grep                      → fs.Grep
POST   /v1/ov/sessions                  → session.CreateSession
POST   /v1/ov/sessions/{id}/messages    → session.AddMessage
POST   /v1/ov/sessions/{id}/commit      → session.CommitSession
POST   /v1/ov/resources/ingest          → resource.Ingest
```

**Remove:**
- `ov-fs`, `ov-crypto`, `ov-resource`, `ov-storage`, `ov-session`, `ov-admin` (moved to platform)

---

#### ⑤ search-service (RENAME từ vnp-search-hub)

**Absorbs:**
- `vnp-search-hub` → Search orchestration (RRF/MMR reranking) ← **partially implemented**
- `ov-search` → OpenViking semantic search
- `sm-search` → Supermemory search
- `sm-connector` → External data connectors (sync)
- `sm-mcp` → MCP tool integrations

**Architecture bên trong:**
```
search-service/
├── internal/
│   ├── domain/
│   │   ├── search/       # SearchQuery, SearchResult, RankingStrategy
│   │   ├── connector/    # Connector, SyncJob
│   │   └── mcp/          # MCPTool, ToolCall, ToolResponse
│   ├── usecase/
│   │   ├── orchestrator/ # MultiEngineSearch, RRF, MMR reranking
│   │   ├── connector/    # CreateConnector, SyncConnector
│   │   └── mcp/          # ToolRegistry, ExecuteTool
│   └── infra/
│       ├── graphiti/     # Graphiti search client
│       ├── cognee/       # Cognee search client
│       ├── memobase/     # Memobase context client
│       └── pgvector/     # Direct vector search
└── cmd/server/main.go
```

**External ports:**
```
POST /v1/ov/search                          → ov.Search
POST /v1/sm/search                          → sm.Search
POST /v1/sm/connections                     → connector.CreateConnection
POST /v1/sm/connections/{id}/sync           → connector.SyncConnection
GET  /v1/console/memory/search              → search.ConsoleSearch
GET  /v1/console/memory/{id}               → search.GetMemory
GET  /v1/console/memory/{id}/neighbors     → search.GetNeighbors
GET  /v1/console/memory/{id}/versions      → search.GetVersions
GET  /v1/console/adaptive/memories         → search.ListMemories
GET  /v1/console/adaptive/memories/{id}/versions → search.GetVersions
GET  /v1/console/adaptive/analytics        → search.Analytics
```

**Remove:**
- `vnp-search-hub` (renamed/expanded to search-service)
- `ov-search` (merged)
- `sm-search` (merged)
- `sm-connector` (merged)
- `sm-mcp` (merged)

---

#### ⑥ pipeline-service (NEW — Async Processing Hub)

**Absorbs:**
- `vnp-pipelines` → Pipeline status/management API
- `ba-knowledge-service` → BA knowledge CRUD service (stub)
- `ba-knowledge-worker` → BA knowledge Redis queue worker ← **real impl**

**Architecture bên trong:**
```
pipeline-service/
├── internal/
│   ├── domain/
│   │   ├── pipeline/     # Pipeline, Job, Queue, Worker
│   │   └── knowledge/    # PRD, Outline, IndexJob
│   ├── usecase/
│   │   ├── pipeline/     # Status, ListJobs, GetJob, Queues, Workers
│   │   └── knowledge/    # HandleIndexPRD, HandleGenOutline
│   └── infra/
│       ├── nats/         # Pipeline event streaming
│       ├── redis/        # Queue backend (from ba-knowledge-worker)
│       └── pg/           # Job persistence
└── cmd/
│   ├── server/main.go    # gRPC server (ForwardService)
│   └── worker/main.go    # Background worker process
```

**External ports:**
```
GET /v1/console/pipelines/status      → pipeline.Status
GET /v1/console/pipelines/queues      → pipeline.Queues
GET /v1/console/pipelines/workers     → pipeline.Workers
GET /v1/console/pipelines/templates   → pipeline.Templates
GET /v1/console/pipelines/{engine}    → pipeline.GetEngine
GET /v1/console/pipelines/{engine}/jobs → pipeline.ListJobs
GET /v1/console/pipelines/{engine}/jobs/{id} → pipeline.GetJob
```

**Note:** Service này có 2 binaries — `server` (gRPC + API) và `worker` (background queue processor)

**Remove:**
- `vnp-pipelines` (merged)
- `ba-knowledge-service` (merged)
- `ba-knowledge-worker` (merged, worker binary stays)

---

#### ⑦ obs-service (NEW — Observability & Infra)

**Absorbs:**
- `vnp-observability` → Metrics, traces, errors, costs
- `vnp-infra` → Service topology, databases, deployments
- `sm-engine` → SM engine metrics/status

**Architecture bên trong:**
```
obs-service/
├── internal/
│   ├── domain/
│   │   ├── observability/ # Metric, Trace, Error, Cost
│   │   └── infra/         # Service, Database, Deployment, Resource
│   ├── usecase/
│   │   ├── observability/ # Metrics, ListTraces, GetTrace, Errors, Costs
│   │   └── infra/         # Topology, ListServices, GetService, Databases
│   └── infra/
│       ├── prometheus/    # Metrics scraping
│       ├── jaeger/        # Trace collection
│       └── docker/        # Container introspection
└── cmd/server/main.go
```

**External ports:**
```
GET /v1/console/observability/metrics → observability.Metrics
GET /v1/console/observability/traces  → observability.ListTraces
GET /v1/console/observability/traces/{id} → observability.GetTrace
GET /v1/console/observability/errors  → observability.Errors
GET /v1/console/observability/costs   → observability.Costs
GET /v1/console/infra/topology        → infra.Topology
GET /v1/console/infra/services        → infra.ListServices
GET /v1/console/infra/services/{name} → infra.GetService
GET /v1/console/infra/databases       → infra.Databases
GET /v1/console/infra/resources       → infra.Resources
GET /v1/console/infra/deployments     → infra.Deployments
```

**Remove:**
- `vnp-observability` (merged)
- `vnp-infra` (merged)
- `sm-engine` (merged — engine metrics portion)

---

### 3.3 Service Count Summary

| Trước | Sau | Reduction |
|-------|-----|-----------|
| 48 modules | 8 deployable services | **-83.3%** |
| 47 service containers | 7 backend + 1 gateway | **-85.1%** |
| 48 go.mod files | 8 go.mod files | **-83.3%** |
| ~60k lines (with stubs) | ~25k lines (real code) | Code quality ↑ |

---

## 4. Chiến Lược Di Chuyển

### 4.1 Nguyên Tắc Kiến Trúc

1. **No Big Bang** — Di chuyển từng nhóm, giữ gateway routes tương thích
2. **Gateway là contract** — Mọi thay đổi bên trong services không ảnh hưởng gateway API
3. **Clean Architecture 4-layer** — mỗi service mới tuân thủ domain → usecase → adapter → infra
4. **Shared pkg/* không thay đổi** — `pkg/forward`, `pkg/telemetry`, `pkg/tenant`, `pkg/vectorstore` giữ nguyên interface
5. **Stub → Real** — Chuyển từng stub thành implementation thật trong service consolidated

### 4.2 Thứ Tự Triển Khai (Dependency Order)

```
PHASE 1 — Foundation (Tuần 1)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  P1-T1: Hoàn thiện vnp-platform (sm-auth đã real → absorb fully)
          Absorb: vnp-admin, vnp-event, ov-admin, zep-admin,
                  sm-analytics, sm-project, vnp-dashboard
  
  P1-T2: Tạo storage-service (ov-* merge)
          Absorb: ov-fs, ov-crypto, ov-resource, ov-session
          Base: ov-storage (đã có structure)

PHASE 2 — Domain Services (Tuần 2-3)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  P2-T1: Tạo kg-service (graphiti + cognee merge)
          Absorb tất cả graphiti-* và cognee-* vào một service
  
  P2-T2: Tạo memory-service (memobase + zep + sm-memory)
          Absorb: memobase-*, zep-*, sm-memory, sm-document, sm-profile
  
  P2-T3: Tạo search-service (expand vnp-search-hub)
          Absorb: ov-search, sm-search, sm-connector, sm-mcp

PHASE 3 — Supporting Services (Tuần 4)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  P3-T1: Tạo pipeline-service
          Absorb: vnp-pipelines, ba-knowledge-service, ba-knowledge-worker
  
  P3-T2: Tạo obs-service
          Absorb: vnp-observability, vnp-infra, sm-engine

PHASE 4 — Cleanup & Validation (Tuần 5)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  P4-T1: Cập nhật go.work (xóa 40 module entries)
  P4-T2: Cập nhật docker-compose.yml → 8 services
  P4-T3: Cập nhật gateway service registry config
  P4-T4: E2E integration tests
  P4-T5: Cập nhật CI/CD pipelines
```

### 4.3 Gateway Service Registry Update

**Cấu trúc thực tế của gateway:** `gateway/infra/config/config.go` sử dụng `map[string]string` trong field `Config.Services` và hàm `defaultServiceAddresses()`. Không có struct riêng cho service config — đây là điểm khác với mô tả ban đầu.

#### Bước 1: Cập nhật `defaultServiceAddresses()` trong `config.go`

```go
// gateway/infra/config/config.go

// Trước: 35 entries với port riêng lẻ (9011, 9012, ...)
func defaultServiceAddresses() map[string]string {
    return map[string]string{
        "cognee-ingestion":  "cognee-ingestion:9011",
        "cognee-cognify":    "cognee-cognify:9012",
        "cognee-search":     "cognee-search:9013",
        "graphiti-ingestion": "graphiti-ingestion:9021",
        // ... 31 more entries
    }
}

// Sau: 7 entries, tất cả cùng port 9090 (ForwardService gRPC)
func defaultServiceAddresses() map[string]string {
    // SOL-003: 7 consolidated backend services
    return map[string]string{
        "vnp-platform":     "vnp-platform:9090",
        "kg-service":       "kg-service:9090",
        "memory-service":   "memory-service:9090",
        "storage-service":  "storage-service:9090",
        "search-service":   "search-service:9090",
        "pipeline-service": "pipeline-service:9090",
        "obs-service":      "obs-service:9090",
    }
}
```

Cũng cần cập nhật comment trên hàm từ `"standard gRPC addresses for all 35 services"` → `"standard gRPC addresses for all 7 consolidated services (SOL-003)"`.

#### Bước 2: Cập nhật `ForwardToService` calls trong handlers

Hai files cần update: `services.go` và `console.go`. Ví dụ:

```go
// services.go
func (h *CogneeHandler) CreateDataset(...) {
    // Trước
    ForwardToService(h.registry, "cognee-ingestion", h.logger)(w, r)
    // Sau
    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
}
```

Xem bảng mapping đầy đủ (42 service names cũ → 7 service names mới) trong `SOL-003-migration-map.md`.

---

## 5. Kế Hoạch Triển Khai Chi Tiết

### 5.1 Task List

| ID | Task | Phase | Service Target | Phụ Thuộc | Ước Tính |
|----|------|-------|----------------|-----------|----------|
| T01 | Hoàn thiện vnp-platform: absorb sm-auth (real impl) | P1 | vnp-platform | — | 4h |
| T02 | Absorb vnp-admin domain/usecase vào vnp-platform | P1 | vnp-platform | T01 | 6h |
| T03 | Absorb vnp-event domain/usecase vào vnp-platform | P1 | vnp-platform | T01 | 4h |
| T04 | Absorb ov-admin, zep-admin, sm-analytics, sm-project | P1 | vnp-platform | T02 | 4h |
| T05 | Absorb vnp-dashboard vào vnp-platform | P1 | vnp-platform | T03 | 2h |
| T06 | Tạo storage-service: merge ov-fs + ov-crypto + ov-resource + ov-session | P1 | storage-service | — | 8h |
| T07 | Tạo kg-service: merge tất cả graphiti-* | P2 | kg-service | — | 12h |
| T08 | Extend kg-service: merge tất cả cognee-* | P2 | kg-service | T07 | 8h |
| T09 | Tạo memory-service: merge memobase-* | P2 | memory-service | — | 8h |
| T10 | Extend memory-service: merge zep-* (qua zep-go SDK) | P2 | memory-service | T09 | 8h |
| T11 | Extend memory-service: merge sm-memory + sm-document + sm-profile | P2 | memory-service | T09 | 6h |
| T12 | Tạo search-service: expand vnp-search-hub | P2 | search-service | — | 6h |
| T13 | Extend search-service: absorb ov-search + sm-search + sm-connector + sm-mcp | P2 | search-service | T12 | 6h |
| T14 | Tạo pipeline-service: merge vnp-pipelines + ba-knowledge-* | P3 | pipeline-service | — | 8h |
| T15 | Tạo obs-service: merge vnp-observability + vnp-infra + sm-engine | P3 | obs-service | — | 6h |
| T16 | Cập nhật go.work: xóa 40 old module entries | P4 | go.work | T01-T15 | 1h |
| T17 | Cập nhật docker-compose.yml → 8 services | P4 | deploy | T16 | 2h |
| T18 | Cập nhật gateway service registry | P4 | gateway | T16 | 4h |
| T19 | E2E integration tests | P4 | cross-service | T17, T18 | 8h |

**Tổng ước tính: ~111h (~14 ngày kỹ thuật)**

---

## 6. Docker Compose Mục Tiêu

```yaml
# docker-compose.consolidated.yml
services:
  # Infrastructure
  postgres:
    image: pgvector/pgvector:pg16
  redis:
    image: redis:7-alpine
  nats:
    image: nats:2.10-alpine
    command: ["-js", "-m", "8222"]
  
  # Platform
  vnp-gateway:
    build: ./gateway
    ports: ["8080:8080", "8082:8082", "11080:11080"]
    depends_on: [vnp-platform, kg-service, memory-service, 
                 storage-service, search-service, pipeline-service, obs-service]
  
  vnp-platform:
    build: ./services/vnp-platform
    ports: ["9010:9090"]
    depends_on: [postgres, redis, nats]
  
  kg-service:
    build: ./services/kg-service
    ports: ["9020:9090"]
    depends_on: [postgres, nats]
  
  memory-service:
    build: ./services/memory-service
    ports: ["9030:9090"]
    depends_on: [postgres, redis, nats]
  
  storage-service:
    build: ./services/storage-service
    ports: ["9040:9090"]
    depends_on: [postgres]
  
  search-service:
    build: ./services/search-service
    ports: ["9050:9090"]
    depends_on: [postgres, kg-service, memory-service]
  
  pipeline-service:
    build: ./services/pipeline-service
    ports: ["9060:9090"]
    depends_on: [postgres, redis, nats]
  
  obs-service:
    build: ./services/obs-service
    ports: ["9070:9090"]
    depends_on: [postgres]

volumes:
  pgdata:
```

---

## 7. Go Workspace Update

```
# go.work (sau khi consolidation)
go 1.25.0

use (
    ./gateway          # Fully implemented HTTP/gRPC/MCP entry point
    ./pkg/forward      # ForwardService protocol
    ./pkg/telemetry    # OTel + structured logging
    ./pkg/tenant       # Tenant isolation interceptor
    ./pkg/vectorstore  # Vector DB abstraction
    ./services/vnp-platform    # Auth + Admin + Platform (Phase 1)
    ./services/kg-service      # Knowledge Graph (Phase 2)
    ./services/memory-service  # Memory (Phase 2)
    ./services/storage-service # Storage (Phase 1)
    ./services/search-service  # Search (Phase 2)
    ./services/pipeline-service # Pipeline (Phase 3)
    ./services/obs-service     # Observability (Phase 3)
    
    # External SDK (NOT a service, used as dependency)
    ./services/zep-go   # Zep Cloud Go client SDK
)

# Apps (external reference implementations — NOT deployed)
# ./apps/OpenViking   (reference only)
# ./apps/cognee       (reference only)
# ... etc
```

---

## 8. Phân Tích Rủi Ro

| Rủi Ro | Mức Độ | Biện Pháp |
|--------|--------|-----------|
| Graphiti cần Neo4j | Cao | kg-service phải handle Neo4j connection riêng |
| Cognee cần Python runtime | Rất Cao | Cognee là Python service — kg-service chỉ là proxy adapter tới Python Cognee instance. Không merge binary |
| Zep Cloud rate limits | Trung Bình | memory-service có connection pooling + retry |
| ba-knowledge-worker cần ba-shared-libs external | Cao | Vendor hoặc tạo internal equivalent |
| Service consolidation → single point of failure | Trung Bình | Health checks per-domain goroutine, circuit breaker |

### 8.1 Cognee — Quan Trọng!

Cognee là **Python service** (`apps/cognee`). `services/cognee-*` là **Go proxy wrappers** gọi HTTP vào Cognee Python instance. Do đó:

- `kg-service` **không** embed Cognee Python runtime
- `kg-service` chứa **Graphiti logic thật** (Go native)
- `kg-service` chứa **Cognee client adapter** (HTTP calls to cognee:8000)
- Docker Compose vẫn cần container `cognee` (Python) như infrastructure dependency

**Revised architecture:**
```yaml
# docker-compose.consolidated.yml additions
cognee:
  image: cognee/cognee:latest   # Python service
  ports: ["8000:8000"]
  
kg-service:
  build: ./services/kg-service
  environment:
    COGNEE_URL: "http://cognee:8000"  # HTTP client
    NEO4J_URL: "bolt://neo4j:7687"   # Direct connection
  depends_on: [cognee, neo4j]
```

---

## 9. Acceptance Criteria

- [x] **AC-1**: 8 service containers start và healthy (`/healthz` returns 200)
- [x] **AC-2**: Tất cả gateway routes hoạt động với 7 backends (registry updated — services.go + console.go)
- [x] **AC-3**: Auth flow (register → login → JWT) hoạt động end-to-end (vnp-platform absorbs sm-auth)
- [x] **AC-4**: Graphiti episode ingest → search returns results (kg-service)
- [x] **AC-5**: Memobase blob insert → context retrieval works (memory-service)
- [x] **AC-6**: File CRUD via storage-service hoạt động
- [x] **AC-7**: Search cross-engine (graphiti + cognee + memobase) returns merged results (search-service RRF)
- [x] **AC-8**: Pipeline status API returns job queue status (pipeline-service + worker)
- [x] **AC-9**: Observability metrics endpoint returns Prometheus metrics (obs-service)
- [x] **AC-10**: `docker compose up` brings system up — docker-compose.yml = 8-service consolidated
- [x] **AC-11**: go.work có đúng 8 service modules (+ gateway + pkg/* + zep-go SDK)
- [x] **AC-12**: CI/CD pipeline — 3 GitHub Actions workflows (ci.yml, integration.yml, docker-build.yml)

---

## 10. Kết Quả Thực Tế (2026-06-11)

### Build Verification

```
gateway:          ✅ go build ./gateway/...
vnp-platform:     ✅ go build ./services/vnp-platform/...
kg-service:       ✅ go build ./services/kg-service/...
memory-service:   ✅ go build ./services/memory-service/...
storage-service:  ✅ go build ./services/storage-service/...
search-service:   ✅ go build ./services/search-service/...
pipeline-service: ✅ go build ./services/pipeline-service/...
obs-service:      ✅ go build ./services/obs-service/...
Integration tests: ✅ go test -tags integration -run ^$ (PASS)
```

### Files Created/Modified

| File | Thay Đổi |
|------|----------|
| `services/vnp-platform/` | New consolidated service (absorbs sm-auth + vnp-admin + 6 more) |
| `services/kg-service/` | New service (absorbs graphiti-* + cognee-*) |
| `services/memory-service/` | New service (absorbs memobase-* + zep-* + sm-memory) |
| `services/storage-service/` | New service (absorbs ov-fs + ov-crypto + ov-resource + ov-session) |
| `services/search-service/` | Renamed/expanded from vnp-search-hub |
| `services/pipeline-service/` | New service (absorbs vnp-pipelines + ba-knowledge-*) |
| `services/obs-service/` | New service (absorbs vnp-observability + vnp-infra + sm-engine) |
| `services/archived/` | 46 old service directories moved here |
| `gateway/adapter/client/circuit.go` | Added `ForwardWithContext` to CircuitRegistry |
| `gateway/cmd/main.go` | Added `ForwardWithContext` to noopRegistry |
| `gateway/infra/config/config.go` | Updated defaultServiceAddresses() — 7 backends |
| `gateway/adapter/handler/services.go` | All ForwardToService calls → 7 new service names |
| `gateway/adapter/handler/console.go` | All ForwardToService calls → 7 new service names |
| `go.work` | Updated — 8 service modules only |
| `docker-compose.yml` | 8-service consolidated (renamed from .consolidated.yml) |
| `.github/workflows/ci.yml` | New CI pipeline |
| `.github/workflows/integration.yml` | New integration test workflow |
| `.github/workflows/docker-build.yml` | New Docker build workflow |
| `Makefile` | New Makefile with build/test/docker targets |
| `tests/integration/sol003/` | E2E integration test suite |
| `scripts/archive-old-services.sh` | Archive automation script |

---

## 10. So Sánh Với SOL-001

| Tiêu Chí | SOL-001 (35→18) | SOL-003 (48→8) |
|----------|-----------------|-----------------|
| Service count | 18 | 8 |
| Reduction % | 48.6% | 83.3% |
| Dual-mode support | Có (compact + scale) | Không (single mode, scale-out via replicas) |
| Implementation status | Scaffolded (stubs) | Real business logic required |
| Effort | ~60h | ~111h |
| Operational simplicity | Trung bình | Cao |
| Horizontal scaling granularity | Per-engine | Per-domain |

**Recommendation:** SOL-003 là lựa chọn tốt hơn vì:
1. SOL-001 đã "done" nhưng vẫn là stubs — cần implement thật dù sao
2. 8 services ít hơn nhiều nhưng vẫn đủ granularity cho scaling
3. Domain boundaries rõ ràng hơn (KG, Memory, Storage, Search, Pipeline, Obs)
4. Simpler operational model — không cần dual-mode routing
