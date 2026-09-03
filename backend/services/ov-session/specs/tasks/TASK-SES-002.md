---
id: TASK-SES-002
title: Infrastructure and Persistence Layer Setup
status: Done
---

# Task: Infrastructure and Persistence Layer Setup

## Objective
Implement the Infrastructure Layer (Layer 4) including PostgreSQL persistence, configuration, dependency injection, and initial gRPC/NATS clients for `ov-session`.

## Requirements
1. **Database Schema & Migrations**:
   - Create migration scripts for PostgreSQL tables: `ov_sessions`, `ov_messages`, `ov_working_memory`, and `ov_extracted_memories` exactly as specified in `data-model.md`.
   - Ensure all constraints, indexes (`idx_sessions_account_user`, etc.) and default values are included.
2. **Repository Implementations** (`internal/infra/persistence/`):
   - Implement `SessionRepository` for PostgreSQL (`session_repo.go`).
   - Implement `MessageRepository` for PostgreSQL (`message_repo.go`).
3. **Infrastructure Setup** (`internal/infra/`):
   - Set up `config/config.go` for environment variables and service configurations.
   - Scaffold Wire dependency injection (`wire/wire.go`).
4. **Adapter Scaffolding** (`internal/adapter/`):
   - Create initial client stubs for `fs_client.go` (ov-fs gRPC) and `llm_client.go` (Bifrost LLM).
   - Create initial publisher stub for `publisher.go` (NATS event publisher).

## Acceptance Criteria
- [x] Database migrations execute successfully against PostgreSQL.
- [x] Repository implementations fulfill the domain interfaces using SQL/ORM correctly mapping to the JSONB fields.
- [x] Wire dependency injection resolves properly for the repository and config layers.
