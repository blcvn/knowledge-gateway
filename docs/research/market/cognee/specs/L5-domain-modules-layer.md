# L5 — Domain Modules Layer

> **Layer**: 5 (Domain Logic)  
> **Responsibility**: Business logic, domain models, and strategy implementations  
> **Dependencies**: L6 (Infrastructure Adapters)  
> **Path**: `cognee/modules/` (trừ `pipelines/`)

---

## 1. Tổng Quan

Layer 5 chứa **domain logic** — các module chuyên biệt thực hiện các chiến lược truy xuất, chunking, quản lý người dùng, ontology, và agent memory. Đây là tầng "thông minh" của hệ thống.

```
modules/
├── retrieval/           # Search retriever implementations
├── search/              # Search type registry + dispatch
├── chunking/            # Text splitting strategies
├── ontology/            # OWL ontology resolver
├── users/               # Auth, roles, permissions, ACLs
├── data/                # Dataset + Data management
├── cognify/             # Cognify configuration
├── memify/              # Graph enrichment
├── agent_memory/        # Runtime memory injection for agents
├── observability/       # OpenTelemetry tracing
├── session_lifecycle/   # Session tracking + metrics
├── graph/               # Graph operations + models
├── engine/              # Engine setup operations
├── ingestion/           # Data classification + identification
├── storage/             # Storage utilities
├── visualization/       # Network graph visualization
├── settings/            # Runtime settings management
├── sync/                # Data synchronization
├── cloud/               # Cloud service operations
├── metrics/             # System metrics
├── notebooks/           # Notebook integrations
├── run_custom_pipeline/ # Custom pipeline runner
└── pipelines/           # → L3 (excluded from L5)
```

---

## 2. Retrieval Module (`modules/retrieval/`)

### 2.1 Architecture

```
BaseRetriever (ABC)
  ├── GraphCompletionRetriever
  ├── GraphCompletionCotRetriever        (Chain-of-Thought)
  ├── GraphCompletionDecompositionRetriever
  ├── GraphCompletionContextExtensionRetriever
  ├── GraphSummaryCompletionRetriever
  ├── TripletRetriever
  ├── CompletionRetriever                (RAG)
  ├── ChunksRetriever
  ├── LexicalRetriever                   (Jaccard)
  ├── SummariesRetriever
  ├── CypherSearchRetriever
  ├── NaturalLanguageRetriever
  ├── TemporalRetriever
  └── CodingRulesRetriever
```

### 2.2 BaseRetriever Pipeline (3-step)

```python
class BaseRetriever(ABC):
    async def get_retrieved_objects(query, query_batch)   # Step 1: Fetch raw data
    async def get_context_from_objects(query, objects)     # Step 2: Format for LLM
    async def get_completion_from_context(query, context)  # Step 3: LLM completion
    
    # Convenience method
    async def get_completion(query):  # Runs all 3 steps
```

### 2.3 Key Retrievers

| Retriever | File | Kích thước | Đặc điểm |
|-----------|------|-----------|----------|
| `GraphCompletionRetriever` | `graph_completion_retriever.py` | 14KB | Vector search → graph traversal → LLM |
| `GraphCompletionCotRetriever` | `graph_completion_cot_retriever.py` | 12KB | Multi-step chain-of-thought reasoning |
| `GraphCompletionDecompositionRetriever` | `graph_completion_decomposition_retriever.py` | 10KB | Query decomposition → parallel search |
| `TripletRetriever` | `triplet_retriever.py` | 6.5KB | Subject-predicate-object triplet matching |
| `CompletionRetriever` | `completion_retriever.py` | 6.3KB | Traditional RAG |
| `NaturalLanguageRetriever` | `natural_language_retriever.py` | 7KB | NL → structured query |
| `TemporalRetriever` | `temporal_retriever.py` | 7KB | Time-aware graph traversal |

### 2.4 Context Providers (`modules/retrieval/context_providers/`)

Cung cấp context từ nhiều nguồn cho retrievers (e.g., `TripletSearchContextProvider`).

### 2.5 Entity Extractors (`modules/retrieval/entity_extractors/`)

Extract entities từ queries cho graph lookup.

### 2.6 Community Retrievers

Extension point cho custom retrievers:

```python
from cognee.modules.retrieval import register_retriever

register_retriever("my_retriever", MyCustomRetriever)
```

---

## 3. Search Module (`modules/search/`)

### 3.1 Structure

```
search/
├── types/         # SearchType enum, SearchResult model
├── methods/       # search() dispatcher
├── operations/    # Search operations
├── models/        # Search data models
├── utils/         # Search utilities
└── exceptions/    # Search-specific errors
```

### 3.2 Dispatcher

`search/methods/search()` maps `SearchType` → `*Retriever` instance → gọi retriever.

---

## 4. Chunking Module (`modules/chunking/`)

### 4.1 Chunker Implementations

| Chunker | Strategy | Use case |
|---------|----------|----------|
| `TextChunker` | Paragraph-based | Default, most reliable |
| `LangchainChunker` | Recursive character splitting with overlap | LangChain compatibility |
| `CsvChunker` | Row-based splitting | Tabular data |
| `text_chunker_with_overlap` | Sliding window with overlap | Fine-grained context |

### 4.2 Chunker Interface

```python
class Chunker(ABC):
    def chunk(self, text: str) -> List[Chunk]:
        ...
```

### 4.3 Chunk Models (`chunking/models/`)

Data models cho chunk metadata: text, position, source, tokens.

---

## 5. Ontology Module (`modules/ontology/`)

### 5.1 Architecture

```
ontology/
├── base_ontology_resolver.py     # ABC for ontology resolvers
├── get_default_ontology_resolver.py  # Factory
├── matching_strategies.py        # Fuzzy + exact matching
├── models.py                     # Ontology data models
├── ontology_config.py           # Config type aliases
├── ontology_env_config.py       # Environment config
└── rdf_xml/                     # RDFLib-based OWL resolver
```

### 5.2 Configuration

```bash
ONTOLOGY_RESOLVER=rdflib
MATCHING_STRATEGY=fuzzy      # 80% similarity threshold
ONTOLOGY_FILE_PATH=/path/to/ontology.owl
```

### 5.3 Matching Strategies

| Strategy | Behavior |
|----------|----------|
| `exact` | Exact string match |
| `fuzzy` | Fuzzy matching with 80% similarity threshold |

---

## 6. Users & Permissions Module (`modules/users/`)

### 6.1 Structure

```
users/
├── models/          # User, Role models
├── methods/         # get_default_user, get_user
├── authentication/  # Auth strategies
├── permissions/     # Permission models + methods
├── roles/           # Role hierarchy
├── tenants/         # Multi-tenant support
├── api_key/         # API key management
├── get_fastapi_users.py   # FastAPI Users integration
├── get_user_db.py         # User DB adapter
└── get_user_manager.py    # User CRUD manager
```

### 6.2 Data Hierarchy

```
Tenant (optional)
  └─ User
       └─ Dataset (many)
            └─ Data (many)
                 └─ Graph/Vector indices
```

### 6.3 Permissions

| Permission | Scope |
|------------|-------|
| `read` | View dataset data |
| `write` | Add data to dataset |
| `delete` | Remove dataset data |
| `share` | Grant permissions to others |

### 6.4 Multi-Tenant Isolation

Khi `ENABLE_BACKEND_ACCESS_CONTROL=True`:
- Mỗi user+dataset → isolated graph + vector DB instances
- Supported backends: Kuzu, LanceDB, SQLite, Postgres
- `context_global_variables.py` — ContextVar-based DB config switching

---

## 7. Agent Memory Module (`modules/agent_memory/`)

### 7.1 Architecture

```
agent_memory/
├── __init__.py      # Public export: agent_memory decorator
├── decorator.py     # @agent_memory() decorator definition
├── runtime.py       # AgentMemoryContext, persist_trace, retrieve_memory
└── sanitization.py  # Input sanitization utilities
```

### 7.2 Decorator Pattern

```python
@agent_memory(with_memory=True, session_id="chat_1")
async def my_agent_function(query: str):
    ...  # LLMGateway auto-prepends relevant memory context
```

### 7.3 Runtime Flow

1. `resolve_agent_user()` — get/create user
2. `resolve_agent_dataset_scope()` — resolve dataset context
3. `set_current_agent_memory_context()` — set ContextVar
4. `retrieve_memory_context()` — query Cognee for relevant context
5. Execute wrapped function
6. `persist_trace()` — save execution trace for learning

### 7.4 Key Parameters

| Parameter | Type | Default | Mô tả |
|-----------|------|---------|--------|
| `with_memory` | bool | True | Enable memory injection |
| `with_session_memory` | bool | False | Include session history |
| `save_session_traces` | bool | False | Persist traces |
| `memory_query_fixed` | str | None | Fixed query for memory retrieval |
| `memory_query_from_method` | str | None | Extract query from method param |
| `memory_top_k` | int | 5 | Number of memory results |
| `session_memory_last_n` | int | 5 | Recent session entries |
| `persist_session_trace_after` | int | None | Auto-persist threshold |

---

## 8. Observability Module (`modules/observability/`)

### 8.1 Architecture

```
observability/
├── __init__.py         # Public exports: new_span, attribute constants
├── tracing.py          # OpenTelemetry span creation + management
├── trace_context.py    # enable_tracing(), get_last_trace()
├── get_observe.py      # Observer factory
└── observers.py        # Observer registry
```

### 8.2 Span Attributes

| Constant | Attribute Key |
|----------|--------------|
| `COGNEE_SEARCH_QUERY` | `cognee.search.query` |
| `COGNEE_SEARCH_TYPE` | `cognee.search.type` |
| `COGNEE_LLM_MODEL` | `cognee.llm.model` |
| `COGNEE_RESULT_COUNT` | `cognee.result.count` |
| `COGNEE_RESULT_SUMMARY` | `cognee.result.summary` |
| `COGNEE_PIPELINE_NAME` | `cognee.pipeline.name` |
| `COGNEE_SESSION_ID` | `cognee.session.id` |

### 8.3 Integration Points

- **Langfuse** — via `monitoring` extra
- **Sentry** — error tracking
- **PostHog** — analytics
- **structlog** — structured logging

---

## 9. Session Lifecycle Module (`modules/session_lifecycle/`)

```
session_lifecycle/
├── __init__.py       # SessionModelUsage, SessionRecord exports
├── models.py         # SessionRecord, SessionModelUsage models
├── metrics.py        # Session metrics tracking (16KB)
└── usage_tracking.py # Usage/billing tracking
```

---

## 10. Data Module (`modules/data/`)

```
data/
├── models/       # Dataset, Data models
├── methods/      # CRUD operations, authorization
├── processing/   # Data processing utilities
├── deletion/     # Data deletion logic
└── exceptions/   # DatasetNotFoundError, etc.
```

---

## 11. Visualization Module (`modules/visualization/`)

`cognee_network_visualization.py` (72KB) — comprehensive network graph visualization engine.

---

## 12. Other Modules

| Module | Path | Chức năng |
|--------|------|-----------|
| `cognify/` | `modules/cognify/config.py` | Cognify config: triplet_embedding, chunks_per_batch |
| `memify/` | `modules/memify/memify.py` | Graph enrichment logic |
| `graph/` | `modules/graph/` | Graph operations, legacy utilities |
| `engine/` | `modules/engine/operations/` | DB setup operations |
| `ingestion/` | `modules/ingestion/` | Data classification, identification |
| `storage/` | `modules/storage/utils/` | Storage utility functions |
| `settings/` | `modules/settings/` | Runtime settings CRUD |
| `sync/` | `modules/sync/` | Data sync between sources |
| `cloud/` | `modules/cloud/` | Cloud operations |
| `metrics/` | `modules/metrics/` | System metrics |
| `run_custom_pipeline/` | `modules/run_custom_pipeline/` | Custom pipeline execution |
