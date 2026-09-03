# 08 — Graphiti Knowledge Service

> **gRPC**: 9023 | **Health**: 9097

---

## 1. Purpose

LLM-powered knowledge processing: entity extraction, entity resolution (deduplication), community detection + summarization, embedding generation. Central AI service cho Graphiti domain.

---

## 2. Clean Architecture

```
services/graphiti-knowledge/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # EntityCandidate, EdgeCandidate, Community, EmbeddingBatch
│   │   ├── value_object.go     # ExtractionMode, ResolutionStrategy
│   │   ├── event.go            # EntityResolvedEvent, CommunityRebuiltEvent
│   │   └── errors.go
│   ├── usecase/
│   │   ├── extract_entities.go     # LLM structured extraction
│   │   ├── resolve_entities.go     # Dedup entity candidates against existing
│   │   ├── extract_edges.go        # LLM relationship extraction
│   │   ├── invalidate_edges.go     # Temporal edge invalidation
│   │   ├── build_communities.go    # Louvain/Leiden community detection
│   │   ├── summarize_community.go  # LLM summary per community
│   │   ├── generate_embeddings.go  # Batch embedding generation
│   │   ├── rerank_results.go       # Cross-encoder reranking
│   │   ├── port/
│   │   │   └── output.go          # LLMClient, EmbedderClient, StoreClient, RerankerClient
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/handler.go     # graphiti.knowledge.v1.KnowledgeService impl
│   │   ├── client/
│   │   │   ├── llm_client.go       # Uses pkg/adapters/llm with structured output
│   │   │   ├── embedder_client.go  # Uses pkg/adapters/embedder
│   │   │   ├── reranker_client.go  # Uses pkg/adapters/reranker
│   │   │   └── store_client.go     # gRPC → graphiti-store
│   │   └── event/
│   │       └── publisher.go        # NATS: graphiti.entity.resolved, graphiti.community.rebuilt
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
```

---

## 3. Key Operations

### Entity Extraction (LLM)
```
ExtractEntities(episode_text, source_type) → EntityCandidate[]
  - JSON mode structured output
  - Schema: {name, type, description, properties}
  - Supports custom extraction prompts per domain
```

### Entity Resolution (LLM + Embedding)
```
ResolveEntities(candidates[], existing_entities[]) → ResolvedEntity[]
  - Step 1: Embedding similarity to find candidates
  - Step 2: LLM pairwise comparison for borderline cases
  - Step 3: Merge properties, update descriptions
  - Step 4: Track provenance (source episode IDs)
```

### Edge Invalidation (Temporal)
```
InvalidateEdges(new_edges[], existing_edges[]) → InvalidatedEdge[]
  - LLM determines if new facts contradict existing edges
  - Sets invalid_at timestamp on contradicted edges
  - Maintains full edge history (never hard-delete)
```

### Community Detection
```
BuildCommunities(group_id) → Community[]
  - Fetch full graph from store-svc
  - Run Louvain/Leiden algorithm locally
  - Hierarchical multi-level communities
  - LLM-generate summary per community
```

---

## 4. LLM Provider Abstraction

```go
// Uses pkg/adapters/llm with provider-specific configs
type LLMConfig struct {
    Provider       string  // openai, anthropic, openrouter, bifrost
    Model          string  // gpt-4o, claude-3-opus, etc.
    Temperature    float64
    MaxTokens      int
    StructuredMode bool    // JSON mode for extraction
}

// Token usage tracking for billing
type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
    Provider         string
    Model            string
    GroupID          string
    OperationType    string  // extract, resolve, summarize, rerank
}
```

---

## 5. NATS Events

| Subject | Direction | Payload |
|---------|-----------|---------|
| `graphiti.entity.resolved` | Publish | `{entity_ids[], group_id}` |
| `graphiti.community.rebuilt` | Publish | `{community_ids[], group_id}` |
