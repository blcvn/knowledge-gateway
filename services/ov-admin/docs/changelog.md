---

## [DEPRECATED] - 2026-05-10

### Deprecated
- This service has been consolidated into `vnp-platform` per SOL-001
- All domain logic, gRPC handlers, and NATS subscribers moved to `vnp-platform`
- This service directory will be archived after Phase 4 completion

id: DOC-S07
service: ov-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-admin — Changelog

All notable changes to this service will be documented here.
Format: [Keep a Changelog](https://keepachangelog.com/).

## [1.1.0] — 2026-05-09

### Added
- Complete documentation suite: README, API, Architecture, Data Model, Configuration, Runbook
- Full Technical Design Document (TDD) with Clean Architecture layer breakdown
- gRPC service definition with all endpoints documented
- NATS event contracts (published + subscribed)
- Data model with entity-relationship diagrams and index strategy
- Configuration reference with all environment variables
- Operational runbook with troubleshooting guide

### Changed
- Upgraded from skeleton docs (v1.0.0) to production-grade documentation (v1.1.0)

## [1.0.0] — 2026-05-09

### Added
- Initial service scaffold with skeleton documentation
- Basic directory structure following monorepo conventions
