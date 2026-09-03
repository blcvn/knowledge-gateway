---
id: DOC-S07
service: memobase-context
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-context — Changelog

All notable changes to this service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]
### Added
### Changed
### Fixed

## [1.1.0] - 2026-05-09
### Added
- Complete documentation suite (DOC-S01 through DOC-S07)
- Context assembly algorithm with token budget allocation
- Redis caching strategy (TTL 20min, event-driven invalidation)
- Profile truncation with topic priority ordering
- Event gist semantic search via pgvector
- SLA targets (< 100ms p95 context retrieval)

## [1.0.0] - 2026-05-09
### Added
- Initial service scaffold
- Clean Architecture 4-layer structure
- Basic README and TDD stub
