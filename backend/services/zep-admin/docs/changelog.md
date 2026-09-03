---

## [DEPRECATED] - 2026-05-10


id: DOC-S07
service: zep-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-admin — Changelog

## [1.1.0] - 2026-05-10

### Added
- Complete gRPC API with 11 RPCs (Health, Project CRUD, API Key lifecycle, Schema migration)
- Parallel health aggregation across 5 Zep domain services
- API Key management with SHA-256 hashing and prefix identification
- Project settings (rate limit, request timeout, telemetry, graphiti)
- NATS events for project lifecycle (created, deleted)
- Complete data model with projects and api_keys tables

## [1.0.0] - 2026-05-09

### Added
- Initial service scaffold
- Basic project entity
