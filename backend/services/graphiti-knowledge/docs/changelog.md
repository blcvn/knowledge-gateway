---
id: DOC-S07
service: graphiti-knowledge
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-knowledge — Changelog

All notable changes to this service will be documented in this file.

## [0.1.0] — 2026-05-10 — Initial Release

### Added

- **Domain Layer**: ExtractedEntity, ExtractedEdge, Resolution, DuplicateDecision types
- **Domain Layer**: PromptTemplate, TokenUsage, ModelConfig, EmbeddingVector types
- **Domain Layer**: CommunityNode, RerankRequest/Result types
- **Usecase Layer**: 7 core usecases (extract entities, resolve entities, extract edges, resolve edges, generate embedding, update community, rerank)
- **Adapter Layer**: Bifrost LLM client with circuit breaker + bulkhead
- **Adapter Layer**: Bifrost embedder with batch support
- **Adapter Layer**: graphiti-store read-only gRPC client
- **Adapter Layer**: 7 prompt templates for extraction/resolution tasks
- **gRPC Service**: 9 RPCs on port :9023
- **Infrastructure**: Viper config, Wire DI, OTel tracing, Prometheus metrics
- **Operations**: Health checks on :9096, Dockerfile

### Architecture

- Stateless LLM processing engine — no own database
- All AI calls routed through Bifrost multi-provider gateway
- Bulkhead pattern for concurrent LLM request limiting
- Same proto interface as graphiti-pipeline (swappable deployment)
