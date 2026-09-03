---
id: DOC-S07
service: cognee-app
version: 0.1.0
status: Active
created: 2026-05-12
updated: 2026-05-12
format: Keep a Changelog (keepachangelog.com)
---

# Changelog — Cognee App

All notable changes to this app will be documented in this file.

## [0.1.0] — 2026-05-12

### Added
- **TASK-001**: Go module setup with unified config (ENV-first loading, validation)
- **TASK-002**: In-process event bus replacing NATS (Bus, IngestionPublisher, CognifyPublisher)
- **TASK-003**: REST handlers as thin adapters (ingestion, cognify, search, health)
- **TASK-004**: chi/v5 router with structured logging middleware
- **TASK-005**: main.go entry point with graceful shutdown
- **TASK-006**: Dockerfile (multi-stage), Makefile, config.yaml
- **TASK-007**: Service documentation (README, architecture, API, config, runbook)
