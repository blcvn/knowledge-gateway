---
id: DOC-S07
service: cognee-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# cognee-pipeline — Changelog

All notable changes to this service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial consolidated service merging cognee-ingestion + cognee-cognify
- Dual gRPC service registration (CogneeIngestionService + CogneeCognifyService)
- 8-stage cognify pipeline (classify → chunk → extract → dedup → build → embed → summarize)
- File upload with streaming gRPC (PDF/DOCX/PPTX/CSV/HTML/TXT)
- URL scraping + text extraction
- Dataset management (CRUD with tenant isolation)
- Local ingestion → cognify trigger (no NATS hop)
- PostgreSQL + pgvector for metadata + embeddings
- Neo4j knowledge graph construction
- MinIO file storage
- Bifrost LLM integration (entity extraction, dedup, summarization)
- NATS JetStream: `cognee.pipeline.completed` event publishing
- OTel observability (traces + metrics)
- gRPC + HTTP health checks
- Google Wire dependency injection
- Multi-stage Dockerfile

### Changed
- (none)

### Fixed
- (none)
