# Changelog — Graphiti App

## v1.0.0 (2026-05-12)

### Added
- **Monolith Architecture**: Single binary embedding 5 Graphiti services + gateway
- **Supervisor Pattern**: 4-phase startup (Data→Intelligence→Application→Gateway) with reverse-order shutdown
- **Unified Config**: ENV-based configuration with `SetServiceEnvVars()` injection for embedded services
- **Health Aggregation**: Dedicated `/healthz` + `/readyz` endpoints on port 9090
- **Gateway**: REST API with 11 Graphiti routes + CORS + recovery + logging middleware
- **gRPC Registry**: Localhost-based proxy to embedded gRPC services
- **Deployment**: Multi-stage Dockerfile, Makefile with 7 targets, Docker Compose with Neo4j/Redis/NATS
- **Zero-change constraint**: Existing `services/graphiti-*` and `gateway/` code remains untouched

### Embedded Services
- `graphiti-store` (Phase 0, :9024) — Neo4j graph storage
- `graphiti-knowledge` (Phase 1, :9023) — LLM-powered knowledge extraction
- `graphiti-ingestion` (Phase 2, :9021) — Episode ingestion pipeline
- `graphiti-search` (Phase 2, :9022) — Semantic search + community search
- `graphiti-pipeline` (Phase 2, :9025) — Orchestration pipeline

### Infrastructure
- Neo4j 5 Community with APOC plugin
- Redis 7 (caching, rate limiting)
- NATS 2.10 with JetStream (async events)
