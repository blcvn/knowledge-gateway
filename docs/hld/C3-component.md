# C3 — Component Diagram

> **C4 Level 3:** Các module bên trong từng Container quan trọng nhất.

---

## C3.1 — API Gateway Components

```mermaid
C4Component
    title Component Diagram — API Gateway

    Container_Boundary(gw, "API Gateway (backend/gateway)") {

        Component(router, "HTTP Router", "Go stdlib net/http + Go 1.22 routing",
            "50+ REST routes\nMethod + path matching\nRequestID injection")

        Component(auth, "Auth Middleware", "Go middleware",
            "API Key: SHA-256 hash lookup\nJWT RS256 validation\nDev mode bypass (localhost only)\nTenantID injection into context")

        Component(ratelimit, "Rate Limiter", "Go + Redis sliding window",
            "Per-tenant limits by tier\n(free / pro / enterprise)\nNATS event on exceeded")

        Component(registry, "InProcessRegistry", "Go bufconn",
            "Service discovery (dev mode)\nZero-latency in-process gRPC\nFallback to network gRPC (prod)")

        Component(mcp_handler, "MCP Handler", "JSON-RPC 2.0",
            "37+ tool definitions\nSSE stream (GET /mcp/sse)\nHTTP Streamable (POST /mcp/message)\nTranslate to internal calls")

        Component(middleware, "Middleware Stack", "Go middleware chain",
            "CORS\nRequest logging (slog)\nPanic recovery\nSecret redaction in logs\nOpenTelemetry tracing")
    }

    Rel(router, auth, "Every request passes auth")
    Rel(auth, ratelimit, "After auth, check rate limit")
    Rel(ratelimit, registry, "Lookup target service")
    Rel(registry, mcp_handler, "MCP routes → handler")
    Rel(middleware, router, "Wrap router")
```

---

## C3.2 — Memory Engine Layer Components

```
Memory Engine Layer
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  COGNEE CLUSTER (Semantic Memory — F03)                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │cognee-        │  │cognee-        │  │cognee-pipeline           │  │
│  │ingestion      │  │search         │  │(7-step cognify)          │  │
│  │Add, classify  │  │15+ strategies │  │classify→chunk→extract    │  │
│  │chunk, index   │  │Graph/RAG/     │  │→relations→graph→embed    │  │
│  └──────────────┘  │Chunks/Entity  │  └──────────────────────────┘  │
│                     └──────────────┘                                │
│                                                                     │
│  GRAPHITI CLUSTER (Episodic Memory — F02)                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │graphiti-      │  │graphiti-      │  │graphiti-knowledge        │  │
│  │ingestion      │  │search         │  │Entity CRUD               │  │
│  │Episodes:      │  │Temporal-aware │  │valid_at / invalid_at     │  │
│  │text/JSON/fact │  │< 200ms        │  │Fact management           │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
│                                                                     │
│  MEMOBASE CLUSTER (Profile Memory — F05)                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │memobase-      │  │memobase-      │  │memobase-context          │  │
│  │ingestion      │  │engine         │  │GET /context < 100ms      │  │
│  │Blob → Buffer  │  │YOLO: 3 LLM    │  │Prompt-ready string       │  │
│  │auto-flush@20  │  │calls per flush│  │Summary+Profile+Events    │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
│                                                                     │
│  ZEP CLUSTER (Conversational Memory — F04)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │zep-memory     │  │zep-graph      │  │zep-search                │  │
│  │Session CRUD   │  │Knowledge graph│  │Temporal-aware search     │  │
│  │Message ingest │  │Custom ontology│  │Graph + vector            │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
│                                                                     │
│  OPENVIKING CLUSTER (Procedural Memory — F06)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │ov-fs          │  │ov-search      │  │ov-session                │  │
│  │VikingFS CRUD  │  │Hierarchical   │  │2-phase commit            │  │
│  │L0/L1/L2 tiers │  │Semantic grep  │  │Working memory            │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
│                                                                     │
│  SUPERMEMORY CLUSTER (Adaptive Memory — F07)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │sm-memory      │  │sm-search      │  │sm-connector              │  │
│  │Living KG      │  │RAG + semantic │  │Drive/Gmail/Notion        │  │
│  │isLatest chain │  │forgetAfter    │  │OAuth2 sync               │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## C3.3 — AgentMemory Layer Components

```
AgentMemory Layer
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  OBSERVE SERVICE (F08 — Agent Hook Capture)                         │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Hook Receiver                                               │  │
│  │  (12 hook types: session_start, llm_prompt, tool_call...)    │  │
│  │          │                                                   │  │
│  │  14-Step Pipeline:                                           │  │
│  │  Validate → Auth → Dedup(30s TTL) → Redact(PII/secrets)     │  │
│  │  → Parse → Enrich → Classify → Store(postgres)              │  │
│  │  → Index(BM25) → Embed(vector) → Publish(NATS)              │  │
│  │  → Update Session State → Stream SSE                        │  │
│  │          │                                                   │  │
│  │  Session Manager                 SSE Streamer               │  │
│  │  (active/completed/abandoned)    (Console real-time)        │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  MEMORY SERVICE (F09 — Agent Memory Lifecycle)                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Jaccard Versioning Engine          Eviction Manager         │  │
│  │  (similarity threshold → merge      (salience score =        │  │
│  │   or create new version)             importance × recency    │  │
│  │                                       × frequency)           │  │
│  │  6 Memory Types:                    Memory Decay             │  │
│  │  episodic/semantic/conversational   (time-based score decay) │  │
│  │  profile/procedural/adaptive                                 │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ORCHESTRATION SERVICE (F11 — Multi-Agent)                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Lease Manager          Signal Router          Action DAG    │  │
│  │  (distributed mutex     (NATS-backed:          (state machine│  │
│  │   with TTL)              handoff/alert/         pending→     │  │
│  │                          update/query)          in_progress  │  │
│  │  Sentinel Watchers      Sketch→Crystal          →completed)  │  │
│  │  (event-driven          (draft→commit                        │  │
│  │   condition triggers)    pattern)                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  PIPELINE SERVICE (F12 — Memory Consolidation)                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Tier 1: LLM Compression     Tier 2: Session Summary         │  │
│  │  (group by 5min window,      (What attempted, succeeded,     │  │
│  │   compress batch → 70-90%    failed, decisions, entities)    │  │
│  │   reduction, circuit breaker)                                │  │
│  │                                                              │  │
│  │  Tier 3: Procedure Extract   Tier 4: Lessons & Insights      │  │
│  │  (multi-session → generic    (cross-agent patterns,          │  │
│  │   procedures, higher         cross-session insights,         │  │
│  │   durability)                share across agents)            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## C3.4 — Search Hub Components

```
vnp-search-hub
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  Query Router                                                       │
│  ├── Parse query + filters (type, time_range, engine, limit)        │
│  └── Fan-out to engines in parallel (gRPC with 500ms timeout)       │
│                                                                     │
│  Engine Adapters (parallel goroutines)                              │
│  ├── cognee-search adapter    → GRAPH_COMPLETION / RAG / CHUNKS    │
│  ├── graphiti-search adapter  → temporal-aware graph search         │
│  ├── memobase-context adapter → profile context assembly            │
│  ├── ov-search adapter        → hierarchical L0/L1/L2 retrieval     │
│  ├── sm-search adapter        → semantic + RAG search               │
│  └── zep-search adapter       → session + graph search              │
│                                                                     │
│  Hybrid Search Fusion (F10)                                         │
│  ├── BM25 in-memory index     → keyword relevance score            │
│  ├── Vector similarity        → pgvector cosine distance           │
│  └── RRF Fusion               → Reciprocal Rank Fusion merge       │
│                                                                     │
│  Result Merger                                                      │
│  ├── Deduplicate by content hash                                    │
│  ├── Sort by fused score                                            │
│  └── Truncate to limit + token_budget                               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## C3.5 — Platform & Console Components

```
Platform Services
┌────────────────────────────────────────────────────────────────────┐
│ vnp-platform                    vnp-event                          │
│ ├── Tenant CRUD                 ├── UserEvent store                │
│ ├── API Key lifecycle           ├── Timeline query                 │
│ │   (SHA-256, prefix, TTL)      ├── GistText LLM summary          │
│ ├── User management             └── EventType filter               │
│ └── Subscription tiers                                             │
│                                                                    │
│ vnp-observability               vnp-dashboard (Console UI)         │
│ ├── Prometheus metrics          ├── SPA embedded in binary         │
│ ├── OpenTelemetry traces        ├── Memory Explorer                │
│ ├── LLM cost tracking           ├── Graph Studio                   │
│ └── Error rate aggregation      ├── Session Replay                 │
│                                 ├── Governance Center              │
│ vnp-admin                       └── WebSocket SSE stream           │
│ ├── GDPR Forget (cascading)                                        │
│ ├── OPA Policy enforcement                                         │
│ └── Audit Trail (searchable)                                       │
└────────────────────────────────────────────────────────────────────┘
```

---

## C3.6 — Shared Packages (backend/shared/pkg/)

| Package | Responsibility | Used by |
|---|---|---|
| `config` | Config loading (env vars, YAML) | All services |
| `adapters` | gRPC client wrappers per engine | Gateway, search-hub |
| `forward` | Request forwarding middleware | Gateway |
| `graph` | Graph traversal utilities | Cognee, Graphiti, Zep |
| `privacy` | PII/secret redaction | Observe pipeline |
| `resilience` | Circuit breaker, retry, timeout | All services |
| `search` | BM25 index, RRF fusion | search-hub, observe-search |
| `telemetry` | OpenTelemetry setup | All services |
| `tenant` | TenantID injection, multi-tenancy | Gateway, all services |
| `tokenizer` | Token counting for budget | MCP, context assembly |
| `vectorstore` | pgvector + Qdrant abstraction | Search, ingestion |

---

*[← C2 Container](./C2-container.md) | [→ C4 Code](./C4-code.md)*
