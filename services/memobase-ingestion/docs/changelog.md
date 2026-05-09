---
id: DOC-S07
service: memobase-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-ingestion — Changelog

All notable changes to this service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]
### Added
### Changed
### Fixed

## [1.1.0] - 2026-05-09
### Added
- Complete documentation suite (DOC-S01 through DOC-S07)
- gRPC API definition (InsertBlob, GetBufferStatus, FlushBuffer, DeleteBlob)
- Buffer Zone FSM specification (IDLE → PROCESSING → DONE/FAILED)
- NATS event publishing (`memobase.buffer.ready`)
- Token-aware buffer threshold (1024 tokens default)
- Architecture documentation with component diagram

## [1.0.0] - 2026-05-09
### Added
- Initial service scaffold
- Clean Architecture 4-layer structure
- Basic README and TDD stub
