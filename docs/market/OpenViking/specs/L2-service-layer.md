# L2 — Service Layer

> **Layer**: 2 (Service / Business Logic)  
> **Responsibility**: Orchestrate business operations via sub-services  
> **Dependencies**: L3 (Core Domain), L4 (Retrieval & Processing)

---

## 1. Tổng Quan

Layer 2 chứa toàn bộ business logic cấp cao của OpenViking. `OpenVikingService` là composition root quản lý lifecycle của tất cả infrastructure và sub-services.

**Path:** `openviking/service/`

| Service | File | Size | Chức năng |
|---------|------|------|-----------|
| **OpenVikingService** | `core.py` | 16KB | Composition root, lifecycle |
| **FSService** | `fs_service.py` | 10KB | File CRUD qua VikingFS |
| **SearchService** | `search_service.py` | 4KB | Semantic search (find/search) |
| **SessionService** | `session_service.py` | 10KB | Session lifecycle, commit |
| **ResourceService** | `resource_service.py` | 21KB | Resource ingestion |
| **RelationService** | `relation_service.py` | 3KB | Context relation CRUD |
| **PackService** | `pack_service.py` | 2KB | Context packing/export |
| **DebugService** | `debug_service.py` | 7KB | IO recording, diagnostics |
| **TaskTracker** | `task_tracker.py` | 13KB | Background task tracking |

---

## 2. OpenVikingService — Composition Root

### 2.1 Infrastructure Components

```python
class OpenVikingService:
    # Config
    _config: OpenVikingConfig        # ov.conf loaded
    _user: UserIdentifier            # Default user identity

    # Infrastructure (initialized in stages)
    _agfs_client: AGFSClient         # RAGFS filesystem binding
    _queue_manager: QueueManager     # Background task queue
    _vikingdb_manager: VikingDBManager  # Vector collection management
    _viking_fs: VikingFS             # Virtual filesystem (singleton)
    _embedder: BaseEmbedder          # Embedding model
    _resource_processor: ResourceProcessor  # Resource pipeline
    _skill_processor: SkillProcessor  # Skill loading
    _session_compressor: SessionCompressor  # VLM-based WM compression
    _lock_manager: LockManager       # Distributed path locks
    _encryptor: FileEncryptor        # AES-256-GCM encryption
    _watch_scheduler: WatchScheduler  # Resource auto-refresh
    _privacy_config_service: UserPrivacyConfigService
    _directory_initializer: DirectoryInitializer

    # Sub-services (created at __init__)
    _fs_service: FSService
    _search_service: SearchService
    _session_service: SessionService
    _resource_service: ResourceService
    _relation_service: RelationService
    _pack_service: PackService
    _debug_service: DebugService
```

### 2.2 Initialization Sequence

```
__init__(path, user):
  1. Load config from ov.conf
  2. Init RAGFS client (Rust binding)
  3. Init QueueManager
  4. Init embedder (from config provider)
  5. Create 7 sub-service instances

initialize():
  6. Acquire data directory lock (process guard)
  7. Bootstrap encryption (key provider setup)
  8. Init VikingFS singleton (RAGFS + embedder + encryptor)
  9. Create VikingDB collection (vector schema)
  10. Start QueueManager workers
  11. Init ResourceProcessor + SkillProcessor
  12. Create SessionCompressor (VLM-backed)
  13. Init DirectoryInitializer (root dirs, built-in skills)
  14. Start WatchScheduler (resource auto-refresh)
  15. Wire VikingFS into all sub-services
  16. Set _initialized = True
```

### 2.3 Deferred Initialization Pattern

Sub-services are created at `__init__()` but receive their VikingFS dependency later via `set_viking_fs()`:

```python
class SearchService:
    def __init__(self, viking_fs=None):
        self._viking_fs = viking_fs

    def set_viking_fs(self, viking_fs: VikingFS):
        self._viking_fs = viking_fs

    def _ensure_initialized(self) -> VikingFS:
        if not self._viking_fs:
            raise NotInitializedError("VikingFS")
        return self._viking_fs
```

This allows the server to start accepting health checks before heavy initialization completes.

---

## 3. FSService — Filesystem Operations

**File:** `fs_service.py` (10KB)

Delegates CRUD operations to VikingFS with request context.

| Method | Params | Delegate | Chức năng |
|--------|--------|----------|-----------|
| `read` | uri, ctx | `viking_fs.read()` | Read file content |
| `write` | uri, data, ctx | `viking_fs.write()` | Write file content |
| `ls` | uri, ctx | `viking_fs.ls()` | List directory |
| `tree` | uri, format, ctx | `viking_fs.tree()` | Recursive listing |
| `mkdir` | uri, ctx | `viking_fs.mkdir()` | Create directory |
| `rm` | uri, recursive, ctx | `viking_fs.rm()` | Delete (+ vector cleanup) |
| `mv` | old_uri, new_uri, ctx | `viking_fs.mv()` | Move (cp + rm + vector update) |
| `cp` | src, dst, ctx | `viking_fs.cp()` | Copy |
| `stat` | uri, ctx | `viking_fs.stat()` | File/dir metadata |

---

## 4. SearchService — Semantic Search

**File:** `search_service.py` (4KB)

| Method | Params | Chức năng |
|--------|--------|-----------|
| `search` | query, ctx, target_uri, session, limit, score_threshold, filter | Session-aware semantic search |
| `find` | query, ctx, target_uri, limit, score_threshold, filter | Semantic search without session |

**search vs find:** `search()` accepts a Session object for context-aware retrieval (session history influences ranking). `find()` is stateless.

```python
async def search(self, query, ctx, target_uri="", session=None, limit=10, ...):
    session_info = None
    if session:
        session_info = await session.get_context_for_search(query)
    return await viking_fs.search(query, ctx, session_info=session_info, ...)
```

---

## 5. SessionService — Session Lifecycle

**File:** `session_service.py` (10KB)

| Method | Chức năng |
|--------|-----------|
| `create_session` | Create new session directory + metadata |
| `get_session` | Load session by ID |
| `list_sessions` | List all sessions for user |
| `add_messages` | Append messages to session |
| `record_used` | Record URIs used during session |
| `commit` | Trigger 2-phase commit (archive + memory extract) |
| `delete_session` | Delete session and cleanup |
| `get_session_info` | Session metadata + stats |

**Commit delegates to L4 Session class** for the actual 2-phase commit logic.

---

## 6. ResourceService — Resource Ingestion

**File:** `resource_service.py` (21KB) — Largest sub-service.

| Method | Chức năng |
|--------|-----------|
| `add_resource` | Full ingestion pipeline (detect → clone → parse → embed) |
| `get_resource` | Resource metadata by name |
| `list_resources` | List all ingested resources |
| `delete_resource` | Remove resource + vector index |
| `refresh_resource` | Re-process existing resource |
| `get_resource_status` | Processing status (queued/processing/done) |

**Ingestion pipeline** (delegates to L4 ResourceProcessor):

```
add_resource(url_or_path):
  1. Detect source type (git repo / HTTP URL / local path / file)
  2. Clone/download to viking://temp/
  3. Parse files (tree-sitter for code, VLM for images/PDFs)
  4. Build directory tree under viking://resources/{name}/
  5. Queue background processing:
     a. Chunk content
     b. Embed (dense + sparse vectors)
     c. Generate L0 (.abstract.md) + L1 (.overview.md) via VLM
     d. Upsert to vector index
     e. Move from temp → final location
```

---

## 7. RelationService — Context Relations

**File:** `relation_service.py` (3KB)

| Method | Chức năng |
|--------|-----------|
| `get_relations` | Get relations for a URI |
| `add_relation` | Link two URIs with reason |
| `remove_relation` | Unlink relation by ID |

Relations are stored in `.relations.json` files within the VikingFS.

---

## 8. PackService — Context Packing

**File:** `pack_service.py` (2KB)

| Method | Chức năng |
|--------|-----------|
| `pack` | Export context subtree as portable package |
| `unpack` | Import context package |

---

## 9. DebugService — Diagnostics

**File:** `debug_service.py` (7KB)

| Method | Chức năng |
|--------|-----------|
| `start_recording` | Enable IO recorder (wrap VikingFS) |
| `stop_recording` | Disable IO recorder |
| `get_recording` | Retrieve recorded operations |
| `replay_search` | Replay a recorded search for comparison |
| `get_diagnostics` | System diagnostics snapshot |

---

## 10. TaskTracker — Background Task Management

**File:** `task_tracker.py` (13KB)

| Method | Chức năng |
|--------|-----------|
| `create_task` | Register new background task |
| `update_status` | Update task progress |
| `get_task` | Get task by ID |
| `list_tasks` | List tasks with filters |
| `cleanup_old_tasks` | Remove completed tasks |

Task states: `QUEUED` → `PROCESSING` → `DONE` | `FAILED`

---

## 11. Key Design Patterns

| Pattern | Implementation |
|---------|----------------|
| **Composition** | OpenVikingService composes 7 sub-services + infrastructure |
| **Deferred Init** | Sub-services created early, wired with VikingFS later |
| **Delegation** | Services delegate to L4 processing and L5 storage |
| **Context Passing** | `RequestContext` flows from L1 through all service methods |
| **Singleton** | VikingFS is a process-wide singleton |
