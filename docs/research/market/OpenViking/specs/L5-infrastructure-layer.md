# L5 — Infrastructure Layer

> **Layer**: 5 (Infrastructure)  
> **Responsibility**: Storage, AI model adapters, encryption, observability  
> **Dependencies**: L6 (External Services)

---

## 1. Tổng Quan

Layer 5 chứa toàn bộ infrastructure adapters — từ virtual filesystem, vector index, encryption, đến AI model integrations. Đây là tầng duy nhất trực tiếp tương tác với external services (L6).

```
openviking/
├── storage/                          # Storage subsystem
│   ├── viking_fs.py         (82KB)   # Virtual filesystem (2199 lines)
│   ├── vikingdb_manager.py  (18KB)   # Vector collection management
│   ├── viking_vector_index_backend.py (39KB)  # Vector search
│   ├── collection_schemas.py (29KB)  # Vector schemas
│   ├── content_write.py     (22KB)   # Content write pipeline
│   ├── local_fs.py          (13KB)   # Local FS helpers
│   ├── queuefs/                      # Async task queue
│   ├── transaction/                  # Lock manager + redo log
│   ├── vectordb/                     # Vector DB core
│   ├── vectordb_adapters/            # VDB adapter implementations
│   └── observers/                    # IO recording
├── models/                           # AI model adapters
│   ├── embedder/            (13 files) # 11 embedding providers
│   ├── vlm/                           # Vision-Language Models
│   └── rerank/                        # Reranking providers
├── crypto/                           # Encryption subsystem
│   ├── encryptor.py         (324 lines)
│   ├── config.py
│   └── providers.py                  # KMS adapters
├── telemetry/                        # OpenTelemetry
├── metrics/                          # Prometheus
└── pyagfs/                           # RAGFS Python bindings
```

---

## 2. VikingFS — Virtual Filesystem

**File:** `storage/viking_fs.py` (82KB, 2199 lines) — Largest module in the system.

### 2.1 Architecture

```
VikingFS (Python)
    │
    ├── URI normalization + access control
    ├── Transparent encryption (encrypt on write, decrypt on read)
    ├── Semantic search (→ HierarchicalRetriever)
    ├── Relation management (.relations.json)
    │
    └── Delegates to:
        RAGFS Client (Rust binding)
            │
            └── Local filesystem operations
```

**Singleton:** `init_viking_fs()` → `get_viking_fs()`

### 2.2 Operation Categories

| Category | Methods | Delegate |
|----------|---------|----------|
| **Basic CRUD** | `read`, `write`, `mkdir`, `rm`, `mv`, `stat`, `exists` | RAGFS + encrypt |
| **Directory** | `ls`, `tree` (original/agent) | RAGFS |
| **Tiered Read** | `abstract`, `overview`, `read_batch` | RAGFS + L0/L1 files |
| **Pattern** | `grep`, `glob` | Python regex + RAGFS |
| **Semantic** | `find`, `search` | HierarchicalRetriever |
| **Relations** | `get_relations`, `add_relation`, `remove_relation` | `.relations.json` |
| **Utility** | `move_file`, `_collect_uris`, `_ensure_parent_dirs` | Internal |

### 2.3 Transparent Encryption

```python
async def read(self, uri, offset=0, size=-1, ctx=None):
    raw = self.agfs.read(path, 0, -1)   # Read full file
    raw = await self._decrypt_content(raw, ctx)  # Auto-decrypt
    if offset > 0 or size != -1:
        raw = raw[offset:offset+size]    # Slice on plaintext
    return raw

async def write(self, uri, data, ctx=None):
    data = await self._encrypt_content(data, ctx)  # Auto-encrypt
    return self.agfs.write(path, data)
```

### 2.4 URI Security

Rejects unsafe path components:
- `..` (traversal)
- `\` (Windows separator)
- Drive letters (`C:`)

### 2.5 Lock-Protected Operations

`rm` and `mv` acquire PathLocks before modifying vector index + filesystem:

```python
async def rm(self, uri, recursive=False, ctx=None):
    async with LockContext(lock_manager, [path], lock_mode="subtree"):
        uris = await self._collect_uris(path, recursive)
        await self._delete_from_vector_store(uris)
        self.agfs.rm(path, recursive=recursive)

async def mv(self, old_uri, new_uri, ctx=None):
    async with LockContext(lock_manager, [old_path], lock_mode="mv",
                           mv_dst_parent_path=dst_parent):
        agfs_cp(self.agfs, old_path, new_path)
        await self._update_vector_store_uris(uris, old_uri, new_uri)
        self.agfs.rm(old_path, recursive=True)
```

### 2.6 Grep Implementation

VikingFS-level grep supports encrypted files:

```python
async def grep(self, uri, pattern, case_insensitive=False, ...):
    file_uris = await self._collect_grep_files(uri, level_limit=5)
    results = await self._grep_files_parallel(
        file_uris, compiled_pattern, node_limit,
        concurrency=32)  # Batch parallel grep
    return {"matches": results, "count": len(results)}
```

---

## 3. VikingDBManager — Vector Collections

**File:** `storage/vikingdb_manager.py` (18KB)

| Method | Chức năng |
|--------|-----------|
| `collection_exists` | Check if tenant collection exists |
| `create_collection` | Create vector collection with schema |
| `search_global_roots_in_tenant` | Global search across L0/L1 |
| `search_children_in_tenant` | Search children under parent URI |
| `upsert_context` | Insert/update context with vectors |
| `delete_context` | Remove context from index |
| `update_active_count` | Increment usage counter |

### 3.1 Vector Schema

```python
CONTEXT_COLLECTION_SCHEMA = {
    "uri": str,           # Primary key
    "parent_uri": str,    # Parent directory
    "context_type": int,  # ContextType enum
    "level": int,         # 0/1/2
    "owner_account_id": str,
    "owner_user_id": str,
    "abstract": str,      # L0 text
    "active_count": int,  # Hotness
    "dense_vector": List[float],   # 768-3072 dims
    "sparse_vector": Dict[str, float],  # BM25/SPLADE
}
```

---

## 4. Embedding Providers

**Path:** `models/embedder/` (13 files)

### 4.1 Base Interface

```python
class BaseEmbedder(ABC):
    @property
    def dimension(self) -> int: ...
    @property
    def is_sparse(self) -> bool: ...

    async def embed(self, text: str, is_query: bool = False) -> EmbedResult: ...
    async def embed_batch(self, texts: List[str]) -> List[EmbedResult]: ...

@dataclass
class EmbedResult:
    dense_vector: List[float]
    sparse_vector: Optional[Dict[str, float]]
```

### 4.2 Provider Matrix

| Provider | Dense | Sparse | Hybrid | File Size |
|----------|-------|--------|--------|-----------|
| Volcengine | ✅ | ✅ | ✅ | 24KB |
| VikingDB | ✅ | ✅ | ✅ | 23KB |
| DashScope | ✅ | ✅ | ✅ | 17KB |
| OpenAI | ✅ | ❌ | ❌ | 18KB |
| Gemini | ✅ | ❌ | ❌ | 18KB |
| Jina | ✅ | ❌ | ❌ | 12KB |
| Local (ONNX) | ✅ | ❌ | ❌ | 12KB |
| MiniMax | ✅ | ❌ | ❌ | 11KB |
| Cohere | ✅ | ❌ | ❌ | 10KB |
| LiteLLM | ✅ | ❌ | ❌ | 10KB |
| Voyage | ✅ | ❌ | ❌ | 8KB |

### 4.3 Factory

```python
# In config
embedding_config.get_embedder() → BaseEmbedder
# Routes to correct provider based on config.embedding.dense.provider
```

---

## 5. Encryption Subsystem

**Path:** `crypto/`

### 5.1 Envelope Encryption

```
Root Key (from KMS)
  └── Account Key (derived per account)
       └── File Key (random 32B per file)
            └── AES-256-GCM encrypt content
```

### 5.2 Binary Format (OVE1)

| Offset | Size | Field |
|--------|------|-------|
| 0 | 4B | Magic: `"OVE1"` |
| 4 | 1B | Version: `0x01` |
| 5 | 1B | Provider type |
| 6 | 2B | Encrypted File Key length |
| 8 | 2B | Key IV length |
| 10 | 2B | Data IV length |
| 12 | var | Encrypted File Key |
| — | var | Key IV |
| — | 12B | Data IV |
| — | var | AES-GCM ciphertext + 16B auth tag |

### 5.3 KMS Providers

| Provider | Config | Mô tả |
|----------|--------|--------|
| `LocalFileProvider` | `encryption.local_key_path` | Key file on disk |
| `VaultProvider` | `encryption.vault_*` | HashiCorp Vault |
| `VolcengineKMSProvider` | `encryption.volcengine_*` | Cloud KMS |

---

## 6. Concurrency Primitives

**Path:** `storage/transaction/`, `utils/`

| Primitive | File | Chức năng |
|-----------|------|-----------|
| **LockManager** | `transaction/lock_manager.py` | Distributed file-based locks |
| **PathLock** | `transaction/` | Point/subtree/mv lock modes |
| **LockContext** | `transaction/` | Async context manager for locks |
| **RedoLog** | `transaction/` | Phase 2 crash recovery |
| **ProcessLock** | `utils/process_lock.py` | Single-instance data dir guard |
| **CircuitBreaker** | `utils/circuit_breaker.py` | External API resilience |
| **ModelRetry** | `utils/model_retry.py` | LLM/embedding retry with backoff |

### 6.1 Lock Modes

| Mode | Scope | Use Case |
|------|-------|----------|
| `point` | Single file/dir | Session commit Phase 1 |
| `subtree` | Directory tree | Recursive delete (`rm -r`) |
| `mv` | Source + dest parent | Move operations |

### 6.2 Redo Log

```
Phase 2 starts → write redo-log marker
Phase 2 completes → mark committed
Server crashes → on restart, replay uncommitted redo-logs
```

---

## 7. QueueManager — Background Tasks

**Path:** `storage/queuefs/`

| Method | Chức năng |
|--------|-----------|
| `enqueue` | Add task to queue |
| `start_workers` | Start N worker coroutines |
| `stop` | Graceful shutdown |

Task types: resource embedding, L0/L1 generation, memory extraction.

---

## 8. Observability

### 8.1 Telemetry (`telemetry/`)

| Signal | Backend | Export |
|--------|---------|--------|
| Traces | OpenTelemetry | OTLP endpoint |
| Logs | Structured JSON | stdout + OTel handler |
| Metrics | Prometheus | `/metrics` endpoint |

### 8.2 Observers (`storage/observers/`)

IO recording for debug/replay:
- Record all VikingFS operations
- Replay search queries for result comparison

### 8.3 Stats Aggregator (`storage/stats_aggregator.py`)

Aggregates retrieval statistics:
- Vector searches count
- Scored/passed ratios
- Latency distributions
- Memory extraction counts

---

## 9. RAGFS — Rust Filesystem Engine

**Path:** `crates/ragfs/`

RAGFS provides the on-disk filesystem operations via Python bindings:

| Operation | Binding |
|-----------|---------|
| `read(path, offset, size)` | Read file bytes |
| `write(path, data)` | Write file bytes |
| `mkdir(path)` | Create directory |
| `rm(path, recursive)` | Delete file/directory |
| `stat(path)` | File metadata |
| `ls(path)` | List directory entries |

Python bindings via `openviking/pyagfs/` (PyO3/Maturin).
