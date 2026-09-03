# L2 — Core Operations Layer

> **Layer**: 2 (Application Logic)  
> **Responsibility**: Implement high-level business operations that orchestrate the data pipeline  
> **Dependencies**: L3 (Pipeline Orchestration), L5 (Domain Modules)

---

## 1. Tổng Quan

Layer 2 chứa các **hàm nghiệp vụ cấp cao** — mỗi hàm đại diện cho một thao tác hoàn chỉnh mà người dùng thực hiện. Layer này có 2 surface:

| Surface | Kiểu | Mô tả |
|---------|------|--------|
| **V1 API** | Low-level | `add()`, `cognify()`, `search()`, `memify()`, `delete()` |
| **V2 Memory API** | High-level | `remember()`, `recall()`, `improve()`, `forget()`, `serve()` |

V2 API là abstraction trên V1 — `remember()` = `add()` + `cognify()` + optional `improve()`.

---

## 2. V1 Core Operations

### 2.1 `add(data, dataset_name, ...)` — Data Ingestion

**File**: `cognee/api/v1/add/add.py`

**Chức năng**: Nhận dữ liệu thô (text, file, URL, DLT resource) và lưu vào dataset.

**Input types hỗ trợ**:
- `str` — text content, file path, S3 path, URL
- `BinaryIO` — file stream
- `list[str]` / `list[BinaryIO]` — batch
- `DataItem` — structured data item
- `DltResource` — Data Load Tool resource

**Processing flow**:
1. Route to remote nếu `serve()` đã connect
2. Transform `preferred_loaders` config
3. Build pipeline: `[resolve_data_directories, ingest_data]`
4. `setup()` — ensure relational DB schemas exist
5. `resolve_authorized_user_dataset()` — resolve user + dataset + permissions
6. `resolve_dlt_sources()` — expand DLT/CSV/connection strings
7. `reset_dataset_pipeline_run_status()` — clear previous pipeline states
8. Execute pipeline (blocking hoặc background)

**Key parameters**:
- `dataset_name: str = "main_dataset"`
- `node_set: List[str]` — graph organization
- `incremental_loading: bool = True` — skip already-processed data
- `run_in_background: bool = False`
- `importance_weight: float = 0.5`

**Return**: `PipelineRunInfo`

---

### 2.2 `cognify(datasets, graph_model, ...)` — Knowledge Graph Construction

**File**: `cognee/api/v1/cognify/cognify.py`

**Chức năng**: Chuyển đổi dữ liệu đã ingest thành knowledge graph có cấu trúc.

**Processing pipeline (default)**:
```
classify_documents
  → extract_chunks_from_documents (TextChunker, max_chunk_size)
  → extract_graph_and_summarize (LLM entity extraction + summarization)
  → add_data_points (write to graph + vector DBs)
  → extract_dlt_fk_edges (FK-based edges cho tabular data)
```

**Processing pipeline (temporal variant)**:
```
classify_documents
  → extract_chunks_from_documents
  → extract_events_and_timestamps
  → extract_knowledge_graph_from_events
  → add_data_points
```

**Key parameters**:
- `graph_model: BaseModel = KnowledgeGraph` — custom graph model
- `chunker = TextChunker` — chunking strategy
- `chunk_size: int` — tokens per chunk (auto-calculated if None)
- `config: Config` — ontology configuration
- `custom_prompt: str` — custom LLM prompt
- `temporal_cognify: bool = False` — enable temporal processing
- `chunks_per_batch: int` — batch size cho cognify tasks

**Observability**: Wrapped trong `new_span("cognee.api.cognify")` với attributes:
- `COGNEE_PIPELINE_NAME`
- `COGNEE_RESULT_SUMMARY`

---

### 2.3 `search(query_text, query_type, ...)` — Multi-Strategy Retrieval

**File**: `cognee/api/v1/search/search.py`

**Chức năng**: Query knowledge graph với nhiều chiến lược tìm kiếm khác nhau.

**SearchType options** (15 types):

| SearchType | Strategy | Use case |
|------------|----------|----------|
| `GRAPH_COMPLETION` | Graph traversal + LLM | Default, complex Q&A |
| `GRAPH_COMPLETION_COT` | Chain-of-thought over graph | Multi-step reasoning |
| `GRAPH_COMPLETION_CONTEXT_EXTENSION` | Extended context retrieval | Broad context |
| `GRAPH_COMPLETION_DECOMPOSITION` | Query decomposition | Compound questions |
| `GRAPH_SUMMARY_COMPLETION` | Pre-computed summaries + graph | Fast overviews |
| `TRIPLET_COMPLETION` | Subject-predicate-object | Relationship queries |
| `RAG_COMPLETION` | Traditional vector + LLM | Direct chunk retrieval |
| `CHUNKS` | Vector similarity only | Fast passage search |
| `CHUNKS_LEXICAL` | Jaccard / token-based | Exact-term matching |
| `SUMMARIES` | Pre-computed summaries | Document abstracts |
| `CYPHER` | Raw Cypher query | Advanced graph |
| `NATURAL_LANGUAGE` | NL → structured query | Non-expert users |
| `TEMPORAL` | Time-aware traversal | Event timelines |
| `CODING_RULES` | Code-specific patterns | Code intelligence |
| `FEELING_LUCKY` | Auto-select best | General purpose |

**Processing flow**:
1. Validate `neighborhood_depth`, `neighborhood_seed_top_k`
2. Resolve user (default user nếu None)
3. Set session context variable
4. Transform dataset names → UUIDs
5. Check permissions (`read` access)
6. Dispatch to `search_function()` → Retriever
7. Record observability spans

**Key parameters**:
- `top_k: int = 10`
- `node_type`, `node_name` — entity filtering
- `session_id` — session memory context
- `feedback_influence: float = 0.0` — feedback weighting
- `verbose: bool = False`
- `neighborhood_depth`, `neighborhood_seed_top_k` — graph scope

---

### 2.4 `memify()` — Graph Enrichment

**File**: `cognee/modules/memify/memify.py`

**Chức năng**: Enrich knowledge graph với extraction và enrichment tasks bổ sung.

---

### 2.5 `delete()` / `prune()` / `update()`

| Function | File | Chức năng |
|----------|------|-----------|
| `delete()` | `api/v1/delete/` | Remove data by ID or dataset |
| `prune()` | `api/v1/prune/` | Remove obsolete data |
| `update()` | `api/v1/update/` | Update existing data |

---

## 3. V2 Memory API

### 3.1 `remember(data, dataset_name, session_id)`

**File**: `cognee/api/v1/remember/`

| Mode | Behavior |
|------|----------|
| Without `session_id` | `add()` + `cognify()` + optional `improve()` — permanent graph |
| With `session_id` | Cache to session memory, background-bridge to graph |

**Return**: `RememberResult` — promise-like object (printable, awaitable, inspectable)

### 3.2 `recall(query, session_id)`

**File**: `cognee/api/v1/recall/`

| Mode | Behavior |
|------|----------|
| Without `session_id` | Direct graph search |
| With `session_id` | Session cache first, fall-through to graph |

### 3.3 `improve(dataset)`

**File**: `cognee/api/v1/improve/`

Triplet embedding + index refresh cho self-improvement.

### 3.4 `forget(dataset)`

**File**: `cognee/api/v1/forget/`

Delete dataset and all associated data.

### 3.5 `serve(url, api_key)` / `disconnect()`

**File**: `cognee/api/v1/serve/`

Route tất cả SDK calls to a remote Cognee Cloud instance.

---

## 4. Remote Client Pattern

Mọi V1/V2 function đều check remote client trước khi xử lý local:

```python
from cognee.api.v1.serve.state import get_remote_client

client = get_remote_client()
if client is not None:
    return await client.add(data, dataset_name)  # delegate to cloud

# ... local processing
```

---

## 5. Key Design Decisions

1. **Dual API surface** — V1 (granular control) coexists with V2 (ergonomic memory) 
2. **Remote-first routing** — Mọi function tự động proxy to cloud nếu `serve()` đã gọi
3. **Async-first** — Tất cả functions đều `async`
4. **Pipeline delegation** — Core operations delegate processing to L3 Pipeline Orchestration
5. **Permission enforcement** — Dataset access checked trước khi xử lý
