---

## [DEPRECATED] - 2026-05-10


id: DOC-S07
service: graphiti-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-ingestion — Changelog

All notable changes to this service will be documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]
### Added
### Changed
### Fixed

## [1.0.0] - 2026-05-09
### Added
- Initial service scaffold with Clean Architecture (4-layer)
- gRPC API: `IngestEpisode`, `BulkIngest`, `GetEpisodeStatus`
- Saga orchestrator with 7-step pipeline
- Per-group serialization for consistency
- Compensating actions for pipeline failures
- NATS event publishing (`graphiti.episode.ingested`)
- PostgreSQL-based saga state persistence
- OTel tracing integration
- Prometheus metrics
- Health check endpoints (gRPC + HTTP)
- Docker + Docker Compose support
