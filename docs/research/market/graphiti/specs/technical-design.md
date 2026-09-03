# Graphiti — Technical Design Document

**Version:** 1.0  
**Date:** 2026-05-02  
**Source:** getzep/graphiti (Apache 2.0)  
**Maintainer:** Zep Software, Inc.

---

## 1. Overview

Graphiti is an open-source Python framework for building and querying **temporal context graphs** for AI agents. Unlike static knowledge graphs or traditional RAG pipelines, Graphiti continuously integrates unstructured and structured data into a queryable graph where every fact carries an explicit **validity window** — a `valid_at` / `invalid_at` timestamp pair that models when the fact was true, not just when it was recorded.

The system is designed to answer: *"What did the agent know, and when did it know it?"*

Key capabilities:
- **Temporal fact management** — old facts are invalidated (not deleted) when superseded.
- **Provenance** — every entity and relationship traces back to the raw episode that produced it.
- **Prescribed & learned ontology** — entity/edge types via Pydantic models, or free-form extraction.
- **Incremental construction** — new episodes integrate immediately without full graph recomputation.
- **Hybrid retrieval** — cosine similarity (ANN) + BM25 fulltext + graph BFS, reranked by RRF / MMR / cross-encoder.
- **Pluggable backends** — Neo4j, FalkorDB, Kuzu, Amazon Neptune (+ OpenSearch).

---

## 2. High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                          Public Interfaces                                │
│  ┌──────────────┐   ┌──────────────────┐   ┌──────────────────────────┐ │
│  │  Python SDK  │   │  REST Server     │   │  MCP Server              │ │
│  │  (graphiti)  │   │  (FastAPI/server)│   │  (mcp_server/)           │ │
│  └──────┬───────┘   └────────┬─────────┘   └──────────┬───────────────┘ │
└─────────┼────────────────────┼──────────────────────── ┼────────────────┘
          │                    │                          │
          └────────────────────┴──────────────────────────┘
                                     │
                    ┌────────────────▼─────────────────┐
                    │         Graphiti Core Engine      │
                    │         graphiti_core/graphiti.py │
                    │                                   │
                    │  add_episode()  search()          │
                    │  add_episode_bulk()  search_()    │
                    │  add_triplet()  build_communities()│
                    └──┬──────────┬──────────┬──────────┘
                       │          │          │
          ┌────────────▼──┐  ┌───▼───┐  ┌──▼──────────┐
          │  LLM Gateway  │  │Embedder│  │ Cross-Encoder│
          │  (llm_client/)│  │(embed/)│  │(cross_encoder│
          └───────────────┘  └───────┘  └─────────────┘
                       │
          ┌────────────▼─────────────────────────┐
          │         Graph Driver Layer            │
          │  GraphDriver (ABC)                    │
          │  ┌──────────┐ ┌─────────┐ ┌────────┐ │
          │  │Neo4j     │ │FalkorDB │ │Kuzu    │ │
          │  │Driver    │ │Driver   │ │Driver  │ │
          │  └──────────┘ └─────────┘ └────────┘ │
          │  ┌──────────────────────┐             │
          │  │  Neptune Driver      │             │
          │  └──────────────────────┘             │
          └──────────────────────────────────────┘
```

---

## 3. Graph Data Model

### 3.1 Node Types

| Class | Graph Label | Purpose |
|---|---|---|
| `EpisodicNode` | `Episodic` | Raw ingested data (ground truth). Carries `source`, `content`, `valid_at`, `entity_edges[]`. |
| `EntityNode` | `Entity` | Named entity with evolving `summary` and `name_embedding`. Optionally typed via labels. |
| `CommunityNode` | `Community` | Cluster of co-occurring entities; carries aggregate `summary` and `name_embedding`. |
| `SagaNode` | `Saga` | Named sequence of episodes (e.g. a user conversation); tracks `first_episode_uuid`, `last_episode_uuid`, incremental `summary`. |

All nodes share: `uuid`, `name`, `group_id` (partition key), `labels[]`, `created_at`.

### 3.2 Edge Types

| Class | Relationship | Connects | Purpose |
|---|---|---|---|
| `EpisodicEdge` | `MENTIONS` | Episodic → Entity | Links a raw episode to every entity it mentions. |
| `EntityEdge` | `RELATES_TO` | Entity → Entity | Temporal fact triple. Carries `fact`, `fact_embedding`, `valid_at`, `invalid_at`, `expired_at`. |
| `CommunityEdge` | `HAS_MEMBER` | Community → Entity | Membership of entity in a community. |
| `HasEpisodeEdge` | `HAS_EPISODE` | Saga → Episodic | Links a saga to its constituent episodes. |
| `NextEpisodeEdge` | `NEXT_EPISODE` | Episodic → Episodic | Ordered chain of episodes within a saga. |

### 3.3 Temporal Model

`EntityEdge` implements **bi-temporal tracking**:

```
valid_at      — when the fact became true in the real world
invalid_at    — when the fact stopped being true (set during conflict resolution)
expired_at    — when the record was logically removed from the active set
reference_time— timestamp from the source episode
```

When a new episode contradicts an existing fact, the old `EntityEdge.invalid_at` is set to the new episode's `reference_time`. Old facts are **never deleted**; point-in-time queries are possible via `valid_at` filters.

### 3.4 Partitioning

`group_id` is the logical partition key applied to every node and edge. It maps to a database/graph name in FalkorDB and Kuzu, and acts as a filter property in Neo4j and Neptune. This enables multi-tenant isolation with a single driver instance.

---

## 4. Core Engine — `graphiti_core`

### 4.1 Module Map

```
graphiti_core/
├── graphiti.py           # Public Graphiti class — all top-level APIs
├── graphiti_types.py     # GraphitiClients dataclass (driver, llm, embedder, cross_encoder, tracer)
├── nodes.py              # Node models (EpisodicNode, EntityNode, CommunityNode, SagaNode)
├── edges.py              # Edge models (EpisodicEdge, EntityEdge, CommunityEdge, …)
├── driver/               # Graph driver abstraction + implementations
├── llm_client/           # LLM provider adapters
├── embedder/             # Embedding provider adapters
├── cross_encoder/        # Reranker adapters
├── search/               # Search pipeline (config, recipes, filters, execution)
├── prompts/              # All LLM prompt templates
├── utils/
│   ├── maintenance/      # Node/edge extraction, dedup, community ops
│   ├── ontology_utils/   # Custom entity-type validation
│   ├── bulk_utils.py     # Batch episode ingestion helpers
│   └── content_chunking.py
├── namespaces.py         # Namespace API (graphiti.nodes.entity.save())
├── migrations/           # Schema migration scripts
├── telemetry/            # Anonymous PostHog telemetry
└── tracer.py             # OpenTelemetry integration
```

### 4.2 `Graphiti` Class

The `Graphiti` class (in `graphiti.py`, ~1741 lines) is the single entry point for all operations. Constructor signature:

```python
Graphiti(
    uri, user, password,          # Neo4j credentials (if no graph_driver)
    llm_client,                   # LLMClient (default: OpenAIClient)
    embedder,                     # EmbedderClient (default: OpenAIEmbedder)
    cross_encoder,                # CrossEncoderClient (default: OpenAIRerankerClient)
    store_raw_episode_content,    # bool — whether to persist raw episode text
    graph_driver,                 # GraphDriver override
    max_coroutines,               # Semaphore limit (env: SEMAPHORE_LIMIT, default 10)
    tracer,                       # OpenTelemetry tracer
    trace_span_prefix,            # Span name prefix
)
```

On init, all clients are assembled into `GraphitiClients`, and `NodeNamespace` / `EdgeNamespace` objects are attached as `graphiti.nodes` / `graphiti.edges`.

### 4.3 Public API Methods

| Method | Description |
|---|---|
| `add_episode(...)` | Ingest a single episode: extract nodes/edges → resolve → save. |
| `add_episode_bulk(...)` | Batch ingestion: extract all → dedupe in-memory → resolve → save. |
| `add_triplet(source, edge, target)` | Directly add a (Node, Edge, Node) triple with conflict resolution. |
| `search(query, ...)` | Basic hybrid edge search (BM25 + cosine + RRF). |
| `search_(query, config, ...)` | Advanced search returning `SearchResults` with configurable recipes. |
| `retrieve_episodes(...)` | Fetch recent episodic nodes, optionally filtered by saga or source type. |
| `build_communities(...)` | Run community detection → create/update `CommunityNode`s. |
| `summarize_saga(saga_id)` | Incrementally summarise saga episodes using LLM. |
| `build_indices_and_constraints(...)` | Initialize DB indices/vector indexes. |
| `remove_episode(uuid)` | Delete episode and orphaned nodes/edges it created. |
| `get_nodes_and_edges_by_episode(uuids)` | Retrieve graph objects linked to given episodes. |

---

## 5. Episode Ingestion Pipeline

### 5.1 Single Episode Flow (`add_episode`)

```
Input: name, episode_body, reference_time, source, group_id, entity_types, edge_types
  │
  ├─► Retrieve previous episodes (context window, default 10)
  │
  ├─► LLM: extract_nodes()
  │     └─ Prompt: extract_nodes.py — outputs list of {name, label, summary}
  │
  ├─► resolve_extracted_nodes()
  │     ├─ Semantic search for existing entities (cosine + BM25)
  │     └─ LLM: dedupe_nodes.py — merge or keep as new
  │
  ├─► LLM: extract_edges()  (parallel with node attribute extraction)
  │     └─ Prompt: extract_edges.py — outputs fact triples
  │
  ├─► resolve_extracted_edges()
  │     ├─ Match existing edges by semantic similarity
  │     └─ LLM: dedupe_edges.py — resolve contradictions, set invalid_at
  │
  ├─► extract_attributes_from_nodes()
  │     └─ LLM: summarize_nodes.py — update entity summaries
  │
  ├─► add_nodes_and_edges_bulk()
  │     ├─ Generate embeddings (name_embedding, fact_embedding)
  │     └─ Upsert to graph DB
  │
  └─► Optional: update_community(), saga association
```

All LLM calls in the pipeline use **structured output** (Pydantic `response_model`), enforcing schema compliance. The pipeline is fully `async` and uses `semaphore_gather()` to bound concurrency via `SEMAPHORE_LIMIT`.

### 5.2 Bulk Episode Flow (`add_episode_bulk`)

The bulk path processes a list of `RawEpisode` objects in a single pass:

1. **Parallel extraction** — `extract_nodes_and_edges_bulk()` runs all episodes concurrently.
2. **In-memory deduplication** — `dedupe_nodes_bulk()` resolves duplicates across the batch before any DB write.
3. **Graph resolution** — `resolve_extracted_nodes/edges()` per episode against the existing graph.
4. **Bulk save** — single `add_nodes_and_edges_bulk()` call for all nodes and edges.

The bulk path intentionally skips community updates (communities must be rebuilt separately via `build_communities()`).

### 5.3 Conflict Resolution

When a new edge contradicts an existing edge (detected by LLM in `dedupe_edges.py`):

- The old edge's `invalid_at` is set to the new episode's `reference_time`.
- Both edges are persisted; queries can filter by `invalid_at IS NULL` for current facts.
- The new edge's `valid_at` reflects when the new fact became true.

---

## 6. Driver Layer

### 6.1 Abstraction

`GraphDriver` (ABC in `driver/driver.py`) defines the contract:

```python
class GraphDriver(QueryExecutor, ABC):
    provider: GraphProvider           # NEO4J | FALKORDB | KUZU | NEPTUNE
    fulltext_syntax: str              # Provider-specific BM25 query prefix
    default_group_id: str
    search_interface: SearchInterface | None
    graph_operations_interface: GraphOperationsInterface | None

    # Core methods
    execute_query(cypher_query, **kwargs) -> Coroutine
    session(database) -> GraphDriverSession
    build_indices_and_constraints(delete_existing)
    transaction() -> AsyncContextManager[Transaction]

    # Operation property accessors (return None by default; overridden by drivers)
    entity_node_ops, episode_node_ops, community_node_ops, saga_node_ops
    entity_edge_ops, episodic_edge_ops, community_edge_ops
    search_ops, graph_ops
```

### 6.2 Implementations

| Driver | File | Notes |
|---|---|---|
| `Neo4jDriver` | `neo4j_driver.py` | Default. Uses neo4j async driver. Full Cypher + vector index support. |
| `FalkorDriver` | `falkordb_driver.py` | Redis-compatible graph DB. Custom fulltext syntax prefix. |
| `KuzuDriver` | `kuzu_driver.py` | Embedded columnar graph DB. Edges modelled as intermediate `RelatesToNode_` nodes. |
| `NeptuneDriver` | `neptune_driver.py` | AWS managed. Uses OpenSearch Serverless (AOSS) for fulltext. Custom embedding serialization (CSV string). |

### 6.3 IoC Pattern

Each node/edge class first checks `driver.graph_operations_interface` before falling back to inline Cypher. This allows drivers to provide optimised native implementations while keeping a universal Cypher fallback — a key extensibility point for new backends.

```python
async def save(self, driver: GraphDriver):
    if driver.graph_operations_interface:
        try:
            return await driver.graph_operations_interface.node_save(self, driver)
        except NotImplementedError:
            pass
    # Fallback: inline Cypher
    ...
```

---

## 7. LLM Client Layer

### 7.1 Interface

```python
class LLMClient(ABC):
    async def generate_response(
        self,
        messages: list[Message],
        response_model: type[BaseModel] | None,
        prompt_name: str,
    ) -> dict
```

The `token_tracker` property exposes per-prompt-type usage statistics.

### 7.2 Implementations

| Class | Provider |
|---|---|
| `OpenAIClient` | OpenAI (GPT-4o default, gpt-4o-mini small model) |
| `OpenAIGenericClient` | Any OpenAI-compatible endpoint (Ollama, LM Studio) |
| `AzureOpenAILLMClient` | Azure OpenAI via v1 compatibility endpoint |
| `AnthropicClient` | Anthropic Claude |
| `GeminiClient` | Google Gemini |
| `GroqClient` | Groq |
| `GLiNER2Client` | Local NER via GLiNER2 (entity extraction only) |

LLM config lives in `LLMConfig`: `model`, `small_model`, `base_url`, `api_key`, `max_tokens`, `temperature`.

The `small_model` field allows selecting a cheaper model for classification subtasks (e.g., reranking).

### 7.3 Prompt Library

All prompts are in `graphiti_core/prompts/`. Each module returns a `list[Message]` for the LLM call:

| Module | Purpose |
|---|---|
| `extract_nodes.py` | Entity extraction from episode text |
| `extract_edges.py` | Fact triple extraction |
| `extract_nodes_and_edges.py` | Combined extraction (bulk path) |
| `dedupe_nodes.py` | Merge/disambiguate entity candidates |
| `dedupe_edges.py` | Conflict resolution for edges |
| `summarize_nodes.py` | Update entity summary from new edges |
| `summarize_sagas.py` | Incremental saga summarization |
| `eval.py` | Evaluation/scoring prompts |

`prompt_library` (in `lib.py`) is the central registry accessed as `prompt_library.<module>.<function>(context)`.

---

## 8. Embedder & Cross-Encoder Layer

### 8.1 Embedder

```python
class EmbedderClient(ABC):
    async def create(self, input_data: list[str]) -> list[float]
```

Implementations: `OpenAIEmbedder`, `AzureOpenAIEmbedderClient`, `GeminiEmbedder`, `VoyageAIEmbedder`.

Embeddings are generated for:
- `EntityNode.name_embedding` (768–3072 dims depending on model)
- `EntityEdge.fact_embedding`
- `CommunityNode.name_embedding`

### 8.2 Cross-Encoder (Reranker)

```python
class CrossEncoderClient(ABC):
    async def rank(self, query: str, passages: list[str]) -> list[float]
```

Implementations: `OpenAIRerankerClient` (log-prob classification), `GeminiRerankerClient`, `LocalCrossEncoderClient` (sentence-transformers).

---

## 9. Search Pipeline

### 9.1 Search Configuration

`SearchConfig` composes four optional sub-configs:

```python
SearchConfig(
    edge_config   = EdgeSearchConfig(search_methods, reranker, sim_min_score, mmr_lambda, bfs_max_depth),
    node_config   = NodeSearchConfig(...),
    episode_config= EpisodeSearchConfig(...),
    community_config = CommunitySearchConfig(...),
    limit         = 10,
    reranker_min_score = 0.0,
)
```

**Search methods** per entity type:

| Method | Description |
|---|---|
| `cosine_similarity` | ANN over stored embeddings |
| `bm25` | Fulltext (provider-specific Lucene syntax) |
| `breadth_first_search` | Graph traversal from matched nodes |

**Rerankers:**

| Reranker | Description |
|---|---|
| `rrf` | Reciprocal Rank Fusion of multiple result lists |
| `node_distance` | Boost edges closer (in hops) to a `center_node_uuid` |
| `episode_mentions` | Boost edges mentioned in more recent episodes |
| `mmr` | Maximum Marginal Relevance for diversity |
| `cross_encoder` | Neural reranking via `CrossEncoderClient` |

### 9.2 Built-in Recipes

`search_config_recipes.py` provides ready-made configs:

| Constant | Strategy |
|---|---|
| `EDGE_HYBRID_SEARCH_RRF` | BM25 + cosine, RRF rerank (default `search()`) |
| `EDGE_HYBRID_SEARCH_NODE_DISTANCE` | BM25 + cosine, node-distance rerank |
| `EDGE_HYBRID_SEARCH_CROSS_ENCODER` | BM25 + cosine + BFS, cross-encoder rerank |
| `COMBINED_HYBRID_SEARCH_CROSS_ENCODER` | All entity types, cross-encoder (default `search_()`) |
| `NODE_HYBRID_SEARCH_*` | Node-focused variants |
| `COMMUNITY_HYBRID_SEARCH_*` | Community-focused variants |

### 9.3 Search Filters

`SearchFilters` allows fine-grained result scoping:
- `node_uuids`, `edge_uuids` — restrict to specific graph objects
- `group_ids` — partition filter
- `valid_at` / `invalid_at` — temporal window filter
- `entity_labels` — filter by entity type labels

---

## 10. Community Detection

Communities are higher-level clusters of co-occurring entities. The algorithm (`utils/maintenance/community_operations.py`):

1. Retrieve all `EntityNode`s and their connections.
2. Apply a graph clustering algorithm to group densely connected entities.
3. For each cluster, create a `CommunityNode` with an LLM-generated summary.
4. Persist `CommunityEdge` (`HAS_MEMBER`) from community to each member.
5. Generate and store `name_embedding` for each `CommunityNode`.

Community nodes are searchable via `CommunitySearchMethod.cosine_similarity` and `bm25`, enabling broad topic-level retrieval.

---

## 11. Saga Management

A `Saga` is a named, ordered sequence of episodes (e.g. a user session, a document processing workflow).

```
SagaNode ──HAS_EPISODE──► EpisodicNode₁ ──NEXT_EPISODE──► EpisodicNode₂ ──► ...
```

Key operations:
- `add_episode(..., saga="conversation-42")` — auto-creates saga on first use; links episode via `HAS_EPISODE` and threads `NEXT_EPISODE` edges.
- `summarize_saga(saga_id)` — incremental LLM summary; only processes episodes added since `last_summarized_at`.
- `saga_previous_episode_uuid` param — optimization to avoid DB lookup when adding multiple episodes to the same saga in sequence.

---

## 12. REST Server (`server/`)

A lightweight **FastAPI** service wrapping the Graphiti core, intended for deployment scenarios where the Python SDK cannot be used directly.

### Routers

| Router | Endpoints |
|---|---|
| `ingest.py` | `POST /messages` (async queue), `POST /entity-node`, `DELETE /entity-edge/{uuid}`, `DELETE /group/{group_id}`, `DELETE /episode/{uuid}`, `POST /clear` |
| `retrieve.py` | `GET /search`, `GET /episodes` |

The ingest router uses an `AsyncWorker` (asyncio queue + single consumer task) to serialize episode processing — critical for maintaining graph consistency since episodes must be processed sequentially.

### Configuration

`ZepGraphitiDep` (FastAPI `Depends`) provides a shared `Graphiti` instance per request. Environment variables: `NEO4J_URI`, `NEO4J_USER`, `NEO4J_PASSWORD`, `OPENAI_API_KEY`, `PORT`, `db_backend`.

---

## 13. MCP Server (`mcp_server/`)

A **Model Context Protocol** server exposing Graphiti capabilities to MCP-compatible AI clients (Claude, Cursor, etc.).

### Capabilities exposed as MCP tools

- Episode add/retrieve/delete
- Entity node management
- Hybrid search
- Group management
- Graph maintenance (clear, rebuild indices)

The MCP server is independently deployable via Docker + Neo4j (see `mcp_server/docker/`).

---

## 14. Observability

### 14.1 OpenTelemetry Tracing

`Tracer` (in `tracer.py`) wraps an OpenTelemetry `tracer` instance. Usage:

```python
with self.tracer.start_span('add_episode') as span:
    span.add_attributes({...})
    span.set_status('error', str(e))
    span.record_exception(e)
```

Configured via standard OTEL environment variables (see `OTEL_TRACING.md`). No-op by default if no tracer is provided.

### 14.2 Telemetry

Anonymous PostHog telemetry on `Graphiti.__init__()`. Collects: OS, Python version, Graphiti version, LLM/embedder/database provider types. No personal data, queries, or graph content. Disable: `GRAPHITI_TELEMETRY_ENABLED=false`.

### 14.3 Token Tracking

`TokenUsageTracker` (in `llm_client/token_tracker.py`) aggregates token counts per prompt type. Access via `graphiti.token_tracker.get_usage()`.

---

## 15. Deployment

### 15.1 Docker Compose (Default — Neo4j)

```yaml
services:
  graph:    # Graphiti REST server (port 8000)
  neo4j:    # Neo4j 5.26.2 (ports 7474/7687, volume neo4j_data)
```

### 15.2 Docker Compose (FalkorDB profile)

```yaml
services:
  graph-falkordb:   # Graphiti server (port 8001)
  falkordb:         # FalkorDB (port 6379, volume falkordb_data)
```

Activated with: `docker compose --profile falkordb up`

### 15.3 Environment Variables

| Variable | Default | Description |
|---|---|---|
| `OPENAI_API_KEY` | — | Required for default LLM/embedder |
| `NEO4J_URI` | `bolt://localhost:7687` | Graph DB connection |
| `NEO4J_USER` | `neo4j` | DB user |
| `NEO4J_PASSWORD` | `password` | DB password |
| `SEMAPHORE_LIMIT` | `10` | Max concurrent LLM/embedder calls |
| `GRAPHITI_TELEMETRY_ENABLED` | `true` | Opt-out telemetry |
| `ENTITY_INDEX_NAME` | `entities` | Vector index name |
| `EPISODE_INDEX_NAME` | `episodes` | |
| `COMMUNITY_INDEX_NAME` | `communities` | |
| `ENTITY_EDGE_INDEX_NAME` | `entity_edges` | |

### 15.4 Python Package

```bash
pip install graphiti-core                          # Core (Neo4j)
pip install graphiti-core[falkordb]                # + FalkorDB
pip install graphiti-core[kuzu]                    # + Kuzu
pip install graphiti-core[neptune]                 # + Amazon Neptune
pip install graphiti-core[anthropic,google-genai]  # + LLM providers
```

---

## 16. Extension Points

### Adding a New Graph Backend

1. Create `driver/<name>_driver.py` implementing `GraphDriver`.
2. Add `GraphProvider.<NAME>` to the enum.
3. Implement `execute_query`, `session`, `build_indices_and_constraints`, `delete_all_indexes`.
4. Optionally implement `GraphOperationsInterface` for optimized per-entity-type operations.
5. Handle provider-specific branching in `nodes.py` and `edges.py` `save()` methods.

### Adding a New LLM Provider

1. Create `llm_client/<name>_client.py` extending `LLMClient`.
2. Implement `generate_response(messages, response_model, prompt_name)`.
3. Use `LLMConfig` for configuration.

### Custom Ontology

```python
from pydantic import BaseModel
from graphiti_core.nodes import EpisodeType

class Person(BaseModel):
    age: int | None = None
    occupation: str | None = None

class WorksFor(BaseModel):
    role: str | None = None

await graphiti.add_episode(
    ...,
    entity_types={"Person": Person},
    edge_types={"WORKS_FOR": WorksFor},
    edge_type_map={("Person", "Person"): ["WORKS_FOR"]},
)
```

---

## 17. Key Design Decisions

| Decision | Rationale |
|---|---|
| **Async-first** | All I/O (DB, LLM, embedder) is `async`; `semaphore_gather` manages concurrency at scale. |
| **Pydantic structured output** | Enforces schema compliance from LLMs; fails loudly on hallucinated schemas. |
| **Cypher + IoC fallback** | Universal Cypher queries run on any provider; optimized IoC interfaces unlock provider-specific performance. |
| **Invalidation over deletion** | Temporal history is preserved; enables point-in-time queries and full auditability. |
| **group_id as partition key** | Simple multi-tenancy without schema changes; maps to DB-level isolation where supported (FalkorDB, Kuzu). |
| **Sequential episode processing** | Graph consistency requires episodes to be ingested in order; the REST server enforces this via an async queue. |
| **Semaphore concurrency limit** | LLM provider rate limits require bounding parallel requests; `SEMAPHORE_LIMIT=10` is a safe conservative default. |

---

## 18. Limitations & Known Constraints

- **Sequential ingestion requirement** — `add_episode` must not be called concurrently for the same `group_id`; doing so risks deduplication races. The REST server serializes via `AsyncWorker`; SDK users must manage this themselves.
- **LLM structured output dependency** — providers without structured output (function calling) support may produce schema errors during extraction. The README explicitly warns about this.
- **Community detection cost** — `build_communities()` is O(N) in entities; not suitable for real-time updates on large graphs. Intended as a periodic batch job.
- **Neptune fulltext** — requires a separate Amazon OpenSearch Serverless collection; not available in-process.
- **Kuzu edge representation** — entity edges are materialized as intermediate `RelatesToNode_` nodes due to Kuzu's property graph model, requiring specialized query patterns throughout the codebase.
