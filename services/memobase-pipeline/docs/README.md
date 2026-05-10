# memobase-pipeline — Profile Memory Ingestion + Engine Service

> **Service**: `memobase-pipeline` | **gRPC Port**: 9031 | **Health**: 9098  
> **Origin**: Consolidated from memobase-ingestion + memobase-engine  
> **Status**: Proposed | **Version**: 0.1.0

---

## Purpose

Unified ingestion and profile extraction service for the Memobase profile memory engine. Manages the Buffer Zone FSM (token-aware batching) and YOLO merge pipeline (3 fixed LLM calls for cost-predictable extraction). Converts raw conversational data into structured user profiles with event gists and topic summaries.

## Business Capability

- **Buffer Zone FSM**: Token-aware batching (IDLE → ACCUMULATING → READY → PROCESSING → DONE)
- **YOLO Merge**: 3 fixed LLM calls per flush — cost-predictable profile extraction
- **Profile Management**: Structured profiles with topics, preferences, behavioral patterns
- **Event Gist Generation**: Summarized session events for timeline
- **Token Economics**: Precise token counting with configurable threshold (default: 1024)

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| RPC | gRPC (MemobaseIngestionService) |
| Database | PostgreSQL 17 + pgvector |
| Cache | Redis 7+ (buffer state) |
| Async | NATS JetStream |
| LLM | Bifrost (YOLO merge: 3 calls per flush) |

## Quick Start

```bash
cd services/memobase-pipeline
go run cmd/server/main.go
# gRPC: :9031 | Health: :9098
```

## Links

- [Architecture](./architecture.md)
- [Changelog](./changelog.md)

## Owner

Memobase Engine Team
