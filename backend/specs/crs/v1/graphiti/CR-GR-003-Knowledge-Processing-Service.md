# Change Request: CR-GR-003 — Knowledge Processing Service (LLM + Entity/Edge Extraction)

**CR ID:** CR-GR-003  
**Component:** `services/graphiti-knowledge` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** graphiti PRD §5.4, SRS §3.3–3.5, specs/services/04-knowledge-service.md  
**Maps to Python:** `llm_client/`, `embedder/`, `cross_encoder/`, `prompts/`, `utils/maintenance/`

---

## 1. Mô tả

Xây dựng **graphiti-knowledge** service — service duy nhất encapsulates toàn bộ AI/ML intelligence:
1. **Entity Extraction** — LLM trích xuất entities từ episode content.
2. **Entity Resolution** — Dedup entities (deterministic fast path + LLM ambiguous path).
3. **Edge Extraction** — LLM trích xuất fact triples với temporal validity.
4. **Edge Resolution** — Conflict detection + temporal invalidation decisions.
5. **Attribute Extraction** — Cập nhật entity summaries từ facts mới.
6. **Community Detection** — Label Propagation + LLM summarization.
7. **Embedding Generation** — Vector embeddings cho names, facts, queries.
8. **Neural Reranking** — Cross-encoder reranking cho search.
9. **Saga Summarization** — Incremental LLM summary của saga sequences.

---

## 2. Vấn đề hiện tại

`services/graph-service` / `services/knowledge-service` hiện tại:
- ✅ Có basic LLM extraction (cognee-based).
- ❌ Không có **entity resolution** với LLM disambiguation (two-phase).
- ❌ Không có **edge resolution** với temporal invalidation logic.
- ❌ Không có **community detection** (Label Propagation + LLM summarization).
- ❌ Không có **cross-encoder reranking**.
- ❌ Không có **token usage tracking** per prompt type.
- ❌ Không có **LLM response caching** (Redis-backed).
- ❌ Không có **Bifrost gateway adapter** cho production.
- ❌ Prompt management thiếu template registry + versioning.
- ❌ Không có **bulk extraction** (parallel across episodes).
- ❌ Không có **input sanitization** (Unicode cleaning).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/graphiti-knowledge/`

**Port:** `9003` (gRPC internal)

### 3.2. LLM Client Interface

```go
// internal/adapter/client/llm/client.go

type LLMClient interface {
    GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error)
}

type GenerateOpts struct {
    ResponseSchema interface{}  // JSON schema for structured output
    PromptName     string       // for token tracking
    ModelSize      ModelSize    // ModelSizeMedium (default) | ModelSizeSmall
    MaxTokens      int
    Temperature    float64
}

type LLMResponse struct {
    Content    json.RawMessage
    TokenUsage TokenUsage
    Cached     bool
    Provider   string
    Model      string
}

type Message struct {
    Role    string  // system | user | assistant
    Content string
}

type ModelSize int
const (
    ModelSizeMedium ModelSize = iota  // gpt-4o, claude-3-5-sonnet, gemini-2.0-flash
    ModelSizeSmall                    // gpt-4o-mini, haiku, flash-lite
)
```

### 3.3. LLM Provider Implementations

| Adapter | Provider | Models | Notes |
|---|---|---|---|
| `openai.go` | OpenAI | GPT-4o / gpt-4o-mini | Native structured output |
| `anthropic.go` | Anthropic | claude-3-5-sonnet | Tool use for structured output |
| `gemini.go` | Google Gemini | gemini-2.0-flash | Native structured output |
| `groq.go` | Groq | Llama 3.1 70B | Fast inference |
| `bifrost.go` | Bifrost gateway | Via proxy | Unified + failover (production recommended) |
| `generic.go` | Any OpenAI-compat | Configurable | Ollama, LM Studio, vLLM |

### 3.4. Retry & Circuit Breaker

```go
// Exponential backoff retry per LLM call
type RetryConfig struct {
    MaxAttempts   int           // default: 4
    InitialDelay  time.Duration // default: 5s
    MaxDelay      time.Duration // default: 120s
    RetryableHTTP []int         // [429, 500, 502, 503, 504]
}

// LLM response cache (Redis, opt-in)
// Cache key = MD5(provider + model + messages_json)
// TTL: 3600s default
```

### 3.5. Entity Extraction (5A)

```go
// internal/usecase/extract_entities.go
// Prompt: extract_nodes (per source type)
// Input: chunks[], prev_episodes[], entity_types (optional ontology)
// Output: []ExtractedEntity{name, label, summary}

// Phase-specific prompts:
// - extract_message: for chat message episodes
// - extract_text: for document text episodes
// - extract_json: for structured JSON episodes
// - extract_nodes_and_edges: combined prompt for bulk extraction

// Validation: filter empty names, collapse exact duplicates
// Custom ontology: validate extracted labels against entity_types schema
// Track token usage per prompt type
```

### 3.6. Entity Resolution (5B) — Two-Phase

```go
// internal/usecase/resolve_entities.go

// Phase 1: Deterministic (fast, NO LLM cost)
// - Exact name match (case-insensitive)
// - High cosine similarity (> 0.95) → auto-merge
// → 60-70% of cases resolved without LLM

// Phase 2: LLM-based (ambiguous cases)
// - Max 15 candidates per entity (NODE_DEDUP_COSINE_MIN_SCORE = 0.6)
// - Prompt: dedupe_nodes
// - LLM decision: merge (existing UUID) | keep (new entity)
// - Cost: ~1 LLM call per ambiguous entity

// Resolution output:
// - existing_uuid: merge with existing node
// - new_entity: create as new EntityNode
```

### 3.7. Edge Extraction (5C)

```go
// internal/usecase/extract_edges.go
// Extracts fact triples: (source_entity, relation_type, target_entity, fact_text)
// Prompt: extract_edges
// Features:
// - Temporal parsing: valid_at, invalid_at from LLM output
// - Self-edge detection and removal (source == target → skip)
// - Custom edge_types with node signature validation
// - Multi-episode attribution (which chunks contributed to each edge)
```

### 3.8. Edge Resolution (5D) — Temporal Invalidation

```go
// internal/usecase/resolve_edges.go
// KEY INVARIANT: Old facts are NEVER deleted, only invalidated

// Resolution pipeline:
// 1. Fast path: exact fact text match → reuse existing edge (DUPLICATE)
// 2. Search existing edges between same nodes (EdgeSimilaritySearch)
// 3. LLM (dedupe_edges) → categorize as:
//    - DUPLICATE: exact match, no action
//    - NEW: no conflict, create edge
//    - CONTRADICTION: new info replaces old → mark old invalid_at
//    - UPDATE: partial update → supersede old, create new

// Contradiction handling:
// → store.InvalidateEntityEdge(old_uuid, reference_time)
// → Create new edge with valid_at = reference_time
// → Both persist for point-in-time historical queries
```

### 3.9. Community Detection (5E) — Label Propagation

```go
// internal/usecase/build_community.go

// Algorithm:
// 1. Get community clusters from Store (adjacency lists)
// 2. Label Propagation in-memory (O(N)):
//    - Each node adopts plurality community of neighbors
//    - Iterate until convergence
// 3. For each cluster: Hierarchical pairwise LLM summarization
//    - Merge summaries bottom-up until single summary per community
//    - Max concurrency: 10 (semaphore)
// 4. Generate name_embedding for community search
// 5. Persist CommunityNode + CommunityEdge via Store

// Incremental update (UpdateCommunity):
// - Only process affected entities after episode ingest
// - Determine which community each entity belongs to
// - Re-summarize only affected communities
```

### 3.10. Embedding Generation

```go
// internal/adapter/client/embedder/client.go

type EmbedderClient interface {
    Create(ctx, input string) ([]float32, error)
    CreateBulk(ctx, inputs []string) ([][]float32, error)
    Dimensions() int
}

// Embeddings generated for:
// - EntityNode.name_embedding        (for node similarity search)
// - EntityEdge.fact_embedding        (for edge similarity search)
// - CommunityNode.name_embedding     (for community search)
// - Search query embedding           (for hybrid search)

// Providers: OpenAI (text-embedding-3-small, 1536d), Gemini, Voyage AI, Ollama
```

### 3.11. Cross-Encoder Reranking

```go
// internal/adapter/client/reranker/client.go

type CrossEncoderClient interface {
    Rank(ctx, query string, passages []string) ([]float64, error)
}

// Providers: OpenAI (via chat API), Gemini, BGE local
// Used by: Search service for neural reranking
```

### 3.12. Prompt Registry

```go
// internal/adapter/prompt/registry.go

type PromptTemplate struct {
    Name    string
    Version int
    System  string
    User    func(ctx PromptContext) string
    Schema  interface{}  // expected response JSON schema
}

// Registry entries:
var DefaultPrompts = map[string]PromptTemplate{
    "extract_nodes":            extractNodesPrompt,
    "extract_edges":            extractEdgesPrompt,
    "extract_nodes_and_edges":  extractCombinedPrompt,
    "dedupe_nodes":             dedupeNodesPrompt,
    "dedupe_edges":             dedupeEdgesPrompt,
    "summarize_nodes":          summarizeNodesPrompt,
    "summarize_sagas":          summarizeSagasPrompt,
}

// Multilingual support: auto-append language instruction
// based on content language detection or per group_id config
```

### 3.13. Token Usage Tracking

```go
// internal/infra/telemetry/token_tracker.go

type TokenTracker struct {
    mu    sync.RWMutex
    usage map[string]*TokenUsage  // key: prompt_type
}

type TokenUsage struct {
    PromptTokens     int64
    CompletionTokens int64
    TotalTokens      int64
    CallCount        int64
}

// Tracked per: extract_nodes, extract_edges, dedupe_nodes, dedupe_edges,
//              summarize_nodes, summarize_sagas, extract_combined
// Exposed via: GetTokenUsage RPC + Prometheus metric
```

### 3.14. gRPC API

```protobuf
service KnowledgeService {
    // Entity
    rpc ExtractEntities(ExtractEntitiesRequest) returns (ExtractEntitiesResponse);
    rpc ResolveEntity(ResolveEntityRequest) returns (ResolveEntityResponse);
    rpc ExtractAttributes(ExtractAttributesRequest) returns (ExtractAttributesResponse);

    // Edge
    rpc ExtractEdges(ExtractEdgesRequest) returns (ExtractEdgesResponse);
    rpc ResolveEdge(ResolveEdgeRequest) returns (ResolveEdgeResponse);

    // Bulk
    rpc ExtractEntitiesAndEdgesBulk(ExtractBulkRequest) returns (ExtractBulkResponse);
    rpc DedupeEntitiesBulk(DedupeBulkRequest) returns (DedupeBulkResponse);

    // Community
    rpc BuildCommunities(BuildCommunitiesRequest) returns (BuildCommunitiesResponse);
    rpc UpdateCommunity(UpdateCommunityRequest) returns (UpdateCommunityResponse);

    // Embedding
    rpc GenerateEmbedding(GenerateEmbeddingRequest) returns (GenerateEmbeddingResponse);
    rpc GenerateEmbeddingBulk(GenerateEmbeddingBulkRequest) returns (GenerateEmbeddingBulkResponse);

    // Reranking
    rpc Rerank(RerankRequest) returns (RerankResponse);

    // Saga
    rpc SummarizeSaga(SummarizeSagaRequest) returns (SummarizeSagaResponse);

    // Telemetry
    rpc GetTokenUsage(GetTokenUsageRequest) returns (GetTokenUsageResponse);
}
```

### 3.15. NATS Events Published

| Subject | Trigger |
|---|---|
| `graphiti.entity.resolved` | After entity dedup completed |
| `graphiti.community.rebuilt` | After community detection |

---

## 4. Configuration

```yaml
server:
  grpc_port: 9003
  max_concurrent_llm: 20
  max_concurrent_community: 10

llm:
  provider: "openai"  # openai | anthropic | gemini | groq | bifrost | generic
  model: "gpt-4o"
  small_model: "gpt-4o-mini"
  api_key: "${OPENAI_API_KEY}"
  base_url: ""         # for ollama/custom endpoints
  max_tokens: 4096
  temperature: 0.0
  retry:
    max_attempts: 4
    initial_delay: 5s
    max_delay: 120s
  cache:
    enabled: false
    redis_url: "redis://redis:6379/2"
    ttl: 3600s

embedder:
  provider: "openai"
  model: "text-embedding-3-small"
  dimensions: 1536
  api_key: "${OPENAI_API_KEY}"

reranker:
  provider: "openai"
  model: "gpt-4o-mini"

resolution:
  node_dedup_cosine_min_score: 0.6
  node_dedup_max_candidates: 15
  edge_dedup_cosine_min_score: 0.5
```

---

## 5. Acceptance Criteria

- [ ] `ExtractEntities` với "Alice joined engineering in March" → `[]ExtractedEntity{name: "Alice", label: "Person", ...}`.
- [ ] `ResolveEntity("Alice")` khi "Alice" đã tồn tại trong graph → trả về `existing_uuid` (Phase 1 fast path).
- [ ] `ResolveEdge` khi new fact contradicts existing → `resolution: CONTRADICTION`, `invalidated_edge_uuids: [old_uuid]`.
- [ ] `BuildCommunities` với 10 connected entities → `CommunityNode` được tạo với `summary` từ LLM.
- [ ] `GenerateEmbedding("database performance")` → `[]float32` với `dimensions: 1536`.
- [ ] `Rerank("auth middleware", passages)` → scores array sorted by relevance.
- [ ] Token usage sau 10 `ExtractEntities` calls → `GetTokenUsage()` trả về `extract_nodes.total_tokens > 0`.
- [ ] LLM cache hit: same prompt gọi 2 lần → 2nd call `response.cached = true` (0 tokens used).
- [ ] Custom ontology: `ExtractEntities` với `entity_types = {"Company": {...}}` → chỉ extract Company entities.
- [ ] `SummarizeSaga` với 5 episodes → `SagaSummary.summary` là text tóm tắt.
