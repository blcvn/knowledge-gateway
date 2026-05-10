---

## [DEPRECATED] - 2026-05-10

### Deprecated
- This service has been consolidated into `graphiti-pipeline` per SOL-001
- All domain logic, gRPC handlers, and NATS subscribers moved to `graphiti-pipeline`
- This service directory will be archived after Phase 4 completion

id: DOC-S07
service: graphiti-knowledge
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-knowledge — Changelog

Format: [Keep a Changelog](https://keepachangelog.com/)

## [Unreleased]
### Added
### Changed
### Fixed

## [1.0.0] - 2026-05-09
### Added
- Initial service scaffold with 4-layer Clean Architecture
- gRPC API implementation
- NATS event integration
- OTel tracing + Prometheus metrics
- Health check endpoints
- Docker + Docker Compose support
