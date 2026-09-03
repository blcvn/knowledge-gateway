# zep-core — Context Engineering Core Service

> **Service**: `zep-core` | **gRPC Port**: 9061 | **Health**: 9110  
> **Origin**: Consolidated from zep-user + zep-thread + zep-memory  
> **Status**: Proposed | **Version**: 0.1.0

---

## Purpose

Unified core service for the Zep context engineering platform. Handles user management, thread/session lifecycle, and memory operations (message ingestion + sub-200ms context assembly). The critical hot path — PutMemory → context assembly with fact overlay — benefits from local function calls replacing cross-service gRPC.

## Business Capability

- **User Management**: CRUD with JSONB metadata, deletion cascade across threads/memories
- **Thread Lifecycle**: Create/end threads, session management with `ended_at` semantics
- **Memory Operations**: PutMemory (message ingestion), GetMemory (sub-200ms context assembly)
- **Context Assembly**: Fact-enriched context with priority-based overlay
- **Sub-200ms Hot Path**: Critical performance requirement for real-time context retrieval

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| RPC | gRPC (3 services: ZepUserService + ZepThreadService + ZepMemoryService) |
| Database | PostgreSQL 17 + pgvector |
| Cache | Redis 7+ (recent context cache) |
| Async | NATS JetStream |

## Quick Start

```bash
cd services/zep-core
go run cmd/server/main.go
# gRPC: :9061 | Health: :9110
```

## Links

- [Architecture](./architecture.md)
- [Changelog](./changelog.md)

## Owner

Zep Engine Team
