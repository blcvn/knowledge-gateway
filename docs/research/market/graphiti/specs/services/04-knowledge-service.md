# graphiti-knowledge — Knowledge Processing Service

**Version:** 2.0 | **Date:** 2026-05-09  
**Origin:** Python L5 (Knowledge Processing Layer) + L3 (AI Services Layer)  
**Architecture:** Clean Architecture | **Protocol:** gRPC

---

## 1. Service Overview

Knowledge Service encapsulates tất cả **AI/ML intelligence** của hệ thống: LLM inference, entity/edge extraction, deduplication, conflict resolution, community detection, embedding generation, và neural reranking. Nó là service duy nhất giao tiếp trực tiếp với LLM/Embedding providers.

### Responsibilities

| Concern | Description |
|---------|-------------|
| **Entity Extraction** | LLM-based entity identification from episode content |
| **Entity Resolution** | Semantic dedup via search + LLM disambiguation |
| **Edge Extraction** | LLM-based fact triple extraction |
| **Edge Resolution** | Conflict detection + temporal invalidation decisions |
| **Attribute Extraction** | Entity summary updates from new facts |
| **Community Detection** | Label Propagation + LLM summarization |
| **Embedding Generation** | Vector embeddings for names, facts, queries |
| **Neural Reranking** | Cross-encoder based passage reranking |
| **Prompt Management** | Template registry, versioning, multilingual support |
| **Token Tracking** | Per-prompt-type token usage aggregation |

---

## 2. Clean Architecture Layout

```
services/graphiti-knowledge/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/                         # Layer 1: Entities
│   │   ├── entity.go                   #   ExtractedEntity, ResolvedEntity
│   │   ├── edge.go                     #   ExtractedEdge, ResolvedEdge
│   │   ├── community.go                #   Community, CommunityCluster
│   │   ├── prompt.go                   #   Prompt, PromptTemplate, Message
│   │   ├── embedding.go                #   EmbeddingVector, EmbeddingModel
│   │   ├── token_usage.go              #   TokenUsage, UsageByPromptType
│   │   ├── model_config.go             #   LLMConfig, ModelSize
│   │   ├── ontology.go                 #   EntityTypeSchema, EdgeTypeSchema
│   │   └── errors.go
│   ├── usecase/                        # Layer 2: Use Cases
│   │   ├── extract_entities.go         #   Entity extraction from content
│   │   ├── resolve_entities.go         #   Entity deduplication
│   │   ├── extract_edges.go            #   Fact triple extraction
│   │   ├── resolve_edges.go            #   Edge conflict resolution
│   │   ├── extract_attributes.go       #   Entity summary updates
│   │   ├── extract_bulk.go             #   Combined bulk extraction
│   │   ├── dedupe_bulk.go              #   Cross-batch dedup
│   │   ├── build_community.go          #   Community detection + summarization
│   │   ├── update_community.go         #   Incremental community update
│   │   ├── generate_embedding.go       #   Embedding generation
│   │   ├── rerank.go                   #   Cross-encoder reranking
│   │   ├── summarize_saga.go           #   Saga summarization
│   │   ├── port/
│   │   │   ├── input.go               #   KnowledgeUseCase interfaces
│   │   │   └── output.go             #   LLMPort, EmbedderPort, RerankerPort, StorePort
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── grpc/
│   │   │   ├── handler.go             #   gRPC service implementation
│   │   │   └── mapper.go
│   │   ├── client/
│   │   │   ├── llm/                   #   LLM provider adapters
│   │   │   │   ├── client.go          #     LLMClient interface
│   │   │   │   ├── openai.go          #     OpenAI adapter
│   │   │   │   ├── anthropic.go       #     Anthropic adapter
│   │   │   │   ├── gemini.go          #     Google Gemini adapter
│   │   │   │   ├── groq.go            #     Groq adapter
│   │   │   │   ├── generic.go         #     OpenAI-compatible (Ollama, etc.)
│   │   │   │   └── bifrost.go         #     Bifrost AI Gateway adapter
│   │   │   ├── embedder/             #   Embedding provider adapters
│   │   │   │   ├── client.go          #     EmbedderClient interface
│   │   │   │   ├── openai.go
│   │   │   │   ├── gemini.go
│   │   │   │   ├── voyage.go
│   │   │   │   └── bifrost.go
│   │   │   ├── reranker/             #   Reranker adapters
│   │   │   │   ├── client.go          #     CrossEncoderClient interface
│   │   │   │   ├── openai.go
│   │   │   │   ├── gemini.go
│   │   │   │   └── local_bge.go
│   │   │   └── store_client.go        #   Store Service gRPC client
│   │   ├── prompt/                    #   Prompt template repository
│   │   │   ├── registry.go           #     Template registry + versioning
│   │   │   ├── extract_nodes.go      #     Entity extraction prompts
│   │   │   ├── extract_edges.go      #     Edge extraction prompts
│   │   │   ├── dedupe_nodes.go       #     Entity dedup prompts
│   │   │   ├── dedupe_edges.go       #     Edge conflict prompts
│   │   │   ├── summarize_nodes.go    #     Node summarization prompts
│   │   │   ├── summarize_sagas.go    #     Saga summarization prompts
│   │   │   └── extract_combined.go   #     Bulk extraction prompts
│   │   ├── cache/
│   │   │   └── llm_cache.go          #   LLM response cache (Redis)
│   │   └── event/
│   │       └── publisher.go          #   Domain event publisher
│   └── infra/
│       ├── config/
│       │   └── config.go
│       ├── server/
│       │   └── grpc.go
│       ├── telemetry/
│       │   ├── token_tracker.go       #   Per-prompt token aggregation
│       │   ├── tracer.go
│       │   └── metrics.go
│       └── wire/
├── api/
│   └── proto/
│       └── knowledge/v1/
│           └── knowledge.proto
├── Dockerfile
└── Makefile
```

---

## 3. gRPC API (Protobuf)

```protobuf
syntax = "proto3";
package graphiti.knowledge.v1;

import "google/protobuf/timestamp.proto";
import "common/temporal.proto";

service KnowledgeService {
  // Entity operations
  rpc ExtractEntities(ExtractEntitiesRequest) returns (ExtractEntitiesResponse);
  rpc ResolveEntity(ResolveEntityRequest) returns (ResolveEntityResponse);
  rpc ExtractAttributes(ExtractAttributesRequest) returns (ExtractAttributesResponse);
  
  // Edge operations
  rpc ExtractEdges(ExtractEdgesRequest) returns (ExtractEdgesResponse);
  rpc ResolveEdge(ResolveEdgeRequest) returns (ResolveEdgeResponse);
  
  // Bulk operations
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
  
  // Token usage
  rpc GetTokenUsage(GetTokenUsageRequest) returns (GetTokenUsageResponse);
}

// --- Entity Extraction ---

message ExtractEntitiesRequest {
  repeated ContentChunk chunks = 1;
  repeated EpisodeContext previous_episodes = 2;
  map<string, EntityTypeSchema> entity_types = 3;    // optional ontology
  EpisodeSource source = 4;
  string group_id = 5;
}

message ExtractEntitiesResponse {
  repeated ExtractedEntity entities = 1;
  TokenUsage token_usage = 2;
}

message ExtractedEntity {
  string name = 1;
  string label = 2;
  string summary = 3;
  repeated int32 episode_indices = 4;      // multi-episode attribution
}

// --- Entity Resolution ---

message ResolveEntityRequest {
  ExtractedEntity entity = 1;
  repeated EntityCandidate candidates = 2;  // from search
  string group_id = 3;
}

message ResolveEntityResponse {
  oneof resolution {
    string existing_uuid = 1;              // merged with existing
    ExtractedEntity new_entity = 2;        // kept as new
  }
  TokenUsage token_usage = 3;
}

// --- Edge Extraction ---

message ExtractEdgesRequest {
  repeated ContentChunk chunks = 1;
  repeated ResolvedNode resolved_nodes = 2;
  map<string, EdgeTypeSchema> edge_types = 3;
  string group_id = 4;
}

message ExtractEdgesResponse {
  repeated ExtractedEdge edges = 1;
  TokenUsage token_usage = 2;
}

message ExtractedEdge {
  string source_name = 1;
  string target_name = 2;
  string relation_type = 3;
  string fact = 4;
  optional google.protobuf.Timestamp valid_at = 5;
  optional google.protobuf.Timestamp invalid_at = 6;
}

// --- Edge Resolution ---

message ResolveEdgeRequest {
  ExtractedEdge edge = 1;
  repeated ExistingEdge existing_edges = 2;     // from store search
  google.protobuf.Timestamp reference_time = 3;
  string group_id = 4;
}

message ResolveEdgeResponse {
  EdgeResolution resolution = 1;
  repeated string invalidated_edge_uuids = 2;  // edges to set invalid_at
  TokenUsage token_usage = 3;
}

enum EdgeResolution {
  EDGE_RESOLUTION_UNSPECIFIED = 0;
  EDGE_RESOLUTION_NEW = 1;              // new edge, no conflicts
  EDGE_RESOLUTION_DUPLICATE = 2;        // exact match with existing
  EDGE_RESOLUTION_CONTRADICTION = 3;    // contradicts existing, invalidate old
  EDGE_RESOLUTION_UPDATE = 4;           // partial update to existing
}

// --- Embedding ---

message GenerateEmbeddingRequest {
  string text = 1;
}

message GenerateEmbeddingResponse {
  repeated float embedding = 1;
  int32 dimensions = 2;
}

// --- Reranking ---

message RerankRequest {
  string query = 1;
  repeated string passages = 2;
}

message RerankResponse {
  repeated float scores = 1;           // one score per passage
}
```

---

## 4. Knowledge Processing Functions

### 4.1 Entity Extraction (5A)

```go
// internal/usecase/extract_entities.go

type ExtractEntitiesUseCase struct {
    llmClient   port.LLMPort
    promptRepo  port.PromptRepository
    logger      *slog.Logger
}

func (uc *ExtractEntitiesUseCase) Execute(ctx context.Context, req *dto.ExtractEntitiesReq) (*dto.ExtractEntitiesResp, error) {
    // 1. Select prompt based on source type
    //    - extract_message (chat), extract_text (document), extract_json (structured)
    
    // 2. Build prompt with chunks + previous episodes context
    
    // 3. Call LLM with structured output (JSON schema enforcement)
    //    Response model: []ExtractedEntity{name, label, summary}
    
    // 4. Filter empty names, collapse exact duplicates
    
    // 5. If entity_types specified: validate against ontology schema
    
    // 6. Track token usage per prompt type
}
```

### 4.2 Entity Resolution (5B)

```go
// Two-phase resolution:
// Phase 1: Deterministic (fast)
//   - Exact name match (case-insensitive)
//   - High similarity threshold (cosine > 0.95)
// Phase 2: LLM-based (for ambiguous cases)
//   - Max 15 candidates per entity
//   - NODE_DEDUP_COSINE_MIN_SCORE = 0.6
//   - LLM prompt: dedupe_nodes → merge/keep decision
```

### 4.3 Edge Extraction (5C)

```go
// Extracts fact triples: (source_entity, relation_type, target_entity, fact_text)
// Temporal parsing: valid_at, invalid_at from LLM output
// Self-edge detection and removal
// Custom edge_types with node signature validation
```

### 4.4 Edge Resolution (5D)

```go
// Key invariant: Old facts are NEVER deleted
// When contradiction detected:
//   1. Set old edge invalid_at = reference_time
//   2. Create new edge with valid_at
//   3. Both edges persist for point-in-time queries
//
// Resolution pipeline:
//   1. Fast path: exact text match → reuse existing
//   2. Search existing edges between same nodes
//   3. LLM (dedupe_edges) → identify duplicates + contradictions
//   4. resolve_edge_contradictions() → mark invalidations
```

### 4.5 Community Detection (5E)

```go
// internal/usecase/build_community.go

type BuildCommunityUseCase struct {
    llmClient    port.LLMPort
    embedder     port.EmbedderPort
    storeClient  port.StorePort
    semaphore    chan struct{}  // max 10 concurrent community builds
}

func (uc *BuildCommunityUseCase) Execute(ctx context.Context, groupIDs []string) error {
    // 1. Get community clusters from Store
    //    store.GetCommunityClusters(group_ids)
    
    // 2. Label Propagation Algorithm (in-memory)
    //    Each node adopts plurality community of neighbors
    //    Iterate until convergence
    
    // 3. For each cluster: Hierarchical pairwise LLM summarization
    //    Merge summaries bottom-up until single summary remains
    //    Max concurrency: 10
    
    // 4. Generate name_embedding for community search
    
    // 5. Persist CommunityNode + CommunityEdge via Store
}
```

---

## 5. LLM Client Architecture

### 5.1 Provider Interface

```go
// internal/adapter/client/llm/client.go

type LLMClient interface {
    GenerateResponse(ctx context.Context, messages []Message, opts GenerateOpts) (*LLMResponse, error)
}

type GenerateOpts struct {
    ResponseSchema  interface{}   // JSON schema for structured output
    PromptName      string        // for token tracking
    ModelSize       ModelSize     // medium (default) or small (classification)
    MaxTokens       int
    Temperature     float64
}

type LLMResponse struct {
    Content    json.RawMessage
    TokenUsage TokenUsage
    Cached     bool
    Provider   string
    Model      string
}

type Message struct {
    Role    string // system, user, assistant
    Content string
}
```

### 5.2 Provider Implementations

| Provider | Adapter | Models | Features |
|----------|---------|--------|----------|
| **OpenAI** | `openai.go` | GPT-4o (medium), gpt-4o-mini (small) | Structured output, function calling |
| **Anthropic** | `anthropic.go` | Claude 3.5 Sonnet | Tool use for structured output |
| **Gemini** | `gemini.go` | Gemini 2.0 Flash | Native structured output |
| **Groq** | `groq.go` | Llama 3.1 70B | Fast inference |
| **Bifrost** | `bifrost.go` | Via Bifrost gateway | Unified provider access |
| **Generic** | `generic.go` | Any OpenAI-compatible | Ollama, LM Studio, vLLM |

### 5.3 Resilience

```go
// Built-in retry with exponential backoff
type RetryConfig struct {
    MaxAttempts    int           // default: 4
    InitialDelay   time.Duration // default: 5s
    MaxDelay       time.Duration // default: 120s
    RetryableHTTP  []int         // 429, 500, 502, 503, 504
}

// Built-in response cache (Redis)
type LLMCache interface {
    Get(ctx context.Context, key string) (*LLMResponse, error)
    Set(ctx context.Context, key string, resp *LLMResponse, ttl time.Duration) error
}
// Cache key = MD5(provider + model + messages_json)
```

### 5.4 Input Sanitization

```go
// Strip zero-width chars, control chars, invalid Unicode
// Applied to all LLM inputs before sending
func SanitizeInput(text string) string
```

---

## 6. Prompt Management

### 6.1 Prompt Registry

```go
// internal/adapter/prompt/registry.go

type PromptRegistry struct {
    templates map[string]PromptTemplate
}

type PromptTemplate struct {
    Name     string
    Version  int
    System   string          // system prompt template
    User     func(ctx PromptContext) string  // user prompt builder
    Schema   interface{}     // expected response JSON schema
}

// Registry entries (mapped from Python prompts/)
var DefaultPrompts = map[string]PromptTemplate{
    "extract_nodes":            extractNodesPrompt,
    "extract_edges":            extractEdgesPrompt,
    "extract_nodes_and_edges":  extractCombinedPrompt,
    "dedupe_nodes":             dedupeNodesPrompt,
    "dedupe_edges":             dedupeEdgesPrompt,
    "summarize_nodes":          summarizeNodesPrompt,
    "summarize_sagas":          summarizeSagasPrompt,
}
```

### 6.2 Multilingual Support

```go
// Auto-append language instruction based on content language detection
// or per group_id configuration
func AppendLanguageInstruction(messages []Message, groupID string) []Message
```

---

## 7. Embedder Architecture

```go
// internal/adapter/client/embedder/client.go

type EmbedderClient interface {
    Create(ctx context.Context, input string) ([]float32, error)
    CreateBulk(ctx context.Context, inputs []string) ([][]float32, error)
    Dimensions() int
}

// Embeddings generated for:
// - EntityNode.name_embedding
// - EntityEdge.fact_embedding
// - CommunityNode.name_embedding
// - Search query embedding
```

---

## 8. Token Usage Tracking

```go
// internal/infra/telemetry/token_tracker.go

type TokenTracker struct {
    mu     sync.RWMutex
    usage  map[string]*TokenUsage  // key: prompt_type
}

type TokenUsage struct {
    PromptTokens     int64
    CompletionTokens int64
    TotalTokens      int64
    CallCount        int64
}

// Aggregated per prompt type: extract_nodes, dedupe_edges, etc.
// Exposed via GetTokenUsage RPC
```

---

## 9. Configuration

```yaml
# config/knowledge.yaml
server:
  grpc_port: 9003
  max_concurrent_llm: 20              # semaphore for LLM calls
  max_concurrent_community: 10

llm:
  provider: "openai"                   # openai | anthropic | gemini | groq | bifrost | generic
  model: "gpt-4o"                      # medium model
  small_model: "gpt-4o-mini"          # small model (classification)
  api_key: "${OPENAI_API_KEY}"
  base_url: ""                         # override for generic/ollama
  max_tokens: 4096
  temperature: 0.0

  # Bifrost mode (recommended for production)
  # provider: "bifrost"
  # base_url: "http://bifrost:8002/v1"

  retry:
    max_attempts: 4
    initial_delay: 5s
    max_delay: 120s

  cache:
    enabled: false                     # opt-in LLM response caching
    redis_url: "redis://redis:6379/2"
    ttl: 3600s

embedder:
  provider: "openai"                   # openai | gemini | voyage | bifrost
  model: "text-embedding-3-small"
  dimensions: 1536
  api_key: "${OPENAI_API_KEY}"

reranker:
  provider: "openai"                   # openai | gemini | local_bge
  model: "gpt-4o-mini"
  api_key: "${OPENAI_API_KEY}"

resolution:
  node_dedup_cosine_min_score: 0.6
  node_dedup_max_candidates: 15
  edge_dedup_cosine_min_score: 0.5

services:
  store:
    address: "graphiti-store:9004"
    timeout: 15s

events:
  nats_url: "nats://nats:4222"
  stream: "graphiti-knowledge"

telemetry:
  otel_endpoint: "otel-collector:4317"
  service_name: "graphiti-knowledge"
```

---

## 10. Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `knowledge_llm_calls_total` | Counter | provider, model, prompt_type, status | Total LLM calls |
| `knowledge_llm_duration_seconds` | Histogram | provider, prompt_type | LLM latency |
| `knowledge_llm_tokens_total` | Counter | prompt_type, token_type | Token usage |
| `knowledge_llm_cache_hits_total` | Counter | prompt_type | Cache hit count |
| `knowledge_embedding_calls_total` | Counter | provider, status | Embedding calls |
| `knowledge_embedding_duration_seconds` | Histogram | provider | Embedding latency |
| `knowledge_rerank_calls_total` | Counter | provider, status | Rerank calls |
| `knowledge_entities_extracted_total` | Counter | source | Entities extracted |
| `knowledge_edges_extracted_total` | Counter | source | Edges extracted |
| `knowledge_resolution_decisions` | Counter | decision_type | merge/new/contradiction |
| `knowledge_community_build_duration_seconds` | Histogram | group_id | Community build time |

---

## 11. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Single AI service** | Centralizes all LLM/embedding provider management, API key rotation, model versioning |
| **Bifrost adapter for production** | Unified provider access with failover, rate limiting, cost tracking |
| **Prompt as code (not templates)** | Go functions for prompt building — compile-time type safety, testable |
| **Structured output via JSON schema** | Provider-agnostic; OpenAI native, Anthropic via tool_use, Gemini native |
| **Two-phase entity resolution** | Fast deterministic first (no LLM cost), LLM only for ambiguous cases |
| **Semantic + LLM dedup** | Search finds candidates efficiently; LLM makes nuanced decisions |
| **Label Propagation in-memory** | Simple, deterministic, O(N) — no external graph algorithm dependency |
| **Token tracking per prompt type** | Enables cost attribution, optimization, and quota enforcement per tenant |
