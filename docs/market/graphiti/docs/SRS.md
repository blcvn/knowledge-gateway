# Software Requirements Specification (SRS)

## Graphiti — Temporal Context Graph Engine

| Field | Value |
|-------|-------|
| **Product** | Graphiti (graphiti-core v0.28.2) |
| **Owner** | Zep Software, Inc. |
| **Last Updated** | 2026-05-07 |

---

## 1. System Overview

Graphiti là engine xây dựng và truy vấn Temporal Context Graph cho AI agents, cho phép tích hợp dữ liệu episodic, trích xuất entities/facts với validity windows, và thực hiện hybrid retrieval kết hợp semantic, keyword, và graph traversal.

---

## 2. Module Architecture

### 2.1 Module Map

```
graphiti_core/
├── graphiti.py              # Core engine (Graphiti class)
├── nodes.py                 # Node types (Entity, Episodic, Community, Saga)
├── edges.py                 # Edge types (Entity, Episodic, Community, Has/NextEpisode)
├── models/                  # Pydantic schemas
├── driver/                  # Graph DB drivers (Neo4j, FalkorDB, Kuzu, Neptune)
│   ├── driver.py            # Abstract GraphDriver interface
│   ├── neo4j_driver.py
│   ├── falkordb_driver.py
│   ├── kuzu_driver.py
│   └── neptune_driver.py
├── llm_client/              # LLM provider clients
│   ├── client.py            # Abstract LLMClient
│   ├── openai_client.py
│   ├── anthropic_client.py
│   ├── gemini_client.py
│   └── groq_client.py
├── embedder/                # Embedding providers
│   ├── client.py            # Abstract EmbedderClient
│   ├── openai.py
│   ├── gemini.py
│   └── voyage.py
├── cross_encoder/           # Reranking models
│   ├── client.py            # Abstract CrossEncoderClient
│   ├── openai_client.py
│   └── bge_client.py
├── search/                  # Search engine
│   ├── search.py            # Core search orchestrator
│   ├── search_config.py     # Configuration structures
│   ├── search_config_recipes.py  # Pre-built configs
│   └── search_filters.py    # Query filters
├── utils/                   # Utilities
│   ├── maintenance/         # Graph maintenance ops
│   │   ├── node_operations.py
│   │   ├── edge_operations.py
│   │   └── graph_data_operations.py
│   └── bulk_utils.py        # Bulk processing helpers
├── prompts/                 # LLM prompt templates
├── telemetry.py             # Anonymous usage tracking
└── tracer.py                # OpenTelemetry integration
```

### 2.2 Server Modules

```
server/graph_service/
├── main.py                  # FastAPI app entrypoint
├── config.py                # Pydantic Settings
├── zep_graphiti.py          # ZepGraphiti subclass
├── routers/
│   ├── ingest.py            # Ingestion endpoints
│   └── retrieve.py          # Retrieval endpoints
└── dto/
    ├── common.py            # Shared DTOs
    ├── ingest.py            # Ingestion request/response
    └── retrieve.py          # Retrieval request/response

mcp_server/src/
└── graphiti_mcp_server.py   # MCP Server (FastMCP)
```

---

## 3. Core Components

### 3.1 Graphiti Engine (`graphiti.py`)

| Method | Signature | Mô tả |
|--------|-----------|-------|
| `__init__` | `(uri, user, password, driver, llm_client, embedder, ...)` | Khởi tạo engine |
| `build_indices_and_constraints` | `() → None` | Build DB indices |
| `add_episode` | `(name, body, source, ref_time, group_id, ...) → AddEpisodeResults` | Ingest single episode |
| `add_episode_bulk` | `(bulk_episodes, group_id) → list[AddEpisodeResults]` | Bulk ingestion |
| `add_triplet` | `(source, edge, target, group_id) → AddTripletResults` | Add S-P-O triplet |
| `search` | `(query, group_ids, num_results, ...) → SearchResults` | Hybrid search |
| `remove_episode` | `(episode_uuid) → None` | Delete episode + cascade |
| `close` | `() → None` | Cleanup resources |

**AddEpisodeResults Schema:**
```python
class AddEpisodeResults(BaseModel):
    episode: EpisodicNode
    entity_nodes: list[EntityNode]
    entity_edges: list[EntityEdge]
    community_nodes: list[CommunityNode]
```

### 3.2 Graph Driver Interface (`driver/driver.py`)

| Method | Mô tả |
|--------|-------|
| `execute_query(query, params)` | Execute Cypher/GQL query |
| `close()` | Close connection |
| `database` | Database name property |

**Implementations:**

| Driver | DB | Protocol | Notes |
|--------|----|----------|-------|
| `Neo4jDriver` | Neo4j | Bolt | Primary, full-featured |
| `FalkorDBDriver` | FalkorDB | Redis protocol | In-memory, Docker-ready |
| `KuzuDriver` | Kuzu | Embedded | File-based, no server |
| `NeptuneDriver` | Neptune | HTTP/WebSocket | AWS managed + OpenSearch |

### 3.3 LLM Client Interface (`llm_client/client.py`)

| Method | Mô tả |
|--------|-------|
| `generate_response(messages, response_model, max_tokens, model_size)` | Generate LLM response |
| `_generate_response(...)` | Abstract method per provider |
| `_clean_input(input)` | Sanitize Unicode/control chars |
| `set_tracer(tracer)` | Attach OTel tracer |

**Config (`LLMConfig`):**
```python
class LLMConfig(BaseModel):
    api_key: str | None
    model: str = "gpt-4o-mini"
    small_model: str = "gpt-4o-mini"
    temperature: float = 0
    max_tokens: int = DEFAULT_MAX_TOKENS
    base_url: str | None
```

**Features:**
- Retry with exponential backoff (max 4 attempts)
- Response caching (optional, file-based)
- Token usage tracking per prompt type
- Multilingual extraction support

### 3.4 Embedder Interface (`embedder/client.py`)

| Method | Mô tả |
|--------|-------|
| `create(input) → list[float]` | Generate single embedding |
| `create_bulk(inputs) → list[list[float]]` | Batch embeddings |

**Implementations:** OpenAI, Gemini, Voyage AI, Ollama (via OpenAI compat)

### 3.5 Cross-Encoder Interface (`cross_encoder/client.py`)

| Method | Mô tả |
|--------|-------|
| `rank(query, passages) → list[float]` | Score query-passage pairs |

**Implementations:** OpenAI (via chat), BGE (local), Gemini

### 3.6 Search Engine (`search/`)

**SearchConfig Structure:**
```python
class SearchConfig(BaseModel):
    edge_config: EdgeSearchConfig | None
    node_config: NodeSearchConfig | None
    episode_config: EpisodeSearchConfig | None
    community_config: CommunitySearchConfig | None
    limit: int = 10
```

**Search Methods (enum):**
- `cosine_similarity` — Vector search
- `bm25` — Full-text keyword
- `bfs` — Breadth-first graph traversal

**Reranker Types (enum):**
- `rrf` — Reciprocal Rank Fusion
- `mmr` — Maximal Marginal Relevance
- `cross_encoder` — Neural reranking
- `node_distance` — Graph proximity
- `episode_mentions` — Mention frequency

**SearchResults:**
```python
class SearchResults(BaseModel):
    edges: list[EntityEdge]
    nodes: list[EntityNode]
    episodes: list[EpisodicNode]
    communities: list[CommunityNode]
```

---

## 4. Data Model

### 4.1 Node Types

**EntityNode:**
```python
class EntityNode(Node):
    uuid: str
    name: str
    labels: list[str]            # ["Person", "Developer"]
    summary: str                 # LLM-generated description
    attributes: dict[str, Any]   # Custom key-value pairs
    name_embedding: list[float]  # Vector for name
    group_id: str                # Multi-tenant partition
    created_at: datetime
    updated_at: datetime
```

**EpisodicNode:**
```python
class EpisodicNode(Node):
    uuid: str
    name: str
    content: str                    # Raw episode content
    source: EpisodeType             # text|json|message|fact_triple
    source_description: str
    valid_at: datetime              # When the event occurred
    entity_edges: list[str]         # UUIDs of related EntityEdges
    episode_metadata: dict | None
    group_id: str
    created_at: datetime
```

**CommunityNode:**
```python
class CommunityNode(Node):
    uuid: str
    name: str
    summary: str              # LLM-generated cluster summary
    name_embedding: list[float]
    group_id: str
    created_at: datetime
```

### 4.2 Edge Types

**EntityEdge (RELATES_TO):**
```python
class EntityEdge(Edge):
    uuid: str
    source_node_uuid: str
    target_node_uuid: str
    name: str                 # Relationship label
    fact: str                 # Natural language fact description
    fact_embedding: list[float]
    episodes: list[str]       # Source episode UUIDs (provenance)
    valid_at: datetime | None # When fact became true
    invalid_at: datetime | None  # When fact became false
    expired_at: datetime | None  # When edge was superseded
    group_id: str
    created_at: datetime
```

### 4.3 Temporal Semantics

| Field | Meaning |
|-------|---------|
| `valid_at` | Timestamp when the fact became true in the real world |
| `invalid_at` | Timestamp when the fact ceased to be true |
| `expired_at` | System timestamp when the edge was superseded by newer info |
| `created_at` | System timestamp when the edge was created in the graph |

**Invalidation Logic:** When a new episode contradicts an existing fact, the LLM-based edge resolution marks the old edge with `expired_at` and creates a new edge with updated `valid_at`.

---

## 5. API Specifications

### 5.1 REST API (FastAPI Server)

**Ingest Router (`/v1/`):**

```
POST /v1/episodes
  Body: { name, body, source, source_description, reference_time, group_id, saga_id? }
  Response: AddEpisodeResults

POST /v1/episodes/bulk
  Body: { episodes: [...], group_id }
  Response: list[AddEpisodeResults]
```

**Retrieve Router (`/v1/`):**

```
POST /v1/search
  Body: { query, group_ids, num_results?, center_node_uuid?, search_config? }
  Response: SearchResults

GET /v1/entities/{uuid}
  Response: EntityNode

GET /v1/edges/{uuid}
  Response: EntityEdge

DELETE /v1/episodes/{uuid}
  Response: 204 No Content
```

### 5.2 MCP Protocol Tools

All tools exposed via FastMCP (`graphiti_mcp_server.py`):

| Tool | Parameters | Returns |
|------|-----------|---------|
| `add_memory` | content, source_description, group_id | episode UUID |
| `search_memory` | query, group_ids, num_results | edges + nodes |
| `get_episodes` | group_id, last_n | episode list |
| `delete_episode` | episode_uuid | success |
| `delete_entity_node` | node_uuid | success |
| `delete_entity_edge` | edge_uuid | success |
| `get_entity_edge` | edge_uuid | edge details |
| `clear_graph` | group_id | success |
| `get_status` | — | server status |

---

## 6. Infrastructure Requirements

### 6.1 Runtime

| Requirement | Specification |
|-------------|---------------|
| **Python** | ≥ 3.10, < 4.0 |
| **OS** | Linux, macOS, Windows |
| **Memory** | ≥ 4GB RAM (varies with graph size) |
| **Network** | Access to LLM APIs + Graph DB |

### 6.2 Database Backends

| Backend | Version | Port | Storage |
|---------|---------|------|---------|
| Neo4j | ≥ 5.x | 7687 (Bolt) | Disk-based |
| FalkorDB | ≥ 4.x | 6379 | In-memory + RDB |
| Kuzu | ≥ 0.5 | N/A (embedded) | File-based |
| Neptune | Latest | 8182 | AWS managed |

### 6.3 LLM Provider Requirements

| Provider | Credential | Models Used |
|----------|-----------|-------------|
| OpenAI | `OPENAI_API_KEY` | gpt-4o-mini (default), text-embedding-3-small |
| Azure OpenAI | `AZURE_OPENAI_*` | Configurable |
| Anthropic | `ANTHROPIC_API_KEY` | claude-3-5-sonnet |
| Google Gemini | `GOOGLE_API_KEY` | gemini-2.0-flash |
| Groq | `GROQ_API_KEY` | Configurable |

### 6.4 Docker Deployment

```yaml
services:
  neo4j:
    image: neo4j:5-community
    ports: ["7474:7474", "7687:7687"]
    environment:
      NEO4J_AUTH: neo4j/password

  graphiti-server:
    build: ./server
    ports: ["8000:8000"]
    environment:
      NEO4J_URI: bolt://neo4j:7687
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    depends_on: [neo4j]

  graphiti-mcp:
    build: ./mcp_server
    ports: ["8001:8001"]
    environment:
      NEO4J_URI: bolt://neo4j:7687
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    depends_on: [neo4j]
```

---

## 7. Concurrency & Performance

### 7.1 Concurrency Control

| Mechanism | Configuration | Default |
|-----------|--------------|---------|
| `SEMAPHORE_LIMIT` | Max concurrent LLM calls | 10 |
| Async I/O | All DB + LLM calls are async | — |
| Bulk batching | Configurable batch size | Sequential per episode |

### 7.2 Caching

| Cache Layer | Scope | Storage |
|-------------|-------|---------|
| LLM Response Cache | Per-prompt hash | File-based (`./llm_cache/`) |
| Embedding Cache | Built into embedder | Provider-dependent |

### 7.3 Retry Policy

- **LLM Calls:** Max 4 attempts, exponential backoff (min 5s, max 120s)
- **Retry on:** HTTP 5xx, Rate Limit (429), JSON decode errors
- **No retry on:** 4xx client errors (except 429)

---

## 8. Observability

### 8.1 Tracing (OpenTelemetry)

```python
from graphiti_core.tracer import Tracer

tracer = Tracer(service_name="graphiti", endpoint="http://jaeger:4318")
graphiti.set_tracer(tracer)
```

**Instrumented Operations:**
- `llm.generate` — LLM calls with provider, model, cache hit
- `embedder.create` — Embedding generation
- `driver.execute_query` — Database queries
- `search.*` — Search pipeline stages

### 8.2 Token Usage Tracking

```python
usage = graphiti.llm_client.token_tracker.get_usage()
# Returns: { prompt_name: { input_tokens, output_tokens, total_tokens } }
```

### 8.3 Telemetry

- Anonymous usage via PostHog
- Opt-out: set `GRAPHITI_TELEMETRY_DISABLED=true`
- Tracks: ingestion counts, search patterns, backend type

---

## 9. Security Considerations

| Aspect | Implementation |
|--------|---------------|
| **Data Isolation** | `group_id` partitioning — search filters by group |
| **Credential Management** | Environment variables, no hard-coded secrets |
| **Input Sanitization** | Unicode cleaning, control character removal |
| **PII Protection** | Truncated logging (no full message content in logs) |
| **Network Security** | TLS for all external API calls (LLM, DB) |

---

## 10. Dependencies

### Core Dependencies
```
pydantic >= 2.0
neo4j (default driver)
openai (default LLM + embedder)
numpy
tenacity (retry logic)
httpx (HTTP client)
diskcache (LLM caching)
```

### Optional Extras
```
[anthropic]  → anthropic SDK
[gemini]     → google-genai
[groq]       → groq SDK
[voyage]     → voyageai SDK
[falkordb]   → falkordb, redis
[kuzu]       → kuzu
[neptune]    → boto3, opensearch-py
[ollama]     → (uses openai compat)
```

---

## 11. Testing Requirements

| Category | Framework | Scope |
|----------|-----------|-------|
| Unit Tests | pytest + pytest-asyncio | Models, utils, search logic |
| Integration Tests | pytest | Full pipeline with live DB |
| Parallel Execution | pytest-xdist | `-n auto` for CI |
| Code Quality | Ruff | E, F, UP, B, SIM, I rules |
| Type Checking | Pyright | Basic mode |
| Coverage | pytest-cov | Minimum 80% target |
