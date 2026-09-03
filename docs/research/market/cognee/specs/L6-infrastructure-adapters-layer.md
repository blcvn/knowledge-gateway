# L6 — Infrastructure Adapters Layer

> **Layer**: 6 (Infrastructure)  
> **Responsibility**: Abstract interfaces + concrete adapters for databases, LLMs, file I/O  
> **Dependencies**: L7 (External Services)  
> **Path**: `cognee/infrastructure/`

---

## 1. Tổng Quan

Layer 6 implement **Adapter pattern** — mỗi loại service external đều có một abstract interface (ABC/Protocol) và nhiều concrete implementations. Layer trên gọi qua interface, không biết implementation cụ thể.

```
infrastructure/
├── databases/
│   ├── graph/           # GraphDBInterface + adapters
│   ├── vector/          # VectorDBInterface + adapters
│   ├── relational/      # SQLAlchemy relational DB
│   ├── hybrid/          # Combined query operations
│   ├── unified/         # Unified DB operations
│   ├── cache/           # Caching layer
│   ├── dataset_database_handler/  # Per-dataset DB routing
│   ├── dataset_queue/   # Concurrent access queue
│   ├── utils/           # DB utilities
│   └── exceptions/      # DB-specific errors
├── llm/
│   ├── LLMGateway.py    # Unified LLM interface
│   ├── config.py        # LLM provider config
│   ├── extraction/      # Structured output extraction
│   ├── structured_output_framework/  # Instructor + BAML
│   ├── prompts/         # Prompt templates
│   ├── tokenizer/       # Tokenization utilities
│   └── utils.py         # LLM utilities
├── engine/
│   ├── models/          # DataPoint, Edge, FieldAnnotations
│   └── utils/           # Engine utilities
├── files/
│   ├── storage/         # File storage backends (local, S3)
│   └── utils/           # File utilities
├── loaders/             # Document loaders (PDF, DOCX, audio, etc.)
├── entities/            # Entity models
├── session/             # Session management
├── context/             # Context management
├── data/                # Data infrastructure
├── locks/               # Distributed locking
└── utils/               # General infrastructure utilities
```

---

## 2. Graph Database (`infrastructure/databases/graph/`)

### 2.1 `GraphDBInterface` (ABC)

**File**: `graph_db_interface.py` (364 lines, 12KB)

Abstract interface cho tất cả graph database operations:

| Method | Signature | Mô tả |
|--------|-----------|--------|
| `is_empty()` | `→ bool` | Check if graph is empty |
| `query(query, params)` | `→ List[Any]` | Raw query execution |
| `add_node(node, properties)` | `→ None` | Add single node |
| `add_nodes(nodes)` | `→ None` | Batch add nodes |
| `delete_node(node_id)` | `→ None` | Delete node |
| `delete_nodes(node_ids)` | `→ None` | Batch delete nodes |
| `get_node(node_id)` | `→ NodeData` | Get single node |
| `get_nodes(node_ids)` | `→ List[NodeData]` | Batch get nodes |
| `add_edge(src, tgt, rel, props)` | `→ None` | Add edge |
| `add_edges(edges)` | `→ None` | Batch add edges |
| `delete_graph()` | `→ None` | Delete entire graph |
| `get_graph_data()` | `→ (nodes, edges)` | Get all data |
| `get_graph_metrics(include_optional)` | `→ Dict` | Graph statistics |
| `has_edge(src, tgt, rel)` | `→ bool` | Check edge existence |
| `has_edges(edges)` | `→ List[EdgeData]` | Batch edge check |
| `get_edges(node_id)` | `→ List[EdgeData]` | Get node edges |
| `get_neighbors(node_id)` | `→ List[NodeData]` | Get neighbor nodes |
| `get_nodeset_subgraph(type, names)` | `→ (nodes, edges)` | Filtered subgraph |
| `get_connections(node_id)` | `→ List[Tuple]` | Full connection info |
| `get_neighborhood(node_ids, depth)` | `→ (nodes, edges)` | K-hop neighborhood |
| `get_filtered_graph_data(filters)` | `→ (nodes, edges)` | Attribute-filtered data |
| `get_node_feedback_weights(ids)` | `→ Dict[str,float]` | Feedback weights |
| `set_node_feedback_weights(weights)` | `→ Dict[str,bool]` | Set feedback weights |
| `get_edge_feedback_weights(ids)` | `→ Dict[str,float]` | Edge feedback |
| `set_edge_feedback_weights(weights)` | `→ Dict[str,bool]` | Set edge feedback |
| `get_triplets_batch(offset, limit)` | `→ List[Dict]` | Batch triplet retrieval |

### 2.2 Concrete Adapters

| Backend | Path | Default | Notes |
|---------|------|---------|-------|
| **Kuzu** | `graph/kuzu/` | ✅ | Embedded, per-user isolation |
| **Neo4j** | `graph/neo4j_driver/` | — | Cypher native, requires extra |
| **Neptune** | `graph/neptune_driver/` | — | AWS managed graph |
| **PostgreSQL** | `graph/postgres/` | — | Adjacency tables, no Cypher |

### 2.3 Factory

```python
from cognee.infrastructure.databases.graph import get_graph_engine

graph_engine = await get_graph_engine()  # returns configured adapter
```

`get_graph_engine.py` (10KB) — factory that reads config và instantiates adapter.

### 2.4 Config

`graph/config.py` (6KB) — `GraphConfig` Pydantic model with env vars:
- `GRAPH_DATABASE_PROVIDER` — kuzu | neo4j | neptune | postgres | kuzu-remote
- `GRAPH_DATABASE_URL`, `GRAPH_DATABASE_NAME`, etc.

---

## 3. Vector Database (`infrastructure/databases/vector/`)

### 3.1 `VectorDBInterface` (Protocol)

**File**: `vector_db_interface.py` (273 lines, 10KB)

| Method | Signature | Mô tả |
|--------|-----------|--------|
| `has_collection(name)` | `→ bool` | Check collection existence |
| `create_collection(name, schema)` | | Create new collection |
| `create_data_points(name, points)` | | Insert data points |
| `retrieve(name, ids)` | | Get by IDs |
| `search(name, text, vector, limit)` | | Semantic search |
| `batch_search(name, texts, limit)` | | Multi-query search |
| `delete_data_points(name, ids)` | | Delete points |
| `prune()` | | Remove obsolete data |
| `embed_data(data)` | `→ List[List[float]]` | Text → vectors |
| `run_migrations()` | | Optional migrations |
| `create_vector_index(name, prop)` | | Create search index |
| `index_data_points(name, prop, points)` | | Index for search |
| `create_dataset(dataset_id, user)` | `→ dict` | Multi-tenant dataset creation |
| `delete_dataset(dataset_id, user)` | | Multi-tenant dataset deletion |

### 3.2 Concrete Adapters

| Backend | Path | Default | Notes |
|---------|------|---------|-------|
| **LanceDB** | `vector/lancedb/` | ✅ | Embedded, per-user isolation |
| **ChromaDB** | `vector/chromadb/` | — | Self-hosted or managed |
| **PGVector** | `vector/pgvector/` | — | PostgreSQL extension |

*Additional supported (via external packages): Qdrant, Weaviate, Milvus*

### 3.3 Factory

```python
from cognee.infrastructure.databases.vector import get_vector_engine

vector_engine = await get_vector_engine()
```

`create_vector_engine.py` (10KB) — factory.

### 3.4 Embedding Engine (`vector/embeddings/`)

Factory pattern cho embeddings: `get_embedding_engine.py`

| Provider | Notes |
|----------|-------|
| OpenAI | Default |
| Ollama | Local inference |
| HuggingFace | Transformer models |
| Cohere | Cohere API |
| Custom | OpenAI-compatible |

Config: `EMBEDDING_PROVIDER`, `EMBEDDING_MODEL`, `EMBEDDING_DIMENSIONS`

### 3.5 Vector Models (`vector/models/`)

- `PayloadSchema` — schema for vector metadata

---

## 4. Relational Database (`infrastructure/databases/relational/`)

SQLAlchemy-based relational DB cho metadata storage:

| Backend | Default | Notes |
|---------|---------|-------|
| **SQLite** | ✅ | Metadata, pipeline runs, users |
| **PostgreSQL** | — | Production, async via `asyncpg` |

Config: `DB_PROVIDER`, `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_NAME`

---

## 5. LLM Gateway (`infrastructure/llm/`)

### 5.1 `LLMGateway` Class

**File**: `LLMGateway.py` (3.4KB) — Static class, unified LLM interface.

| Method | Chức năng |
|--------|-----------|
| `acreate_structured_output()` | Entity/graph extraction via Instructor or BAML |
| `create_transcript()` | Audio transcription (Whisper) |
| `transcribe_image()` | OCR / image description |
| `_inject_agent_memory()` | Auto-prepend memory context to prompts |

### 5.2 Structured Output Frameworks

| Framework | Config Value | Backend |
|-----------|-------------|---------|
| **Instructor** | `STRUCTURED_OUTPUT_FRAMEWORK=instructor` | Default, via litellm |
| **BAML** | `STRUCTURED_OUTPUT_FRAMEWORK=baml` | Alternative DSL |

### 5.3 Supported LLM Providers

| Provider | Config | Notes |
|----------|--------|-------|
| OpenAI | `LLM_PROVIDER=openai` | Default, recommended |
| Azure OpenAI | `LLM_PROVIDER=azure` | Enterprise |
| Anthropic | `LLM_PROVIDER=anthropic` | Claude models |
| Google Gemini | `LLM_PROVIDER=gemini` | Gemini models |
| Ollama | `LLM_PROVIDER=ollama` | Local inference |
| AWS Bedrock | `LLM_PROVIDER=bedrock` | AWS managed |
| Mistral | `LLM_PROVIDER=mistral` | Mistral AI |
| Groq | `LLM_PROVIDER=groq` | Fast inference |
| Custom | `LLM_PROVIDER=custom` | Any OpenAI-compatible |

### 5.4 Rate Limiting

Client-side token-bucket:
- `LLM_RATE_LIMIT_ENABLED=true`
- `LLM_RATE_LIMIT_REQUESTS=60`
- `LLM_RATE_LIMIT_INTERVAL=60`

### 5.5 Config (`llm/config.py`, 11KB)

`LLMConfig` Pydantic model with all provider configs.

### 5.6 Prompts (`llm/prompts/`)

Template files cho various LLM operations.

### 5.7 Tokenizer (`llm/tokenizer/`)

Token counting utilities cho chunk size calculation.

---

## 6. Engine Models (`infrastructure/engine/models/`)

### 6.1 `DataPoint` (11.6KB)

Base class cho tất cả graph nodes:
- UUID-based identity
- Versioning support
- Metadata fields
- Pydantic BaseModel extension

### 6.2 `Edge` (1.4KB)

Graph relationship model:
- `source_id`, `target_id`
- `relationship_name`
- Properties dict

### 6.3 `FieldAnnotations` (2.8KB)

Field metadata for DataPoint fields.

### 6.4 `ExtendableDataPoint` (166B)

Marker class cho extensible data points.

---

## 7. File Infrastructure (`infrastructure/files/`)

### 7.1 Storage Backends

| Backend | Config | Notes |
|---------|--------|-------|
| **Local FS** | `STORAGE_BACKEND=local` | Default |
| **AWS S3** | `STORAGE_BACKEND=s3` | Requires `aws` extra |

Config: `DATA_ROOT_DIRECTORY`, `SYSTEM_ROOT_DIRECTORY`

### 7.2 Document Loaders (`infrastructure/loaders/`)

| Loader | Format |
|--------|--------|
| TextLoader | `.txt`, `.md` |
| PdfLoader | `.pdf` |
| DocxLoader | `.docx` |
| PptxLoader | `.pptx` |
| CsvLoader | `.csv` |
| CodeLoader | `.py`, `.js`, `.ts`, etc. |
| ImageLoader | `.png`, `.jpg` (OCR/vision) |
| AudioLoader | `.mp3`, `.wav` (transcription) |

---

## 8. Context & Session (`infrastructure/context/`, `infrastructure/session/`)

### 8.1 Context Variables (`context_global_variables.py`)

ContextVar-based configuration for async task isolation:

```python
vector_db_config = ContextVar("vector_db_config", default=None)
graph_db_config = ContextVar("graph_db_config", default=None)
session_user = ContextVar("session_user", default=None)
```

### 8.2 `DatabaseContextManager`

Dual-mode helper (awaitable + async context manager):

```python
# Scoped mode (preferred)
async with set_database_global_context_variables(dataset, user_id):
    # graph/vector DB config applied for this block
    ...
# Config released automatically
```

### 8.3 Dataset Queue (`infrastructure/databases/dataset_queue/`)

Concurrent access management — ensures only one pipeline per dataset at a time.

---

## 9. Other Infrastructure

| Module | Path | Chức năng |
|--------|------|-----------|
| `locks/` | `infrastructure/locks/` | Distributed locking |
| `data/` | `infrastructure/data/` | Data infrastructure |
| `entities/` | `infrastructure/entities/` | Entity models |
| `utils/` | `infrastructure/utils/` | General utilities |

---

## 10. Adapter Selection (Factory Pattern)

```python
# Graph DB
get_graph_engine() → reads GRAPH_DATABASE_PROVIDER → returns adapter instance

# Vector DB
get_vector_engine() → reads VECTOR_DB_PROVIDER → returns adapter instance

# LLM
get_llm_client() → reads LLM_PROVIDER → returns LLM client

# Embeddings
get_embedding_engine() → reads EMBEDDING_PROVIDER → returns embedding engine
```

Tất cả factories đều đọc config từ environment variables và trả về instance phù hợp.
