# Software Requirements Specification (SRS)

**Product:** Cognee — AI Knowledge Engine  
**Version:** 1.0.3  
**Last Updated:** 2026-05-07  
**Source:** Codebase analysis of `topoteretes/cognee`  
**License:** Apache-2.0

---

## 1. Giới Thiệu

### 1.1 Mục đích

Tài liệu SRS định nghĩa các yêu cầu phần mềm chi tiết cho hệ thống Cognee, bao gồm functional requirements, interface specifications, data models, và infrastructure constraints.

### 1.2 Phạm vi hệ thống

Cognee là AI Knowledge Engine mã nguồn mở sử dụng pipeline **ECL (Extract-Cognify-Load)** để chuyển đổi dữ liệu thô thành knowledge graph kết hợp vector search, graph traversal, và LLM completion. Hệ thống hỗ trợ multi-tenant isolation, pluggable backends, và persistent agent memory.

### 1.3 Kiến trúc tổng quan (7-Layer)

```
L1 — PUBLIC API LAYER         (Python SDK · FastAPI · CLI · MCP)
L2 — CORE OPERATIONS LAYER    (V1: add/cognify/search · V2: remember/recall/forget)
L3 — PIPELINE ORCHESTRATION   (Task · BoundTask · run_pipeline · Queue)
L4 — TASK EXECUTION           (ingestion · graph · storage · summarization)
L5 — DOMAIN MODULES           (retrieval · chunking · ontology · users · search)
L6 — INFRASTRUCTURE ADAPTERS  (LLM Gateway · Graph/Vector/Relational DB · Loaders)
L7 — EXTERNAL SERVICES        (OpenAI · Kuzu/Neo4j · LanceDB · PostgreSQL · S3)
```

---

## 2. Functional Requirements

### 2.1 Data Ingestion (FR-ING)

#### FR-ING-01: Unified Data Input

| Attribute | Value |
|-----------|-------|
| **Function** | `cognee.add(data, dataset_name, node_set)` |
| **Input types** | `str`, `BinaryIO`, `list`, `DataItem`, DLT resource |
| **Formats** | `.txt`, `.md`, `.pdf`, `.docx`, `.pptx`, `.csv`, `.py`, `.js`, `.ts`, images (OCR), audio (transcription) |

**Processing flow:**
```
add(data, dataset_name)
  → resolve_authorized_user_dataset()
  → resolve_dlt_sources()
  → run_pipeline([resolve_data_directories, ingest_data])
  → returns PipelineRunInfo
```

**Optional loaders (plugins):**

| Loader | Extra | Formats |
|--------|-------|---------|
| TextLoader | built-in | `.txt`, `.md` |
| PdfLoader | built-in | `.pdf` |
| DocxLoader | built-in | `.docx` |
| UnstructuredLoader | `cognee[docs]` | DOC, EPUB, RTF, RST, XLSX, ORG, etc. |
| DoclingLoader | `cognee[docling]` | Advanced document processing |
| BeautifulSoupLoader | `cognee[scraping]` | HTML parsing |
| AudioLoader | built-in | `.mp3`, `.wav` (Whisper transcription) |
| ImageLoader | built-in | `.png`, `.jpg` (OCR/vision) |
| CodeLoader | built-in | `.py`, `.js`, `.ts`, etc. |

#### FR-ING-02: NodeSet Tagging

```python
await cognee.add(data, dataset_name="crm", node_set=["customer_123", "preferences"])
```

NodeSets cho phép scoped search: `node_name=["customer_123"]`

#### FR-ING-03: DataPoint Custom Schema

```python
class ScientificPaper(DataPoint):
    title: str
    authors: list[str]
    findings: list[str]
    metadata: dict = {"index_fields": ["title", "findings"]}
```

`DataPoint` base class: UUID identity, versioning, Pydantic BaseModel extension.

---

### 2.2 Knowledge Processing (FR-PROC)

#### FR-PROC-01: Cognify Pipeline

| Attribute | Value |
|-----------|-------|
| **Function** | `cognee.cognify(datasets, graph_model, chunker, chunk_size, ...)` |
| **Default graph model** | `KnowledgeGraph(nodes: List[Node], edges: List[Edge])` |

**Default pipeline tasks:**
1. `classify_documents` — LLM classifies document type (50+ content sub-types)
2. `extract_chunks_from_documents` — Text chunking (configurable chunk_size)
3. `extract_graph_from_data` — LLM entity & relationship extraction
4. `summarize_text` — LLM hierarchical summarization
5. `add_data_points` — Write to graph + vector DBs (embed_triplets=True)
6. `extract_dlt_fk_edges` — FK-based edges for tabular data

**Temporal variant** replaces step 3 with:
- `extract_events_and_timestamps`
- `extract_knowledge_graph_from_events`

#### FR-PROC-02: Memify (Graph Enrichment)

```python
await cognee.memify(dataset="research")
```

Non-destructive enrichment: derived facts, rules, patterns, triplet embeddings.

#### FR-PROC-03: Content Classification Model

`DefaultContentPrediction` classifies into: `TextContent`, `AudioContent`, `ImageContent`, `VideoContent`, `MultimediaContent`, `Model3DContent`, `ProceduralContent` — each with fine-grained subclass enums.

---

### 2.3 Search & Retrieval (FR-SEARCH)

#### FR-SEARCH-01: SearchType Registry

| SearchType | Retriever Class | Strategy |
|------------|----------------|----------|
| `GRAPH_COMPLETION` | GraphCompletionRetriever | Vector → graph traversal → LLM |
| `GRAPH_COMPLETION_COT` | GraphCompletionCotRetriever | Chain-of-thought reasoning |
| `GRAPH_COMPLETION_DECOMPOSITION` | GraphCompletionDecompositionRetriever | Query decomposition |
| `GRAPH_COMPLETION_CONTEXT_EXTENSION` | GraphCompletionContextExtensionRetriever | Broad context |
| `GRAPH_SUMMARY_COMPLETION` | GraphSummaryCompletionRetriever | Summaries + graph |
| `TRIPLET_COMPLETION` | TripletRetriever | Subject-predicate-object |
| `RAG_COMPLETION` | CompletionRetriever | Traditional RAG |
| `CHUNKS` | ChunksRetriever | Vector similarity only |
| `CHUNKS_LEXICAL` | LexicalRetriever | Jaccard token-based |
| `SUMMARIES` | SummariesRetriever | Pre-computed summaries |
| `CYPHER` | CypherSearchRetriever | Raw Cypher queries |
| `NATURAL_LANGUAGE` | NaturalLanguageRetriever | NL → structured query |
| `TEMPORAL` | TemporalRetriever | Time-aware traversal |
| `CODING_RULES` | CodingRulesRetriever | Code patterns |
| `FEELING_LUCKY` | Auto-select | Best strategy |
| `FEEDBACK` | — | Reinforce retrieval |

#### FR-SEARCH-02: BaseRetriever Pipeline (3-step)

```python
class BaseRetriever(ABC):
    async def get_retrieved_objects(query, query_batch)   # Step 1: Fetch raw data
    async def get_context_from_objects(query, objects)     # Step 2: Format for LLM
    async def get_completion_from_context(query, context)  # Step 3: LLM completion
    async def get_completion(query)                        # Convenience: all 3 steps
```

Extension point: `register_retriever("name", CustomRetriever)` cho community retrievers.

#### FR-SEARCH-03: Search Function

```python
results = await cognee.search(
    query_text="...",
    query_type=SearchType.GRAPH_COMPLETION,
    datasets="research",
    node_name=["customer_123"],
    top_k=10,
    save_interaction=True
)
# Returns List[SearchResult]
```

---

### 2.4 V2 Memory API (FR-MEM)

#### FR-MEM-01: Remember

| Mode | Behavior |
|------|----------|
| Permanent | `remember(data)` → `add()` + `cognify()` + optional `improve()` |
| Session | `remember(data, session_id=...)` → cache, background bridge to graph |
| Structured | `remember(QAEntry(...))` → `SessionManager.add_qa` |
| Trace | `remember(TraceEntry(...))` → `SessionManager.add_agent_trace_step` |
| Feedback | `remember(FeedbackEntry(...))` → `SessionManager.add_feedback` |

`RememberResult`: promise-like object — printable, awaitable, inspectable.

#### FR-MEM-02: Recall

```python
results = await cognee.recall(query, session_id="chat_1", scope="auto")
```

**RecallScope enum:** `auto`, `graph`, `session`, `trace`, `graph_context`, `all`

- `auto`: session cache first → fall-through to graph search
- `all`: expands to `["graph", "session", "trace", "graph_context"]`

#### FR-MEM-03: Forget

```python
await cognee.forget(dataset="main_dataset")  # Delete dataset + all data
```

#### FR-MEM-04: Improve

```python
await cognee.improve(dataset="research")  # Triplet embedding + index refresh
```

#### FR-MEM-05: Serve (Cloud)

```python
await cognee.serve(url="https://cloud.cognee.ai", api_key="...")
# Routes all SDK calls to Cognee Cloud
```

---

### 2.5 Authentication & Authorization (FR-AUTH)

#### FR-AUTH-01: Authentication Schemes

| Scheme | Implementation |
|--------|---------------|
| JWT Bearer | `fastapi-users` — `REQUIRE_AUTHENTICATION=True` |
| API Key | `X-Api-Key` header — `cognee/api/v1/api_keys/` |
| Cookie | Session-based via FastAPI Users |
| Unauthenticated | Default user `default_user@example.com` |

JWT config: `FASTAPI_USERS_JWT_SECRET`, `JWT_LIFETIME_SECONDS=3600`  
API Key hashing: `HASH_API_KEY=True` → SHA-256

#### FR-AUTH-02: Permission Model

```
Tenant (optional)
  └─ User
       └─ UserTenant (many-to-many)
       └─ UserRole (many-to-many)
       └─ Dataset (many)
            └─ Permission (read | write | delete | share)
            └─ Data (many)
                 └─ Graph/Vector indices
```

#### FR-AUTH-03: Backend Access Control

Khi `ENABLE_BACKEND_ACCESS_CONTROL=True`:
- Mỗi user+dataset → isolated graph + vector DB instances
- `DatabaseContextManager` (async context manager) handles scoped DB switching
- `ContextVar`-based config: `vector_db_config`, `graph_db_config`, `session_user`
- `DatasetQueue` ensures one pipeline per dataset at a time

---

### 2.6 Pipeline Orchestration (FR-PIPE)

#### FR-PIPE-01: Task Model

```python
from cognee.modules.pipelines.tasks.task import Task

task = Task(my_async_function, arg1, arg2)
# Task wraps pure async functions for pipeline composition
```

Components: `Task`, `BoundTask`, `TaskSpec`, `run_pipeline`

#### FR-PIPE-02: Pipeline Execution

```python
await cognee.run_custom_pipeline(
    tasks=[Task(step_1), Task(step_2)],
    data="input",
    dataset="research"
)
```

Supports blocking and background execution modes. Pipeline queue manages concurrent access.

---

## 3. Interface Requirements

### 3.1 REST API (FastAPI)

**Base URL:** `http://localhost:8000`

| Route | Method | Tag | Description |
|-------|--------|-----|-------------|
| `/health` | GET | health | Health check |
| `/api/v1/auth` | POST | auth | Login, register, reset, verify, API keys |
| `/api/v1/add` | POST | add | Data ingestion |
| `/api/v1/cognify` | POST | cognify | Knowledge graph construction |
| `/api/v1/search` | POST | search | Multi-mode search |
| `/api/v1/memify` | POST | memify | Graph enrichment |
| `/api/v1/remember` | POST | remember | V2 memory store |
| `/api/v1/recall` | POST | recall | V2 memory retrieve |
| `/api/v1/improve` | POST | improve | V2 self-improvement |
| `/api/v1/forget` | POST | forget | V2 memory deletion |
| `/api/v1/datasets` | GET/POST/DEL | datasets | Dataset CRUD |
| `/api/v1/users` | GET/POST | users | User management |
| `/api/v1/permissions` | GET/POST | permissions | ACL management |
| `/api/v1/ontologies` | GET/POST | ontologies | Ontology management |
| `/api/v1/settings` | GET/POST | settings | Runtime config |
| `/api/v1/activity` | GET | activity | Audit trail |
| `/api/v1/sessions` | GET | sessions | Session lifecycle |
| `/api/v1/responses` | POST | responses | LLM responses |
| `/api/v1/llm` | GET/POST | llm | LLM configuration |
| `/api/v1/sync` | POST | sync | Data sync |
| `/api/v1/delete` | POST | delete | Data deletion |
| `/api/v1/update` | POST | update | Data updates |
| `/api/v1/visualize` | GET | visualize | Graph visualization |
| `/api/v1/notebooks` | GET | notebooks | Jupyter integration |
| `/api/v1/checks` | GET | checks | Cloud health checks |

**CORS:** `CORS_ALLOWED_ORIGINS` env var (comma-separated)  
**OpenAPI:** Auto-generated with BearerAuth + ApiKeyAuth security schemes

### 3.2 Python SDK

```python
import cognee

# V1 API
cognee.add, cognee.cognify, cognee.search, cognee.memify, cognee.delete, cognee.update, cognee.prune

# V2 Memory API
cognee.remember, cognee.recall, cognee.improve, cognee.forget, cognee.serve

# Configuration
cognee.config

# Observability
cognee.enable_tracing, cognee.disable_tracing, cognee.get_last_trace, cognee.get_all_traces

# Agent
cognee.agent_memory  # decorator

# Types
cognee.SearchType, cognee.MemoryEntry, cognee.QAEntry, cognee.TraceEntry, cognee.FeedbackEntry
```

### 3.3 CLI

```bash
cognee-cli remember "text"
cognee-cli recall "query"
cognee-cli forget --all
cognee-cli -ui
```

Entry point: `cognee.cli._cognee:main` (registered in pyproject.toml)

### 3.4 MCP Server

| Tool | Description |
|------|-------------|
| `cognify` | Transform data into knowledge graph |
| `search` | Query knowledge graph |
| `save_interaction` | Log user-agent interactions |
| `list_data` | List datasets and items |
| `delete_dataset` | Delete a dataset |
| `cognify_status` | Check background task status |

Transport: `stdio` (default), `sse`, `http`  
Modes: Direct (local cognee) / API (remote endpoint)

---

## 4. Infrastructure Requirements

### 4.1 Database Backends

#### 4.1.1 Graph Databases (`GraphDBInterface`)

| Backend | Default | Extra | Abstract Methods |
|---------|---------|-------|-----------------|
| **Kuzu** | ✅ | built-in | `add_node`, `add_nodes`, `add_edge`, `add_edges`, `get_node`, `get_nodes`, `get_neighbors`, `get_connections`, `get_neighborhood`, `get_graph_data`, `delete_graph`, `get_triplets_batch`, feedback weights |
| **Neo4j** | — | `cognee[neo4j]` | Same interface |
| **Neptune** | — | `cognee[neptune]` | Same interface |
| **PostgreSQL** | — | `cognee[postgres]` | Same interface (adjacency tables) |

Factory: `get_graph_engine()` → reads `GRAPH_DATABASE_PROVIDER`

#### 4.1.2 Vector Databases (`VectorDBInterface`)

| Backend | Default | Extra |
|---------|---------|-------|
| **LanceDB** | ✅ | built-in |
| **PGVector** | — | `cognee[postgres]` |
| **ChromaDB** | — | `cognee[chromadb]` |

Abstract methods: `has_collection`, `create_collection`, `create_data_points`, `retrieve`, `search`, `batch_search`, `delete_data_points`, `prune`, `embed_data`

Factory: `get_vector_engine()` → reads `VECTOR_DB_PROVIDER`

#### 4.1.3 Relational Database

| Backend | Default | Notes |
|---------|---------|-------|
| **SQLite** | ✅ | Metadata, pipeline runs, users |
| **PostgreSQL** | — | Production, `asyncpg` + SQLAlchemy |

Migrations: Alembic with backward compatibility

#### 4.1.4 Storage Backend

| Backend | Config |
|---------|--------|
| **Local FS** | `STORAGE_BACKEND=local` (default) |
| **AWS S3** | `STORAGE_BACKEND=s3` + `cognee[aws]` |

### 4.2 LLM Gateway

`LLMGateway` static class:

| Method | Purpose |
|--------|---------|
| `acreate_structured_output()` | Entity/graph extraction (Instructor or BAML) |
| `create_transcript()` | Audio transcription (Whisper) |
| `transcribe_image()` | OCR / image description |
| `_inject_agent_memory()` | Auto-prepend memory context |

**Providers:** OpenAI, Azure, Anthropic, Gemini, Ollama, Bedrock, Mistral, Groq, Custom  
**Structured output:** Instructor (default) / BAML  
**Rate limiting:** Token-bucket via `LLM_RATE_LIMIT_ENABLED`

### 4.3 Embedding Engine

Factory: `get_embedding_engine()` → reads `EMBEDDING_PROVIDER`  
Providers: OpenAI (default), Ollama, HuggingFace, Cohere, Custom  
Config: `EMBEDDING_MODEL`, `EMBEDDING_DIMENSIONS`

---

## 5. Data Models

### 5.1 Core Graph Models

```python
class Node(BaseModel):
    id: str; name: str; type: str; description: str

class Edge(BaseModel):
    source_node_id: str; target_node_id: str; relationship_name: str

class KnowledgeGraph(BaseModel):
    nodes: List[Node]; edges: List[Edge]
```

### 5.2 DataPoint (Engine Base)

UUID identity, versioning, Pydantic BaseModel, metadata fields, `index_fields` support.

### 5.3 Memory Entry Types

```python
MemoryEntry = Union[QAEntry, TraceEntry, FeedbackEntry]

QAEntry:     type="qa",       question, answer, context, feedback_text/score
TraceEntry:  type="trace",    origin_function, status, method_params/return_value
FeedbackEntry: type="feedback", qa_id, feedback_text/score
```

### 5.4 Pipeline Models

- **Dataset**: id, name, owner user, permissions
- **Data**: content hash, MIME type, token count
- **PipelineRunInfo**: status, run_id, timestamps, payload

---

## 6. Observability Requirements

### 6.1 OpenTelemetry Tracing

Span attributes: `cognee.search.query`, `cognee.search.type`, `cognee.llm.model`, `cognee.llm.provider`, `cognee.result.count`, `cognee.pipeline.name`, `cognee.session.id`, `cognee.dataset.name`, `cognee.vector.collection`, `cognee.operation.mode`

API: `enable_tracing()`, `disable_tracing()`, `get_last_trace()`, `get_all_traces()`, `clear_traces()`

OTLP export: Dash0, Grafana, Datadog, Jaeger compatible

### 6.2 Monitoring Integrations

| Tool | Extra | Config |
|------|-------|--------|
| Langfuse | `cognee[monitoring]` | `LANGFUSE_PUBLIC_KEY`, `LANGFUSE_SECRET_KEY` |
| Sentry | `cognee[monitoring]` | `SENTRY_REPORTING_URL` |
| PostHog | `cognee[posthog]` | PostHog API key |

### 6.3 Logging

- Library: `structlog`
- Level: `LOG_LEVEL` env var (DEBUG, INFO, WARNING, ERROR, CRITICAL)
- Rotation: `COGNEE_LOG_MAX_BYTES=52428800` (50MB), `COGNEE_LOG_BACKUP_COUNT=5`
- Secret redaction: API keys (`sk-*`), bearer tokens, `api_key=...`, `password=...`

---

## 7. Deployment Specifications

### 7.1 Docker Compose Services

| Service | Profile | Port | Resources |
|---------|---------|------|-----------|
| `cognee` | default | 8000, 5678 | 4 CPU, 8GB RAM |
| `cognee-mcp` | `mcp` | 8000, 5678 | 2 CPU, 4GB RAM |
| `frontend` | `ui` | 3000 | — |
| `neo4j` | `neo4j` | 7474, 7687 | — |
| `chromadb` | `chromadb` | 3002 | — |
| `postgres` | `postgres` | 5432 | — |
| `redis` | `redis` | 6379 | — |

### 7.2 Default Stack (Zero Config)

| Layer | Default | Setup |
|-------|---------|-------|
| Relational | SQLite | File-based |
| Vector | LanceDB | File-based |
| Graph | Kuzu | File-based |
| Storage | Local FS | — |
| LLM | OpenAI `gpt-4o-mini` | API key only |

### 7.3 Cloud Targets

Modal (serverless), Railway, Fly.io, Render, Daytona, Cognee Cloud

### 7.4 System Requirements

- **Python**: 3.10 – 3.13
- **OS**: macOS, Linux, Windows
- **Architecture**: ARM64, x86_64

---

## 8. Security Requirements

### 8.1 Authentication

- JWT: `FASTAPI_USERS_JWT_SECRET`, `JWT_LIFETIME_SECONDS=3600`
- API Key hashing: `HASH_API_KEY=True` → SHA-256 (irreversible)
- Default mode: unauthenticated (`REQUIRE_AUTHENTICATION=false`)

### 8.2 Access Control

- `ENABLE_BACKEND_ACCESS_CONTROL=True`: per-user+dataset isolated DBs
- `ACCEPT_LOCAL_FILE_PATH=True/False`: control local file ingestion
- `ALLOW_CYPHER_QUERY=True/False`: control raw Cypher access
- `ALLOW_HTTP_REQUESTS=True/False`: control outbound HTTP

### 8.3 MCP Security

- DNS rebinding protection (default on)
- `MCP_ALLOWED_HOSTS` whitelist
- `MCP_CORS_ALLOW_ORIGINS` for SSE/HTTP

### 8.4 Secret Redaction

Auto-redaction patterns: `sk-*`, bearer tokens, `api_key=...`, `password=...`

---

## 9. Configuration Reference

### 9.1 Tier 1 — Quick Start (1 variable)

```bash
LLM_API_KEY="your-api-key"
```

### 9.2 Tier 2 — Common Overrides

```bash
LLM_MODEL, LLM_PROVIDER, LLM_ENDPOINT
EMBEDDING_PROVIDER, EMBEDDING_MODEL, EMBEDDING_DIMENSIONS
DB_PROVIDER, DB_HOST, DB_PORT, DB_USERNAME, DB_PASSWORD, DB_NAME
GRAPH_DATABASE_PROVIDER, VECTOR_DB_PROVIDER
```

### 9.3 Tier 3 — Advanced

```bash
# LLM: STRUCTURED_OUTPUT_FRAMEWORK, LLM_RATE_LIMIT_*, LLM_ARGS
# Storage: STORAGE_BACKEND, STORAGE_BUCKET_NAME, DATA_ROOT_DIRECTORY
# Ontology: ONTOLOGY_RESOLVER, MATCHING_STRATEGY, ONTOLOGY_FILE_PATH
# Security: REQUIRE_AUTHENTICATION, ENABLE_BACKEND_ACCESS_CONTROL, HASH_API_KEY
# Observability: COGNEE_TRACING_ENABLED, OTEL_EXPORTER_OTLP_ENDPOINT, LOG_LEVEL
# Translation: TRANSLATION_PROVIDER, TARGET_LANGUAGE, CONFIDENCE_THRESHOLD
# Queue: DATASET_QUEUE_ENABLED, DATASET_QUEUE_MAX_CONCURRENT
```

---

## 10. Testing Requirements

| Layer | Location | Tool |
|-------|----------|------|
| Unit | `cognee/tests/unit/` | pytest |
| Integration | `cognee/tests/integration/` | pytest-asyncio |
| CLI | `cognee/tests/cli_tests/` | pytest |
| Tasks | `cognee/tasks/*/tests/` | pytest |
| Evaluation | `cognee/eval_framework/` | deepeval |
| Benchmarks | `evals/` | Custom |

**Quality tools:** ruff (lint), ty (type check), pre-commit, coverage

---

## 11. Extension Points

| Extension | Method |
|-----------|--------|
| Custom pipeline task | `Task(my_async_func)` |
| Custom graph model | Extend `DataPoint` |
| New graph backend | Implement `GraphDBInterface` ABC |
| New vector backend | Implement `VectorDBInterface` Protocol |
| New LLM provider | Configure via litellm (no code) |
| New document loader | Add in `cognee/infrastructure/files/` |
| New search type | Add `SearchType` enum + implement `*Retriever` |
| Community retrievers | `register_retriever("name", Class)` |

---

## 12. Traceability Matrix

| SRS Requirement | URD Requirement | PRD Section |
|----------------|-----------------|-------------|
| FR-ING-01..03 | UR-ING-01..03 | §4.1.1, §4.2, §4.3 |
| FR-PROC-01..03 | UR-PROC-01..03 | §4.1.2, §4.1.5 |
| FR-SEARCH-01..03 | UR-SEARCH-01..03 | §4.1.3 |
| FR-MEM-01..05 | UR-MEM-01..04 | §4.1.4 |
| FR-AUTH-01..03 | UR-AUTH-01..02 | §12 |
| FR-PIPE-01..02 | UR-DX-04 | §4.4 |
| §3 Interfaces | UR-DX-01..05 | §6, §13 |
| §4 Infrastructure | UR-OPS-01..04 | §5, §9 |
| §6 Observability | UR-OPS-04 | §7 |
| §8 Security | UR-AUTH-01..02 | §12 |
