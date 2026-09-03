# VNP Memory — System Architecture

> **Version**: 4.0 | **Date**: 2026-06-18
> **Status**: Active (reflects current codebase)
> **Source**: `apps/memory/`, `gateway/`, `services/`

---

## 1. Deployment Modes

VNP Memory hỗ trợ hai chế độ triển khai:

| Mode | Description | Entry Point |
|------|-------------|-------------|
| **Monolith** | Single binary, all services in-process | `apps/memory/cmd/server/main.go` |
| **Gateway Only** | API gateway kết nối external services qua gRPC | `gateway/cmd/main.go` |

---

## 2. Monolith Architecture (`apps/memory`)

```
┌──────────────────────────────────────────────────────────────────────┐
│                      VNP Memory Monolith                             │
│                                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ REST API │  │ MCP SSE  │  │ Health   │  │ gRPC Bus │            │
│  │ :8080    │  │ :8082    │  │ :8083    │  │ bufconn  │            │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘            │
│       │              │              │              │                  │
│  ┌────▼──────────────▼──────────────▼──────────────▼──────────┐     │
│  │             Gateway (Router + Handlers)                     │     │
│  │  150+ REST routes · 53 MCP tools · WebSocket · SSE         │     │
│  └──────────────────────────┬──────────────────────────────────┘     │
│                             │ InProcessRegistry (bufconn)            │
│  ┌──────────────────────────▼──────────────────────────────────┐     │
│  │                  Engine Services                             │     │
│  │  ┌─ Memory Engines ─────────────────────────────────────┐   │     │
│  │  │  Cognee(4)  Graphiti(6)  Memobase(6)                 │   │     │
│  │  │  OpenViking(7) Zep(6)  Supermemory(9)                │   │     │
│  │  └─────────────────────────────────────────────────────┘   │     │
│  │  ┌─ AgentMemory Engine ─────────────────────────────────┐   │     │
│  │  │  am-observe  am-memory  am-search  am-orchestration  │   │     │
│  │  └─────────────────────────────────────────────────────┘   │     │
│  │  ┌─ Platform ───────────────────────────────────────────┐   │     │
│  │  │  vnp-admin  vnp-event  vnp-search-hub  vnp-platform  │   │     │
│  │  │  vnp-dashboard  kg-service  ba-knowledge-service     │   │     │
│  │  └─────────────────────────────────────────────────────┘   │     │
│  └───────────────────────────┬──────────────────────────────────┘     │
│                              │                                        │
│  ┌───────────────────────────▼──────────────────────────────────┐     │
│  │              Embedded NATS + JetStream                        │     │
│  │   7+ streams · WorkQueue retention · in-process              │     │
│  └──────────────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────────────┘
           │           │           │         │         │
      PostgreSQL    Neo4j       Qdrant    Redis    MinIO
      (pgvector)
```

### 2.1 In-Process Service Registry

Gateway giao tiếp với engine services qua `InProcessRegistry` sử dụng `bufconn` (gRPC in-memory transport). Không có network hop giữa gateway và services trong monolith mode.

### 2.2 Bootstrap Modules

| Bootstrap File | Engine Initialized |
|---------------|-------------------|
| `cognee.go` | cognee-ingestion, cognee-cognify, cognee-search, cognee-pipeline |
| `graphiti.go` | graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store, graphiti-admin, graphiti-pipeline |
| `memobase.go` | memobase-ingestion, memobase-engine, memobase-context, memobase-event, memobase-admin, memobase-pipeline |
| `openviking.go` | ov-fs, ov-search, ov-session, ov-resource, ov-crypto, ov-admin, ov-storage |
| `zep.go` | zep-user, zep-thread, zep-memory, zep-graph, zep-search, zep-admin |
| `supermemory.go` | sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-mcp, sm-auth, sm-analytics, sm-project |
| `memory.go` | am-observe, am-memory, am-search (AgentMemory engine) |
| `orchestration.go` | am-orchestration (Actions, Leases, Signals, Routines, Checkpoints, Sentinels, Sketches) |
| `platform.go` | vnp-dashboard, vnp-search-hub, graphiti-store (console APIs via ForwardService) |
| `observe_search.go` | observe-search (BM25 + vector hybrid search for observations) |

### 2.3 Services Inventory

#### Memory Engines (35 services — legacy)

| Engine | Services | Count |
|--------|----------|-------|
| **Cognee** | cognee-ingestion, cognee-cognify, cognee-search, cognee-pipeline | 4 |
| **Graphiti** | graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store, graphiti-admin, graphiti-pipeline | 6 |
| **Memobase** | memobase-ingestion, memobase-engine, memobase-context, memobase-event, memobase-admin, memobase-pipeline | 6 |
| **OpenViking** | ov-fs, ov-search, ov-session, ov-resource, ov-crypto, ov-admin, ov-storage | 7 |
| **Zep** | zep-user, zep-thread, zep-memory, zep-graph, zep-search, zep-admin | 6 |
| **Supermemory** | sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-mcp, sm-auth, sm-analytics, sm-project | 9 |
| **Platform** | vnp-admin, vnp-event, vnp-search-hub, vnp-platform | 4 |

> **Note**: Số service đã tăng từ 35 lên 42 do thêm các service mới (pipeline, admin, event, storage cho từng engine).

#### AgentMemory Engine (4 services — mới)

| Service | Module | Chức năng |
|---------|--------|-----------|
| **am-observe** | `observe-service` | Thu thập observations từ AI agent hooks (tool calls, prompts, responses) |
| **am-memory** | `memory-service/agentmemory` | Long-term agent memory với decay scoring và eviction |
| **am-search** | `observe-search` | BM25 + vector hybrid search cho observations |
| **am-orchestration** | `orchestration-service` | Actions, Leases, Signals, Routines, Checkpoints, Sentinels, Sketches, Crystals |

#### Platform Services (mới thêm)

| Service | Module | Chức năng |
|---------|--------|-----------|
| **vnp-dashboard** | `vnp-dashboard` | Console dashboard: health, metrics, throughput, heatmap |
| **kg-service** | `kg-service` | Knowledge graph service (BA/OSS layer) |
| **ba-knowledge-service** | `ba-knowledge-service` | BA knowledge integration service |
| **ba-knowledge-worker** | `ba-knowledge-worker` | Background worker cho BA knowledge indexing |

---

## 3. Gateway Architecture (`gateway/`)

```
┌────────────────────────────────────────────────────────────┐
│                    VNP Gateway :8080                       │
│                                                            │
│  ┌─ REST ──┐  ┌─ gRPC ──┐  ┌─ MCP ───┐  ┌─ WebDAV ─┐    │
│  │  :8080  │  │  :8081  │  │  :8082  │  │  :8080   │    │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬─────┘    │
│       │            │            │             │           │
│  ┌────┴────────────┴────────────┴─────────────┴──┐       │
│  │           Middleware Pipeline                  │       │
│  │  Recovery → RequestID → Logger → CORS → Auth  │       │
│  └───────────────────┬──────────────────────────┘       │
│                      │                                    │
│  ┌───────────────────┴──────────────────────────────┐    │
│  │            Usecase Layer                          │    │
│  │   RouteUC   │  AuthUC   │  RateLimitUC           │    │
│  └──────┬──────┴──────┬────┴───────┬────────────────┘    │
│         │             │            │                      │
│  ┌──────┴─────────────┴────────────┴──────────────────┐  │
│  │       Infrastructure Layer                          │  │
│  │  GRPCRegistry → CircuitBreaker → All Services      │  │
│  │  Redis (RateLimit) │ Postgres (Tenants/Keys)       │  │
│  │  NATS (Events)     │ Prometheus (Metrics)          │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

### 3.1 Gateway Domain Model

```
domain/entity.go
├── AuthContext          — JWT/API Key identity (TenantID, UserID, Roles, Scopes, RateTier)
├── TenantContext        — Tenant config (ID, Name, RateTier, Enabled, CreatedAt)
├── RouteTarget          — Downstream gRPC endpoint (Service, Address, Timeout, Method)
├── ProtocolType         — REST | gRPC | MCP | WebDAV | WebSocket
├── StoreRequest         — Unified memory store request (Type, Content, Metadata, SourceID, UserID)
├── RouteResult          — Route outcome (ID, Engine, Status, Body, LatencyMs)
├── ForwardRequest       — HTTP→downstream forward (Path, HTTPMethod, Body, PathParams, QueryParams)
└── MemoryType constants — semantic | episodic | conversational | profile | procedural | auto

domain/console.go
├── ConsoleRouteTarget   — Routes tới console-tier services
├── DashboardMetrics     — Aggregated dashboard data
└── MemoryGraphNode      — Graph node for console explorer

domain/event.go (NATS subjects)
├── RequestReceived      — gateway.request.received
├── RequestRouted        — gateway.request.routed
├── AuthFailed           — gateway.auth.failed
├── RateLimitExceeded    — gateway.ratelimit.exceeded
└── CircuitOpened        — gateway.circuit.opened
```

---

## 4. API Namespaces & Routes

### 4.1 Engine API Routes (50+ routes)

| Prefix | Engine | Routes | Handler |
|--------|--------|--------|---------|
| `/v1/memory/*` | Auto-route | 4 | store, recall, forget, timeline |
| `/v1/cognee/*` | Cognee | 4 | datasets, cognify, search |
| `/v1/graphiti/*` | Graphiti | 4 | episodes, search, nodes, edges |
| `/v1/memobase/*` | Memobase | 5 | blobs, flush, context, profiles, events |
| `/v1/ov/*` | OpenViking | 10 | files, tree, grep, sessions, resources |
| `/v1/zep/*` | Zep | 9 | users, memory, graph search, facts, ontology |
| `/v1/sm/*` | Supermemory | 9 | documents, memories, search, RAG, profiles, connectors |
| `/v1/admin/*` | Platform | 4 | tenants, API keys, health, metrics |

### 4.2 AgentMemory API Routes (mới — 50+ routes)

| Prefix | Engine | Routes | Description |
|--------|--------|--------|-------------|
| `/v1/observe/sessions` | am-observe | 8 | Session CRUD, observations, stream events |
| `/v1/observe/session/{start,end}` | am-observe | 2 | Hook-style session lifecycle |
| `/v1/observe/replay/*` | am-observe | 2 | Session replay timeline |
| `/v1/observe/search/*` | am-search | 8 | Smart, BM25, vector search; index management |
| `/v1/observe` | am-observe | 1 | Single hook endpoint |
| `/v1/memory/agent/*` | am-memory | 8 | Remember, list, get, delete, retention, evict, auto-forget |
| `/v1/memory/slots/*` | am-memory | 4 | MemorySlot CRUD (project/global scope) |
| `/v1/memory/agent/governance` | am-memory | 2 | GDPR delete, audit log |
| `/v1/memory/compress` | am-memory | 1 | Compress raw observations |
| `/v1/memory/summarize` | am-memory | 1 | Summarize session |
| `/v1/memory/consolidate` | am-memory | 1 | Run consolidation pipeline |
| `/v1/memory/procedural` | am-memory | 2 | List, get procedural memories |
| `/v1/memory/lessons` | am-memory | 4 | Lessons CRUD + decay sweep |
| `/v1/memory/insights` | am-memory | 1 | List insights |
| `/v1/stream` | am-observe | 1 | SSE real-time session events |
| `/v1/orchestration/actions` | am-orchestration | 5 | Actions CRUD |
| `/v1/orchestration/leases` | am-orchestration | 4 | Lease acquire/renew/release |
| `/v1/orchestration/signals` | am-orchestration | 4 | Signal send/list/read/delete |
| `/v1/orchestration/routines` | am-orchestration | 3 | Routines + execute |
| `/v1/orchestration/checkpoints` | am-orchestration | 4 | Checkpoint CRUD + approve/reject |
| `/v1/orchestration/sentinels` | am-orchestration | 3 | Sentinel CRUD |
| `/v1/orchestration/sketches` | am-orchestration | 5 | Sketches + promote to Crystal |
| `/v1/orchestration/crystals` | am-orchestration | 2 | Crystal list, get |
| `/v1/health` | platform | 1 | Health snapshot |
| `/v1/admin/doctor` | platform | 1 | Doctor check |
| `/v1/admin/snapshot` | platform | 2 | Snapshot create/list |
| `/v1/admin/plugin/*` | platform | 4 | Plugin config (claude-code, codex, opencode) + install |

### 4.3 Console API Routes (FEAT-006 to FEAT-017)

| Prefix | Feature | Routes | Description |
|--------|---------|--------|-------------|
| `/v1/console/dashboard/*` | FEAT-006 | 4 | health, metrics, throughput, heatmap |
| `/v1/console/memory/*` | FEAT-007 | 4 | search, get, neighbors, versions |
| `/v1/console/graph/*` | FEAT-013 | 6 | subgraph, entity, timeline, ontology, query |
| `/v1/console/profiles/*` | FEAT-008 | 7 | list, config, get, events, context, buffers |
| `/v1/console/adaptive/*` | FEAT-009 | 8 | memories, versions, connectors, sync, analytics, forget-rules |
| `/v1/console/debugger/*` | FEAT-010 | 3 | traces CRUD |
| `/v1/console/sessions/*` | FEAT-014 | 7 | list, live, get, timeline, diff, working-memory, user-summary |
| `/v1/console/governance/*` | FEAT-011 | 9 | tenants, policies, audit, GDPR forget |
| `/v1/console/pipelines/*` | FEAT-015 | 7 | status, queues, workers, templates, jobs |
| `/v1/console/infra/*` | FEAT-016 | 6 | topology, services, databases, resources, deployments |
| `/v1/console/observability/*` | FEAT-017 | 5 | metrics, traces, errors, costs |
| `/v1/console/ws` | FEAT-012 | 1 | WebSocket real-time stream |

### 4.4 MCP Server (53 Tools)

MCP server phục vụ trên port `:8082`. Hỗ trợ **SSE** và **HTTP Streamable** transport. Tổng cộng 53 tools: 16 legacy + 37 AgentMemory.

#### Legacy MCP Tools (16 tools)

| Tool | Target Service | Description |
|------|---------------|-------------|
| `memory_store` | cognee-ingestion | Store memory với auto-classification |
| `memory_recall` | vnp-search-hub | Cross-engine semantic recall |
| `memory_search` | cognee-search | Knowledge graph search |
| `memory_timeline` | vnp-event | Temporal event query |
| `memory_profile` | memobase-context | User profile từ memory context |
| `memory_forget` | vnp-event | Cascading delete across engines |
| `graph_query` | graphiti-store | Knowledge graph với filters |
| `ov_read_file` | ov-fs | Đọc file từ context DB |
| `ov_write_file` | ov-fs | Ghi content vào context DB |
| `ov_search` | ov-search | Hierarchical semantic search |
| `ov_list_dir` | ov-fs | List directory contents |
| `ov_grep` | ov-fs | Regex file search |
| `ov_tree` | ov-fs | Directory tree structure |
| `ov_session_commit` | ov-session | Commit editing session |
| `ov_ingest` | ov-resource | Ingest resource vào context DB |
| `ov_delete` | ov-fs | Xóa file hoặc resource |

#### AgentMemory MCP Tools (37 tools — mới)

Đăng ký qua `RegisterAllAgentMemoryTools()` với 9 nhóm:

| Nhóm | Tools | Description |
|------|-------|-------------|
| **MemoryCore** | remember, list, get, delete, evict, auto-forget, retention | Agent memory CRUD + lifecycle |
| **Session** | list_slots, get_slot, write_slot, delete_slot | MemorySlot management |
| **Observe** | start_session, observe, end_session, get_session, list_sessions, get_observations, stream | Session observation |
| **Governance** | governance_delete, list_audit | GDPR + audit |
| **Graph** | (graph-linked tools) | Graph queries cho agent context |
| **Orchestration** | create_action, get_action, list_actions, update_action, delete_action | Action management |
| **Signal** | acquire_lease, renew_lease, release_lease, send_signal, list_signals | Inter-agent communication |
| **ReplaySlot** | list_replay_sessions, load_replay_timeline | Session replay |
| **Admin** | get_health, doctor_check, create_snapshot, get_plugin_config, install_plugin | System admin |

---

## 5. Service Layer Architecture (`services/`)

### 5.1 Directory Map

```
services/
├── memory-service/          — AgentMemory + Memobase + Zep + Supermemory domain models
│   └── internal/
│       ├── domain/
│       │   ├── agentmemory/ — AgentMemory, MemorySlot, ProceduralMemory, Lesson, Insight
│       │   ├── memobase/    — Blob, UserContext, Profile, Event, Buffer
│       │   ├── zep/         — ZepUser, ZepSession, ZepMemory, ZepMessage, GraphFact
│       │   └── sm/          — SMMemory, SMDocument, SMProfile, RAGResponse
│       ├── consolidation/   — Session consolidation pipeline
│       ├── usecase/
│       │   ├── agentmemory/ — RememberUC, EvictUC, AutoForgetUC, RetentionUC, SlotsUC
│       │   ├── memobase/    — IngestUseCase, ContextUseCase
│       │   ├── zep/         — UserService, MemoryService
│       │   └── sm/          — MemoryService
│       └── infra/           — PostgreSQL repos, migrations
│
├── observe-service/         — AI agent session observation
│   └── internal/
│       ├── domain/          — Session, RawObservation, CompressedObservation
│       ├── observe/         — Observation pipeline (dedup, redact, compress)
│       ├── replay/          — Session replay timeline
│       └── usecase/         — ObserveUC, CreateSessionUC, EndSessionUC
│
├── orchestration-service/   — Multi-agent coordination primitives
│   └── internal/
│       ├── domain/          — Action, Lease, Signal, Routine, Checkpoint, Sentinel, Sketch, Crystal
│       ├── orchestration/   — Service implementations
│       └── usecase/         — Action/Lease/Signal/Checkpoint/Sentinel/Sketch use cases
│
├── pipeline-service/        — Pipeline & Job management
│   └── internal/domain/pipeline/
│       └── Pipeline, Job, Queue, Worker, PipelineTemplate
│
├── obs-service/             — Observability infrastructure
│   └── internal/domain/observability/
│
├── observe-search/          — BM25 + vector hybrid search cho observations
│   └── internal/
│       ├── domain/          — SearchResult, SearchQuery, IndexStats
│       └── usecase/         — SmartSearchUC, BM25SearchUC, VectorSearchUC
│
├── search-service/          — Cross-engine search & connectors
│   └── internal/domain/
│       ├── search/          — Multi-strategy search
│       ├── connector/       — External data connectors
│       └── mcp/             — MCP tool definitions
│
├── storage-service/         — Filesystem & resource management
│   └── internal/domain/
│       ├── fs/              — File entities (VikingFS)
│       ├── resource/        — Resource ingestion
│       └── session/         — Session lifecycle
│
├── vnp-platform/            — Admin, Auth, Events, Analytics
│   └── internal/domain/
│       ├── admin/           — Tenant, User, APIKey, HealthStatus
│       ├── auth/            — JWT, API key validation
│       ├── event/           — UserEvent, Timeline, EventType
│       ├── analytics/       — Usage metrics
│       ├── model/           — Shared models
│       └── project/         — Project/space management
│
├── kg-service/              — Knowledge Graph service (BA/OSS layer)
│   ├── graph/               — Graph engine integration
│   ├── parsers/             — Document parsers
│   └── templates/           — Graph templates
│
├── ba-knowledge-service/    — BA knowledge integration API
├── ba-knowledge-worker/     — Background worker cho BA knowledge indexing
│
└── [per-engine microservices] — Standalone services (khi deploy distributed)
    ├── cognee-{ingestion,cognify,search,pipeline}
    ├── graphiti-{ingestion,search,knowledge,store,admin,pipeline}
    ├── memobase-{ingestion,engine,context,event,admin,pipeline}
    ├── ov-{fs,search,session,resource,crypto,admin,storage}
    ├── zep-{user,thread,memory,graph,search,admin}
    ├── sm-{document,memory,search,profile,connector,mcp,auth,analytics,project}
    └── vnp-{admin,event,search-hub,platform,dashboard,infra,observability,pipelines}
```

### 5.2 Domain Entity Summary

#### AgentMemory Domain (`memory-service/domain/agentmemory`) — **MỚI**

```go
AgentMemory   — ID, TenantID, Project, Type(pattern|preference|architecture|bug|workflow|fact)
              — Title, Content, Concepts[], Files[], SessionIDs[], Strength(0-1)
              — Version, ParentID, Supersedes[], RelatedIDs[], SourceObservationIDs[]
              — IsLatest, ForgetAfter, AgentID, FlaggedEviction

MemorySlot    — TenantID, Project, Scope(project|global), Label, Content
              — Description, SizeLimit, Pinned, ReadOnly

ProceduralMemory — ID, TenantID, Project, Name, Steps[], StepHash(dedup)
                 — TriggerCondition, ExpectedOutcome, Frequency, Confidence

Lesson        — ID, TenantID, Project, Content, Confidence, Source
              — Categories[], AccessCount

Insight       — ID, TenantID, Content, LessonIDs[], Confidence

RawObs        — ID, SessionID, TenantID, HookType, ToolName, ToolInput, ToolOutput
              — UserPrompt, AssistantResponse

CompressedObs — ID, SessionID, ObsType, Title, Subtitle, Facts[], Narrative
              — Concepts[], Files[], Importance, Confidence
```

#### Observe Domain (`observe-service/domain`) — **MỚI**

```go
Session             — ID, TenantID, Project, CWD, Model, AgentID
                    — Status(active|completed|abandoned), FirstPrompt, Summary
                    — ObservationCount, Tags[], CommitSHAs[], StartedAt, EndedAt

RawObservation      — ID, SessionID, TenantID, HookType, ToolName
                    — ToolInput/Output(JSON), UserPrompt, AssistantResponse
                    — Modality(text|image), ImageData, AgentID, Raw, Timestamp

CompressedObservation — ID, SessionID, ObsType, Title, Subtitle
                      — Facts[], Narrative, Concepts[], Files[]
                      — Importance, Confidence, ImageRef, AgentID, Timestamp
```

#### Orchestration Domain (`orchestration-service/domain`) — **MỚI**

```go
Action        — ID, TenantID, Project, AgentID, Title, Description
              — Status(pending|in_progress|done|failed|cancelled), Priority
              — Requires[], ConflictsWith[], Tags[], Result

Lease         — ID, ActionID, AgentID, Status(active|expired|released)
              — AcquiredAt, ExpiresAt, RenewedAt

Signal        — ID, TenantID, FromAgent, ToAgent
              — SignalType(handoff|update|cancel|request|response|alert)
              — Content, ThreadID, ReplyTo, IsRead, ExpiresAt

Routine       — ID, TenantID, Project, Name, Description, Steps[]

Checkpoint    — ID, TenantID, Project, AgentID, ActionID, Title, Description
              — Status(pending|approved|rejected|expired), ApprovedBy, RejectedBy

Sentinel      — ID, TenantID, Name, Condition(SentinelCondition), ActionID, SignalTo
              — Status(watching|triggered|expired), ExpiresAt, TriggeredAt

SentinelCondition — Type(action_done|signal_received|time), Target, Value

Sketch        — ID, TenantID, Project, SessionID, Title
              — ActionIDs[], Status(active|promoted|expired), ExpiresAt

Crystal       — ID, TenantID, SourceActionIDs[], Narrative
              — KeyOutcomes[], FilesAffected[], Lessons[]
```

#### Admin Domain (`vnp-platform/domain/admin`)
```go
Tenant        — ID, Name, Slug, Tier(free|pro|enterprise), Status(active|suspended|deleted)
User          — ID, TenantID, Email, Role(admin|editor|viewer)
APIKey        — ID, TenantID, KeyHash(SHA-256), Permissions, ExpiresAt
HealthStatus  — Service, Status(SERVING|NOT_SERVING|UNKNOWN), Latency
```

#### Event Domain (`vnp-platform/domain/event`)
```go
UserEvent  — TenantID, UserID, Engine, EventType(ingestion|search|memory|profile|admin), Action, GistText
Timeline   — TenantID, UserID, Events[]
```

#### Memobase Domain (`memory-service/domain/memobase`)
```go
Blob        — ID, UserID, TenantID, Type(conversation|fact|document|image), Content, Embedding
UserContext — UserID, TenantID, Summary, Profiles[], Events[], Tokens
Profile     — Key, Value, Category(preference|fact|goal|habit), Score
Event       — ID, UserID, EventType, Content
Buffer      — UserID, Blobs[], TokenCount, FlushThreshold(default:20)
```

#### Zep Domain (`memory-service/domain/zep`)
```go
ZepUser     — UserID, Email, FirstName, LastName
ZepSession  — SessionID, UserID, Metadata
ZepMemory   — SessionID, Messages[], Summary, Facts[]
ZepMessage  — Role(user|assistant), Content
GraphFact   — UUID, Name, Fact, Episodes[]
```

#### Supermemory Domain (`memory-service/domain/sm`)
```go
SMMemory   — ID, TenantID, Content, Tags[], Embedding
SMDocument — ID, TenantID, Title, Content, Type(markdown|pdf|html), URL
SMProfile  — UserID, TenantID, Memories[], Stats
RAGResponse — Context, Sources[], Tokens
```

#### Pipeline Domain (`pipeline-service/domain/pipeline`)
```go
Pipeline         — Engine, Status(idle|running|paused|error), JobCount, Workers[]
Job              — ID, Engine, Type(ingest|index|sync|cognify), Status, Priority
Queue            — Name, Engine, Size, MaxSize, Workers
Worker           — ID, Engine, Status(idle|busy|offline)
PipelineTemplate — ID, Name, Engine, Config
```

---

## 6. Data Flow

### 6.1 Memory Store Flow

```
Client → POST /v1/memory/store
    → Gateway (MemoryHandler.Store)
        → Auto-route by type:
            semantic      → cognee-ingestion
            episodic      → graphiti-ingestion
            conversational → zep-memory
            profile       → memobase-ingestion (Blob → Buffer)
            procedural    → ov-fs
            adaptive      → sm-memory
        → NATS publish: memory.blob.inserted
```

### 6.2 AgentMemory Observation Flow (mới)

```
AI Agent Hook → POST /v1/observe/sessions/{id}/observe
    → am-observe (ObserveUseCase)
        → DedupMap.Check()       — loại bỏ duplicate observations
        → PrivacyRedactor        — xóa PII/sensitive data
        → KVStore.Save()         — lưu RawObservation
        → SearchClient.Index()   — đánh index BM25+vector
        → Publisher.Publish("agentmemory.observation.captured")
        → StreamBroker → SSE clients

agentmemory.observation.captured (NATS)
    → am-memory-consolidator     — consolidate → AgentMemory records
    → am-search-indexer          — update search index
```

### 6.3 Memory Consolidation Flow (mới)

```
POST /v1/memory/consolidate
    → am-memory (ConsolidationPipeline)
        → CompressObservations   — RawObs → CompressedObs (via LLM Bifrost)
        → ExtractPatterns        → ProceduralMemory records
        → ExtractLessons         → Lesson records
        → BuildInsights          → Insight records (cross-lesson synthesis)
        → SessionSummary         → Narrative + KeyDecisions + FilesModified

POST /v1/orchestration/sketches/{id}/promote
    → am-orchestration (SketchService)
        → LLM synthesis (Bifrost) → Crystal (Narrative + KeyOutcomes + Lessons)
```

### 6.4 Orchestration Flow (mới)

```
Agent A → POST /v1/orchestration/actions       — tạo Action
Agent A → POST /v1/orchestration/leases/acquire — lock Action
Agent A → POST /v1/orchestration/signals/send   — gửi Signal đến Agent B
Agent B → POST /v1/orchestration/leases/renew   — gia hạn Lease
Agent B → POST /v1/orchestration/checkpoints    — yêu cầu human approval

Background Sweeper:
    → LeaseService.ExpireLeases()
    → SignalService.PurgeExpired()
    → SentinelService.EvaluateSentinels()
    → SketchService.ExpireSketches()
    → CheckpointService.ExpireCheckpoints()
```

### 6.5 Memobase Buffer Flow

```
Blob Insert → Buffer (in-memory)
    → TokenCount >= FlushThreshold(20) → Auto-flush
    → ProcessBuffer:
        extract → merge → profile update
    → UserContext assembled < 100ms
```

### 6.6 Cross-Engine Recall Flow

```
Client → POST /v1/memory/recall
    → vnp-search-hub
        → [parallel gRPC via bufconn]
            cognee-search + graphiti-search + memobase-context
            ov-search + zep-search + sm-search
        → Merge + rerank → Response
```

### 6.7 Pipeline Completion Events (NATS)

| Engine | NATS Subject |
|--------|-------------|
| Cognee | `cognee.pipeline.completed` |
| Graphiti | `graphiti.episode.completed` |
| Memobase | `memobase.pipeline.completed`, `memobase.profile.changed` |
| OpenViking | `ov.content.written`, `ov.content.deleted`, `ov.resource.ingested`, `ov.session.*` |
| Zep | `zep.memory.messages.ingested`, `zep.user.deleted`, `zep.graph.*`, `zep.search.*` |
| Supermemory | `sm.engine.*`, `sm.connector.synced`, `sm.document.created`, `sm.memory.updated` |
| Admin | `admin.tenant.created`, `admin.tenant.deleted` |
| Gateway | `gateway.request.received`, `gateway.request.routed`, `gateway.auth.failed`, `gateway.ratelimit.exceeded`, `gateway.circuit.opened` |
| **AgentMemory** | `agentmemory.observation.captured`, `agentmemory.session.started`, `agentmemory.session.ended` |
| **Orchestration** | `orchestration.action.*`, `orchestration.signal.*`, `orchestration.checkpoint.*` |

---

## 7. Technology Stack

| Category | Technology | Version/Notes |
|----------|-----------|---------------|
| Language | Go | 1.23+ |
| HTTP Router | net/http ServeMux | stdlib (Go 1.22+ pattern matching) |
| RPC | gRPC + Protobuf | v3, bufconn for in-process |
| Message Broker | NATS JetStream | 2.10+ (embedded or external) |
| Database (relational) | PostgreSQL + pgvector | 17, vector extension |
| Database (graph) | Neo4j | 5+ |
| Vector Store | Qdrant | External |
| Cache | Redis | 7+ |
| Object Storage | MinIO (S3 compat) | — |
| Custom Storage | VikingFS | Go-native FS |
| LLM Gateway | Bifrost | Multi-provider routing (consolidation + sketch→crystal) |
| Observability | OpenTelemetry + Prometheus | — |
| Logging | slog (structured JSON) | stdlib |
| Auth | JWT RS256 + API Key (SHA-256) | — |
| Search | BM25 + pgvector | Hybrid trong observe-search |

---

## 8. Infrastructure Dependencies

| Backend | Used By | Notes |
|---------|---------|-------|
| PostgreSQL + pgvector | All services | Primary relational + vector store |
| Neo4j | cognee, graphiti, zep-graph | Graph-native storage |
| Redis | All services | Cache, rate limit, job queues |
| NATS JetStream | All pipeline + search services | Async events (embedded in monolith) |
| MinIO/S3 | cognee-ingestion, sm-connector | Object/document storage |
| VikingFS | ov-fs, ov-storage | Go-native filesystem |
| Qdrant | Vector search (optional) | Can use pgvector instead |
| Bifrost | am-memory, am-orchestration | LLM Gateway cho consolidation + synthesis |

---

## 9. Deployment Architecture

### 9.1 Development (Monolith + Docker Compose)

```bash
# Start infra (5 containers)
make infra-up   # PostgreSQL, Neo4j, Qdrant, Redis, MinIO

# Run monolith (single process, all services in-memory)
make dev
```

Ports:
- `:8080` — REST API (150+ routes)
- `:8082` — MCP Server (SSE + HTTP Streamable, 53 tools)
- `:8083` — Health + Metrics

### 9.2 Production (Gateway + External Services)

```
vnp-gateway (:8080-8082)
    └── gRPC → distributed engine services
              → PostgreSQL cluster
              → Neo4j cluster
              → Redis cluster
              → NATS cluster
              → Bifrost LLM Gateway
```

### 9.3 Configuration

```
Pattern: VNP_MEMORY_<SECTION>_<KEY>
VNP_MEMORY_SERVER_REST_PORT=8080
VNP_MEMORY_SERVER_MCP_PORT=8082
VNP_MEMORY_SERVER_HEALTH_PORT=8083
VNP_MEMORY_AUTH_DEV_MODE=true
VNP_MEMORY_NATS_MODE=embedded|external
VNP_MEMORY_POSTGRES_DSN=postgres://...
VNP_MEMORY_BIFROST_URL=http://bifrost:8080
```

---

## 10. Project Structure

```
vnp-memory/
├── apps/memory/              # Monolith entry point
│   ├── cmd/server/main.go
│   ├── configs/config.yaml
│   └── internal/
│       ├── bootstrap/        # Service initialization
│       │   ├── gateway.go    #   Gateway + MCP + Auth middleware
│       │   ├── cognee.go     #   Cognee engine (4 services)
│       │   ├── graphiti.go   #   Graphiti engine (6 services)
│       │   ├── memobase.go   #   Memobase engine (6 services)
│       │   ├── openviking.go #   OpenViking engine (7 services)
│       │   ├── zep.go        #   Zep engine (6 services)
│       │   ├── supermemory.go#   Supermemory engine (9 services)
│       │   ├── memory.go     #   AgentMemory engine (am-observe/memory/search)
│       │   ├── orchestration.go # am-orchestration
│       │   ├── observe_search.go# observe-search service
│       │   └── platform.go   #   vnp-dashboard, vnp-search-hub
│       ├── bus/              # InProcessRegistry, gRPC bufconn, embedded NATS
│       ├── config/           # Unified config
│       └── ui/               # Embedded SPA assets
│
├── gateway/                  # Standalone gateway
│   ├── adapter/
│   │   ├── handler/          # HTTP handlers
│   │   │   ├── router.go     #   150+ routes (engine + agentmemory + console)
│   │   │   ├── services.go   #   Engine handlers (Cognee, Graphiti, Memobase, OV, Zep, SM, Admin)
│   │   │   ├── agentmemory.go#   AgentMemory handler (~80 route methods)
│   │   │   ├── console.go    #   Console API handlers
│   │   │   └── ws.go         #   WebSocket handler
│   │   ├── mcp/              # MCP server
│   │   │   ├── server.go     #   MCP SSE + HTTP Streamable (16 legacy tools)
│   │   │   └── tools/agentmemory/ # 37 AgentMemory tools
│   │   ├── client/           # gRPC client adapters
│   │   └── webdav/           # WebDAV proxy
│   └── internal/
│       ├── domain/           # AuthContext, RouteTarget, ConsoleRoute, MemoryType
│       ├── usecase/          # RouteUC, AuthUC, RateLimitUC + ports
│       └── infra/            # config, middleware, persistence, server
│
└── services/                 # Domain service implementations
    ├── memory-service/       # AgentMemory + Memobase + Zep + Supermemory domains
    ├── observe-service/      # Session observation (am-observe)
    ├── orchestration-service/# Multi-agent coordination
    ├── observe-search/       # BM25 + vector search (am-search)
    ├── pipeline-service/     # Pipeline, Job, Queue management
    ├── obs-service/          # Observability infra
    ├── search-service/       # Cross-engine search, connectors, MCP
    ├── storage-service/      # VikingFS, resource, session
    ├── vnp-platform/         # Admin, Auth, Events, Analytics
    ├── kg-service/           # Knowledge Graph service
    ├── ba-knowledge-service/ # BA knowledge integration
    ├── ba-knowledge-worker/  # BA knowledge background worker
    └── [per-engine services] # cognee-*, graphiti-*, memobase-*, ov-*, zep-*, sm-*, vnp-*
```

---

## 11. AgentMemory Engine Architecture (mới)

AgentMemory là engine chuyên biệt cho AI agent memory, tách biệt hoàn toàn khỏi user-facing memory engines.

```
┌──────────────────────────────────────────────────────────────────┐
│                      AgentMemory Engine                          │
│                                                                  │
│  ┌─ am-observe ────────────────────────────────────────────┐    │
│  │  Hook → DedupMap → PrivacyRedactor → KVStore            │    │
│  │       → SearchIndexer → StreamBroker → NATS             │    │
│  └─────────────────────────────────────────────────────────┘    │
│           │ agentmemory.observation.captured                     │
│  ┌─ am-memory ─────────────────────────────────────────────┐    │
│  │  Consolidation Pipeline:                                 │    │
│  │    RawObs → CompressedObs (LLM) → AgentMemory           │    │
│  │    Sessions → ProceduralMemory, Lessons, Insights        │    │
│  │  Retention: Decay scoring → FlaggedEviction → AutoForget │    │
│  │  MemorySlots: project|global scoped key-value store      │    │
│  └─────────────────────────────────────────────────────────┘    │
│           │ search index updates                                  │
│  ┌─ am-search ─────────────────────────────────────────────┐    │
│  │  BM25 (full-text) + pgvector (semantic) hybrid           │    │
│  │  Smart/BM25/Vector/Context search endpoints              │    │
│  │  Index: add, remove, rebuild, stats                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ┌─ am-orchestration ──────────────────────────────────────┐    │
│  │  Actions (CRUD + Dependencies + ConflictsWith)           │    │
│  │  Leases (Distributed locking for actions)               │    │
│  │  Signals (Inter-agent messaging)                         │    │
│  │  Routines (Reusable step sequences)                      │    │
│  │  Checkpoints (Human-in-the-loop approval gates)          │    │
│  │  Sentinels (Condition watchers → trigger signals)        │    │
│  │  Sketches (Ephemeral action plans)                       │    │
│  │  Crystals (LLM-distilled knowledge from action history)  │    │
│  │  BackgroundSweeper (expiry + cleanup goroutine)          │    │
│  └─────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
                        │
                 PostgreSQL (all entities)
                 NATS (event bus)
                 Bifrost (LLM: compress + synthesize)
```
