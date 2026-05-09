# Software Requirements Specification (SRS)

## OpenViking — Context Database for AI Agents

| Field           | Value                          |
| --------------- | ------------------------------ |
| **Product**     | OpenViking                     |
| **Version**     | 0.1.x (Alpha)                  |
| **License**     | AGPL-3.0 / Apache 2.0 (CLI)   |
| **Last Updated**| 2026-05-07                     |

---

## 1. System Overview

OpenViking is an Agent-native Context Database using a filesystem paradigm (`viking://` URI protocol) to unify memories, resources, and skills management for AI Agents. The system replaces flat vector storage with hierarchical, tiered context loading (L0/L1/L2).

### 1.1 Architecture Layers

```
┌────────────────────────────────────────────────────────┐
│  Presentation: FastAPI HTTP + MCP Streamable HTTP      │
├────────────────────────────────────────────────────────┤
│  Service: FS, Search, Session, Resource, Relations     │
├────────────────────────────────────────────────────────┤
│  Core Domain: Context, Namespace, ContextType/Level    │
├────────────────────────────────────────────────────────┤
│  Infrastructure: VikingFS, VikingDB, RAGFS, Embedders  │
│  Crypto, Queue, Lock, Telemetry, Parsers               │
└────────────────────────────────────────────────────────┘
```

### 1.2 Technology Stack

| Component        | Technology                                            |
|------------------|-------------------------------------------------------|
| Core Language    | Python 3.10+ (FastAPI/Pydantic)                       |
| CLI/FS Engine    | Rust (ov_cli, ragfs crates)                           |
| Native Extensions| C++ (vector engine bindings)                          |
| API Framework    | FastAPI + Uvicorn                                     |
| Storage          | RAGFS (custom FS), embedded vector index              |
| Embedding        | OpenAI, Volcengine, Gemini, Jina, Cohere, etc. (12+) |
| VLM              | OpenAI, Volcengine, Gemini, Kimi, GLM                 |
| Encryption       | AES-256-GCM (cryptography lib)                        |
| Observability    | OpenTelemetry + Prometheus                            |
| Containerization | Docker (multi-stage) + Helm                           |

---

## 2. Module Specifications

### 2.1 VikingFS — Virtual Filesystem

**Module**: `openviking.storage.viking_fs`

**Responsibility**: Unified virtual filesystem managing all context under `viking://` URIs.

**URI Namespace**:
- `viking://resources/` — Project documents, repos, web pages
- `viking://user/{account}/{user}/` — User memories, privacy configs
- `viking://agent/{account}/{user}/{agent}/` — Agent skills, memories, instructions
- `viking://session/{session_id}/` — Active session data

**Core Operations**:

| Operation   | Method       | Description                          |
|-------------|-------------|--------------------------------------|
| `read`      | Read file    | Returns content string               |
| `read_batch`| Batch read   | Multiple URIs, level-aware (l0/l1/l2)|
| `write_file`| Write file   | Create/overwrite file content        |
| `mkdir`     | Create dir   | With `exist_ok` option               |
| `ls`        | List dir     | Returns entries with metadata        |
| `tree`      | Tree listing | Recursive with depth limit           |
| `rm`        | Delete       | Removes file/directory               |
| `stat`      | File info    | Returns metadata (size, dates, etc.) |
| `mv`/`cp`   | Move/Copy    | URI-based file operations            |
| `find`      | Search       | Semantic search within scope         |
| `grep`      | Pattern match| Regex search in file content         |
| `glob`      | Glob match   | Filename pattern matching            |
| `get_relations`| Relations | Retrieve related context URIs        |

**Data Model — Context**:

```python
@dataclass
class Context:
    uri: str                    # viking:// URI
    parent_uri: str             # Parent directory URI
    context_type: ContextType   # MEMORY | RESOURCE | SKILL | SESSION
    level: int                  # 0=Abstract, 1=Overview, 2=Detail
    owner_account_id: str       # Tenant account
    owner_user_id: str          # User within account
    owner_agent_id: str         # Agent within user scope
    abstract: str               # L0 summary (~100 tokens)
    category: str               # Sub-classification
    active_count: int           # Usage counter
    created_at: datetime
    updated_at: datetime
    meta: Dict[str, Any]        # Extensible metadata
```

### 2.2 VikingDBManager — Vector Storage

**Module**: `openviking.storage.vikingdb_manager`

**Responsibility**: Manages vector index collections for semantic search.

**Key Operations**:
- `collection_exists()` — Check if tenant collection exists
- `search_global_roots_in_tenant()` — Global vector search across L0/L1
- `search_children_in_tenant()` — Search children under a parent URI
- `upsert_context()` — Insert/update context with embeddings
- `delete_context()` — Remove context from vector index

**Vector Types**:
- Dense vectors (float32, 768-3072 dims depending on model)
- Sparse vectors (BM25/SPLADE-style keyword features)
- Hybrid search combining both

### 2.3 HierarchicalRetriever — Search Engine

**Module**: `openviking.retrieve.hierarchical_retriever`

**Algorithm**:

```
1. Parse query → TypedQuery (query, context_type, target_directories)
2. Embed query → dense_vector + sparse_vector
3. Global search → top-K candidates across all L0/L1 nodes
4. Merge starting points (root URIs + global hits)
5. Recursive directory search:
   a. Pop highest-score directory from priority queue
   b. Search children in tenant (vector similarity)
   c. Optional rerank (model-based scoring)
   d. Score propagation: final = α·child + (1-α)·parent
   e. Push sub-directories to queue, collect L2 files
   f. Convergence check: stop after 3 rounds with stable top-K
6. Convert to MatchedContext with hotness blending
7. Return QueryResult (matched_contexts, searched_directories)
```

**Configuration** (`RetrievalConfig`):
- `hotness_alpha` — Weight for recency/activity boosting (0 disables)
- `score_propagation_alpha` — Parent→child score blending factor
- `threshold` — Minimum relevance score

**Rerank Providers**: Volcengine, OpenAI, Cohere, Jina, local models

### 2.4 Session Manager

**Module**: `openviking.session.session`

**Session Data Model**:

```python
class SessionMeta:
    session_id: str
    created_at: str
    participant_user_ids: List[str]
    participant_agent_ids: List[str]
    message_count: int
    commit_count: int
    memories_extracted: Dict[str, int]  # 8 categories
    llm_token_usage: Dict[str, int]
    pending_tokens: int          # WM v2 sliding window
    keep_recent_count: int       # Retained tail size
```

**Two-Phase Commit**:

| Phase | Scope          | Protection       | Action                              |
|-------|----------------|-------------------|-------------------------------------|
| 1     | Archive        | PathLock (distributed) | Split messages, write archive JSONL |
| 2     | Memory Extract | Background task   | WM v2 update, memory extraction, active_count |

**Working Memory v2** — 7-section structured document:
1. Session Title
2. Current State
3. Task & Goals
4. Key Facts & Decisions
5. Files & Context
6. Errors & Corrections
7. Open Issues

Section-level operations: `KEEP`, `UPDATE`, `APPEND`

### 2.5 Authentication & Authorization

**Module**: `openviking.server.auth`

**Auth Modes**:

| Mode      | Mechanism                        | Use Case               |
|-----------|----------------------------------|------------------------|
| `DEV`     | No auth, ROOT role, localhost    | Local development      |
| `API_KEY` | Root key + per-user keys, RBAC   | Production deployment  |
| `TRUSTED` | Trust gateway headers            | Behind API gateway     |

**RBAC Roles**:

| Role    | Scope                          | Permissions                    |
|---------|--------------------------------|--------------------------------|
| `ROOT`  | Global                         | All operations, admin APIs     |
| `ADMIN` | Account-scoped                 | User/key management within acct|
| `USER`  | User-scoped                    | Data CRUD within own namespace |

**Identity Headers**:
- `X-OpenViking-Account` — Tenant account ID
- `X-OpenViking-User` — User ID within account
- `X-OpenViking-Agent` — Agent ID within user scope
- `X-Api-Key` / `Authorization: Bearer` — API key

### 2.6 Encryption Subsystem

**Module**: `openviking.crypto.encryptor`

**Envelope Encryption Format** (`OVE1`):

```
┌──────────┬─────────┬──────────┬───────────┬──────────┬──────────┐
│ Magic 4B │ Ver 1B  │ Prov 1B  │ EFK Len 2B│ KIV Len 2B│ DIV Len 2B│
├──────────┴─────────┴──────────┴───────────┴──────────┴──────────┤
│ Encrypted File Key │ Key IV │ Data IV │ AES-GCM Ciphertext      │
└────────────────────┴────────┴─────────┴─────────────────────────┘
```

**Key Hierarchy**: Root Key → Account Key → File Key (random per-file)

**KMS Providers**:
- `LocalFileProvider` — Key file on disk
- `VaultProvider` — HashiCorp Vault
- `VolcengineKMSProvider` — Volcengine Cloud KMS

### 2.7 MCP Endpoint

**Module**: `openviking.server.mcp_endpoint`

**Transport**: Streamable HTTP at `/mcp` path

**9 Tools**: `search`, `read`, `list`, `store`, `add_resource`, `grep`, `glob`, `forget`, `health`

**Identity Propagation**: ASGI middleware extracts identity headers → `contextvars` → tool handlers

### 2.8 Embedding Providers

**Module**: `openviking.models.embedder`

| Provider     | Dense | Sparse | Hybrid |
|-------------|-------|--------|--------|
| OpenAI       | ✅    | ❌     | ❌     |
| Volcengine   | ✅    | ✅     | ✅     |
| Gemini       | ✅    | ❌     | ❌     |
| Jina         | ✅    | ❌     | ❌     |
| Cohere       | ✅    | ❌     | ❌     |
| DashScope    | ✅    | ✅     | ✅     |
| MiniMax      | ✅    | ❌     | ❌     |
| Voyage       | ✅    | ❌     | ❌     |
| LiteLLM      | ✅    | ❌     | ❌     |
| VikingDB     | ✅    | ✅     | ✅     |
| Local (ONNX) | ✅    | ❌     | ❌     |

### 2.9 VLM Providers

**Module**: `openviking.models.vlm`

Used for: L0/L1 summary generation, image content extraction, complex document parsing.

Providers: OpenAI, Volcengine, Gemini, Kimi, GLM, LiteLLM (proxy)

### 2.10 Privacy Service

**Module**: `openviking.privacy.service`

**Features**:
- Per-user privacy configuration with version history
- Category/target-key scoping
- Version activation/rollback
- Stored under `viking://user/{id}/privacy/`

### 2.11 VikingBot

**Module**: `bot/vikingbot/`

**Components**: Agent framework, multi-channel bus, sandbox, FUSE mount, console UI

**Channels**: Telegram, Feishu/Lark, DingTalk, Slack, QQ, Discord

---

## 3. API Specification

### 3.1 Core API Routes

| Group       | Path Prefix                    | Methods                        |
|-------------|-------------------------------|--------------------------------|
| Filesystem  | `/api/v1/filesystem`          | GET/POST (ls, tree, mkdir, rm) |
| Content     | `/api/v1/content`             | GET/POST (read, write, mv, cp)|
| Search      | `/api/v1/search`              | POST (find, grep, glob)       |
| Resources   | `/api/v1/resources`           | POST/GET/DELETE                |
| Sessions    | `/api/v1/sessions`            | POST/GET/DELETE (CRUD, commit) |
| Relations   | `/api/v1/relations`           | GET/POST/DELETE                |
| Admin       | `/api/v1/admin`               | Account/user/key management   |
| Observer    | `/api/v1/observer`            | Retrieval stats/replay        |
| Privacy     | `/api/v1/privacy-configs`     | GET/POST (config CRUD)        |
| Tasks       | `/api/v1/tasks`               | GET (task status tracking)    |
| System      | `/api/v1/system`              | GET (status, wait, debug)     |
| MCP         | `/mcp`                        | POST (Streamable HTTP)        |
| Metrics     | `/metrics`                    | GET (Prometheus)              |
| WebDAV      | `/webdav`                     | Standard WebDAV methods       |

### 3.2 Error Response Format

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found: viking://resources/example",
    "details": {}
  }
}
```

**Error Codes**: `INVALID_ARGUMENT`, `NOT_FOUND`, `PERMISSION_DENIED`, `UNAUTHENTICATED`, `ALREADY_EXISTS`, `FAILED_PRECONDITION`, `INTERNAL`, `UNAVAILABLE`

---

## 4. Infrastructure Requirements

### 4.1 Runtime Dependencies

| Dependency              | Required | Notes                           |
|------------------------|----------|---------------------------------|
| Python 3.10+           | ✅       | Core runtime                    |
| Rust toolchain         | Build    | CLI + RAGFS compilation         |
| C++ compiler           | Build    | Native extensions               |
| Embedding API key      | ✅       | At least one provider           |
| VLM API key            | ✅       | For summary generation          |
| Ollama (optional)      | ❌       | Local model alternative         |

### 4.2 Hardware Recommendations

| Tier         | CPU   | RAM   | Disk   | Users       |
|-------------|-------|-------|--------|-------------|
| Development | 2 CPU | 4 GB  | 10 GB  | 1           |
| Small Team  | 4 CPU | 8 GB  | 50 GB  | 1-10        |
| Production  | 8 CPU | 16 GB | 200 GB | 10-100      |

### 4.3 Deployment

**Docker**:
```bash
docker run -d -p 1933:1933 \
  -v openviking_data:/data \
  -e OPENVIKING_CONFIG_FILE=/data/ov.conf \
  ghcr.io/volcengine/openviking:latest
```

**Kubernetes**: Helm chart at `examples/k8s-helm/`

**Environment Variables**:
- `OPENVIKING_CONFIG_FILE` — Server config path
- `OPENVIKING_CLI_CONFIG_FILE` — CLI config path

---

## 5. Observability

### 5.1 Telemetry

| Type     | Backend          | Key Signals                          |
|----------|------------------|--------------------------------------|
| Traces   | OpenTelemetry    | Request spans, retrieval path        |
| Metrics  | Prometheus       | Latency, throughput, error rates     |
| Logs     | Structured JSON  | Operation logs with trace correlation|

### 5.2 Key Metrics

- `vector.searches` — Vector search count
- `vector.scored` / `vector.passed` — Result filtering stats
- `memory.extracted` — Memories extracted per commit
- Request latency histograms per route
- Session lifecycle events

---

## 6. Configuration Schema

**File**: `ov.conf` (JSON)

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 1933,
    "auth_mode": "dev|api_key|trusted",
    "root_api_key": "",
    "workers": 1
  },
  "storage": {
    "workspace": "~/.openviking/data"
  },
  "embedding": {
    "dense": {
      "provider": "openai",
      "model": "text-embedding-3-small",
      "api_key": ""
    }
  },
  "vlm": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "api_key": ""
  },
  "rerank": {
    "provider": "jina",
    "model": "jina-reranker-v2-base-multilingual",
    "threshold": 0.35
  },
  "retrieval": {
    "hotness_alpha": 0.1,
    "score_propagation_alpha": 0.7
  },
  "encryption": {
    "enabled": false,
    "provider": "local"
  },
  "observability": {
    "telemetry_enabled": true,
    "metrics_enabled": true
  }
}
```

---

## 7. Concurrency & Safety

| Mechanism          | Implementation                                    |
|--------------------|----------------------------------------------------|
| Filesystem Locks   | PathLock (distributed, file-based)                 |
| Session Commit     | Point lock on session path during Phase 1          |
| Redo Log           | Write-ahead log for Phase 2 crash recovery         |
| Queue Manager      | Async task queue for background processing         |
| Data Directory Lock| Single-instance lock on workspace directory        |
| Convergence        | Retrieval stops after 3 stable rounds              |

---

## 8. Testing & Quality

| Type               | Framework / Approach                |
|--------------------|-------------------------------------|
| Unit Tests         | pytest + pytest-asyncio             |
| Integration Tests  | Docker-based end-to-end             |
| CLI Tests          | Rust `cargo test`                   |
| Load Testing       | Configurable concurrent sessions    |
| Retrieval Quality  | OpenClaw benchmark (task completion)|
