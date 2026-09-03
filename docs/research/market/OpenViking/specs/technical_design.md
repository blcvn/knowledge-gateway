# OpenViking: Technical Design Document

> **Repository**: `github.com/volcengine/OpenViking`  
> **Generated**: 2026-05-07  
> **Status**: Active (Alpha 0.1.x)

---

## 1. Executive Summary

OpenViking is an **Agent-native Context Database** that organizes AI Agent context (memories, resources, skills) under a virtual filesystem with `viking://` URIs. It employs three-tier context loading (Abstract/Overview/Detail), hierarchical recursive retrieval, and automated session-to-memory extraction to deliver high-precision, token-efficient context to LLM-based Agents.

---

## 2. System Overview

### 2.1 Core Value Proposition

| Capability | Description |
|---|---|
| **Filesystem Paradigm** | Unified `viking://` URI protocol replaces fragmented vector stores |
| **Tiered Context (L0/L1/L2)** | Abstract→Overview→Detail loading reduces token consumption by 80-90% |
| **Hierarchical Retrieval** | Directory-aware recursive search with score propagation and convergence |
| **Automated Memory** | Two-phase session commit with background memory extraction |
| **Multi-tenancy** | Account/User/Agent namespace isolation with RBAC |

### 2.2 Technology Stack

| Layer | Technology |
|---|---|
| **Core** | Python 3.10+ (FastAPI, Pydantic, asyncio) |
| **CLI + FS** | Rust (ov_cli, ragfs crates) |
| **Native** | C++ (vector engine bindings) |
| **Embedding** | 12+ providers (OpenAI, Volcengine, Gemini, Jina, Cohere, DashScope, etc.) |
| **VLM** | OpenAI, Volcengine, Gemini, Kimi, GLM |
| **Crypto** | AES-256-GCM (cryptography), Argon2id |

---

## 3. Repository Structure

```
OpenViking/
├── openviking/                  # Python server package
│   ├── server/                  # FastAPI app, routers, auth, MCP
│   │   ├── app.py               # Application factory + lifespan
│   │   ├── bootstrap.py         # Server startup orchestration
│   │   ├── config.py            # ServerConfig schema
│   │   ├── auth.py              # Authentication + RBAC
│   │   ├── mcp_endpoint.py      # MCP Streamable HTTP
│   │   └── routers/             # 17 API routers
│   ├── service/                 # Business logic layer
│   │   ├── core.py              # OpenVikingService composition
│   │   ├── fs_service.py        # Filesystem operations
│   │   ├── search_service.py    # Search operations
│   │   ├── session_service.py   # Session management
│   │   └── resource_service.py  # Resource ingestion
│   ├── core/                    # Domain model
│   │   ├── context.py           # Context record type
│   │   ├── namespace.py         # URI resolution + policies
│   │   └── directories.py       # Root directory bootstrap
│   ├── storage/                 # Storage backends
│   │   ├── viking_fs.py         # Virtual filesystem (2199 lines)
│   │   ├── vikingdb_manager.py  # Vector index management
│   │   ├── queuefs/             # Async task queue
│   │   └── transaction/         # Lock manager + redo log
│   ├── retrieve/                # Retrieval engine
│   │   ├── hierarchical_retriever.py  # Main retriever (627 lines)
│   │   └── memory_lifecycle.py  # Hotness scoring
│   ├── session/                 # Session subsystem
│   │   ├── session.py           # Session class (2629 lines)
│   │   └── compressor.py        # VLM-based compression
│   ├── crypto/                  # Encryption subsystem
│   │   ├── encryptor.py         # Envelope encryption
│   │   └── providers.py         # KMS provider adapters
│   ├── models/                  # AI model integrations
│   │   ├── embedder/            # 12 embedding providers
│   │   ├── vlm/                 # VLM backends
│   │   └── rerank.py            # Reranking client
│   ├── privacy/                 # Privacy config service
│   ├── resource/                # Resource processing
│   ├── telemetry/               # OpenTelemetry integration
│   └── metrics/                 # Prometheus metrics
├── crates/                      # Rust workspace
│   ├── ov_cli/                  # CLI binary
│   ├── ragfs/                   # Filesystem engine
│   └── openviking-py/           # Python bindings
├── bot/vikingbot/               # AI Agent bot framework
│   ├── agent/                   # Agent logic + tools
│   ├── channels/                # Telegram, Feishu, Slack, etc.
│   ├── sandbox/                 # Code execution sandbox
│   └── console/                 # Web console (Gradio)
├── examples/                    # Integration examples
│   ├── claude-code-memory-plugin/
│   ├── opencode-memory-plugin/
│   ├── codex-memory-plugin/
│   └── k8s-helm/
├── Dockerfile                   # Multi-stage (Rust + Python)
└── pyproject.toml               # Python project config
```

---

## 4. Core Data Model

### 4.1 Context (Primary Record)

```python
@dataclass
class Context:
    uri: str                      # viking:// URI (primary key)
    parent_uri: str               # Parent directory URI
    context_type: ContextType     # MEMORY | RESOURCE | SKILL | SESSION
    level: int                    # 0=Abstract, 1=Overview, 2=Detail
    owner_account_id: str         # Tenant account ID
    owner_user_id: str            # User within account
    owner_agent_id: str           # Agent within user scope
    abstract: str                 # L0 text (~100 tokens)
    category: str                 # Sub-classification
    active_count: int             # Usage counter
    created_at: datetime
    updated_at: datetime
    meta: Dict[str, Any]          # Extensible metadata
```

### 4.2 Session Data Model

```python
class SessionMeta:
    session_id: str
    created_at: str
    updated_at: str
    created_by_user_id: str
    participant_user_ids: List[str]
    participant_agent_ids: List[str]
    message_count: int
    total_message_count: int
    commit_count: int
    memories_extracted: Dict[str, int]   # 8 categories
    llm_token_usage: Dict[str, int]      # prompt/completion/total
    embedding_token_usage: Dict[str, int]
    pending_tokens: int                  # WM v2 sliding window
    keep_recent_count: int               # Retained tail size

class Message:
    id: str                   # msg_{uuid_hex}
    role: str                 # user | assistant | system | tool
    parts: List[Part]         # TextPart, ToolPart, ContextPart
    role_id: str              # User/Agent identifier
    created_at: str
    estimated_tokens: int
```

### 4.3 Working Memory v2 (7-Section Document)

```markdown
## Session Title
(auto-generated title)

## Current State
(what the agent is currently doing)

## Task & Goals
(objectives and sub-goals)

## Key Facts & Decisions
(important facts discovered)

## Files & Context
(files and URIs referenced)

## Errors & Corrections
(mistakes and corrections made)

## Open Issues
(unresolved items)
```

Section update operations: `KEEP` | `UPDATE` (full replace) | `APPEND` (add items)

---

## 5. Component Deep-Dives

### 5.1 VikingFS (Virtual Filesystem)

> Located in `openviking/storage/viking_fs.py` (2199 lines)

**Design**: Wraps a Rust-based RAGFS filesystem engine with Python-level access control, encryption, and semantic search capabilities.

**Singleton pattern**: `init_viking_fs()` → `get_viking_fs()` global instance

**Operation categories**:

| Category | Methods | Notes |
|---|---|---|
| Basic CRUD | `read`, `write`, `mkdir`, `rm`, `mv`, `stat`, `exists` | Delegate to RAGFS |
| Directory | `ls`, `tree` (original/agent format) | With depth/node limits |
| Search | `abstract`, `overview`, `read_batch` (level-aware) | L0/L1/L2 reads |
| Pattern | `grep`, `glob` | Regex/glob matching |
| Semantic | `find` (→ HierarchicalRetriever) | Vector search |
| Relations | `get_relations`, `add_relation`, `remove_relation` | `.relations.json` |

**Encryption integration**: Every `read()` decrypts, every `write()` encrypts — transparent to callers:

```python
async def read(self, uri, ...):
    raw = self.agfs.read(path, 0, -1)
    raw = await self._decrypt_content(raw, ctx=ctx)  # transparent
    return raw

async def write(self, uri, data, ...):
    data = await self._encrypt_content(data, ctx=ctx)  # transparent
    return self.agfs.write(path, data)
```

**URI security**: Rejects `..`, `\`, and drive-letter prefixed components:

```python
for part in parts:
    if part in {".", ".."}:
        raise PermissionDeniedError("Unsafe URI traversal")
    if "\\" in part:
        raise PermissionDeniedError("Unsafe path separator")
    if len(part) >= 2 and part[1] == ":" and part[0].isalpha():
        raise PermissionDeniedError("Unsafe drive-prefixed component")
```

### 5.2 HierarchicalRetriever (Search Engine)

> Located in `openviking/retrieve/hierarchical_retriever.py` (627 lines)

**Algorithm** — 6-step pipeline:

```
Step 1: Determine starting directories
  ├── From target_directories (explicit scope)
  └── From context_type → _get_root_uris_for_type()
         MEMORY  → [user/memories, agent/memories]
         RESOURCE → [viking://resources]
         SKILL   → [agent/skills]
         None    → all of the above

Step 2: Global vector search
  └── search_global_roots_in_tenant()
      → dense + sparse vectors, top-K across L0/L1

Step 3: Merge starting points
  ├── Optional rerank on global results (L0/L1 only)
  └── Combine root URIs + global hits → priority queue

Step 4: Recursive directory search
  ├── Priority queue (max-heap by score)
  ├── For each directory:
  │   ├── search_children_in_tenant() → vector search
  │   ├── Optional rerank
  │   ├── Score propagation: final = α·child + (1-α)·parent
  │   ├── Collect L2 files as candidates
  │   └── Push L0/L1 directories back to queue
  ├── Dedup by URI (keep highest score)
  └── Convergence: stop after 3 rounds with stable top-K

Step 5: Convert to MatchedContext
  ├── Hotness blending: (1-α_hot)·semantic + α_hot·hotness
  ├── Hotness = f(active_count, updated_at)
  └── Fetch related contexts

Step 6: Return QueryResult
```

**Key parameters**:

| Parameter | Default | Description |
|---|---|---|
| `MAX_CONVERGENCE_ROUNDS` | 3 | Stop after N rounds with unchanged top-K |
| `GLOBAL_SEARCH_TOPK` | 10 | Global retrieval candidate count |
| `DIRECTORY_DOMINANCE_RATIO` | 1.2 | Directory must exceed max child score |
| `score_propagation_alpha` | configurable | Parent→child blend weight |
| `hotness_alpha` | configurable | Semantic vs recency blend weight |

**Retriever modes**:
- `THINKING` — Full pipeline with rerank (higher latency, better quality)
- `QUICK` — Vector search only (lower latency, good-enough quality)

### 5.3 Session Manager

> Located in `openviking/session/session.py` (2629 lines)

**Session lifecycle**:

```
create → add_message* → used() → commit → (repeat)
```

**Token accounting (WM v2)**:

- `pending_tokens` — cumulative tokens outside the keep window
- `keep_recent_count` — messages retained after commit
- Sliding window: new messages push older ones into `pending_tokens`
- On commit: pending_tokens resets to 0

**Two-phase commit design**:

| Phase | Lock | Async | Crash Safety |
|---|---|---|---|
| Phase 1 (Archive) | PathLock (distributed) | Synchronous | Write-then-delete |
| Phase 2 (Extract) | None | `asyncio.create_task` | Redo log |

**Memory categories extracted** (8 types):

```python
memories_extracted = {
    "profile": 0,      # User profile facts
    "preferences": 0,  # Preferences and opinions
    "entities": 0,     # Named entities
    "events": 0,       # Time-bound events
    "cases": 0,        # Problem-solving cases
    "patterns": 0,     # Behavioral patterns
    "tools": 0,        # Tool usage patterns
    "skills": 0,       # Learned skills
}
```

### 5.4 Authentication & Authorization

> Located in `openviking/server/auth.py` (433 lines)

**Three auth modes**:

| Mode | Mechanism | Identity Source |
|---|---|---|
| `DEV` | No auth | Implicit ROOT, localhost only |
| `API_KEY` | Root key + per-user keys | `APIKeyManager.resolve(key)` |
| `TRUSTED` | Gateway trust | HTTP headers + optional root key |

**Identity resolution flow**:

```
resolve_identity(request) → ResolvedIdentity
  │
  ├── DEV: → ROOT, account=header|"default"
  │
  ├── TRUSTED:
  │   ├── Validate root key (if configured)
  │   ├── Match header vs URL account/user
  │   └── Lookup role via APIKeyManager (if available)
  │
  └── API_KEY:
      ├── Root key: override account/user/agent from headers
      ├── Admin key: override user/agent, locked to account
      └── User key: locked to account+user, override agent only

get_request_context(request, identity) → RequestContext
  ├── ROOT data APIs require explicit X-OpenViking-Account + User
  └── TRUSTED data APIs require explicit account + user
```

**RBAC decorators**:

```python
# Dependency injection style
@router.post("/admin/accounts")
async def create_account(ctx: RequestContext = Depends(require_role(Role.ROOT))):
    ...

# Decorator style (with auth mode awareness)
@router.post("/admin/accounts")
@require_auth_root
async def create_account(body, request: Request, ctx: RequestContext):
    ...
```

### 5.5 MCP Endpoint

> Located in `openviking/server/mcp_endpoint.py` (412 lines)

**9 tools** exposed via Streamable HTTP at `/mcp`:

| Tool | Input | Output |
|---|---|---|
| `search` | query, target_uri, limit, min_score | Ranked results with URI + abstract |
| `read` | uris (string or list) | Full content, batch-capable |
| `list` | uri, recursive | Directory listing |
| `store` | messages (role+content) | Creates session, commits |
| `add_resource` | path (URL only), description | Async resource processing |
| `grep` | uri, pattern(s), case_insensitive | Line-by-line regex matches |
| `glob` | pattern, uri, node_limit | Filename glob matches |
| `forget` | uri | Permanent deletion |
| `health` | — | Server health status |

**Identity propagation**: ASGI middleware → `contextvars.ContextVar` → tool handlers

```python
class _IdentityASGIMiddleware:
    async def __call__(self, scope, receive, send):
        identity = await resolve_identity(request, ...)  # Same as REST
        ctx = RequestContext(...)
        token = _mcp_ctx.set(ctx)
        try:
            return await self.app(scope, receive, send)
        finally:
            _mcp_ctx.reset(token)
```

### 5.6 Encryption Subsystem

> Located in `openviking/crypto/encryptor.py` (324 lines)

**Envelope encryption** — each file gets an independent random key:

```
Root Key (from KMS provider)
  │
  ▼ derive
Account Key
  │
  ▼ wrap
File Key (random 32 bytes per file)
  │
  ▼ AES-256-GCM encrypt
Ciphertext
```

**Binary envelope format** (`OVE1`):

```
Offset  Size  Field
0       4B    Magic: "OVE1"
4       1B    Version: 0x01
5       1B    Provider Type (0x01=Local, 0x02=Vault, 0x03=Volcengine)
6       2B    Encrypted File Key Length (big-endian)
8       2B    Key IV Length (big-endian)
10      2B    Data IV Length (big-endian)
12      var   Encrypted File Key
var     var   Key IV
var     var   Data IV (12 bytes)
var     var   AES-GCM Ciphertext (includes 16B auth tag)
```

**KMS providers**: `LocalFileProvider`, `VaultProvider`, `VolcengineKMSProvider`

### 5.7 Embedding Providers

> Located in `openviking/models/embedder/` (13 files)

| Provider | Dense | Sparse | Hybrid | Module |
|---|---|---|---|---|
| OpenAI | ✅ | ❌ | ❌ | `openai_embedders.py` |
| Volcengine | ✅ | ✅ | ✅ | `volcengine_embedders.py` |
| Gemini | ✅ | ❌ | ❌ | `gemini_embedders.py` |
| Jina | ✅ | ❌ | ❌ | `jina_embedders.py` |
| Cohere | ✅ | ❌ | ❌ | `cohere_embedders.py` |
| DashScope | ✅ | ✅ | ✅ | `dashscope_embedders.py` |
| MiniMax | ✅ | ❌ | ❌ | `minimax_embedders.py` |
| Voyage | ✅ | ❌ | ❌ | `voyage_embedders.py` |
| LiteLLM | ✅ | ❌ | ❌ | `litellm_embedders.py` |
| VikingDB | ✅ | ✅ | ✅ | `vikingdb_embedders.py` |
| Local (ONNX) | ✅ | ❌ | ❌ | `local_embedders.py` |

**Unified interface**:
```python
result: EmbedResult = await embed_compat(embedder, text, is_query=True)
# result.dense_vector: List[float]
# result.sparse_vector: Optional[Dict[str, float]]
```

### 5.8 Privacy Config Service

> Located in `openviking/privacy/service.py` (217 lines)

**Version-controlled user privacy configs** stored in `viking://user/{id}/privacy/`:

```
viking://user/{account}/{user}/privacy/
└── {category}/
    └── {target_key}/
        ├── .meta.json          # UserPrivacyConfigMeta
        ├── current.json        # Active version snapshot
        └── history/
            ├── v001.json
            ├── v002.json
            └── ...
```

Operations: `upsert`, `get_current`, `get_version`, `activate_version`, `list_categories`

---

## 6. Infrastructure & Deployment

### 6.1 Server Startup Sequence

```
openviking-server (CLI entry)
  → bootstrap.py: load_server_config()
  → bootstrap.py: detect_ollama() (optional local LLM)
  → app.py: create_app(config)
      → Register 17 routers
      → Configure middleware (CORS, timing, error mapping)
      → Register exception handlers
  → Lifespan:
      → MCP session manager start
      → OpenVikingService() + initialize()
          → Lock data directory
          → Bootstrap encryption
          → Init VikingFS + VikingDB
          → Start queue workers
          → Init directory structure
      → (Optional) Start VikingBot subprocess
  → Uvicorn serve (:1933, N workers)
```

### 6.2 Docker Build (Multi-stage)

```dockerfile
# Stage 1: Rust build (RAGFS + CLI)
FROM rust:1.x → cargo build --release

# Stage 2: Python build
FROM python:3.10 → uv pip install

# Stage 3: Runtime
FROM python:3.10-slim
COPY --from=stage1 /ragfs binaries
COPY --from=stage2 /python env
EXPOSE 1933 8020
CMD ["openviking-server"]
```

### 6.3 Configuration Schema (`ov.conf`)

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
    "workspace": "~/.openviking/data",
    "agfs": { "type": "binding", "data_dir": "..." }
  },
  "embedding": {
    "dense": { "provider": "openai", "model": "...", "api_key": "..." },
    "dimension": 1536,
    "max_concurrent": 10
  },
  "vlm": { "provider": "openai", "model": "gpt-4o-mini", "max_concurrent": 100 },
  "rerank": { "provider": "jina", "model": "...", "threshold": 0.35 },
  "retrieval": { "hotness_alpha": 0.1, "score_propagation_alpha": 0.7 },
  "encryption": { "enabled": false, "provider": "local" },
  "observability": { "telemetry_enabled": true, "metrics_enabled": true }
}
```

---

## 7. Design Decisions & Rationale

| Decision | Rationale |
|---|---|
| **Filesystem paradigm over flat vector DB** | Hierarchical organization enables directory-scoped search, natural access control, and human-readable context browsing |
| **Three-tier L0/L1/L2 loading** | Reduces token consumption by 80-90% — agents only load detail when needed |
| **Score propagation in recursive search** | Parent directory relevance boosts child discovery — solves the "needle in haystack" problem |
| **Convergence detection (3 rounds)** | Prevents infinite traversal while ensuring result stability |
| **Two-phase session commit** | Phase 1 (lock-protected) ensures data safety; Phase 2 (background) avoids blocking API |
| **Redo log for Phase 2** | Crash recovery — if server dies during memory extraction, redo log replays on restart |
| **Envelope encryption per file** | Compromising one file key doesn't expose others; KMS rotation only re-wraps, not re-encrypts |
| **RAGFS in Rust** | Performance-critical filesystem operations in Rust; Python for business logic |
| **Deferred initialization** | Heavy infra setup (VikingDB, embedder, VLM) happens after server starts accepting health checks |
| **MCP identity via ASGI middleware** | Reuses the same `resolve_identity()` as REST API — zero auth logic duplication |

---

## 8. Known Limitations & Gotchas

> [!WARNING]
> RAGFS is an embedded filesystem — it does **not** support distributed multi-node writes. Multi-worker mode uses shared local storage with file-based locking.

> [!NOTE]
> Resource ingestion is **asynchronous**. After `add_resource()`, content is processed in the background via QueueManager. Check task status via `/api/v1/tasks/{id}`.

> [!CAUTION]
> Session commit Phase 2 (memory extraction) runs as an `asyncio.create_task`. If the server shuts down during Phase 2, the redo log ensures recovery on next startup.

> [!TIP]
> In `DEV` auth mode, the server binds to localhost only. To expose externally, switch to `api_key` or `trusted` mode and configure `root_api_key`.

---

## 9. Extension Points

### Adding a New Embedding Provider

1. Create `openviking/models/embedder/{provider}_embedders.py`
2. Implement `BaseEmbedder` interface with `embed()` and `embed_batch()`
3. Register in `openviking/models/embedder/__init__.py`
4. Add config section in `EmbeddingConfig`

### Adding a New API Router

1. Create `openviking/server/routers/{feature}.py`
2. Define `FastAPI APIRouter` with prefix
3. Import and register in `openviking/server/routers/__init__.py`
4. Add to router list in `app.py:create_app()`

### Adding a New MCP Tool

1. Define tool function in `openviking/server/mcp_endpoint.py`
2. Decorate with `@mcp.tool()`
3. Use `_get_ctx()` for identity, `get_service()` for business logic
4. Update tool count in `mcp_lifespan()` log message

---

*Document generated from source analysis of `github.com/volcengine/OpenViking` (2026-05-07).*
