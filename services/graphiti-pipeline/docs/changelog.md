---
id: DOC-S07
service: graphiti-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# graphiti-pipeline — Changelog

All notable changes to this service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
### Changed
### Fixed

## [0.1.0] - 2026-05-10

### Added
- Initial service structure with Clean Architecture 4-layer layout
- Consolidated GraphitiIngestionService + GraphitiKnowledgeService in single binary
- Saga orchestrator for 6-step episodic ingestion pipeline
- Entity extraction via Bifrost LLM gateway
- Entity resolution with search + LLM deduplication
- Edge extraction with temporal fact triple support
- Edge resolution with contradiction detection and bi-temporal invalidation
- Embedding generation via configurable providers
- Community detection with label propagation + LLM summarization
- Cross-encoder reranking support
- Per-group_id serialization for consistency
- PostgreSQL saga state persistence
- NATS JetStream event publishing
- gRPC health checks + HTTP health/readyz/livez
- OTel tracing + Prometheus metrics + structured JSON logging
- Circuit breaker for graphiti-store gRPC client
- Docker multi-stage build (distroless runtime)
