---

## [DEPRECATED] - 2026-05-10


id: DOC-S07
service: zep-thread
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-thread — Changelog

## [Unreleased]

## [1.1.0] - 2026-05-10

### Added
- Complete gRPC API with 8 RPCs (Create, Get, Update, Upsert, End, List, ListOrdered, ListUserSessions)
- Advisory lock mechanism with configurable retry policy
- NATS events: session.created, session.ended, session.deleted
- Complete data model documentation with indexes and advisory lock details

## [1.0.0] - 2026-05-09

### Added
- Initial service scaffold with Clean Architecture structure
- Sessions table with JSONB metadata and soft deletes
- Session state management via `ended_at` field
