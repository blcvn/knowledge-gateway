# OpenViking: Architecture Design Document

> **Repository**: `github.com/volcengine/OpenViking`  
> **Generated**: 2026-05-07  
> **Status**: Active (Alpha 0.1.x)

---

## 1. Executive Summary

OpenViking is an **Agent-native Context Database** that uses a filesystem paradigm (`viking://` URI protocol) to unify context management for AI Agents. It replaces flat vector storage with a hierarchical virtual filesystem where every piece of context — memories, resources, skills — is addressable via a URI and organized in a tiered abstraction model (L0/L1/L2).

---

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       Client Layer                              │
│   ov CLI (Rust)  │  Python SDK  │  MCP Clients  │  REST API    │
└────────┬─────────┴──────┬───────┴───────┬───────┴──────┬───────┘
         │                │               │              │
┌────────▼────────────────▼───────────────▼──────────────▼───────┐
│                    FastAPI HTTP Server (:1933)                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │ 17 API   │ │ MCP /mcp │ │Auth/RBAC │ │ WebDAV   │          │
│  │ Routers  │ │(9 tools) │ │Middleware│ │ /webdav  │          │
│  └────┬─────┘ └────┬─────┘ └──────────┘ └──────────┘          │
├───────┼─────────────┼──────────────────────────────────────────┤
│       └──────┬──────┘                                          │
│       ┌──────▼──────┐                                          │
│       │ Service Layer│ ← OpenVikingService (core.py)            │
│       │  ┌────────────────────────────────────────────┐        │
│       │  │ FSService  │ SearchService │ SessionService│        │
│       │  │ ResourceSvc│ RelationSvc   │ DebugService  │        │
│       │  │ PackService│ TaskTracker   │ PrivacySvc    │        │
│       │  └────────────────────────────────────────────┘        │
│       └──────┬──────┘                                          │
├──────────────┼─────────────────────────────────────────────────┤
│       ┌──────▼──────┐                                          │
│       │  Core Domain │                                         │
│       │  Context │ Namespace │ ContextType │ ContextLevel      │
│       │  URI Resolution │ Accessibility │ DirectoryInitializer │
│       └──────┬──────┘                                          │
├──────────────┼─────────────────────────────────────────────────┤
│       ┌──────▼──────┐                                          │
│       │Infrastructure│                                         │
│       │  ┌─────────────────────────────────────────────────┐   │
│       │  │ VikingFS  │ VikingDB  │ RAGFS   │ Embedder     │   │
│       │  │ QueueMgr  │ VectorIdx │ LockMgr │ VLM          │   │
│       │  │ Encryptor │ Rerank    │ Parsers │ Telemetry    │   │
│       │  └─────────────────────────────────────────────────┘   │
│       └─────────────┘                                          │
└────────────────────────────────────────────────────────────────┘
         │                     │                    │
    ┌────▼─────┐        ┌──────▼──────┐      ┌─────▼──────┐
    │ RAGFS    │        │ VikingBot   │      │ External   │
    │ (Rust)   │        │ (:18790)    │      │ AI APIs    │
    │ Local FS │        │ Multi-chan   │      │ (Embed/VLM)│
    └──────────┘        └─────────────┘      └────────────┘
```

---

## 3. Layer Architecture

### 3.1 Layer 1 — Presentation (HTTP Server)

| Component         | Role                                           |
|-------------------|-------------------------------------------------|
| FastAPI App       | ASGI application factory with lifespan management|
| 17 API Routers    | filesystem, content, search, sessions, resources, relations, admin, observer, privacy, tasks, system, debug, bot, pack, maintenance, stats, metrics |
| MCP Endpoint      | Streamable HTTP at `/mcp` — 9 tools for AI IDEs |
| WebDAV Endpoint   | Standard WebDAV for file manager access          |
| Auth Middleware    | 3 modes: DEV / API_KEY / TRUSTED                |
| Observability MW  | Request timing, CORS, error mapping, OTel traces|

**Router count & registration** (from `app.py`):

```python
# 17 routers registered on the FastAPI app
routers = [
    filesystem_router,  content_router,    search_router,
    sessions_router,    resources_router,   relations_router,
    admin_router,       observer_router,    privacy_configs_router,
    tasks_router,       system_router,      debug_router,
    bot_router,         pack_router,        maintenance_router,
    stats_router,       metrics_router,
]
# + MCP (mounted as ASGI sub-app at /mcp)
# + WebDAV (mounted at /webdav)
```

### 3.2 Layer 2 — Service Layer

**Entry point**: `OpenVikingService` (`openviking/service/core.py`)

The service layer composes 7 sub-services and manages the full infrastructure lifecycle:

| Sub-Service        | Responsibility                                 |
|--------------------|-------------------------------------------------|
| `FSService`        | File CRUD operations via VikingFS                |
| `SearchService`    | Semantic search, find, grep, glob                |
| `SessionService`   | Session lifecycle, commit, memory extraction     |
| `ResourceService`  | Resource ingestion pipeline (git/URL/local)      |
| `RelationService`  | Context relation management                      |
| `PackService`      | Context packing for export                       |
| `DebugService`     | IO recording, replay, diagnostics                |

**Initialization sequence** (`OpenVikingService.initialize()`):

```
1. Acquire data directory lock (single-instance guard)
2. Bootstrap encryption (key provider setup)
3. Initialize VikingFS singleton (RAGFS + embedder + encryptor)
4. Create VikingDB vector collection (if needed)
5. Start QueueManager workers
6. Initialize resource/skill processors
7. Create session compressor (VLM-backed)
8. Initialize directory structure (root dirs, skill loading)
9. Start watch scheduler (resource auto-refresh)
10. Set initialized flag
```

### 3.3 Layer 3 — Core Domain

| Component             | Module                          | Purpose                              |
|-----------------------|---------------------------------|--------------------------------------|
| `Context`             | `core/context.py`              | Unified record: URI, type, level, owner, abstract |
| `Namespace`           | `core/namespace.py`            | URI resolution, ownership, canonical roots |
| `ContextType`         | `core/context.py`              | Enum: MEMORY, RESOURCE, SKILL, SESSION |
| `ContextLevel`        | `core/context.py`              | 0=Abstract, 1=Overview, 2=Detail     |
| `DirectoryInitializer`| `core/directories.py`          | Bootstrap root dirs, load built-in skills |
| `RequestContext`      | `server/identity.py`           | User identity + role for request scope |

### 3.4 Layer 4 — Infrastructure

| Component           | Module                         | Technology            |
|---------------------|--------------------------------|-----------------------|
| **VikingFS**        | `storage/viking_fs.py`        | Virtual FS over RAGFS |
| **VikingDBManager** | `storage/vikingdb_manager.py` | Vector index management|
| **RAGFS**           | Rust crate `ragfs`            | On-disk FS engine     |
| **Embedder**        | `models/embedder/`            | 12+ embedding providers|
| **VLM**             | `models/vlm/`                 | Vision-Language Models |
| **Rerank**          | `models/rerank.py`            | Relevance reranking   |
| **QueueManager**    | `storage/queuefs/`            | Async task queue      |
| **LockManager**     | `storage/transaction/`        | Distributed file locks|
| **FileEncryptor**   | `crypto/encryptor.py`         | AES-256-GCM envelope |
| **Telemetry**       | `telemetry/`                  | OTel traces + metrics |

---

## 4. Viking URI Namespace

### 4.1 URI Schema

```
viking://{space}/{account_id}/{user_id}/{agent_id}/{...path}
```

### 4.2 Namespace Topology

```
viking://
├── resources/                          # Shared project resources
│   └── {resource_name}/
│       ├── .abstract.md               # L0 summary
│       ├── .overview.md               # L1 overview
│       └── {file_tree}                # L2 content
├── user/{account_id}/{user_id}/        # User space
│   ├── memories/                      # User long-term memories
│   │   └── {category}/
│   │       ├── .abstract.md
│   │       └── {memory_item}
│   └── privacy/                       # Privacy configs
│       └── {category}/{target}/
├── agent/{account_id}/{user_id}/{agent_id}/  # Agent space
│   ├── skills/                        # Agent skills/tools
│   ├── memories/                      # Agent task memories
│   └── instructions/                  # Agent instructions
├── session/{session_id}/               # Session space
│   ├── messages.jsonl                 # Live messages
│   ├── .meta.json                     # Session metadata
│   └── history/                       # Archived commits
│       └── archive_NNN/
│           ├── messages.jsonl
│           └── .overview.md           # Working Memory v2
└── temp/                               # Temporary staging area
```

### 4.3 Access Control Model

```
ROOT ── can access everything
  │
  ├── ADMIN ── account-scoped (own account + managed users)
  │     │
  │     └── USER ── user-scoped
  │           │
  │           ├── viking://user/{own_account}/{own_user}/*    ✅
  │           ├── viking://agent/{own_account}/{own_user}/*   ✅
  │           ├── viking://resources/*                        ✅ (read)
  │           ├── viking://user/{other}/*                     ❌
  │           └── viking://agent/{other_account}/*            ❌
```

---

## 5. Three-Tier Context Model

### 5.1 Tiered Loading Strategy

```
                    Token Cost
    ┌────────────────────────────────────────┐
    │  L0: .abstract.md                      │  ~100 tokens
    │  ├── "What is this?"                   │  Quick relevance check
    ├────────────────────────────────────────┤
    │  L1: .overview.md                      │  ~2K tokens
    │  ├── "Key points + usage guidance"     │  Planning & reasoning
    ├────────────────────────────────────────┤
    │  L2: Raw content                       │  Full document
    │  └── "Complete file content"           │  Deep reading
    └────────────────────────────────────────┘
```

### 5.2 File-Level Mapping

| Level | File              | Generated By | Token Budget |
|-------|-------------------|--------------|--------------|
| L0    | `.abstract.md`    | VLM          | ~100         |
| L1    | `.overview.md`    | VLM          | ~2,000       |
| L2    | Original file     | User/Ingestion| Full        |

### 5.3 Level-aware Read Operations

```python
# VikingFS methods
await viking_fs.abstract(uri)         # → reads {uri}/.abstract.md
await viking_fs.overview(uri)         # → reads {uri}/.overview.md
await viking_fs.read(uri)             # → reads raw content (L2)
await viking_fs.read_batch(uris, level="l0")  # → batch L0 reads
```

---

## 6. Data Flow Architectures

### 6.1 Resource Ingestion Flow

```
User: ov add-resource https://github.com/org/repo
  │
  ▼
┌─────────────────────────────┐
│ ResourceService.add_resource│
│  1. Detect source type      │
│  2. Clone/download to temp  │
│  3. Parse files (tree-sitter│
│     for code, VLM for docs) │
│  4. Build directory tree    │
└─────────────┬───────────────┘
              ▼
┌─────────────────────────────┐
│ QueueManager (background)   │
│  5. Chunk + embed content   │
│     (dense + sparse vectors)│
│  6. Generate L0/L1 summaries│
│     via VLM                 │
│  7. Upsert to VikingDB      │
│  8. Move from temp → final  │
│     viking://resources/     │
└─────────────────────────────┘
```

### 6.2 Hierarchical Retrieval Flow

```
Agent: "How does authentication work?"
  │
  ▼
┌─────────────────────────────┐
│ SearchService.find()        │
│  1. Parse query → TypedQuery│
│     (query, context_type,   │
│      target_directories)    │
└─────────────┬───────────────┘
              ▼
┌─────────────────────────────┐
│ HierarchicalRetriever       │
│                             │
│  2. Embed query             │
│     → dense_vector          │
│     → sparse_vector         │
│                             │
│  3. Global vector search    │
│     → top-K across L0/L1    │
│                             │
│  4. Merge starting points   │
│     (root URIs + global     │
│      hits excluding L2)     │
│                             │
│  5. Recursive directory     │
│     search (priority queue) │
│     ┌─────────────────┐     │
│     │ Pop top-score dir│     │
│     │ Search children  │     │
│     │ Rerank (optional)│     │
│     │ Score propagation│     │
│     │ Push sub-dirs    │     │
│     │ Convergence check│     │
│     └─────────────────┘     │
│                             │
│  6. Hotness blending        │
│     final = (1-α)·semantic  │
│             + α·hotness     │
└─────────────┬───────────────┘
              ▼
        QueryResult
        (matched_contexts[],
         searched_directories[])
```

### 6.3 Session Commit Flow (Two-Phase)

```
Plugin: POST /sessions/{id}/commit
  │
  ▼
┌─────────────────────────────────────┐
│ Phase 1: Archive (Lock-Protected)   │
│                                     │
│  1. Acquire PathLock on session dir │
│  2. Split messages:                 │
│     [archive | retained_tail]       │
│  3. Write retained tail →           │
│     messages.jsonl                  │
│  4. Write archive →                 │
│     history/archive_NNN/messages.jsonl│
│  5. Update .meta.json               │
│  6. Release lock                    │
└─────────────┬───────────────────────┘
              ▼
┌─────────────────────────────────────┐
│ Phase 2: Memory Extract (Background)│
│                                     │
│  1. Write redo-log (crash safety)   │
│  2. Wait for previous archive done  │
│  3. Generate Working Memory v2      │
│     (7-section structured doc)      │
│  4. Extract memories via VLM        │
│     (profile, preferences, entities,│
│      events, cases, patterns,       │
│      tools, skills)                 │
│  5. Upsert memories →              │
│     viking://user/{id}/memories/    │
│  6. Update active_count on used URIs│
│  7. Mark redo-log committed         │
└─────────────────────────────────────┘
```

---

## 7. Security Architecture

### 7.1 Authentication Flow

```
Request → Extract API Key
  │        (X-Api-Key or Authorization: Bearer)
  ▼
┌─────────────────────────┐
│ AuthMode Dispatch        │
│                          │
│ DEV mode:                │
│   → No auth, ROOT role   │
│   → Localhost only        │
│                          │
│ API_KEY mode:            │
│   → Root key check       │
│   → APIKeyManager lookup │
│   → Role assignment      │
│                          │
│ TRUSTED mode:            │
│   → Trust gateway headers│
│   → Optional root key    │
└─────────┬───────────────┘
          ▼
┌─────────────────────────┐
│ RequestContext           │
│  user: UserIdentifier   │
│    (account, user, agent)│
│  role: ROOT|ADMIN|USER  │
│  namespace_policy: ...   │
└─────────┬───────────────┘
          ▼
┌─────────────────────────┐
│ Namespace Access Check   │
│  URI ↔ Owner matching    │
│  Role permission check   │
└─────────────────────────┘
```

### 7.2 Encryption Architecture (Envelope Encryption)

```
              Root Key Provider
              (Local / Vault / Volcengine KMS)
                    │
                    ▼
              Account Key (derived)
                    │
                    ▼
         ┌──────────────────────┐
         │  Per-File Encryption │
         │                      │
         │  File Key = random   │
         │  32 bytes            │
         │                      │
         │  Encrypted File Key  │
         │  = AES-GCM(Account   │
         │    Key, File Key)    │
         │                      │
         │  Ciphertext          │
         │  = AES-GCM(File Key, │
         │    plaintext)        │
         └──────────────────────┘
                    │
                    ▼
         ┌──────────────────────────────────────┐
         │  OVE1 Envelope Format (binary)       │
         │  Magic(4B) | Ver(1B) | Prov(1B)     │
         │  EFK_Len(2B) | KIV_Len(2B)         │
         │  DIV_Len(2B) | EncryptedFileKey     │
         │  KeyIV | DataIV | Ciphertext        │
         └──────────────────────────────────────┘
```

---

## 8. Concurrency & Distributed Safety

### 8.1 Lock Hierarchy

| Lock Type        | Scope              | Use Case                       |
|------------------|--------------------|--------------------------------|
| Data Dir Lock    | Process-level      | Single-instance guard          |
| PathLock (point) | Single file/dir    | Session commit Phase 1         |
| PathLock (subtree)| Directory tree    | Recursive delete (rm -r)       |
| PathLock (mv)    | Source + dest      | Move operations                |
| Redo Log         | Archive-level      | Phase 2 crash recovery         |

### 8.2 Convergence Detection

The hierarchical retriever uses convergence detection to avoid unnecessary traversal:

```
Round N: topK URIs = {A, B, C, D, E}
Round N+1: topK URIs = {A, B, C, D, E}  → convergence_rounds += 1
Round N+2: topK URIs = {A, B, C, D, E}  → convergence_rounds += 1
Round N+3: convergence_rounds >= MAX_CONVERGENCE_ROUNDS(3) → STOP
```

---

## 9. Deployment Architecture

### 9.1 Standalone Mode

```
┌─────────────────────────┐
│   openviking-server     │
│   (:1933)               │
│                         │
│   FastAPI + Uvicorn     │
│   + RAGFS (embedded)    │
│   + VikingDB (embedded) │
│   + QueueManager        │
└─────────────────────────┘
         │
    ┌────▼────┐
    │ Local   │
    │ Disk    │
    │ Storage │
    └─────────┘
```

### 9.2 With VikingBot

```
┌─────────────────────────┐    ┌─────────────────────────┐
│   openviking-server     │    │   VikingBot             │
│   (:1933)               │◄──►│   (:18790)              │
│                         │    │                         │
│   API + MCP + WebDAV    │    │   Agent Framework       │
│                         │    │   Multi-channel Bot     │
│                         │    │   Sandbox + FUSE        │
└─────────────────────────┘    └─────────────────────────┘
```

### 9.3 Kubernetes (Helm)

```
┌─────────────────────────────────────────┐
│  Kubernetes Cluster                      │
│                                          │
│  ┌──────────────────┐  ┌──────────────┐ │
│  │ OpenViking Pod   │  │ VikingBot Pod│ │
│  │ (N replicas)     │  │ (optional)   │ │
│  └────────┬─────────┘  └──────────────┘ │
│           │                              │
│  ┌────────▼─────────┐                   │
│  │ PersistentVolume  │                   │
│  │ (shared storage)  │                   │
│  └──────────────────┘                   │
└─────────────────────────────────────────┘
```

---

## 10. Cross-Cutting Concerns

### 10.1 Observability Stack

| Signal     | Technology        | Export                         |
|------------|-------------------|--------------------------------|
| Traces     | OpenTelemetry     | OTLP endpoint (configurable)  |
| Logs       | Structured JSON   | stdout + OTel log handler      |
| Metrics    | Prometheus        | `/metrics` endpoint            |
| Stats      | RetrievalStats    | `/api/v1/observer/*` endpoint  |

### 10.2 Error Handling Strategy

```
Application Exception (OpenVikingError)
  │
  ├── InvalidArgumentError    → 400
  ├── UnauthenticatedError    → 401
  ├── PermissionDeniedError   → 403
  ├── NotFoundError           → 404
  ├── AlreadyExistsError      → 409
  ├── FailedPreconditionError → 412
  ├── ResourceBusyError       → 423
  ├── NotInitializedError     → 503
  └── InternalError           → 500
```

All errors return structured JSON:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Resource not found: viking://resources/example",
    "details": {}
  }
}
```

---

*Document generated from source analysis of `github.com/volcengine/OpenViking` (2026-05-07).*
