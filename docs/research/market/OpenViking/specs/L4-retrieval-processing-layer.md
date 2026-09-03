# L4 — Retrieval & Processing Layer

> **Layer**: 4 (Processing)  
> **Responsibility**: Search algorithms, session management, content processing  
> **Dependencies**: L3 (Core Domain), L5 (Infrastructure)

---

## 1. Tổng Quan

Layer 4 chứa các thuật toán và pipeline xử lý phức tạp nhất của OpenViking — từ hierarchical retrieval, session 2-phase commit, đến resource parsing và embedding.

| Module | Path | Size | Chức năng |
|--------|------|------|-----------|
| **HierarchicalRetriever** | `retrieve/hierarchical_retriever.py` | 627 lines | Core search algorithm |
| **MemoryLifecycle** | `retrieve/memory_lifecycle.py` | — | Hotness scoring + decay |
| **Session** | `session/session.py` | 2629 lines | Session lifecycle + 2-phase commit |
| **SessionCompressor** | `session/compressor.py` | — | Working Memory v2 generation |
| **ResourceProcessor** | `utils/resource_processor.py` | 16KB | Resource ingestion pipeline |
| **SkillProcessor** | `utils/skill_processor.py` | 12KB | Skill loading & indexing |
| **ContentWriter** | `storage/content_write.py` | 22KB | Content write + embed + summarize |
| **Parse Engine** | `parse/` | 9 files | File type detection + parsing |
| **EmbeddingUtils** | `utils/embedding_utils.py` | 17KB | Chunk + embed orchestration |
| **Summarizer** | `utils/summarizer.py` | 5KB | L0/L1 summary generation |
| **SearchFilters** | `utils/search_filters.py` | 6KB | Query filter construction |
| **WatchManager** | `resource/watch_manager.py` | 27KB | Resource auto-refresh |
| **WatchScheduler** | `resource/watch_scheduler.py` | 13KB | Periodic refresh scheduling |

---

## 2. HierarchicalRetriever — Core Search

**File:** `retrieve/hierarchical_retriever.py` (627 lines)

### 2.1 Algorithm — 6-Step Pipeline

```
Input: query (str), context_type?, target_directories?, limit, threshold

Step 1: Determine Starting Directories
  ├── From target_directories (explicit scope from user)
  └── From context_type → _get_root_uris_for_type():
         MEMORY  → [user/memories, agent/memories]
         RESOURCE → [viking://resources]
         SKILL   → [agent/skills]
         None    → all of the above

Step 2: Embed Query
  └── embedder.embed(query, is_query=True)
      → dense_vector (float32, 768-3072 dimensions)
      → sparse_vector (BM25/SPLADE-style, if supported)

Step 3: Global Vector Search
  └── vikingdb.search_global_roots_in_tenant(
        dense_vector, sparse_vector,
        account_id, topk=GLOBAL_SEARCH_TOPK)
      → top-K candidates across all L0/L1 nodes

Step 4: Merge Starting Points
  ├── Root URIs from Step 1
  ├── Global hits from Step 3 (L0/L1 only, exclude L2)
  ├── Optional rerank on merged set
  └── Initialize priority queue (max-heap by score)

Step 5: Recursive Directory Search
  ┌─────────────────────────────────────────────┐
  │ while priority_queue not empty:              │
  │   dir = pop highest-score directory          │
  │                                              │
  │   children = vikingdb.search_children(       │
  │     dir.uri, dense, sparse, account_id)      │
  │                                              │
  │   for child in children:                     │
  │     if child is L2 file:                     │
  │       optional rerank                        │
  │       score = α·child_score + (1-α)·dir_score│
  │       add to candidates (dedup by URI)       │
  │     elif child is L0/L1 directory:           │
  │       push to priority_queue                 │
  │                                              │
  │   convergence_check:                         │
  │     if top-K stable for 3 rounds → STOP      │
  └─────────────────────────────────────────────┘

Step 6: Post-Processing
  ├── Hotness blending:
  │   final = (1 - α_hot) · semantic + α_hot · hotness
  │   where hotness = f(active_count, updated_at)
  ├── Sort by final score descending
  ├── Apply limit and threshold
  └── Fetch related contexts (if relations exist)

Output: QueryResult(matched_contexts[], searched_directories[])
```

### 2.2 Key Parameters

| Parameter | Default | Mô tả |
|-----------|---------|--------|
| `GLOBAL_SEARCH_TOPK` | 10 | Global candidate count |
| `MAX_CONVERGENCE_ROUNDS` | 3 | Stop after N stable rounds |
| `DIRECTORY_DOMINANCE_RATIO` | 1.2 | Dir must exceed max child |
| `score_propagation_alpha` | configurable | Parent→child blend weight |
| `hotness_alpha` | configurable | Semantic vs recency weight |

### 2.3 Score Propagation

```
child_final = α · child_raw_score + (1 - α) · parent_score
```

Mục đích: Files trong thư mục "đúng chủ đề" được boost lên, ngay cả khi bản thân file có score trung bình.

### 2.4 Convergence Detection

```
Round N:   topK = {A, B, C, D, E}
Round N+1: topK = {A, B, C, D, E}  → convergence++ (1)
Round N+2: topK = {A, B, C, D, E}  → convergence++ (2)
Round N+3: convergence >= 3         → STOP
```

---

## 3. Session — Two-Phase Commit

**File:** `session/session.py` (2629 lines)

### 3.1 Session Lifecycle

```
create → add_message* → used() → commit → (repeat) → delete
```

### 3.2 Session Data Structure

```
viking://session/{session_id}/
├── messages.jsonl              # Live messages (retained tail)
├── .meta.json                  # SessionMeta
└── history/
    ├── archive_001/
    │   ├── messages.jsonl      # Archived messages
    │   └── .overview.md        # Working Memory v2
    ├── archive_002/
    │   └── ...
    └── archive_NNN/
```

### 3.3 SessionMeta

```python
class SessionMeta:
    session_id: str
    created_at: str
    updated_at: str
    created_by_user_id: str
    participant_user_ids: List[str]
    participant_agent_ids: List[str]
    message_count: int              # Current live messages
    total_message_count: int        # All-time count
    commit_count: int
    memories_extracted: Dict[str, int]  # 8 categories
    llm_token_usage: Dict[str, int]
    embedding_token_usage: Dict[str, int]
    pending_tokens: int             # WM v2 sliding window
    keep_recent_count: int          # Messages to retain
```

### 3.4 Two-Phase Commit

**Phase 1 — Archive (Synchronous, Lock-Protected):**

```
1. Acquire PathLock on session directory
2. Load current messages from messages.jsonl
3. Split: messages[:-keep_recent] → archive
         messages[-keep_recent:] → retained
4. Write retained → messages.jsonl (overwrite)
5. Write archive → history/archive_{N}/messages.jsonl
6. Update .meta.json (commit_count++, message_count)
7. Release PathLock
```

**Phase 2 — Memory Extract (Background, asyncio.create_task):**

```
1. Write redo-log (crash safety marker)
2. Wait for any previous Phase 2 to complete
3. Generate Working Memory v2:
   - Read previous WM v2 (if exists)
   - Feed archived messages to VLM
   - VLM returns section-level operations:
     KEEP / UPDATE / APPEND for each of 7 sections
   - Write updated .overview.md
4. Extract memories via VLM:
   - Analyze archived messages for 8 categories
   - For each extracted memory:
     a. Create file in viking://user/{id}/memories/{category}/
     b. Generate L0 abstract
     c. Embed and upsert to vector index
5. Update active_count on URIs marked as "used"
6. Mark redo-log as committed
```

### 3.5 Working Memory v2 — 7 Sections

| Section | Mô tả |
|---------|--------|
| Session Title | Auto-generated conversation title |
| Current State | What the agent is currently doing |
| Task & Goals | Objectives and sub-goals |
| Key Facts & Decisions | Important discoveries |
| Files & Context | URIs and files referenced |
| Errors & Corrections | Mistakes and fixes |
| Open Issues | Unresolved items |

### 3.6 Memory Categories (8 types)

```python
MEMORY_CATEGORIES = {
    "profile",       # User profile facts
    "preferences",   # Preferences and opinions
    "entities",      # Named entities (people, orgs)
    "events",        # Time-bound events
    "cases",         # Problem-solving cases
    "patterns",      # Behavioral patterns
    "tools",         # Tool usage patterns
    "skills",        # Learned skills
}
```

---

## 4. Parse Engine — File Processing

**Path:** `openviking/parse/`

| Module | File | Chức năng |
|--------|------|-----------|
| **Base** | `base.py` (14KB) | Parser interface, chunk models |
| **Registry** | `registry.py` (11KB) | File type → parser routing |
| **Directory Scanner** | `directory_scan.py` (12KB) | Recursive scan + .gitignore |
| **Tree Builder** | `tree_builder.py` (8KB) | Directory tree construction |
| **VLM Parser** | `vlm.py` (25KB) | Image/PDF/PPTX via VLM |
| **Converter** | `converter.py` (3KB) | Format conversion utilities |
| **Custom** | `custom.py` (7KB) | Custom parser registration |
| **Gitignore** | `gitignore.py` (4KB) | .gitignore pattern matching |
| **Parsers** | `parsers/` | File-type specific parsers |
| **Resource Detector** | `resource_detector/` | Auto-detect resource type |
| **Accessors** | `accessors/` | Content access strategies |

### 4.1 Parser Registry

```python
# File extension → parser mapping
PARSER_REGISTRY = {
    ".py": TreeSitterParser("python"),
    ".js": TreeSitterParser("javascript"),
    ".ts": TreeSitterParser("typescript"),
    ".go": TreeSitterParser("go"),
    ".rs": TreeSitterParser("rust"),
    ".java": TreeSitterParser("java"),
    ".md": MarkdownParser(),
    ".pdf": DocumentParser(),
    ".docx": DocumentParser(),
    ".xlsx": DocumentParser(),
    ".epub": DocumentParser(),
    ".png": VLMParser(),
    ".jpg": VLMParser(),
    ".pptx": VLMParser(),
    # ... 50+ extensions
}
```

### 4.2 Resource Detection

```python
# Auto-detect source type
ResourceDetector.detect(path_or_url):
  "https://github.com/..." → GitRepository
  "https://..."           → WebPage
  "/path/to/dir"          → LocalDirectory
  "/path/to/file.pdf"     → SingleFile
```

---

## 5. ResourceProcessor — Ingestion Pipeline

**File:** `utils/resource_processor.py` (16KB)

### 5.1 Pipeline Steps

```
process_resource(source, name, ctx):
  1. Clone/download to viking://temp/{name}/
  2. Scan directory → file tree
  3. For each file:
     a. Detect parser from registry
     b. Parse → chunks (text segments)
     c. Compute metadata (line count, language)
  4. Build directory tree structure
  5. Queue background processing (via QueueManager):
     a. Embed each chunk (dense + sparse)
     b. Upsert Context records to VikingDB
     c. Generate L0/L1 summaries via VLM
     d. Write .abstract.md + .overview.md
  6. Move from temp → viking://resources/{name}/
  7. Report task completion to TaskTracker
```

---

## 6. ContentWriter — Write + Embed Pipeline

**File:** `storage/content_write.py` (22KB)

Handles the full pipeline of writing content to VikingFS with embedding and summarization:

```
write_content(uri, content, ctx):
  1. Write raw content to VikingFS
  2. Determine context_type + level from URI
  3. Embed content → dense_vector + sparse_vector
  4. Create Context record
  5. Upsert to VikingDB (vector index)
  6. If directory: generate/update L0 + L1 summaries
  7. If parent exists: update parent L0 summary
```

---

## 7. EmbeddingUtils — Chunk & Embed

**File:** `utils/embedding_utils.py` (17KB)

| Function | Chức năng |
|----------|-----------|
| `chunk_and_embed` | Split text → chunks → batch embed |
| `embed_single` | Embed single text |
| `compute_sparse` | Generate sparse vector |
| `merge_embeddings` | Combine multiple embeddings |

---

## 8. WatchManager — Resource Auto-Refresh

**File:** `resource/watch_manager.py` (27KB)

Monitors resources for changes and triggers re-processing:

```
WatchScheduler (periodic timer)
  └── WatchManager.check_all()
      ├── For each watched resource:
      │   ├── Check for changes (git pull / HTTP ETag)
      │   ├── If changed: queue re-process
      │   └── Update watch metadata
      └── Configurable interval
```

---

## 9. Key Files Summary

| File | Size | Chức năng |
|------|------|-----------|
| `retrieve/hierarchical_retriever.py` | 627 lines | 6-step recursive retrieval |
| `session/session.py` | 2629 lines | Session lifecycle + 2-phase commit |
| `storage/content_write.py` | 22KB | Content write + embed pipeline |
| `parse/vlm.py` | 25KB | VLM-based file parsing |
| `resource/watch_manager.py` | 27KB | Resource change monitoring |
| `utils/resource_processor.py` | 16KB | Resource ingestion pipeline |
| `utils/embedding_utils.py` | 17KB | Chunk + embed orchestration |
| `parse/base.py` | 14KB | Parser base + chunk models |
| `resource/watch_scheduler.py` | 13KB | Periodic refresh scheduling |
| `utils/skill_processor.py` | 12KB | Skill loading + indexing |
