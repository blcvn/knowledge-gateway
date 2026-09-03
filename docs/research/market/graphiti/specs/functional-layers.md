# Graphiti — Functional Layer Specification

**Version:** 1.0 | **Date:** 2026-05-07 | **Source:** getzep/graphiti v0.28.2

---

## Overview

Graphiti được phân tách thành **7 functional layers**, mỗi layer có trách nhiệm rõ ràng và chỉ phụ thuộc vào layers bên dưới (unidirectional dependency). Tài liệu này mô tả chi tiết chức năng, interface, và data flow giữa các layers.

```
┌─────────────────────────────────────────────────────────┐
│  L7  API & Protocol Layer        (REST, MCP, SDK)       │
├─────────────────────────────────────────────────────────┤
│  L6  Orchestration Layer         (Graphiti Engine)       │
├─────────────────────────────────────────────────────────┤
│  L5  Knowledge Processing Layer  (Extract, Resolve, Dedup) │
├─────────────────────────────────────────────────────────┤
│  L4  Search & Retrieval Layer    (Hybrid Search, Rerank) │
├─────────────────────────────────────────────────────────┤
│  L3  AI Services Layer           (LLM, Embedder, Reranker) │
├─────────────────────────────────────────────────────────┤
│  L2  Data Access Layer           (Namespace, Operations)  │
├─────────────────────────────────────────────────────────┤
│  L1  Storage & Driver Layer      (Neo4j, FalkorDB, Kuzu)  │
└─────────────────────────────────────────────────────────┘
```

---

## L1 — Storage & Driver Layer

### Trách nhiệm
Cung cấp abstraction thống nhất cho mọi tương tác với graph database, che giấu sự khác biệt giữa các backend.

### Key Files

| File | Chức năng |
|------|-----------|
| `driver/query_executor.py` | `QueryExecutor` ABC — interface tối thiểu (`execute_query`, `session`) |
| `driver/query_executor.py` | `Transaction` ABC — interface cho transactional operations |
| `driver/driver.py` | `GraphDriver` ABC — extends QueryExecutor, composes tất cả Operations |
| `driver/neo4j_driver.py` | Neo4j implementation (Bolt protocol, native vector index) |
| `driver/falkordb_driver.py` | FalkorDB implementation (Redis protocol, graph per group_id) |
| `driver/kuzu_driver.py` | Kuzu implementation (embedded, edges as intermediate nodes) |
| `driver/neptune_driver.py` | Neptune implementation (HTTP, OpenSearch cho fulltext) |
| `driver/record_parsers.py` | Chuyển đổi DB records → domain objects |

### Interfaces

```python
class QueryExecutor(ABC):
    async execute_query(query: str, **kwargs) → Any
    session(database: str | None) → GraphDriverSession

class Transaction(ABC):
    async run(query: str, **kwargs) → Any

class GraphDriver(QueryExecutor, ABC):
    provider: GraphProvider              # NEO4J | FALKORDB | KUZU | NEPTUNE
    transaction() → AsyncContextManager[Transaction]
    build_indices_and_constraints(delete_existing: bool) → None
    close() → None
    # 11 Operations properties (xem L2)
```

### Backend Differences

| Aspect | Neo4j | FalkorDB | Kuzu | Neptune |
|--------|-------|----------|------|---------|
| **Transactions** | Real (commit/rollback) | No-op wrapper | No-op wrapper | No-op wrapper |
| **Fulltext** | Lucene native | Custom prefix syntax | Cypher-based | OpenSearch Serverless |
| **Vector Index** | Native | Built-in | Property-based | OpenSearch |
| **Edge Model** | Direct relationships | Direct relationships | Intermediate `RelatesToNode_` | Direct relationships |
| **Multi-tenant** | `group_id` property filter | Separate graph per group | Separate DB per group | Property filter |
| **Embedding Storage** | Native list | Native list | Native list | CSV string serialization |

### Data Flow
```
L2 Operations → GraphDriver.execute_query(cypher) → DB Backend → Records → L2
```

---

## L2 — Data Access Layer

### Trách nhiệm
Cung cấp typed CRUD operations cho mọi graph object, tách biệt DB I/O logic khỏi business logic. Bao gồm hai sub-layers: **Operations ABCs** (pure DB I/O) và **Namespaces** (orchestration wrapper).

### Key Files — Operations ABCs

| File | Object | Key Methods |
|------|--------|-------------|
| `driver/operations/entity_node_ops.py` | EntityNode | `save`, `save_bulk`, `delete`, `get_by_uuid`, `get_by_group_ids`, `load_embeddings` |
| `driver/operations/episode_node_ops.py` | EpisodicNode | `save`, `retrieve_episodes`, `get_by_entity_node_uuid` |
| `driver/operations/community_node_ops.py` | CommunityNode | `save`, `load_name_embedding` |
| `driver/operations/saga_node_ops.py` | SagaNode | `save`, `get_by_group_ids` |
| `driver/operations/entity_edge_ops.py` | EntityEdge | `save`, `get_between_nodes`, `get_by_node_uuid`, `load_embeddings` |
| `driver/operations/episodic_edge_ops.py` | EpisodicEdge | `save`, `save_bulk` |
| `driver/operations/community_edge_ops.py` | CommunityEdge | `save`, `delete` |
| `driver/operations/has_episode_edge_ops.py` | HasEpisodeEdge | `save`, `save_bulk` |
| `driver/operations/next_episode_edge_ops.py` | NextEpisodeEdge | `save`, `save_bulk` |
| `driver/operations/search_ops.py` | Search | `node/edge/episode/community_fulltext_search`, `*_similarity_search`, `*_bfs_search` |
| `driver/operations/graph_ops.py` | Maintenance | `clear_data`, `build_indices`, `get_community_clusters`, `get_mentioned_nodes` |

### Key Files — Namespace Wrappers

| File | Namespace | Accessed via | Cross-cutting concern |
|------|-----------|-------------|----------------------|
| `namespaces/nodes.py` | `EntityNodeNamespace` | `graphiti.nodes.entity` | Embedding generation before save |
| `namespaces/nodes.py` | `EpisodeNodeNamespace` | `graphiti.nodes.episode` | Direct delegation |
| `namespaces/nodes.py` | `CommunityNodeNamespace` | `graphiti.nodes.community` | Embedding generation before save |
| `namespaces/nodes.py` | `SagaNodeNamespace` | `graphiti.nodes.saga` | Direct delegation |
| `namespaces/edges.py` | `EdgeNamespace` | `graphiti.edges` | Edge embedding orchestration |

### Design Pattern

```
User Code                    Namespace                     Operations ABC
─────────                    ─────────                     ──────────────
graphiti.nodes.entity.save(  EntityNodeNamespace.save(      EntityNodeOperations.save(
    node, tx=tx                  node, tx=tx                   executor, node, tx=tx
)                            )                              )
                             │                              │
                             ├─ generate_name_embedding()   ├─ if tx: tx.run(cypher)
                             └─ delegate to ops             └─ else: executor.execute_query()
```

### Dependency Rule
Operations depend ONLY on `QueryExecutor` + `Transaction` (Layer 0 abstractions) — never on `GraphDriver` directly. This breaks import cycles.

---

## L3 — AI Services Layer

### Trách nhiệm
Cung cấp abstraction cho tất cả AI/ML services: LLM inference, embedding generation, và neural reranking. Các layer trên không gọi trực tiếp providers.

### Key Files

| File | Chức năng |
|------|-----------|
| `llm_client/client.py` | `LLMClient` ABC — `generate_response()`, retry, caching, token tracking |
| `llm_client/config.py` | `LLMConfig` — model, small_model, temperature, max_tokens, base_url |
| `llm_client/openai_client.py` | OpenAI implementation (GPT-4o, gpt-4o-mini) |
| `llm_client/anthropic_client.py` | Anthropic implementation (Claude) |
| `llm_client/gemini_client.py` | Google Gemini implementation |
| `llm_client/groq_client.py` | Groq implementation |
| `llm_client/token_tracker.py` | Per-prompt-type token usage aggregation |
| `llm_client/cache.py` | File-based LLM response cache (MD5 key) |
| `embedder/client.py` | `EmbedderClient` ABC — `create()`, `create_bulk()` |
| `embedder/openai.py` | OpenAI Embedder (text-embedding-3-small) |
| `embedder/gemini.py` | Gemini Embedder |
| `embedder/voyage.py` | Voyage AI Embedder |
| `cross_encoder/client.py` | `CrossEncoderClient` ABC — `rank(query, passages)` |
| `cross_encoder/openai_client.py` | OpenAI Reranker (log-prob classification) |
| `cross_encoder/bge_client.py` | Local BGE cross-encoder |

### LLM Client Features

| Feature | Implementation |
|---------|---------------|
| **Retry** | Exponential backoff, 4 attempts, 5-120s, on HTTP 5xx/429/JSON errors |
| **Caching** | File-based MD5 hash cache (`./llm_cache/`), opt-in |
| **Token Tracking** | Per-prompt-type aggregation via `TokenUsageTracker` |
| **Structured Output** | Pydantic `response_model` appended as JSON schema to prompt |
| **Multilingual** | Auto-append language extraction instruction per `group_id` |
| **Input Sanitization** | Strip zero-width chars, control chars, invalid Unicode |
| **Model Selection** | `ModelSize.medium` (default) vs `ModelSize.small` for classification tasks |
| **Tracing** | OTel span per `generate_response()` call |

### Provider Matrix

| Provider | LLM | Embedder | Cross-Encoder |
|----------|-----|----------|---------------|
| OpenAI | ✅ (default) | ✅ (default) | ✅ (default) |
| Azure OpenAI | ✅ | ✅ | — |
| Anthropic | ✅ | — | — |
| Google Gemini | ✅ | ✅ | ✅ |
| Groq | ✅ | — | — |
| Ollama | ✅ (via OpenAI compat) | ✅ (via OpenAI compat) | — |
| Voyage AI | — | ✅ | — |
| BGE (local) | — | — | ✅ |

---

## L4 — Search & Retrieval Layer

### Trách nhiệm
Thực hiện hybrid search kết hợp nhiều phương pháp tìm kiếm, reranking đa chiến lược, và temporal/property filtering.

### Key Files

| File | Chức năng |
|------|-----------|
| `search/search.py` | Core orchestrator — dispatch search methods, merge, rerank |
| `search/search_config.py` | `SearchConfig`, `SearchResults`, enums (SearchMethod, Reranker) |
| `search/search_config_recipes.py` | Pre-built configurations (RRF, MMR, Cross-Encoder variants) |
| `search/search_filters.py` | `SearchFilters` — temporal, uuid, label, group_id filters |
| `search/search_utils.py` | Helper functions (similarity search, mentioned nodes) |

### Search Pipeline

```
Query Input
  │
  ├─1─► Embedding Generation (L3 Embedder)
  │
  ├─2─► Parallel Search Dispatch:
  │     ├── cosine_similarity → ANN over stored vectors
  │     ├── bm25 → Fulltext keyword search (provider-specific Lucene)
  │     └── bfs → Breadth-first graph traversal from matched nodes
  │
  ├─3─► Per-Type Result Collection:
  │     ├── EdgeSearchConfig → EntityEdge results
  │     ├── NodeSearchConfig → EntityNode results
  │     ├── EpisodeSearchConfig → EpisodicNode results
  │     └── CommunitySearchConfig → CommunityNode results
  │
  ├─4─► Reranking:
  │     ├── rrf (Reciprocal Rank Fusion) — merge multi-method results
  │     ├── mmr (Maximal Marginal Relevance) — diversity optimization
  │     ├── cross_encoder — neural reranking via L3
  │     ├── node_distance — graph proximity to center_node_uuid
  │     └── episode_mentions — recency/frequency boost
  │
  ├─5─► Filter Application:
  │     ├── Temporal: created_at_start/end, valid_at, invalid_at
  │     ├── Identity: node_uuids, edge_uuids
  │     ├── Type: entity_labels
  │     └── Partition: group_ids
  │
  └─6─► Return SearchResults { edges, nodes, episodes, communities }
```

### Pre-built Recipes

| Recipe | Methods | Reranker | Use Case |
|--------|---------|----------|----------|
| `EDGE_HYBRID_SEARCH_RRF` | BM25 + cosine | RRF | Default `search()` — general purpose |
| `EDGE_HYBRID_SEARCH_NODE_DISTANCE` | BM25 + cosine | node_distance | Graph-aware edge retrieval |
| `EDGE_HYBRID_SEARCH_CROSS_ENCODER` | BM25 + cosine + BFS | cross_encoder | High-accuracy edge retrieval |
| `COMBINED_HYBRID_SEARCH_CROSS_ENCODER` | All types | cross_encoder | Default `search_()` — comprehensive |
| `NODE_HYBRID_SEARCH_RRF` | BM25 + cosine | RRF | Entity-focused retrieval |
| `COMMUNITY_HYBRID_SEARCH_RRF` | BM25 + cosine | RRF | Topic-level retrieval |

---

## L5 — Knowledge Processing Layer

### Trách nhiệm
Xử lý core knowledge engineering: trích xuất entities/facts từ raw data, deduplication, conflict resolution, community detection, và saga management. Đây là layer chứa phần lớn "intelligence" của hệ thống.

### Key Files

| File | Chức năng |
|------|-----------|
| `utils/maintenance/node_operations.py` | `extract_nodes()`, `resolve_extracted_nodes()`, `extract_attributes_from_nodes()` |
| `utils/maintenance/edge_operations.py` | `extract_edges()`, `resolve_extracted_edges()`, `resolve_extracted_edge()`, `build_episodic_edges()` |
| `utils/maintenance/community_operations.py` | `build_communities()`, `label_propagation()`, `update_community()` |
| `utils/maintenance/graph_data_operations.py` | `retrieve_episodes()` |
| `utils/maintenance/dedup_helpers.py` | Deterministic dedup utilities (similarity threshold, string normalization) |
| `utils/bulk_utils.py` | `extract_nodes_and_edges_bulk()`, `dedupe_nodes_bulk()`, `dedupe_edges_bulk()` |
| `utils/content_chunking.py` | Density-based content chunking for large inputs |
| `utils/ontology_utils/` | Custom entity type validation |

### Sub-functions

#### 5A — Entity Extraction
```
Episode Content → LLM (extract_nodes prompt) → ExtractedEntities
  → Filter empty names → Create EntityNode objects → Collapse exact duplicates
```
- Prompt varies by source type: `extract_message`, `extract_text`, `extract_json`
- Supports custom `entity_types` (Pydantic models) for prescribed ontology
- Multi-episode attribution via `episode_indices`

#### 5B — Entity Resolution (Dedup)
```
Extracted Nodes → Semantic Search (cosine+BM25 candidates)
  → Deterministic Match (exact name match, high-similarity threshold)
  → LLM Dedup (dedupe_nodes prompt) → Resolved Nodes + UUID Map
```
- Two-phase: deterministic first (fast), then LLM for ambiguous cases
- `NODE_DEDUP_COSINE_MIN_SCORE = 0.6` threshold for candidates
- Max 15 candidates per extracted node

#### 5C — Edge/Fact Extraction
```
Episode + Resolved Nodes → LLM (extract_edges prompt) → ExtractedEdges
  → Validate entity names → Parse temporal fields → Create EntityEdge objects
```
- Fact triples: `(source_entity, relation_type, target_entity, fact_text)`
- Temporal parsing: `valid_at`, `invalid_at` from LLM output
- Self-edge detection and removal
- Custom `edge_types` with node signature validation

#### 5D — Edge Resolution (Conflict)
```
Extracted Edge → Search existing edges (between same nodes + broader)
  → Fast path: exact text match → reuse existing
  → LLM (dedupe_edges prompt) → identify duplicates + contradictions
  → resolve_edge_contradictions() → set invalid_at on superseded edges
```
- **Key invariant:** Old facts are NEVER deleted. `invalid_at` is set on contradicted edges.
- Continuous indexing: duplicate candidates (idx 0-N) + invalidation candidates (idx N+1-M)
- Custom edge attributes extracted via separate LLM call

#### 5E — Community Detection
```
All EntityNodes per group_id → Build adjacency projection
  → Label Propagation Algorithm (iterative)
  → For each cluster: Hierarchical pairwise LLM summarization
  → Create CommunityNode + CommunityEdge (HAS_MEMBER)
```
- Label Propagation: each node adopts plurality community of neighbors
- Pairwise summarization: merge summaries bottom-up until single summary remains
- Max concurrency: 10 concurrent community builds
- O(N) cost — intended as periodic batch job

#### 5F — Saga Management
```
Episodes with same saga_id → SagaNode (auto-created on first use)
  → HAS_EPISODE edges → NEXT_EPISODE chain (temporal order)
  → summarize_saga(): incremental LLM summary since last_summarized_at
```

### Prompt Library (used by L5)

| Module | L5 Sub-function | Output Schema |
|--------|-----------------|---------------|
| `prompts/extract_nodes.py` | 5A Entity Extraction | `ExtractedEntities` |
| `prompts/extract_edges.py` | 5C Edge Extraction | `ExtractedEdges`, `EdgeTimestamps` |
| `prompts/extract_nodes_and_edges.py` | Bulk combined extraction | Merged schema |
| `prompts/dedupe_nodes.py` | 5B Entity Resolution | `NodeResolutions` |
| `prompts/dedupe_edges.py` | 5D Edge Resolution | `EdgeDuplicate` |
| `prompts/summarize_nodes.py` | 5E Community summaries | `Summary`, `SummaryDescription` |
| `prompts/summarize_sagas.py` | 5F Saga summaries | `SagaSummary` |

---

## L6 — Orchestration Layer

### Trách nhiệm
Điều phối toàn bộ pipeline: assembly các L3 clients, quản lý lifecycle, và expose high-level API methods mà L7 sử dụng.

### Key Files

| File | Chức năng |
|------|-----------|
| `graphiti.py` | `Graphiti` class (~1741 LOC) — single entry point |
| `graphiti_types.py` | `GraphitiClients` — bundles driver + llm + embedder + cross_encoder + tracer |
| `helpers.py` | `semaphore_gather()`, validation utilities, chunking config |
| `decorators.py` | `@handle_multiple_group_ids` — fan-out operations across groups |
| `tracer.py` | `Tracer` ABC, `OpenTelemetryTracer`, `NoOpTracer` |
| `telemetry/` | Anonymous PostHog telemetry |

### Graphiti Class — Public API

| Method | Orchestrates | Layers Used |
|--------|-------------|-------------|
| `add_episode()` | Full ingestion pipeline (9 steps) | L5→L4→L3→L2→L1 |
| `add_episode_bulk()` | Batch ingestion with in-memory dedup | L5→L4→L3→L2→L1 |
| `add_triplet()` | Direct (S,P,O) insertion with resolution | L5→L4→L3→L2→L1 |
| `search()` | Basic hybrid edge search (RRF) | L4→L3→L2→L1 |
| `search_()` | Configurable multi-type search | L4→L3→L2→L1 |
| `retrieve_episodes()` | Fetch recent episodes | L2→L1 |
| `build_communities()` | Full community detection | L5→L3→L2→L1 |
| `summarize_saga()` | Incremental saga summary | L5→L3→L2→L1 |
| `build_indices_and_constraints()` | DB index setup | L1 |
| `remove_episode()` | Cascade delete | L2→L1 |

### Episode Ingestion Orchestration (`add_episode`)

```
L6 orchestrates:
  1. L2: retrieve_episodes() — context window
  2. L5: extract_nodes() — LLM entity extraction
  3. L5: resolve_extracted_nodes() — dedup against graph
  4. L5: extract_edges() — LLM fact extraction (parallel with step 5)
  5. L5: extract_attributes_from_nodes() — LLM summary update
  6. L5: resolve_extracted_edges() — conflict resolution
  7. L5: build_episodic_edges() — MENTIONS links
  8. L2: add_nodes_and_edges_bulk() — persist with embeddings
  9. L5: update_community() — community membership
```

### Concurrency Management

| Mechanism | Config | Default |
|-----------|--------|---------|
| `semaphore_gather()` | `SEMAPHORE_LIMIT` env var | 20 |
| Content chunking | `CHUNK_TOKEN_SIZE`, `CHUNK_DENSITY_THRESHOLD` | 3000 tokens, 0.15 |
| Sequential constraint | AsyncWorker (REST server) | 1 consumer per group |

---

## L7 — API & Protocol Layer

### Trách nhiệm
Expose Graphiti capabilities qua multiple protocols: REST API, MCP Protocol, và direct Python SDK.

### Key Files

| File | Chức năng |
|------|-----------|
| `server/graph_service/main.py` | FastAPI app với lifespan context manager |
| `server/graph_service/config.py` | `Settings` (Pydantic BaseSettings) |
| `server/graph_service/zep_graphiti.py` | `ZepGraphiti` subclass — extended CRUD |
| `server/graph_service/routers/ingest.py` | Ingestion endpoints + AsyncWorker queue |
| `server/graph_service/routers/retrieve.py` | Search/retrieval endpoints |
| `server/graph_service/dto/` | Request/Response DTOs |
| `mcp_server/src/graphiti_mcp_server.py` | FastMCP server — 9 tools |

### REST API (FastAPI)

| Endpoint | Method | Router | Mô tả |
|----------|--------|--------|-------|
| `/messages` | POST | ingest | Async episode ingestion (queued) |
| `/entity-node` | POST | ingest | Create entity node |
| `/entity-edge/{uuid}` | DELETE | ingest | Delete edge |
| `/group/{group_id}` | DELETE | ingest | Clear group data |
| `/episode/{uuid}` | DELETE | ingest | Delete episode + cascade |
| `/clear` | POST | ingest | Clear entire graph |
| `/search` | GET | retrieve | Hybrid search |
| `/episodes` | GET | retrieve | List recent episodes |

**Critical design:** Ingest router uses `AsyncWorker` (asyncio queue + single consumer task) to serialize episode processing per group — required for graph consistency.

### MCP Server (FastMCP)

| Tool | Mô tả |
|------|-------|
| `add_memory` | Add episode to graph |
| `search_memory` | Hybrid search |
| `get_episodes` | List recent episodes |
| `delete_episode` | Delete episode |
| `delete_entity_node` | Delete entity |
| `delete_entity_edge` | Delete edge |
| `get_entity_edge` | Get edge details |
| `clear_graph` | Clear graph |
| `get_status` | Server status |

### Python SDK

Direct usage of `Graphiti` class (L6) — no serialization overhead:

```python
graphiti = Graphiti(uri, user, password, llm_client, embedder)
await graphiti.build_indices_and_constraints()
result = await graphiti.add_episode(...)
results = await graphiti.search(query, group_ids)
```

---

## Cross-Layer Data Flows

### Flow 1: Episode Ingestion (Full Pipeline)

```
L7 (REST POST /messages)
  │ deserialize request → RawEpisode
  │ enqueue to AsyncWorker
  ▼
L6 (Graphiti.add_episode)
  │ assemble GraphitiClients
  │ coordinate 9-step pipeline
  ▼
L5 (Knowledge Processing)
  │ extract_nodes() ─────────────────► L3 (LLM: extract entities)
  │ resolve_extracted_nodes() ───────► L3 (LLM: dedup) + L4 (search candidates)
  │ extract_edges() ─────────────────► L3 (LLM: extract facts)
  │ resolve_extracted_edges() ───────► L3 (LLM: resolve conflicts) + L4 (search related)
  │ build_episodic_edges()
  │ update_community() ─────────────► L3 (LLM: summarize)
  ▼
L2 (Namespace/Operations)
  │ generate embeddings ────────────► L3 (Embedder)
  │ save nodes, edges, episodes
  ▼
L1 (GraphDriver)
  │ execute_query(cypher)
  ▼
Graph Database (Neo4j / FalkorDB / Kuzu / Neptune)
```

### Flow 2: Hybrid Search

```
L7 (REST GET /search or SDK graphiti.search())
  ▼
L6 (Graphiti.search / search_)
  │ validate inputs, select config
  ▼
L4 (Search Pipeline)
  │ generate query embedding ───────► L3 (Embedder)
  │ dispatch parallel searches ─────► L2 (SearchOperations) → L1
  │ merge results
  │ apply reranking ────────────────► L3 (CrossEncoder, if cross_encoder reranker)
  │ apply filters
  ▼
L6 → L7 (return SearchResults)
```

### Flow 3: Community Detection

```
L6 (Graphiti.build_communities)
  ▼
L5 (community_operations)
  │ get_community_clusters() ───────► L2 → L1 (query adjacency)
  │ label_propagation()              (in-memory algorithm)
  │ build_community() ──────────────► L3 (LLM: pairwise summarization)
  │ generate embeddings ────────────► L3 (Embedder)
  ▼
L2 (save CommunityNode + CommunityEdge) → L1
```

---

## Observability Across Layers

| Layer | Instrumentation |
|-------|----------------|
| L1 | Query execution timing (via tracer spans) |
| L2 | Operation-level spans |
| L3 | `llm.generate` spans (provider, model, cache hit, tokens) |
| L4 | `search.*` spans (method, result counts) |
| L5 | Pipeline step timing (extraction, resolution durations) |
| L6 | Top-level operation spans (`add_episode`, `search`) |
| L7 | HTTP request logging, health check |

**Token Tracking:** `TokenUsageTracker` (L3) aggregates per prompt type across all L5 calls. Access: `graphiti.token_tracker.get_usage()`.

**Telemetry:** Anonymous PostHog event on L6 initialization. Disable: `GRAPHITI_TELEMETRY_ENABLED=false`.
