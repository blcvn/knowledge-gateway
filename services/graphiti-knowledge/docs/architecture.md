---
id: DOC-S03
service: graphiti-knowledge
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-knowledge — Service Architecture

> **Group**: Graphiti | **Pattern**: 4-layer Clean Architecture | **Stateless LLM Processor**

## Layer Structure

```
services/graphiti-knowledge/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # ExtractedEntity, ExtractedEdge, Resolution
│   │   ├── value_object.go        # EntityType, EdgeType, DuplicateDecision
│   │   ├── event.go               # EntityResolved, CommunityRebuilt
│   │   └── errors.go              # ErrLLMTimeout, ErrInvalidSchema
│   ├── usecase/
│   │   ├── extract_entities.go    # Entity extraction orchestration
│   │   ├── resolve_entities.go    # Entity deduplication against graph
│   │   ├── extract_edges.go       # Edge/relationship extraction
│   │   ├── resolve_edges.go       # Edge dedup + fact invalidation
│   │   ├── generate_embedding.go  # Vector embedding generation
│   │   ├── rerank.go              # Cross-encoder reranking
│   │   ├── update_community.go    # Community detection + summary
│   │   └── port/
│   │       ├── input.go           # Use case interfaces
│   │       └── output.go          # LLMClient, EmbedderClient, CrossEncoder, GraphReader
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC server (7 RPCs)
│   │   │   └── mapper.go          # Proto ↔ Domain
│   │   ├── client/
│   │   │   └── store_client.go    # Read existing entities/edges for resolution
│   │   ├── llm/
│   │   │   ├── bifrost_client.go  # LLM via Bifrost gateway
│   │   │   ├── embedder.go        # Embedding via Bifrost
│   │   │   └── reranker.go        # Cross-encoder via Bifrost
│   │   └── event/
│   │       └── nats_publisher.go  # Publish resolution/community events
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       ├── prompt/                # LLM prompt templates
│       │   ├── extract_entities.go
│       │   ├── resolve_entities.go
│       │   ├── extract_edges.go
│       │   └── summarize_community.go
│       └── wire/wire.go
```

## Design Decisions

- **Stateless**: No persistent state — pure LLM processing engine
- **Bifrost gateway**: All LLM/embedding calls routed through Bifrost for model-agnostic abstraction
- **Structured outputs**: JSON Schema-constrained LLM responses ensure predictable parsing
- **Bulkhead pattern**: Semaphore-limited concurrent LLM calls to prevent overload
- **Prompt templates**: Externalized prompt library supporting EN/ZH localization

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| Bifrost (LLM Gateway) | GPT-4o/4o-mini, text-embedding-3-large, Cross-Encoder |
| graphiti-store (gRPC) | Read existing entities/edges for resolution context |
| NATS JetStream | Publish entity.resolved, community.rebuilt events |
