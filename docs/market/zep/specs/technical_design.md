# Zep: Technical Design Document

> **Repository**: `github.com/getzep/zep`  
> **Generated**: 2026-05-02  
> **Status**: Active (Cloud) / Deprecated (Community Edition)

---

## 1. Executive Summary

Zep is an **end-to-end context engineering platform** for AI agents. It solves the *agent context problem*—assembling comprehensive, relationship-aware context from multiple data sources (chat history, business data, documents, app events) with sub-200ms latency.

The platform is powered by **Graphiti**, an open-source temporal knowledge graph framework that autonomously builds and maintains a graph while reasoning about state changes over time.

---

## 2. System Overview

### 2.1 Core Value Proposition

| Capability | Description |
|---|---|
| **Graph RAG** | Auto-extracts relationships; maintains a temporal KG where each fact has `valid_at` / `invalid_at` timestamps |
| **Context Assembly** | Pre-formats relationship-aware context blocks optimised for LLMs |
| **Sub-200ms Retrieval** | Cloud-managed, production-grade latency SLA |
| **Multi-source Ingestion** | Chat messages, business data, documents, application events |

### 2.2 High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│  Python SDK │ TypeScript SDK │ Go SDK │ REST API │ MCP Server │
└──────────────┬───────────────────────────────────────────────┘
               │
┌──────────────▼───────────────────────────────────────────────┐
│                      Zep Cloud API                           │
│          (gRPC/REST gateway, Auth, Rate-limiting)            │
└──────────────┬───────────────────────────────────────────────┘
               │
      ┌────────┴────────┐
      ▼                 ▼
┌──────────┐    ┌──────────────────┐
│ Postgres │    │    Graphiti      │
│ (users,  │    │  Service (Graph  │
│ sessions,│    │   extraction,    │
│ messages)│    │  fact storage)   │
└──────────┘    └────────┬─────────┘
                         │
                   ┌─────▼──────┐
                   │   Neo4j    │
                   │  (temporal │
                   │   KG store)│
                   └────────────┘
```

---

## 3. Repository Structure

```
zep/
├── ontology/              # Default entity/edge ontology (Python)
├── mcp/
│   └── zep-mcp-server/    # MCP server exposing Zep via Model Context Protocol (Go)
├── integrations/
│   └── python/            # Framework integrations
│       ├── zep_autogen/   # Microsoft AutoGen
│       ├── zep_crewai/    # CrewAI
│       ├── zep_adk/       # Google ADK
│       └── zep_livekit/   # LiveKit
├── benchmarks/
│   ├── locomo/            # LoCoMo long-context conversation benchmark
│   └── longmemeval/       # LongMemEval benchmark
├── examples/
│   ├── go/
│   ├── python/
│   └── typescript/
├── legacy/                # Deprecated Community Edition (Go)
│   └── src/               # Full CE source — Postgres + Graphiti
└── specs/                 # Design documents (this file)
```

---

## 4. Core Data Model

### 4.1 Domain Entities

#### User
```
User {
  uuid         UUID (PK)
  user_id      string (unique, human-readable)
  email        string
  first_name   string
  last_name    string
  project_uuid UUID (FK)
  metadata     JSONB
  created_at   timestamptz
  updated_at   timestamptz
  deleted_at   timestamptz (soft delete)
}
```

#### Session / Thread
```
Session {
  uuid         UUID (PK)
  session_id   string (unique)
  user_id      string (FK → User.user_id, nullable)
  project_uuid UUID (FK)
  metadata     JSONB
  ended_at     timestamptz (nullable — marks session closed)
  created_at   timestamptz
  updated_at   timestamptz
  deleted_at   timestamptz (soft delete)
}
```

#### Message
```
Message {
  uuid         UUID (PK)
  session_id   string (FK → Session)
  project_uuid UUID (FK)
  role         string        -- "user" | "assistant" | "system" | ...
  role_type    role_type_enum -- noRole | system | assistant | user | function | tool
  content      string
  token_count  int
  metadata     JSONB
  created_at   timestamptz
  updated_at   timestamptz
  deleted_at   timestamptz (soft delete)
}
```

#### Memory (API Overlay)
```
Memory {
  messages       []Message
  relevant_facts []Fact       -- from Graphiti
  metadata       map[string]any
}
```

#### Fact (Graphiti Edge)
```
Fact {
  uuid       UUID
  name       string
  fact       string        -- human-readable fact statement
  created_at time.Time
  valid_at   *time.Time    -- when fact became true
  invalid_at *time.Time    -- when fact ceased to be true
  expired_at *time.Time
}
```

### 4.2 Knowledge Graph Entities (Ontology)

The `ontology/default_ontology.py` defines the typed node schema used during graph extraction:

| Node Type | Priority | Description |
|---|---|---|
| `User` | Highest | The Zep user (singleton per conversation) |
| `Assistant` | Highest | The AI assistant (singleton) |
| `Preference` | Very High | User preferences, choices, opinions |
| `Organization` | High | Companies, institutions, groups |
| `Event` | High | Time-bound activities/occurrences |
| `Location` | Medium | Physical or virtual places |
| `Document` | Medium | Content in various forms |
| `Topic` | Low | Subjects of conversation |
| `Object` | Lowest | Physical items, tools, devices |

**Edge types:**

| Edge | Connects | Description |
|---|---|---|
| `LOCATED_AT` | Entity → Location | Entity exists at a location |
| `OCCURRED_AT` | Event → Entity/Location | Event happened at time/place |

---

## 5. Component Deep-Dives

### 5.1 Legacy Community Edition (Go)

> Located in `legacy/src/`. **Deprecated** — included for architectural reference.

**Technology stack:**
- Go 1.21+
- HTTP framework: `go-chi/chi` v5
- ORM: `uptrace/bun` (PostgreSQL)
- Observability: OpenTelemetry (`otelchi`)
- Graph: Graphiti HTTP service

**Startup sequence:**
```
main()
  → config.Load()          # YAML config: zep.yaml
  → logger.InitDefaultLogger()
  → newAppState()           # DB pool, task router, Graphiti client
  → api.Create(appState)   # Chi router + middleware
  → srv.ListenAndServe()
```

**HTTP API routes** (`/api/v2`):

| Endpoint | Method | Handler |
|---|---|---|
| `/sessions` | GET/POST | List/Create sessions |
| `/sessions/{id}` | GET/PATCH | Get/Update session |
| `/sessions/{id}/memory` | GET/POST/DELETE | Memory CRUD |
| `/sessions/{id}/messages` | GET | List messages |
| `/sessions/{id}/messages/{uuid}` | GET/PATCH | Get/Update message |
| `/sessions/search` | POST | Search sessions |
| `/sessions-ordered` | GET | Paginated ordered list |
| `/users` | GET/POST | List/Create users |
| `/users/{id}` | GET/PATCH/DELETE | User CRUD |
| `/users/{id}/sessions` | GET | User's sessions |
| `/facts/{uuid}` | GET/DELETE | Fact retrieval/deletion |

**Middleware stack:**
1. CORS (allow all origins)
2. Request logging (structured, with duration)
3. Request size limiter (5MB max)
4. Request ID injection
5. Timeout (30s)
6. Real IP extraction
7. OpenTelemetry tracing

**Memory DAO flow (PutMemory):**
```
PutMemory(sessionID, messages)
  → upsert Session (update or create)
  → check session.EndedAt (reject if ended)
  → messageDAO.CreateMany(messages)  → INSERT INTO messages
  → _initializeProcessingMemory()
      → publish messages to Graphiti service (POST /messages)
```

**Memory DAO flow (GetMemory):**
```
GetMemory(sessionID, lastN)
  → messageDAO.GetLastN(lastN)      → SELECT from messages
  → graphiti.GetMemory(groupID)     → POST /get-memory → relevant Facts
  → return Memory{messages, facts}
```

**PostgreSQL advisory locks** are used for concurrent metadata updates to prevent race conditions:
- Lock key = SHA-256 hash of session ID
- Retry policy: exponential backoff 200ms→30s, max 15 retries

### 5.2 Graphiti Service Integration

The legacy CE communicates with Graphiti via an internal HTTP client:

| Graphiti Endpoint | Method | Purpose |
|---|---|---|
| `POST /messages` | PutMemory | Ingest messages into the KG |
| `POST /get-memory` | GetMemory | Retrieve relevant facts |
| `POST /search` | Search | Semantic fact search |
| `POST /entity-node` | AddNode | Add entity node |
| `GET /entity-edge/{uuid}` | GetFact | Retrieve specific fact |
| `DELETE /entity-edge/{uuid}` | DeleteFact | Remove a fact |
| `DELETE /group/{id}` | DeleteGroup | Remove all data for a group |
| `DELETE /episode/{uuid}` | DeleteMessage | Remove message episode |

**Group ID strategy**: Messages are grouped by session ID. When `addGroupIDPrefix=true`, episode UUIDs are prefixed with `{groupID}-{messageUUID}` to namespace episodes across groups.

### 5.3 MCP Server (Go)

Located at `mcp/zep-mcp-server/`. A **read-only** Model Context Protocol server exposing Zep Cloud's capabilities to MCP-compatible AI clients (Claude Desktop, Cline, Claude Code).

**Technology stack:**
- Go 1.21+
- MCP SDK: `github.com/modelcontextprotocol/go-sdk/mcp`
- Zep SDK: `github.com/getzep/zep-go/v3`
- Transport: stdio (Claude Desktop) or HTTP Streamable (Claude Code)

**Transport modes:**

| Mode | Flag | Protocol | Use case |
|---|---|---|---|
| stdio | `--stdio` | stdin/stdout JSON-RPC | Claude Desktop, Cline |
| HTTP | default | MCP Streamable HTTP (2025-03-26 spec) | Claude Code, web clients |

**13 exposed tools (all read-only):**

*Phase 1 — Core Search & Retrieval:*
- `search_graph` — semantic search with scope (edges/nodes/episodes), reranking strategies (RRF, MMR, node_distance, episode_mentions, cross_encoder), label/type filters
- `get_user_context` — formatted context string for a thread (supports custom template ID)
- `get_user` — user metadata
- `list_threads` — all threads for a user

*Phase 2 — Graph Query:*
- `get_user_nodes` — entity nodes from user's KG
- `get_user_edges` — relationship edges from user's KG
- `get_episodes` — temporal episode history

*Phase 3 — Detail Retrieval:*
- `get_thread_messages` — messages from a thread
- `get_node` — node by UUID
- `get_edge` — edge by UUID
- `get_episode` — episode by UUID
- `get_node_edges` — all edges for a node
- `get_episode_mentions` — nodes/edges mentioned in an episode

**Configuration:**
```
ZEP_API_KEY=<required>
LOG_LEVEL=info|debug|warn|error  (optional)
--port <N>   (HTTP mode, default 8080)
--stdio      (stdio mode)
```

**Docker deployment:**
```yaml
docker run -e ZEP_API_KEY=<key> -p 8080:8080 zep-mcp-server:latest
```

### 5.4 Python Framework Integrations

Located at `integrations/python/`. Each integration is an independent `uv`-managed package published to PyPI.

**Standardised structure per integration:**
```
integrations/python/{framework}/
├── src/zep_{framework}/
│   ├── __init__.py       # Package entry + version
│   ├── memory.py         # Core memory class
│   └── exceptions.py     # Framework-specific exceptions
├── tests/
├── examples/
├── pyproject.toml
└── Makefile
```

**Storage routing pattern (all integrations):**

| `metadata.type` | Storage destination |
|---|---|
| `"message"` | Zep Thread (via `thread.add_messages()`) |
| `"text"` / `"json"` | Zep User Graph (via `graph.add()`) |

#### AutoGen Integration (`zep_autogen`)
- Implements `autogen_core.memory.Memory` interface
- Supports `CancellationToken` for async operations
- MIME type mapping: TEXT/MARKDOWN → `"text"`, JSON → `"json"`
- Async-first design; caller manages client lifecycle

#### CrewAI Integration (`zep_crewai`)

Two storage classes:

**`ZepUserStorage`** — user-specific memory:
- Thread messages via `thread.add_messages()`
- User graph via `graph.add()`
- Context via `thread.get_user_context()` (mode: `"summary"` | `"raw_messages"`)
- Parallel search across thread + graph

**`ZepGraphStorage`** — shared knowledge graphs:
- Structured knowledge with custom ontologies
- `compose_context_string()` for formatted LLM context
- Multi-scope search: edges, nodes, episodes

**Tool factories:**
- `create_search_tool(client, user_id=...|graph_id=...)` — natural language search
- `create_add_data_tool(client, user_id=...|graph_id=...)` — data ingestion

**Parameters:**
```python
ZepUserStorage(
  client=zep_client,       # required
  user_id="alice_123",     # required
  thread_id="thread_456",  # optional
  facts_limit=20,          # default: 20
  entity_limit=5,          # default: 5
  mode="summary"           # "summary" | "raw_messages"
)
```

---

## 6. Benchmarking Infrastructure

### 6.1 LoCoMo Benchmark (`benchmarks/locomo/`)

Evaluates Zep's long-context conversation memory using the LoCoMo dataset.

**CLI modes:**
```
python benchmark.py --ingest    # Load LoCoMo data into Zep graphs
python benchmark.py --eval      # Run Q&A evaluation
python benchmark.py --cleanup   # List/delete experiment graphs
```

**Evaluation metrics:**
- **Accuracy** — correct answers / total questions
- **Context completeness** — COMPLETE / PARTIAL / INSUFFICIENT
- **Latency** — retrieval time (median, p95, p99)
- **Context tokens** — median, mean, p95, p99

**Graph retrieval configuration:**
```yaml
graph_params:
  edge_limit: N
  edge_reranker: rrf|mmr|node_distance|episode_mentions|cross_encoder
  node_limit: N
  node_reranker: ...
```

**Multi-run support**: Creates an `experiment/` directory with per-run JSON results and an `experiment_summary.json` with aggregated statistics (mean, std dev, min, max accuracy).

### 6.2 LongMemEval Benchmark (`benchmarks/longmemeval/`)

Companion benchmark for extended memory evaluation (structure mirrors LoCoMo).

---

## 7. Infrastructure & Deployment

### 7.1 Legacy CE Docker Compose

Services required for the Community Edition:

| Service | Image | Port | Role |
|---|---|---|---|
| `zep` | `zepai/zep:latest` | 8000 | API server |
| `db` | `ankane/pgvector:v0.5.1` | 5432 | PostgreSQL + pgvector |
| `graphiti` | `zepai/graphiti:0.3` | 8003 | Graph extraction service |
| `neo4j` | `neo4j:5.22.0` | 7474/7687 | Graph database |

**Startup dependencies:**
```
zep → [graphiti (healthy), db (healthy)]
graphiti → [neo4j (healthy)]
```

### 7.2 Legacy CE Configuration (`zep.yaml`)

```yaml
log:
  level: info
  format: json         # or "console"
http:
  host: 0.0.0.0
  port: 8000
  max_request_size: 5242880   # 5MB
postgres:
  user: postgres
  password: postgres
  host: db
  port: 5432
  database: postgres
  schema_name: public
  max_open_connections: 10
graphiti:
  service_url: http://graphiti:8003
api_secret: <shared-secret>
telemetry:
  disabled: false
  organization_name: <optional>
```

### 7.3 MCP Server Docker Compose

```yaml
# docker-compose.yml in mcp/zep-mcp-server/
services:
  zep-mcp-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ZEP_API_KEY=${ZEP_API_KEY}
      - LOG_LEVEL=info
```

---

## 8. Security & Auth

| Concern | Implementation |
|---|---|
| **API Auth (CE)** | Shared secret in `Authorization` header (`api_secret` config) |
| **API Auth (Cloud)** | API Key (`ZEP_API_KEY`) via Bearer token |
| **MCP Server** | Read-only; inherits Zep Cloud API key auth |
| **CORS (CE)** | All origins allowed (configured for development use) |
| **Request size** | 5MB max enforced by middleware |
| **Timeout** | 30s server context timeout |
| **Soft deletes** | All entities use `deleted_at` instead of hard deletion |
| **Multi-tenancy (CE)** | `project_uuid` on all entities; schema-based isolation |

---

## 9. Observability

| Signal | Implementation |
|---|---|
| **Structured logging** | `log/slog` (MCP server); custom structured logger (CE) |
| **Distributed tracing** | OpenTelemetry via `otelchi` middleware |
| **Telemetry** | Anonymous usage events (can be disabled) |
| **Health check** | `GET /healthz` endpoint (chi Heartbeat middleware) |
| **Request logging** | Method, path, duration, status, request ID on every request |

---

## 10. Database Schema Details

### 10.1 PostgreSQL Tables

| Table | PK | Key Columns | Indexes |
|---|---|---|---|
| `users` | `uuid` | `user_id` (unique), `project_uuid` | `user_user_id_idx`, `user_email_idx` |
| `sessions` | `uuid` | `session_id` (unique), `user_id`, `project_uuid` | `session_user_id_idx`, composite `(session_id, project_uuid, deleted_at)` |
| `messages` | `uuid` | `session_id`, `role`, `role_type`, `content`, `project_uuid` | `memstore_session_id_idx`, `memstore_id_idx`, composite `(session_id, project_uuid, deleted_at)` |

### 10.2 Migration System

Migrations are applied at startup via `store.MigrateSchema()` → `migrations.Migrate()`. The `bun` ORM manages the migration runner.

### 10.3 Role Types (enum)

```sql
CREATE TYPE role_type_enum AS ENUM (
  'norole', 'system', 'assistant', 'user', 'function', 'tool'
);
```

---

## 11. SDK & API Reference

### 11.1 Official SDKs

| Language | Package | Version |
|---|---|---|
| Python | `zep-cloud` | `>=3.0.0` |
| TypeScript/JS | `@getzep/zep-cloud` | latest |
| Go | `github.com/getzep/zep-go/v3` | v3 |

### 11.2 Key Python SDK Patterns

```python
from zep_cloud.client import AsyncZep

zep = AsyncZep(api_key=os.getenv("ZEP_API_KEY"))

# Create user
await zep.user.add(user_id="alice", first_name="Alice")

# Create thread
await zep.thread.create(user_id="alice", thread_id="thread_1")

# Add messages
await zep.thread.add_messages(thread_id="thread_1", messages=[...])

# Get context
context = await zep.thread.get_user_context(thread_id="thread_1")

# Graph search
results = await zep.graph.search(user_id="alice", query="preferences")

# Graph add
await zep.graph.add(user_id="alice", data="Alice prefers Python", type="text")

# Set ontology
await zep.graph.set_ontology(graph_id="kg", entities={...}, edges={})
```

---

## 12. Design Decisions & Rationale

| Decision | Rationale |
|---|---|
| **Temporal KG over vector-only store** | Facts need `valid_at`/`invalid_at` to track how context evolves — pure vector search loses temporal reasoning |
| **Graphiti as separate service** | Decouples graph extraction (LLM-heavy) from the API server (low-latency CRUD) |
| **PostgreSQL advisory locks** | Prevents race conditions on concurrent metadata updates without application-level locking infrastructure |
| **Read-only MCP server** | Safety-first for AI assistant access; write operations are done via SDKs with explicit developer intent |
| **CE deprecation** | Operational complexity of self-hosting Neo4j + Graphiti + Postgres; Cloud offers better SLAs |
| **Soft deletes everywhere** | Preserves audit trail; allows temporal analysis of "deleted" data |
| **UV workspace for Python** | Single `uv sync` manages all integration packages; consistent tooling across integrations |
| **MCP Streamable HTTP** | Supports both stateless JSON and streaming SSE; forward-compatible with 2025-03-26 MCP spec |

---

## 13. Known Limitations & Gotchas

> [!WARNING]
> The Community Edition (`legacy/`) is **deprecated and unsupported**. Do not build new features against it.

> [!NOTE]
> Graph data ingestion via Graphiti is **asynchronous** — allow 10-20 seconds after `graph.add()` before search results reflect the new data.

> [!CAUTION]
> The MCP server is **read-only** by design. Write operations (adding messages, ingesting data) must be performed via the Zep SDK or REST API directly.

> [!TIP]
> Use `search_filters` (`node_labels`, `edge_types`) in graph search to improve precision and reduce latency. Searching with `scope="all"` is more expensive than scoped searches.

---

## 14. Extension Points

### Adding a New Framework Integration

1. Create `integrations/python/zep_{framework}/` following the standard structure
2. Implement the framework's memory interface (see `CLAUDE.md` for patterns)
3. Add to `test-integrations.yml` GitHub Actions filter
4. Release tag format: `zep-{framework}-v{version}`

### Adding a New MCP Tool

1. Define input struct in `internal/handlers/types.go`
2. Implement handler in `internal/handlers/{feature}.go`
3. Define tool descriptor in `internal/server/tools.go`
4. Register via `mcp.AddTool[Input, any](...)` in `server.registerTools()`

### Custom Ontology

```python
from ontology.default_ontology import ZEP_NODE_ONTOLOGY, ZEP_EDGE_ONTOLOGY
from pydantic import BaseModel, Field

class CustomEntity(BaseModel):
    """Custom entity for domain-specific extraction."""
    attribute: str = Field(..., description="...")

CUSTOM_NODE_ONTOLOGY = {**ZEP_NODE_ONTOLOGY, "CustomEntity": CustomEntity}
```

---

*Document generated from source analysis of `github.com/getzep/zep` at commit HEAD (2026-05-02).*
