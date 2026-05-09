# Zep — Architecture Design Document

> **Repository**: `github.com/getzep/zep`
> **Generated**: 2026-05-07
> **Status**: Active (Cloud) / Deprecated (Community Edition)

---

## 1. Architecture Overview

Zep is an **end-to-end context engineering platform** for AI agents. It assembles relationship-aware context from chat history, business data, documents, and app events with sub-200ms latency, powered by **Graphiti** — a temporal knowledge graph framework.

### 1.1 System Context Diagram

```
                    ┌──────────────────────────────┐
                    │        AI Applications       │
                    │  (Agents, Copilots, Chatbots) │
                    └──────────┬───────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
        ┌──────────┐   ┌────────────┐   ┌──────────┐
        │ Python   │   │ TypeScript │   │ Go SDK   │
        │ SDK      │   │ SDK        │   │          │
        └────┬─────┘   └─────┬──────┘   └────┬─────┘
             │                │               │
             └────────────────┼───────────────┘
                              ▼
                    ┌─────────────────────┐
                    │   Zep Cloud API     │
                    │  (REST/gRPC Gateway)│
                    └────────┬────────────┘
                             │
                    ┌────────┴────────┐
                    ▼                 ▼
              ┌──────────┐    ┌────────────┐
              │PostgreSQL│    │  Graphiti  │
              │(pgvector)│    │  Service   │
              └──────────┘    └─────┬──────┘
                                    │
                              ┌─────▼──────┐
                              │   Neo4j    │
                              └────────────┘

        ┌──────────────────────────────────────────┐
        │           MCP Server (Go)                │
        │  13 read-only tools for AI assistants    │
        │  Transport: stdio | HTTP Streamable      │
        └──────────────────────────────────────────┘
```

---

## 2. Architectural Layers

### Layer 1 — Client SDKs & Access

| Component | Language | Transport | Purpose |
|---|---|---|---|
| Python SDK (`zep-cloud`) | Python | REST/gRPC | Primary developer SDK |
| TypeScript SDK (`@getzep/zep-cloud`) | TS/JS | REST/gRPC | Web/Node applications |
| Go SDK (`zep-go/v3`) | Go | REST/gRPC | Go applications |
| MCP Server | Go | stdio/HTTP | AI assistant read-only access |
| REST API | — | HTTP | Direct API access |

### Layer 2 — API Gateway

- **Authentication**: API Key (Cloud) / Shared Secret (CE)
- **Rate Limiting**: Cloud-managed per-key limits
- **CORS**: Configurable (CE allows all origins)
- **Request Limits**: 5MB max payload, 30s timeout

### Layer 3 — Core Services

| Service | Responsibility | Technology |
|---|---|---|
| User Service | User CRUD, metadata management | Go + PostgreSQL |
| Session Service | Thread lifecycle, session state | Go + PostgreSQL |
| Memory Service | Message storage, memory overlay | Go + PostgreSQL |
| Graph Service | KG extraction, fact management | Graphiti (Python) |
| Search Service | Semantic search, reranking | Graphiti + Neo4j |

### Layer 4 — Data Stores

| Store | Technology | Data |
|---|---|---|
| Relational DB | PostgreSQL + pgvector | Users, Sessions, Messages, Metadata |
| Graph DB | Neo4j 5.22+ | Temporal Knowledge Graph (nodes, edges, episodes) |

---

## 3. Component Architecture

### 3.1 Legacy CE Server (`legacy/src/`)

```
main.go
  │
  ├── config.Load()              ← zep.yaml
  ├── logger.InitDefaultLogger()
  ├── newAppState()              ← DB pool, TaskRouter, Graphiti client
  └── api.Create(appState)       ← chi Router + middleware
        │
        ├── Middleware Stack (10 layers)
        │   ├── CORS
        │   ├── Request Logging (structured)
        │   ├── Heartbeat (/healthz)
        │   ├── Request Size Limiter (5MB)
        │   ├── Request ID Injection
        │   ├── Timeout (30s)
        │   ├── Real IP Extraction
        │   ├── Clean Path
        │   ├── Version Header
        │   └── OpenTelemetry (otelchi)
        │
        └── Routes (/api/v2)
            ├── /sessions/*        ← Session CRUD + Memory + Messages
            ├── /users/*           ← User CRUD + User Sessions
            └── /facts/*           ← Fact Retrieval/Deletion
```

**Key Design Patterns**:
- **DAO Pattern**: `memoryDAO`, `messageDAO`, `sessionDAO` for data access
- **AppState Singleton**: Shared DB pool, task router, Graphiti client
- **Advisory Locks**: PostgreSQL locks (SHA-256 hash) for concurrent updates
- **Soft Deletes**: `deleted_at` timestamp on all entities

### 3.2 MCP Server (`mcp/zep-mcp-server/`)

```
cmd/server/main.go
  │
  ├── config.Load()              ← .env + ZEP_API_KEY
  ├── zepclient.New()            ← Zep Go SDK v3
  └── server.New()               ← MCP Server
        │
        ├── Transport Selection
        │   ├── --stdio  → stdin/stdout JSON-RPC
        │   └── default  → HTTP Streamable (port 8080)
        │
        └── 13 Read-Only Tools
            ├── Phase 1: search_graph, get_user_context, get_user, list_threads
            ├── Phase 2: get_user_nodes, get_user_edges, get_episodes
            └── Phase 3: get_node, get_edge, get_episode, get_thread_messages,
                         get_node_edges, get_episode_mentions
```

**Handler Pattern**: Each tool → `handlers/{feature}.go` → input validation → Zep SDK call → JSON transform → MCP response.

### 3.3 Framework Integrations (`integrations/python/`)

```
integrations/python/
  ├── CLAUDE.md                  ← Integration development guide
  ├── zep_autogen/               ← AutoGen: autogen_core.memory.Memory
  ├── zep_crewai/                ← CrewAI: ZepUserStorage + ZepGraphStorage
  ├── zep_adk/                   ← Google ADK adapter
  └── zep_livekit/               ← LiveKit adapter
```

**Storage Routing Pattern** (all integrations):

```
metadata.type == "message"  →  thread.add_messages()   → PostgreSQL
metadata.type == "text"     →  graph.add(type="text")  → Neo4j KG
metadata.type == "json"     →  graph.add(type="json")  → Neo4j KG
```

### 3.4 Eval Harness (`zep-eval-harness/`)

```
Pipeline Architecture:

data/users.json ─────────┐
data/conversations/*.json ├→ zep_ingest_users.py    → runs/users/{N}/
data/telemetry/*.json ────┘

data/documents/* → zep_chunk_documents.py → runs/chunk_sets/{N}/chunks.jsonl
                                                      ↓
                   zep_ingest_documents.py → runs/documents/{N}/

data/test_cases/*.json → zep_evaluate.py  → runs/evaluations/{N}/results.json
```

---

## 4. Data Architecture

### 4.1 Entity Relationship Model

```
┌──────────┐     1:N     ┌──────────┐     1:N     ┌──────────┐
│  User    │────────────▶│ Session  │────────────▶│ Message  │
│          │             │ (Thread) │             │          │
│ uuid     │             │ uuid     │             │ uuid     │
│ user_id  │             │session_id│             │session_id│
│ email    │             │ user_id  │             │ role     │
│ metadata │             │ ended_at │             │ content  │
└──────────┘             │ metadata │             │ metadata │
      │                  └──────────┘             └──────────┘
      │
      │  Graph Association
      ▼
┌──────────────────────────────────────────────────────┐
│                Temporal Knowledge Graph               │
│                                                      │
│  ┌──────┐  ──EDGE──▶  ┌──────┐                     │
│  │ Node │              │ Node │     ┌─────────┐     │
│  │(User)│              │(Pref)│     │ Episode │     │
│  └──────┘  ◀──EDGE──  └──────┘     │(temporal)│    │
│                                     └─────────┘     │
│  Each Edge (Fact): valid_at / invalid_at            │
└──────────────────────────────────────────────────────┘
```

### 4.2 Knowledge Graph Ontology

**Node Priority Hierarchy** (extraction classification order):

```
1. User (singleton)        ← Highest priority
2. Assistant (singleton)
3. Preference              ← LOW threshold for classification
4. Organization
5. Event
6. Location
7. Document
8. Topic
9. Object                  ← Last resort only
```

**Edge Types**: `LOCATED_AT` (Entity→Location), `OCCURRED_AT` (Event→Entity/Location), custom types via ontology API.

---

## 5. Key Data Flows

### 5.1 Message Ingestion → Graph Extraction

```
Client POST /sessions/{id}/memory
  │
  ▼
┌─────────────────────────┐
│ 1. Upsert Session       │ ← Create if not exists
│ 2. Check session.EndedAt│ ← Reject if ended
│ 3. INSERT messages      │ ← PostgreSQL batch insert
│ 4. Publish to Graphiti  │ ← Async graph extraction
│    ├─ PutMemory(sessionID, msgs, addPrefix=true)
│    └─ PutMemory(userID, msgs, addPrefix=true)
└─────────────────────────┘
                │
                ▼ (async, 10-20s)
┌─────────────────────────┐
│ Graphiti Service         │
│ ├─ Entity extraction     │ ← LLM-powered
│ ├─ Relationship mapping  │
│ ├─ Temporal annotation   │ ← valid_at / invalid_at
│ └─ Neo4j upsert          │
└─────────────────────────┘
```

### 5.2 Memory Retrieval → Context Assembly

```
Client GET /sessions/{id}/memory?lastN=10
  │
  ▼
┌─────────────────────────────────────────┐
│ 1. Fetch last max(N,4) messages         │ ← PostgreSQL
│ 2. groupID = user_id ?? session_id      │
│ 3. Call Graphiti.GetMemory(             │
│      groupID, maxFacts=5,               │
│      last 4 messages as query context)  │ ← Neo4j search
│ 4. Assemble Memory response:            │
│    { messages: [lastN], facts: [...] }  │
└─────────────────────────────────────────┘
```

### 5.3 Graph Search

```
Client POST /sessions/search  OR  SDK graph.search()
  │
  ▼
┌─────────────────────────────────────────┐
│ 1. Build GraphSearchQuery               │
│    ├─ scope: edges|nodes|episodes       │
│    ├─ reranker: rrf|mmr|cross_encoder   │
│    │            |node_distance           │
│    │            |episode_mentions         │
│    ├─ filters: node_labels, edge_types  │
│    └─ limit, min_fact_rating, mmr_lambda│
│ 2. Execute via Graphiti/Neo4j           │
│ 3. Rerank results                       │
│ 4. Return formatted JSON               │
└─────────────────────────────────────────┘
```

---

## 6. Deployment Architecture

### 6.1 Cloud Deployment (Production)

```
┌─────────────────────────────────────────────┐
│              Zep Cloud (Managed)            │
│                                             │
│  ┌─────────┐  ┌──────────┐  ┌───────────┐ │
│  │API      │  │Graphiti  │  │Background │ │
│  │Gateway  │  │Workers   │  │Processing │ │
│  └────┬────┘  └────┬─────┘  └─────┬─────┘ │
│       │            │              │        │
│  ┌────▼────────────▼──────────────▼─────┐  │
│  │        Managed Data Plane            │  │
│  │  PostgreSQL │ Neo4j │ Object Store   │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  SLA: <200ms retrieval │ SOC2/HIPAA        │
└─────────────────────────────────────────────┘
```

### 6.2 Self-Hosted CE (Legacy/Deprecated)

```yaml
# docker-compose.ce.yaml — 4 services
zep (port 8000) ──depends──▶ graphiti (port 8003) ──depends──▶ neo4j (7474/7687)
                ──depends──▶ db/postgres (port 5432)
```

### 6.3 MCP Server Deployment

```
Option A: stdio (Claude Desktop/Cline)
  claude_desktop_config.json → spawn process → stdin/stdout

Option B: HTTP (Claude Code/web clients)
  docker run -e ZEP_API_KEY=key -p 8080:8080 zep-mcp-server

Option C: Docker Compose
  docker-compose.yml → zep-mcp-server service
```

---

## 7. Security Architecture

```
┌─────────────────────────────────────────────────┐
│                Security Boundaries               │
│                                                  │
│  ┌──────────────┐    ┌───────────────────────┐  │
│  │ Auth Layer   │    │ Data Isolation         │  │
│  │              │    │                        │  │
│  │ Cloud: API   │    │ project_uuid on ALL    │  │
│  │  Key Bearer  │    │  entities (multi-      │  │
│  │              │    │  tenant isolation)     │  │
│  │ CE: Shared   │    │                        │  │
│  │  Secret      │    │ Soft deletes only      │  │
│  │              │    │  (audit trail)         │  │
│  │ MCP: Read-   │    │                        │  │
│  │  only access │    │ Advisory locks for     │  │
│  └──────────────┘    │  concurrent writes     │  │
│                      └───────────────────────┘  │
│                                                  │
│  Middleware: 5MB limit │ 30s timeout │ ReqID    │
│  Compliance: SOC2 Type 2 │ HIPAA (Cloud)        │
└─────────────────────────────────────────────────┘
```

---

## 8. Observability Architecture

```
┌─────────────────────────────────────────────┐
│              Observability Stack             │
│                                             │
│  Logs ─────── log/slog (MCP)               │
│               custom structured (CE)        │
│               Format: JSON | console        │
│                                             │
│  Traces ───── OpenTelemetry (otelchi)       │
│               Per-request spans             │
│                                             │
│  Health ───── GET /healthz                  │
│               chi Heartbeat middleware       │
│                                             │
│  Metrics ──── Request: method, path,        │
│               duration, status, size         │
│               Request ID tracking            │
│                                             │
│  Telemetry ── Anonymous usage events        │
│               (opt-out: telemetry.disabled)  │
└─────────────────────────────────────────────┘
```

---

## 9. Integration Architecture

### 9.1 Framework Integration Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                   Framework Integration Layer                │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌──────┐  ┌─────────┐ │
│  │  AutoGen    │  │  CrewAI     │  │ ADK  │  │ LiveKit │ │
│  │  Memory     │  │  Storage    │  │      │  │         │ │
│  └──────┬──────┘  └──────┬──────┘  └──┬───┘  └────┬────┘ │
│         │                │            │            │       │
│         └────────────────┼────────────┼────────────┘       │
│                          ▼            ▼                     │
│              ┌─────────────────────────────┐               │
│              │    Zep Python SDK           │               │
│              │    (zep-cloud >= 3.0)       │               │
│              │                             │               │
│              │  thread.add_messages()      │               │
│              │  thread.get_user_context()  │               │
│              │  graph.add()                │               │
│              │  graph.search()             │               │
│              └─────────────┬───────────────┘               │
│                            ▼                                │
│                     Zep Cloud API                           │
└─────────────────────────────────────────────────────────────┘
```

### 9.2 CrewAI Dual-Storage Architecture

```
┌─────────────────────────────────┐
│        CrewAI Agent             │
│                                 │
│  ┌────────────────────────────┐ │
│  │ ZepUserStorage             │ │  Per-user memory
│  │  ├─ thread messages        │ │  (user_id + thread_id)
│  │  ├─ user graph data        │ │
│  │  └─ context retrieval      │ │
│  └────────────────────────────┘ │
│                                 │
│  ┌────────────────────────────┐ │
│  │ ZepGraphStorage            │ │  Shared knowledge
│  │  ├─ structured knowledge   │ │  (graph_id)
│  │  ├─ custom ontologies      │ │
│  │  └─ multi-scope search     │ │
│  └────────────────────────────┘ │
│                                 │
│  Tool Factories:                │
│  create_search_tool(client)     │
│  create_add_data_tool(client)   │
└─────────────────────────────────┘
```

---

## 10. Evaluation Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    Eval Harness Pipeline                  │
│                                                          │
│  ┌────────────────┐                                     │
│  │ Config Layer   │  config/                            │
│  │ ├─ ontology    │  ├─ user_ingestion_config/          │
│  │ ├─ instructions│  ├─ document_ingestion_config/      │
│  │ └─ constants   │  ├─ document_chunking_config/       │
│  └────────────────┘  └─ evaluation_config/              │
│          │                                               │
│          ▼                                               │
│  ┌────────────────────────────────────────────────────┐ │
│  │ Ingestion   │ Chunking    │ Evaluation             │ │
│  │             │             │                        │ │
│  │ users.py    │ chunk.py    │ evaluate.py            │ │
│  │ documents.py│             │ ├─ Search (Zep API)    │ │
│  │             │             │ ├─ Context eval (LLM)  │ │
│  │             │             │ ├─ Generate (LLM)      │ │
│  │             │             │ └─ Grade (LLM judge)   │ │
│  └────────────────────────────────────────────────────┘ │
│          │                                               │
│          ▼                                               │
│  ┌────────────────┐                                     │
│  │ Run Tracking   │  runs/                              │
│  │ ├─ users/      │  Numbered, timestamped dirs         │
│  │ ├─ documents/  │  Config snapshots for reproducibility│
│  │ ├─ chunk_sets/ │  Decoupled: mix user_run + doc_run  │
│  │ └─ evaluations/│  Metrics: completeness + accuracy   │
│  └────────────────┘                                     │
└──────────────────────────────────────────────────────────┘
```

---

## 11. Design Decisions

| Decision | Rationale |
|---|---|
| **Temporal KG over vector-only** | Facts need `valid_at`/`invalid_at` for temporal reasoning |
| **Graphiti as separate service** | Decouples LLM-heavy extraction from low-latency API |
| **PostgreSQL advisory locks** | Prevents race conditions without external lock infrastructure |
| **Read-only MCP server** | Safety-first for AI assistant access |
| **CE deprecation** | Operational complexity of self-hosting; Cloud offers better SLAs |
| **Soft deletes everywhere** | Preserves audit trail and temporal analysis |
| **UV workspace** | Single `uv sync` manages all Python integration packages |
| **MCP Streamable HTTP** | Forward-compatible with 2025-03-26 MCP spec |
| **Decoupled eval pipeline** | Combinatorial testing without re-ingesting identical graphs |

---

## 12. Appendix: Technology Matrix

| Category | Technology | Version |
|---|---|---|
| **Languages** | Go, Python, TypeScript | 1.21+, 3.10+, ES2022 |
| **HTTP Framework** | go-chi/chi | v5 |
| **ORM** | uptrace/bun | Latest |
| **MCP SDK** | modelcontextprotocol/go-sdk | Latest |
| **Zep SDK** | getzep/zep-go | v3 |
| **Database** | PostgreSQL + pgvector | v0.5.1+ |
| **Graph DB** | Neo4j | 5.22+ |
| **Observability** | OpenTelemetry (otelchi) | Latest |
| **Python Tooling** | UV, ruff, mypy, pytest | Latest |
| **Container** | Docker, Docker Compose | Latest |
| **CI/CD** | GitHub Actions | — |

---

*Document generated from architectural analysis of `github.com/getzep/zep` repository.*
