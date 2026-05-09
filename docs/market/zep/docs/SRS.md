# Zep — Software Requirements Specification (SRS)

> **Product**: Zep — End-to-End Context Engineering Platform  
> **Version**: 1.0  
> **Generated**: 2026-05-07  
> **Status**: Active

---

## 1. Introduction

### 1.1 Purpose
This SRS defines the technical architecture, component specifications, API contracts, data models, and infrastructure requirements for the Zep context engineering platform.

### 1.2 Scope
Covers: Zep Cloud API, Legacy Community Edition (Go), MCP Server (Go), Python Framework Integrations, Eval Harness, and Benchmarking Infrastructure.

### 1.3 Technology Stack Summary

| Component | Language | Key Dependencies |
|---|---|---|
| Legacy CE API | Go 1.21+ | chi v5, bun ORM, otelchi |
| MCP Server | Go 1.21+ | MCP Go SDK, zep-go/v3 |
| Integrations | Python 3.10+ | zep-cloud ≥3.0, UV workspace |
| Eval Harness | Python 3.13+ | zep-cloud, google-genai |
| Databases | — | PostgreSQL + pgvector, Neo4j 5.22+ |

---

## 2. System Architecture

### 2.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Client Layer                           │
│  Python SDK │ TS SDK │ Go SDK │ REST API │ MCP Server       │
└──────────────┬──────────────────────────────────────────────┘
               │
┌──────────────▼──────────────────────────────────────────────┐
│                    Zep Cloud API                            │
│         (gRPC/REST, Auth, Rate-limiting)                    │
└──────────────┬──────────────────────────────────────────────┘
               │
      ┌────────┴────────┐
      ▼                 ▼
┌──────────┐    ┌──────────────────┐
│ Postgres │    │    Graphiti      │
│ (CRUD)   │    │  (Graph RAG)     │
└──────────┘    └────────┬─────────┘
                         │
                   ┌─────▼──────┐
                   │   Neo4j    │
                   └────────────┘
```

### 2.2 Component Overview

| Component | Responsibility | Location |
|---|---|---|
| Legacy CE API | REST API server for User/Session/Memory CRUD | `legacy/src/` |
| Graphiti Service | Graph extraction, fact storage, temporal KG | External service |
| MCP Server | Read-only MCP tools for AI assistants | `mcp/zep-mcp-server/` |
| Framework Integrations | AutoGen, CrewAI, ADK, LiveKit memory adapters | `integrations/python/` |
| Eval Harness | End-to-end QA evaluation pipeline | `zep-eval-harness/` |
| Ontology | Default entity/edge type definitions | `ontology/` |
| Benchmarks | LoCoMo and LongMemEval benchmarks | `benchmarks/` |

---

## 3. Data Model

### 3.1 Domain Entities

#### User
```sql
CREATE TABLE users (
  uuid          UUID PRIMARY KEY,
  user_id       VARCHAR UNIQUE NOT NULL,
  email         VARCHAR,
  first_name    VARCHAR,
  last_name     VARCHAR,
  project_uuid  UUID NOT NULL,
  metadata      JSONB,
  created_at    TIMESTAMPTZ DEFAULT NOW(),
  updated_at    TIMESTAMPTZ DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ  -- soft delete
);
CREATE INDEX user_user_id_idx ON users(user_id);
CREATE INDEX user_email_idx ON users(email);
```

#### Session
```sql
CREATE TABLE sessions (
  uuid          UUID PRIMARY KEY,
  session_id    VARCHAR UNIQUE NOT NULL,
  user_id       VARCHAR REFERENCES users(user_id),
  project_uuid  UUID NOT NULL,
  metadata      JSONB,
  ended_at      TIMESTAMPTZ,  -- marks session closed
  created_at    TIMESTAMPTZ DEFAULT NOW(),
  updated_at    TIMESTAMPTZ DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ
);
CREATE INDEX session_user_id_idx ON sessions(user_id);
CREATE UNIQUE INDEX ON sessions(session_id, project_uuid, deleted_at);
```

#### Message
```sql
CREATE TYPE role_type_enum AS ENUM (
  'norole', 'system', 'assistant', 'user', 'function', 'tool'
);

CREATE TABLE messages (
  uuid          UUID PRIMARY KEY,
  session_id    VARCHAR NOT NULL REFERENCES sessions(session_id),
  project_uuid  UUID NOT NULL,
  role          VARCHAR NOT NULL,
  role_type     role_type_enum NOT NULL,
  content       TEXT NOT NULL,
  token_count   INTEGER,
  metadata      JSONB,
  created_at    TIMESTAMPTZ DEFAULT NOW(),
  updated_at    TIMESTAMPTZ DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ
);
CREATE INDEX memstore_session_id_idx ON messages(session_id);
CREATE UNIQUE INDEX ON messages(session_id, project_uuid, deleted_at);
```

#### Memory (API Overlay — not persisted)
```
Memory {
  messages:       []Message
  relevant_facts: []Fact
  metadata:       map[string]any
}
```

#### Fact (Graphiti Edge)
```
Fact {
  uuid:       UUID
  name:       string
  fact:       string         -- human-readable statement
  created_at: time.Time
  valid_at:   *time.Time     -- when fact became true
  invalid_at: *time.Time     -- when fact ceased to be true
  expired_at: *time.Time
}
```

### 3.2 Knowledge Graph Ontology

**Node Types** (priority-ordered):

| Type | Priority | Description | Pydantic Fields |
|---|---|---|---|
| `User` | Highest | Zep user (singleton) | `user_id`, `role_type`, `email`, `first_name`, `last_name` |
| `Assistant` | Highest | AI assistant (singleton) | `assistant_name` |
| `Preference` | Very High | Choices, opinions | *(marker class)* |
| `Organization` | High | Companies, groups | *(marker class)* |
| `Event` | High | Time-bound activities | *(marker class)* |
| `Location` | Medium | Physical/virtual places | *(marker class)* |
| `Document` | Medium | Content forms | *(marker class)* |
| `Topic` | Low | Conversation subjects | *(marker class)* |
| `Object` | Lowest | Physical items | *(marker class)* |

**Edge Types**:

| Type | Connects | Description |
|---|---|---|
| `LOCATED_AT` | Entity → Location | Entity exists at a location |
| `OCCURRED_AT` | Event → Entity/Location | Event happened at time/place |

---

## 4. API Specification

### 4.1 REST API (Legacy CE — `/api/v2`)

#### Session Endpoints

| Method | Path | Handler | Description |
|---|---|---|---|
| GET | `/sessions` | `GetSessionListHandler` | List all sessions |
| POST | `/sessions` | `CreateSessionHandler` | Create new session |
| GET | `/sessions-ordered` | `GetOrderedSessionListHandler` | Paginated ordered list |
| POST | `/sessions/search` | `SearchSessionsHandler` | Search sessions |
| GET | `/sessions/{id}` | `GetSessionHandler` | Get session details |
| PATCH | `/sessions/{id}` | `UpdateSessionHandler` | Update session |

#### Memory Endpoints

| Method | Path | Handler | Description |
|---|---|---|---|
| GET | `/sessions/{id}/memory` | `GetMemoryHandler` | Retrieve memory (messages + facts) |
| POST | `/sessions/{id}/memory` | `PostMemoryHandler` | Add messages to memory |
| DELETE | `/sessions/{id}/memory` | `DeleteMemoryHandler` | Delete session memory |

#### Message Endpoints

| Method | Path | Handler | Description |
|---|---|---|---|
| GET | `/sessions/{id}/messages` | `GetMessagesForSessionHandler` | List session messages |
| GET | `/sessions/{id}/messages/{uuid}` | `GetMessageHandler` | Get specific message |
| PATCH | `/sessions/{id}/messages/{uuid}` | `UpdateMessageMetadataHandler` | Update message metadata |

#### User Endpoints

| Method | Path | Handler | Description |
|---|---|---|---|
| GET | `/users` | `ListAllUsersHandler` | List all users |
| POST | `/users` | `CreateUserHandler` | Create user |
| GET | `/users/{id}` | `GetUserHandler` | Get user details |
| PATCH | `/users/{id}` | `UpdateUserHandler` | Update user |
| DELETE | `/users/{id}` | `DeleteUserHandler` | Soft-delete user |
| GET | `/users/{id}/sessions` | `ListUserSessionsHandler` | List user's sessions |

#### Fact Endpoints

| Method | Path | Handler | Description |
|---|---|---|---|
| GET | `/facts/{uuid}` | `GetFactHandler` | Retrieve fact |
| DELETE | `/facts/{uuid}` | `DeleteFactHandler` | Delete fact |

### 4.2 Graphiti Internal API

| Method | Path | Purpose |
|---|---|---|
| POST | `/messages` | Ingest messages into KG |
| POST | `/get-memory` | Retrieve relevant facts |
| POST | `/search` | Semantic fact search |
| POST | `/entity-node` | Add entity node |
| GET | `/entity-edge/{uuid}` | Get specific fact |
| DELETE | `/entity-edge/{uuid}` | Remove fact |
| DELETE | `/group/{id}` | Remove group data |
| DELETE | `/episode/{uuid}` | Remove episode |

### 4.3 MCP Server Tools (13 read-only)

| Tool | Input | Output |
|---|---|---|
| `search_graph` | `user_id`, `query`, `scope`, `limit`, `reranker`, `node_labels[]`, `edge_types[]`, `min_fact_rating`, `mmr_lambda`, `center_node_uuid` | JSON search results |
| `get_user_context` | `thread_id`, `template_id?` | Formatted context string |
| `get_user` | `user_id` | User metadata JSON |
| `list_threads` | `user_id` | Thread list JSON |
| `get_user_nodes` | `user_id`, `limit?` | Node list JSON |
| `get_user_edges` | `user_id`, `limit?` | Edge list JSON |
| `get_episodes` | `user_id`, `lastn?` | Episode list JSON |
| `get_thread_messages` | `thread_id`, `limit?` | Message list JSON |
| `get_node` | `node_uuid` | Node detail JSON |
| `get_edge` | `edge_uuid` | Edge detail JSON |
| `get_episode` | `episode_uuid` | Episode detail JSON |
| `get_node_edges` | `node_uuid` | Connected edges JSON |
| `get_episode_mentions` | `episode_uuid` | Mentioned nodes/edges JSON |

**Search Scope Values**: `edges`, `nodes`, `episodes`

**Reranker Strategies**: `rrf`, `mmr`, `node_distance`, `episode_mentions`, `cross_encoder`

---

## 5. Component Specifications

### 5.1 Legacy CE Server (`legacy/src/`)

**Startup Sequence**:
```
main() → config.Load() → logger.InitDefaultLogger() → newAppState()
       → api.Create(appState) → srv.ListenAndServe()
```

**Middleware Stack** (ordered):
1. CORS (allow all origins)
2. Structured request logging (method, path, duration, status)
3. Health check (`GET /healthz`)
4. Request size limiter (5MB)
5. Request ID injection (UUID)
6. Context timeout (30s)
7. Real IP extraction
8. Clean path
9. Version header
10. OpenTelemetry tracing (`otelchi`)

**Memory DAO — PutMemory Flow**:
```
PutMemory(sessionID, messages)
  → upsert Session
  → check session.EndedAt (reject if ended)
  → messageDAO.CreateMany(messages)
  → _initializeProcessingMemory()
      → graphiti.PutMemory(sessionID, messages)
      → graphiti.PutMemory(userID, messages)  // if user associated
```

**Memory DAO — GetMemory Flow**:
```
GetMemory(sessionID, lastN)
  → messageDAO.GetLastN(max(lastN, 4))
  → graphiti.GetMemory(groupID, maxFacts=5, last4Messages)
  → return Memory{messages[lastN:], relevantFacts}
```

**Concurrency Control**: PostgreSQL advisory locks (SHA-256 hash of session ID) with exponential backoff (200ms→30s, max 15 retries).

### 5.2 MCP Server (`mcp/zep-mcp-server/`)

**Transport Modes**:

| Mode | Flag | Protocol |
|---|---|---|
| HTTP (default) | — | MCP Streamable HTTP (2025-03-26 spec) |
| stdio | `--stdio` | stdin/stdout JSON-RPC |

**Internal Structure**:
```
cmd/server/main.go      — Entry point
internal/config/         — Environment + .env loading
internal/handlers/       — 15 handler files (13 tools + types + tests)
internal/server/         — MCP server setup + tool registration
internal/transform/      — JSON formatting + validation
pkg/zep/                 — Zep SDK client wrapper
```

**Configuration**: `ZEP_API_KEY` (required), `LOG_LEVEL` (optional, default: `info`), `--port` (HTTP mode, default: 8080).

### 5.3 Framework Integrations (`integrations/python/`)

**Available Integrations**:

| Package | Framework | Memory Interface |
|---|---|---|
| `zep-autogen` | Microsoft AutoGen | `autogen_core.memory.Memory` |
| `zep-crewai` | CrewAI | `ZepUserStorage` + `ZepGraphStorage` |
| `zep-adk` | Google ADK | ADK memory interface |
| `zep-livekit` | LiveKit | LiveKit memory interface |

**CrewAI Dual Storage**:
- `ZepUserStorage(client, user_id, thread_id?, facts_limit=20, entity_limit=5, mode="summary"|"raw_messages")`
- `ZepGraphStorage(client, graph_id)` — shared knowledge graphs with custom ontologies
- Tool factories: `create_search_tool()`, `create_add_data_tool()`

**Quality Requirements**: ruff linting, mypy type checking, pytest >90% coverage, CI via GitHub Actions (Python 3.10–3.13).

### 5.4 Eval Harness (`zep-eval-harness/`)

**Pipeline Scripts**:

| Script | Purpose | Output |
|---|---|---|
| `zep_ingest_users.py` | Ingest users + conversations + telemetry | `runs/users/{N}/manifest.json` |
| `zep_chunk_documents.py` | Chunk + LLM-contextualize documents | `runs/chunk_sets/{N}/chunks.jsonl` |
| `zep_ingest_documents.py` | Ingest chunks into document graph | `runs/documents/{N}/manifest.json` |
| `zep_evaluate.py` | Search → Context eval → Generate → Grade | `runs/evaluations/{N}/results.json` |
| `zep_graph_inspect.py` | Print graph nodes and edges | stdout |

**Evaluation Metrics**:
- **Primary**: Context Completeness — `COMPLETE` / `PARTIAL` / `INSUFFICIENT`
- **Secondary**: Answer Accuracy — `CORRECT` / `WRONG`
- **Breakdowns**: Per-category, per-user, correlation analysis

**Resilience**: Exponential backoff (up to 8 retries, max 5-min delay) for rate limits. Configurable concurrency via `--concurrency N`.

---

## 6. Infrastructure Requirements

### 6.1 Legacy CE Docker Services

| Service | Image | Port | Dependencies |
|---|---|---|---|
| `zep` | `zepai/zep:latest` | 8000 | graphiti (healthy), db (healthy) |
| `db` | `ankane/pgvector:v0.5.1` | 5432 | — |
| `graphiti` | `zepai/graphiti:0.3` | 8003 | neo4j (healthy) |
| `neo4j` | `neo4j:5.22.0` | 7474, 7687 | — |

### 6.2 MCP Server Docker

```yaml
services:
  zep-mcp-server:
    build: .
    ports: ["8080:8080"]
    environment:
      - ZEP_API_KEY=${ZEP_API_KEY}
      - LOG_LEVEL=info
```

### 6.3 Configuration (`zep.yaml`)

```yaml
log:
  level: info          # debug|info|warn|error|panic|fatal
  format: json         # json|console
http:
  host: 0.0.0.0
  port: 8000
  max_request_size: 5242880  # 5MB
postgres:
  host: db
  port: 5432
  user: postgres
  password: postgres
  database: postgres
  schema_name: public
  max_open_connections: 10
  read_timeout: 30
  write_timeout: 30
graphiti:
  service_url: http://graphiti:8003
api_secret: <shared-secret>
telemetry:
  disabled: false
```

---

## 7. Security Requirements

| Concern | Specification |
|---|---|
| **API Auth (Cloud)** | Bearer token via `ZEP_API_KEY` |
| **API Auth (CE)** | Shared secret in `Authorization` header |
| **MCP Server** | Read-only; inherits Zep Cloud API key |
| **CORS (CE)** | All origins allowed (dev mode) |
| **Request size** | 5MB max enforced by middleware |
| **Timeout** | 30s server context timeout |
| **Data deletion** | Soft deletes (`deleted_at`) on all entities |
| **Multi-tenancy** | `project_uuid` on all entities |
| **Compliance (Cloud)** | SOC2 Type 2, HIPAA |

---

## 8. Observability Requirements

| Signal | Implementation |
|---|---|
| **Structured logging** | `log/slog` (MCP), custom logger (CE); JSON or console format |
| **Distributed tracing** | OpenTelemetry via `otelchi` middleware |
| **Health check** | `GET /healthz` (chi Heartbeat middleware) |
| **Request logging** | Method, path, duration, status, request ID, response size |
| **Telemetry** | Anonymous usage events (opt-out via config) |

---

## 9. Database Schema

### 9.1 Migration System
Migrations applied at startup via `store.MigrateSchema()` → `migrations.Migrate()` using `bun` ORM migration runner.

### 9.2 Index Strategy

| Table | Index | Type |
|---|---|---|
| `users` | `user_user_id_idx(user_id)` | Unique |
| `users` | `user_email_idx(email)` | Standard |
| `sessions` | `session_user_id_idx(user_id)` | Standard |
| `sessions` | `(session_id, project_uuid, deleted_at)` | Composite unique |
| `messages` | `memstore_session_id_idx(session_id)` | Standard |
| `messages` | `(session_id, project_uuid, deleted_at)` | Composite unique |

---

## 10. Extension Points

### 10.1 Adding a New Framework Integration
1. Create `integrations/python/zep_{framework}/` with standard structure
2. Implement framework's memory interface
3. Add to `test-integrations.yml` GitHub Actions
4. Release tag: `zep-{framework}-v{version}`

### 10.2 Adding a New MCP Tool
1. Define input struct in `internal/handlers/types.go`
2. Implement handler in `internal/handlers/{feature}.go`
3. Define tool descriptor in `internal/server/tools.go`
4. Register via `mcp.AddTool[Input, any](...)` in `server.registerTools()`

### 10.3 Custom Ontology
```python
from ontology.default_ontology import ZEP_NODE_ONTOLOGY, ZEP_EDGE_ONTOLOGY

class CustomEntity(BaseModel):
    """Domain-specific entity."""
    attribute: str = Field(..., description="...")

CUSTOM_NODES = {**ZEP_NODE_ONTOLOGY, "CustomEntity": CustomEntity}
```

---

## 11. Known Constraints

| Constraint | Details |
|---|---|
| **Async ingestion** | Graph extraction takes 10-20s; results not immediately searchable |
| **CE deprecated** | `legacy/` is unsupported; no new features |
| **MCP read-only** | Write operations require SDK/REST API |
| **Search cost** | `scope="all"` is more expensive than scoped searches |
| **Advisory locks** | Concurrent metadata updates use SHA-256 hash-based locks |

---

*Document generated from source analysis of `github.com/getzep/zep` repository.*
