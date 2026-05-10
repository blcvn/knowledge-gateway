---

## [DEPRECATED] - 2026-05-10

### Deprecated
- This service has been consolidated into `sm-engine` per SOL-001
- All domain logic, gRPC handlers, and NATS subscribers moved to `sm-engine`
- This service directory will be archived after Phase 4 completion

id: DOC-S07
service: sm-memory
version: 1.0.0
status: Active
created: 2026-05-09
updated: 2026-05-09
format: Keep a Changelog (keepachangelog.com)
---

# Changelog — sm-memory

All notable changes to this service will be documented in this file.

## [Unreleased]

### Added
- Initial service scaffold with Clean Architecture
- docs/ and specs/ structure per VNP Memory standards

## [1.0.0] - 2026-05-09

### Added
- Service initialization
- gRPC server skeleton on port 9072
- Health check endpoint on port 12072
