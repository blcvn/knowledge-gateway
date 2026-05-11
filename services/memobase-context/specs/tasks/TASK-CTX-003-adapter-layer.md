---
id: TASK-CTX-003
title: "Adapter Layer Implementation"
status: Done
created: 2026-05-11
---

# Task: Implement Adapter Layer

## 1. Objective
Implement the Adapter Layer (Layer 3) to integrate internal usecases with external infrastructure (PostgreSQL, pgvector, Redis, NATS, and gRPC).

## 2. Requirements

### 2.1. Database Adapters (PostgreSQL + pgvector)
- **Profile Repository (`internal/adapter/repository/postgres/profile_repo.go`)**:
  - Read-only SQL queries filtering by `user_id` and `project_id`.
- **Event Gist Repository (`internal/adapter/repository/postgres/event_gist_repo.go`)**:
  - Implement pgvector cosine similarity search using config threshold (default `< 1 - 0.2` distance).
  - Apply config time window filter (`EVENT_SEARCH_WINDOW_DAYS`, default 21 days).
  - Apply result limit (`EVENT_SEARCH_TOPK`, default 10).

### 2.2. Cache Adapter (Redis)
- **Profile Cache (`internal/adapter/repository/redis/profile_cache.go`)**:
  - Implement `GET`, `SET` with JSON serialization.
  - Use cache key pattern `user_profiles::{project_id}::{user_id}` and TTL defined by config (`PROFILE_CACHE_TTL`, default 1200 seconds).

### 2.3. Event Adapter (NATS)
- **NATS Subscriber (`internal/adapter/event/nats_subscriber.go`)**:
  - Subscribe to `memobase.profile.changed` and `memobase.engine.completed`.
  - Trigger Redis cache invalidation (`DEL`) for updated profiles.

### 2.4. gRPC Handler
- **gRPC API (`internal/adapter/grpc/handler.go`)**:
  - Implement `GetContext`, `GetProfiles`, `SearchProfiles` RPCs.
  - Extract tenant context (`x-tenant-id`) from metadata.
  - Map core errors to standard gRPC error codes: `NOT_FOUND` (404), `INVALID_ARGUMENT` (400), `INTERNAL` (500), `UNAVAILABLE` (503).

## 3. Acceptance Criteria
- [x] Postgres repositories implemented with pgvector support and correct topK limits.
- [x] Redis cache adapter functional with configurable TTLs.
- [x] NATS consumer successfully invalidates cache keys.
- [x] gRPC handlers and DTO mappers fully implemented with correct error mappings.
