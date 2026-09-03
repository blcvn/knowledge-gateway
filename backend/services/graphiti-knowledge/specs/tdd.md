---
id: TDD-graphiti-knowledge
title: Technical Design — graphiti-knowledge
service: graphiti-knowledge
version: 3.0.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Graphiti
linked_sol: SOL-001
---

# Technical Design — graphiti-knowledge

> **Group**: Graphiti | **gRPC Port**: 9023 | **Health Port**: 9096

## 1. Service Overview

Standalone LLM-intensive processing engine: entity/edge extraction, entity/edge resolution (deduplication), embedding generation, community detection + summarization, and cross-encoder neural reranking. All AI interactions routed through Bifrost multi-provider gateway. **Stateless** — reads from graphiti-store, produces extracted data returned to caller.

**Key Characteristics:**
- Stateless: reads from graphiti-store, no own database
- 7 core processing pipelines, each backed by a prompt template
- Bifrost gateway for ALL LLM calls (multi-model failover, token management)
- Bulkhead pattern for concurrent LLM request limiting
- Token usage tracking per model per request
- Same proto as graphiti-pipeline → swappable deployment

## 2. Clean Architecture Layers

### 2.1 Domain Layer

```
internal/domain/
├── entity.go          # ExtractedEntity, ExtractedEdge, Resolution, DuplicateDecision
├── value_object.go    # PromptTemplate, TokenUsage, ModelConfig, EmbeddingDimension
├── embedding.go       # EmbeddingVector, EmbeddingRequest, EmbeddingResult
├── community.go       # CommunityNode, CommunityMember, CommunityLevel
├── rerank.go          # RerankRequest, RerankResult, CrossEncoderScore
└── errors.go          # ErrLLMTimeout, ErrPromptTooLong, ErrProviderUnavailable, ErrMalformedResponse
```

### 2.2 Usecase Layer

```
internal/usecase/
├── extract_entities.go    # Content → LLM prompt → parse → validate entities
├── resolve_entities.go    # Search similar → LLM compare → merge/create decision
├── extract_edges.go       # Episode + entities → LLM → temporal fact triples
├── resolve_edges.go       # Find contradictions → invalidate old → decide new
├── generate_embedding.go  # Text → embedder → vector (single + batch)
├── update_community.go    # Label propagation → LLM summarize clusters
├── rerank.go              # Cross-encoder neural reranking
├── port/
│   ├── input.go           # ExtractUseCase, ResolveUseCase, EmbedUseCase, RerankUseCase
│   └── output.go          # LLMClient, EmbedderClient, GraphReader (read-only store client)
└── dto/
    ├── request.go
    └── response.go
```

**Extraction Flow:**
```
Content → BuildPrompt(template, vars) → LLM.Complete(prompt)
       → ParseJSON(response) → Validate(entities) → Return
```

**Resolution Flow:**
```
ExtractedEntity → Embed(name) → Store.FindSimilar(embedding, threshold=0.85)
              → if similar: LLM.Compare(extracted, existing) → merge/create
              → if none:    create new entity
```

### 2.3 Adapter Layer

```
internal/adapter/
├── grpc/
│   ├── handler.go         # GraphitiKnowledgeService — 9 RPCs
│   └── mapper.go          # Proto ↔ Domain bidirectional mapping
├── client/
│   └── store_client.go    # gRPC → graphiti-store:9024 (read-only)
├── llm/
│   ├── bifrost_client.go  # HTTP → Bifrost /v1/chat/completions
│   ├── prompt_registry.go # 7 prompt templates + variable interpolation
│   └── response_parser.go # JSON extraction from markdown LLM responses
├── embedder/
│   └── bifrost_embedder.go # HTTP → Bifrost /v1/embeddings (single + batch)
└── event/
    └── nats_publisher.go  # Publish entity.resolved, community.rebuilt events
```

### 2.4 Infrastructure Layer

```
internal/infra/
├── config/config.go       # LLM, embedder, store, NATS config
├── server/grpc.go         # gRPC server with interceptors
├── telemetry/             # OTel tracer + Prometheus metrics
└── wire/wire.go           # Wire DI providers
```

## 3. gRPC API

```protobuf
service GraphitiKnowledgeService {
  rpc ExtractEntities(ExtractEntitiesRequest) returns (ExtractEntitiesResponse);
  rpc ResolveEntities(ResolveEntitiesRequest) returns (ResolveEntitiesResponse);
  rpc ExtractEdges(ExtractEdgesRequest) returns (ExtractEdgesResponse);
  rpc ResolveEdges(ResolveEdgesRequest) returns (ResolveEdgesResponse);
  rpc GenerateEmbedding(GenerateEmbeddingRequest) returns (GenerateEmbeddingResponse);
  rpc GenerateEmbeddingBulk(GenerateEmbeddingBulkRequest) returns (GenerateEmbeddingBulkResponse);
  rpc Rerank(RerankRequest) returns (RerankResponse);
  rpc UpdateCommunity(UpdateCommunityRequest) returns (UpdateCommunityResponse);
  rpc GetTokenUsage(GetTokenUsageRequest) returns (GetTokenUsageResponse);
}
```

## 4. Prompt Template Architecture

### Template Registry

7 templates loaded at startup from embedded Go files. Each template defines:
- **System prompt**: Role + output format instructions
- **User prompt**: Template with Go text/template syntax (`{{.Content}}`)
- **Model**: Which LLM model to use (gpt-4o for extraction, gpt-4o-mini for resolution)
- **Max tokens**: Response length limit
- **Response schema**: Expected JSON output structure

### Template → Model Mapping

| Template | Model | Avg Tokens | Latency (p95) |
|----------|-------|------------|--------------|
| extract_entities | gpt-4o | ~2000 | ~3s |
| resolve_entities | gpt-4o-mini | ~500 | ~1s |
| extract_edges | gpt-4o | ~2500 | ~4s |
| resolve_edges | gpt-4o-mini | ~600 | ~1s |
| summarize_community | gpt-4o-mini | ~800 | ~2s |
| classify_entity | gpt-4o-mini | ~200 | ~0.5s |
| expand_summary | gpt-4o-mini | ~400 | ~1s |

## 5. Resilience Patterns

| Pattern | Implementation | Config |
|---------|---------------|--------|
| **Circuit Breaker** | gobreaker | Open after 5 failures, 30s timeout |
| **Bulkhead** | Semaphore channel | LLM_MAX_CONCURRENT (default 10) |
| **Retry** | Exponential backoff | 3x on 429/503, base=1s |
| **Timeout** | Context deadline | LLM_TIMEOUT (default 60s) |

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Operations |
|---------|-----------|----------|-----------|
| graphiti-store | Outbound gRPC | :9024 | FindSimilarEntities, GetEntityByName (read-only) |
| Bifrost | Outbound HTTP | LLM gateway | Completion + embedding generation |
| NATS | Outbound | JetStream | Publish entity.resolved, community.rebuilt |

## 7. Observability

- **Metrics**: `graphiti_knowledge_llm_duration_seconds{template,model}`, `graphiti_knowledge_tokens_total{model,type}`, `graphiti_knowledge_extraction_entities_count`, `graphiti_knowledge_resolution_merge_ratio`, `graphiti_knowledge_bulkhead_active`, `graphiti_knowledge_parser_success_ratio`
- **Traces**: OTel span per usecase + LLM call (with model, template, tokens attributes)
- **Logs**: Structured JSON: request_id, tenant_id, template, model, tokens, duration
- **Health**: gRPC health + HTTP /healthz, /readyz on :9096

## Feature Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| FEAT-KNW-001 | Domain layer | ⏳ Draft | P0 |
| FEAT-KNW-002 | Usecase layer (7 usecases) | ⏳ Draft | P0 |
| FEAT-KNW-003 | gRPC handlers | ⏳ Draft | P0 |
| FEAT-KNW-004 | Bifrost LLM client | ⏳ Draft | P0 |
| FEAT-KNW-005 | Bifrost embedder | ⏳ Draft | P0 |
| FEAT-KNW-006 | Store reader client | ⏳ Draft | P0 |
| FEAT-KNW-007 | Prompt registry (7 templates) | ⏳ Draft | P0 |
| FEAT-KNW-008 | Infrastructure | ⏳ Draft | P0 |

---

> **Next Steps**: Implement FEAT specs from SOL-001 in dependency order.
