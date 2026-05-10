---

## [DEPRECATED] - 2026-05-10

### Deprecated
- This service has been consolidated into `zep-core` per SOL-001
- All domain logic, gRPC handlers, and NATS subscribers moved to `zep-core`
- This service directory will be archived after Phase 4 completion

id: DOC-S07
service: zep-user
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-user — Changelog

All notable changes to this service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
### Changed
### Fixed

## [1.1.0] - 2026-05-10

### Added
- Complete gRPC API with 7 RPCs (CreateUser, GetUser, UpdateUser, DeleteUser, ListAllUsers, ListAllOrderedUsers, ListUserSessions)
- NATS event publishing for user lifecycle (created, updated, deleted)
- JSONB metadata merge-patch with advisory lock protection
- Partial indexes for soft-delete filtering
- Full Clean Architecture layout documentation

### Changed
- Updated gRPC port mapping to 9061 (aligned with architecture spec)
- Health port updated to 12061

## [1.0.0] - 2026-05-09

### Added
- Initial service scaffold
- Basic Clean Architecture structure (domain, usecase, adapter, infra)
- PostgreSQL users table with JSONB metadata
- Soft delete pattern with `deleted_at`
- Project-level isolation via `project_uuid`
