# Cognee — Technical Design Document

> **Version**: 1.0  
> **Date**: 2026-05-02  
> **Source**: Derived from codebase analysis of `topoteretes/cognee`

---

## 1. Executive Summary

Cognee is an open-source AI memory platform that replaces traditional RAG (Retrieval-Augmented Generation) with an **ECL (Extract, Cognify, Load)** pipeline. It transforms raw data of any format into persistent, queryable knowledge graphs combining vector embeddings, graph databases, and LLM-powered entity extraction. The platform enables AI agents to maintain persistent, learning memory across sessions with full multi-tenant isolation.

**Core value proposition:**
- Unified ingestion → graph/vector search → context delivery for AI agents
- Persistent, cross-session memory with session-to-graph bridging
- Pluggable backends (LLM, vector DB, graph DB, storage)
- Multi-tenant access control with per-user dataset isolation

---

## 2. System Architecture

### 2.1 Layered Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Public API Layer                      │
│  Python SDK (cognee.__init__)  │  FastAPI REST (v1/)     │
│  CLI (cognee-cli)              │  MCP Server (cognee-mcp) │
├─────────────────────────────────────────────────────────┤
│               V2 Memory API (High-Level)                 │
│   remember()  │  recall()  │  improve()  │  forget()     │
├─────────────────────────────────────────────────────────┤
│               V1 Core Operations                         │
│   add()  │  cognify()  │  search()  │  memify()          │
├─────────────────────────────────────────────────────────┤
│              Pipeline Orchestrator                        │
│         cognee/modules/pipelines/                         │
│   Task  │  run_pipeline  │  pipeline_execution_mode      │
├─────────────────────────────────────────────────────────┤
│               Task Execution Layer                        │
│  cognee/tasks/  — composable, async task functions        │
│  ingestion │ graph │ storage │ summarization │ documents  │
├─────────────────────────────────────────────────────────┤
│          Domain Modules (cognee/modules/)                 │
│  retrieval │ chunking │ ontology │ users │ observability  │
│  search    │ cognify  │ memify   │ agent_memory           │
├─────────────────────────────────────────────────────────┤
│        Infrastructure Adapters (cognee/infrastructure/)  │
│  LLM Gateway │ Graph DB │ Vector DB │ Relational DB       │
│  Files/Loaders │ Session │ Engine   │ Embeddings          │
├─────────────────────────────────────────────────────────┤
│              External Services                            │
│  OpenAI/Anthropic/Gemini  │  Kuzu / Neo4j / Neptune      │
│  LanceDB / ChromaDB       │  SQLite / PostgreSQL          │
│  S3 / Local FS            │  Redis (cache)                │
└─────────────────────────────────────────────────────────┘
```

### 2.2 Component Directory Map

| Path | Responsibility |
|------|---------------|
| `cognee/api/v1/` | FastAPI route handlers + Python SDK entry points |
| `cognee/modules/pipelines/` | Pipeline orchestration, Task wrapper, queue |
| `cognee/tasks/` | Atomic async pipeline task functions |
| `cognee/modules/retrieval/` | Retriever implementations per SearchType |
| `cognee/modules/chunking/` | Text chunking strategies |
| `cognee/modules/ontology/` | OWL ontology resolver & entity grounding |
| `cognee/modules/users/` | Auth, roles, permissions, ACLs |
| `cognee/modules/agent_memory/` | Agent memory context & decorator |
| `cognee/modules/observability/` | OpenTelemetry tracing & span management |
| `cognee/infrastructure/llm/` | LLM gateway, config, structured-output frameworks |
| `cognee/infrastructure/databases/graph/` | Graph DB interface + adapters |
| `cognee/infrastructure/databases/vector/` | Vector DB interface + adapters |
| `cognee/infrastructure/databases/relational/` | SQLAlchemy relational DB |
| `cognee/infrastructure/files/` | Document loaders (PDF, DOCX, audio, image, code) |
| `cognee-mcp/` | Standalone MCP server for IDE integration |
| `cognee-frontend/` | Next.js visualization UI |
| `distributed/` | Modal, Fly.io, Railway, Daytona deployment scripts |

---

## 3. Core Data Pipeline

### 3.1 V1 Pipeline: add → cognify → search

#### ADD — Data Ingestion

```
add(data, dataset_name)
  └─ resolve_authorized_user_dataset()        # create/resolve dataset + user
  └─ resolve_dlt_sources()                    # expand DLT/CSV/connection strings
  └─ run_pipeline([
       Task(resolve_data_directories),         # handle paths, URLs, S3
       Task(ingest_data, dataset_name, user)   # extract text, store metadata
     ])
  └─ returns PipelineRunInfo
```

Supported input types: `str` (text or path), `BinaryIO`, `list`, `DataItem`, DLT resource  
Supported formats: `.txt`, `.md`, `.pdf`, `.docx`, `.pptx`, `.csv`, `.py`, `.js`, `.ts`, images (OCR), audio (transcription)

#### COGNIFY — Knowledge Graph Construction

```
cognify(datasets, graph_model, chunker, chunk_size, ...)
  └─ get_default_tasks() or get_temporal_tasks()
  └─ run_pipeline([
       Task(classify_documents),               # LLM classifies doc type
       Task(extract_chunks_from_documents),    # chunk text semantically
       Task(extract_graph_from_data,           # LLM → entities & relationships
            graph_model=KnowledgeGraph),
       Task(summarize_text),                   # LLM → hierarchical summaries
       Task(add_data_points,                   # write to graph + vector DBs
            embed_triplets=True),
       Task(extract_dlt_fk_edges),             # FK-based edges for tabular data
     ])
```

The **temporal variant** replaces graph extraction with:
`extract_events_and_timestamps` → `extract_knowledge_graph_from_events`

#### SEARCH — Multi-Strategy Retrieval

```
search(query_text, query_type=SearchType.GRAPH_COMPLETION, ...)
  └─ resolve user + dataset permissions
  └─ search_function(query_type) → dispatcher
  └─ Retriever(query_text, top_k, ...) → List[SearchResult]
```

### 3.2 V2 Memory API

| Function | Behaviour |
|----------|-----------|
| `remember(data, dataset_name)` | `add()` + `cognify()` + optional `improve()` |
| `remember(data, session_id=...)` | Cache to session, background-bridge to graph |
| `recall(query, session_id=...)` | Session cache first, fall-through to graph |
| `improve(dataset)` | Triplet embedding + index refresh |
| `forget(dataset)` | Delete dataset and all associated data |
| `serve(url, api_key)` | Route SDK calls to Cognee Cloud |

`RememberResult` is a promise-like object: printable, awaitable, and inspectable.

---

## 4. Search & Retrieval System

### 4.1 SearchType Enum

| SearchType | Strategy | Use Case |
|------------|----------|----------|
| `GRAPH_COMPLETION` | Graph traversal + LLM completion | Default; complex Q&A |
| `GRAPH_COMPLETION_COT` | Chain-of-thought over graph | Multi-step reasoning |
| `GRAPH_COMPLETION_CONTEXT_EXTENSION` | Extended context retrieval | Broad context queries |
| `GRAPH_COMPLETION_DECOMPOSITION` | Query decomposition | Compound questions |
| `GRAPH_SUMMARY_COMPLETION` | Pre-computed summaries + graph | Fast overviews |
| `TRIPLET_COMPLETION` | Subject-predicate-object triplets | Relationship queries |
| `RAG_COMPLETION` | Traditional vector + LLM | Direct chunk retrieval |
| `CHUNKS` | Vector similarity only | Fast passage search |
| `CHUNKS_LEXICAL` | Jaccard / token-based | Exact-term matching |
| `SUMMARIES` | Pre-computed summaries | Document abstracts |
| `CYPHER` | Raw Cypher query | Advanced graph queries |
| `NATURAL_LANGUAGE` | NL → structured query | Non-expert users |
| `TEMPORAL` | Time-aware graph traversal | Event timelines |
| `CODING_RULES` | Code-specific pattern search | Code intelligence |
| `FEELING_LUCKY` | Auto-select best strategy | General purpose |

### 4.2 Retriever Architecture

Each `SearchType` maps to a dedicated `*Retriever` class in `cognee/modules/retrieval/`. All extend `BaseRetriever`. The dispatcher (`cognee/modules/search/methods/`) instantiates the correct retriever and returns `List[SearchResult]`.

---

## 5. Infrastructure Layer

### 5.1 LLM Gateway

`LLMGateway` (static class) provides a unified interface for all LLM operations:

| Method | Purpose |
|--------|---------|
| `acreate_structured_output()` | Entity/graph extraction via Instructor or BAML |
| `create_transcript()` | Audio transcription (Whisper) |
| `transcribe_image()` | OCR / image description |

**Structured Output Frameworks:**
- **Instructor** (default): litellm-based, supports all major providers
- **BAML**: Alternative structured extraction DSL

**Supported LLM Providers:**
`openai` · `azure` · `anthropic` · `gemini` · `ollama` · `bedrock` · `mistral` · `groq` · `custom` (OpenAI-compatible)

**Rate Limiting:** Client-side token-bucket via `LLM_RATE_LIMIT_ENABLED`, `LLM_RATE_LIMIT_REQUESTS`, `LLM_RATE_LIMIT_INTERVAL`

### 5.2 Database Adapters

#### Graph Databases (`GraphDBInterface`)

| Backend | Default | Notes |
|---------|---------|-------|
| **Kuzu** | ✅ | Embedded, local, per-user isolation |
| **Neo4j** | — | Requires `neo4j` extra; Cypher native |
| **AWS Neptune** | — | Managed cloud graph DB |
| **PostgreSQL** | — | Via adjacency tables; no Cypher |
| **Kuzu Remote** | — | Remote Kuzu server |

`GraphDBInterface` abstract methods: `add_node`, `add_nodes`, `add_edge`, `add_edges`, `get_node`, `get_nodes`, `get_neighbors`, `get_connections`, `get_neighborhood`, `get_graph_data`, `delete_graph`, `get_triplets_batch`, feedback weights

#### Vector Databases (`VectorDBInterface`)

| Backend | Default | Notes |
|---------|---------|-------|
| **LanceDB** | ✅ | Embedded, per-user isolation |
| **ChromaDB** | — | Self-hosted or managed |
| **PGVector** | — | PostgreSQL extension |
| **Qdrant** | — | High-performance vector search |
| **Weaviate** | — | Schema-based vector DB |
| **Milvus** | — | Distributed vector search |

#### Relational Database

- **SQLite** (default): Metadata, pipeline runs, user/dataset records
- **PostgreSQL**: Production alternative via `asyncpg` + SQLAlchemy

### 5.3 Embedding Engine

Factory in `cognee/infrastructure/databases/vector/embeddings/get_embedding_engine.py` supports: OpenAI, Ollama, HuggingFace, Cohere, and custom providers. Dimensions are configurable via `EMBEDDING_DIMENSIONS`.

### 5.4 Storage Backend

| Backend | Config | Notes |
|---------|--------|-------|
| **Local FS** | Default | `DATA_ROOT_DIRECTORY`, `SYSTEM_ROOT_DIRECTORY` |
| **AWS S3** | `STORAGE_BACKEND=s3` | Requires `aws` extra |

---

## 6. Data Models

### 6.1 Core Graph Models (`cognee/shared/data_models.py`)

```python
class Node(BaseModel):
    id: str
    name: str
    type: str
    description: str

class Edge(BaseModel):
    source_node_id: str
    target_node_id: str
    relationship_name: str

class KnowledgeGraph(BaseModel):
    nodes: List[Node]
    edges: List[Edge]
```

Custom graph models are created by extending `DataPoint` (engine base class with versioning and metadata).

### 6.2 Content Classification Model

`DefaultContentPrediction` classifies documents into: `TextContent`, `AudioContent`, `ImageContent`, `VideoContent`, `MultimediaContent`, `Model3DContent`, `ProceduralContent` — each with fine-grained subclass enums (50+ content sub-types).

### 6.3 Pipeline Models

- **Dataset**: Project-level container (id, name, owner user, permissions)
- **Data**: Individual data item in a dataset (content hash, MIME type, token count)
- **PipelineRunInfo**: Execution record (status, run_id, timestamps, payload)
- **DataPoint**: Base for all graph nodes (uuid, versioned, with metadata)

---

## 7. Multi-Tenancy & Access Control

### 7.1 Data Hierarchy

```
User → (many) Datasets → (many) Data items → Graph/Vector indices
```

- **Permissions**: `read`, `write`, `delete`, `share` per dataset per user
- **Roles**: Hierarchical role system via `cognee/modules/users/roles/`
- **Tenants**: Optional tenant layer for enterprise isolation

### 7.2 Per-User Database Isolation

When `ENABLE_BACKEND_ACCESS_CONTROL=True` with supported backends (Kuzu, LanceDB, SQLite, Postgres), each user+dataset gets isolated graph and vector database instances, preventing cross-tenant data leakage.

### 7.3 Authentication

Built on **FastAPI Users** (`get_fastapi_users.py`):
- JWT-based bearer token auth (`REQUIRE_AUTHENTICATION=True`)
- Default unauthenticated mode (single user: `default_user@example.com`)
- API key support (`cognee/api/v1/api_keys/`)

---

## 8. Ontology & Semantic Grounding

Cognee supports OWL ontology-based entity extraction via `cognee/modules/ontology/`:

```bash
ONTOLOGY_RESOLVER=rdflib
MATCHING_STRATEGY=fuzzy      # 80% similarity threshold
ONTOLOGY_FILE_PATH=/path/to/ontology.owl
```

During `cognify()`, extracted entities are matched against the ontology using fuzzy or exact matching, grounding the knowledge graph in standardized vocabularies.

---

## 9. Observability & Tracing

The `cognee/modules/observability/` module provides:

- **OpenTelemetry** integration: spans for all pipeline operations, LLM calls, DB queries
- **Span attributes**: `COGNEE_SEARCH_QUERY`, `COGNEE_LLM_MODEL`, `COGNEE_RESULT_COUNT`, `COGNEE_PIPELINE_NAME`, `COGNEE_SESSION_ID`, etc.
- **Trace context**: `enable_tracing()`, `disable_tracing()`, `get_last_trace()`, `get_all_traces()`
- **Langfuse + Sentry** integration via `monitoring` extra
- **PostHog** analytics via `posthog` extra
- **Structured logging** via `structlog` (`LOG_LEVEL` env var)

---

## 10. Agent Memory Module

`cognee/modules/agent_memory/` provides runtime memory injection:

```python
@agent_memory(with_memory=True)
async def my_agent_function(...):
    ...  # LLMGateway automatically prepends relevant memory context
```

The `LLMGateway._inject_agent_memory()` prepends `get_current_agent_memory_context()` to all LLM text inputs when a memory context is active.

---

## 11. MCP Server (cognee-mcp)

`cognee-mcp/` is a standalone **Model Context Protocol** server enabling direct IDE integration (Cursor, Claude Desktop, VS Code via Cline/Roo):

- Exposes Cognee memory operations as MCP tools
- Transport: SSE (`TRANSPORT_MODE=sse`) or stdio
- Dockerized with separate compose profile (`--profile mcp`)
- Port: `8000` (MCP), `5678` (debugger)

---

## 12. REST API Endpoints

FastAPI application under `cognee/api/v1/`:

| Route | Method | Description |
|-------|--------|-------------|
| `/health` | GET | Health check |
| `/add` | POST | Ingest data into dataset |
| `/cognify` | POST | Build knowledge graph |
| `/search` | POST | Query knowledge |
| `/memify` | POST | Enrich graph |
| `/remember` | POST | V2: add + cognify atomically |
| `/recall` | POST | V2: session-aware search |
| `/improve` | POST | V2: self-improvement |
| `/forget` | POST | V2: delete memory |
| `/datasets` | GET/POST/DELETE | Dataset management |
| `/users` | GET/POST | User management |
| `/permissions` | GET/POST | ACL management |
| `/ontologies` | GET/POST | Ontology management |
| `/visualize` | GET | Graph visualization server |
| `/serve` | POST | Connect to Cognee Cloud |
| `/settings` | GET/POST | Runtime configuration |
| `/activity` | GET | Pipeline run history |

---

## 13. Deployment

### 13.1 Docker Compose Services

| Service | Profile | Port | Description |
|---------|---------|------|-------------|
| `cognee` | default | 8000 | Main FastAPI backend |
| `cognee-mcp` | `mcp` | 8000 | MCP server |
| `frontend` | `ui` | 3000 | Next.js visualization UI |
| `neo4j` | `neo4j` | 7474/7687 | Neo4j graph database |
| `chromadb` | `chromadb` | 3002 | ChromaDB vector database |
| `postgres` | `postgres` | 5432 | PostgreSQL + pgvector |
| `redis` | `redis` | 6379 | Session cache |

### 13.2 Deployment Targets

| Platform | Command |
|----------|---------|
| Local Docker | `docker compose up` |
| Modal (serverless) | `bash distributed/deploy/modal-deploy.sh` |
| Railway | `railway init && railway up` |
| Fly.io | `bash distributed/deploy/fly-deploy.sh` |
| Cognee Cloud | `await cognee.serve(url=..., api_key=...)` |

### 13.3 Default Stack (Zero Config)

| Layer | Default |
|-------|---------|
| Relational | SQLite (`.venv/`) |
| Vector | LanceDB (`.venv/`) |
| Graph | Kuzu (`.venv/`) |
| Storage | Local filesystem |
| LLM | OpenAI `gpt-4o-mini` |

---

## 14. Configuration Reference

All configuration via environment variables (`.env` or system env):

```bash
# Minimal
LLM_API_KEY="your-openai-key"

# LLM
LLM_PROVIDER=openai          # openai|azure|anthropic|gemini|ollama|bedrock|custom
LLM_MODEL=openai/gpt-4o-mini
LLM_TEMPERATURE=0.0
LLM_MAX_COMPLETION_TOKENS=16384

# Vector DB
VECTOR_DB_PROVIDER=lancedb   # lancedb|pgvector|chromadb|qdrant|weaviate|milvus

# Graph DB
GRAPH_DATABASE_PROVIDER=kuzu # kuzu|neo4j|neptune|postgres|kuzu-remote

# Relational DB
DB_PROVIDER=sqlite            # sqlite|postgres

# Storage
STORAGE_BACKEND=local         # local|s3

# Security
REQUIRE_AUTHENTICATION=false
ENABLE_BACKEND_ACCESS_CONTROL=true
ALLOW_CYPHER_QUERY=true
ACCEPT_LOCAL_FILE_PATH=true

# Performance
LLM_RATE_LIMIT_ENABLED=false
LLM_RATE_LIMIT_REQUESTS=60
LLM_RATE_LIMIT_INTERVAL=60

# Observability
LOG_LEVEL=INFO
TELEMETRY_DISABLED=0
LITELLM_LOG=ERROR
```

---

## 15. Extension Points

| Extension | How To |
|-----------|--------|
| **Custom pipeline task** | `async def my_task(data): ...` wrapped in `Task(my_task)` |
| **Custom graph model** | Pydantic `BaseModel` extending `DataPoint` |
| **New graph backend** | Implement `GraphDBInterface` ABC |
| **New vector backend** | Implement `VectorDBInterface` ABC |
| **New LLM provider** | Configure via litellm (no code needed) |
| **New document processor** | Add loader in `cognee/infrastructure/files/` |
| **New search type** | Add to `SearchType` enum + implement `*Retriever` |
| **Community retrievers** | Register via `register_retriever()` |

---

## 16. Testing Strategy

| Layer | Location | Notes |
|-------|----------|-------|
| Unit | `cognee/tests/unit/` | Individual module tests |
| Integration | `cognee/tests/integration/` | Full add→cognify→search flow |
| CLI | `cognee/tests/cli_tests/` | CLI command tests |
| Tasks | `cognee/tasks/*/tests/` | Task-specific tests |
| Evaluation | `cognee/eval_framework/` | LLM output quality metrics |
| Evals | `evals/` | Benchmark evaluations |

**Tools**: `pytest`, `pytest-asyncio`, `mypy`, `ruff`, `pylint`, `deepeval`, `pre-commit`

---

## 17. Key Design Decisions

1. **ECL over RAG**: Entity-Cognify-Load creates persistent, relationship-aware knowledge rather than transient retrieval chunks.

2. **Async-first**: All SDK functions are `async`. The pipeline orchestrator supports both blocking and background execution modes.

3. **Adapter pattern**: All database backends implement abstract interfaces (`GraphDBInterface`, `VectorDBInterface`), enabling swap without changing domain logic.

4. **Dual API surface**: V1 (low-level operations) and V2 (memory-oriented `remember/recall/forget`) coexist, allowing both fine-grained control and ergonomic agent integration.

5. **Session-to-graph bridging**: Session memory (fast cache) automatically syncs to the permanent knowledge graph via background `improve()` calls, providing both speed and persistence.

6. **Ontology grounding**: Optional OWL ontology support prevents graph fragmentation by normalizing entity names to canonical forms.

7. **Pipeline composability**: `Task` objects are pure functions that can be composed, reordered, or replaced, making custom pipelines a first-class feature.
