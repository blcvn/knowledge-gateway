# Memobase App — Architecture

> **Version**: 1.0.0 | **Status**: Draft | **Date**: 2026-05-12

## 1. Architecture Overview

Memobase App là **embedded monolith** chạy 4 microservices + gateway trong 1 Go process duy nhất.

```
                    ┌──── SINGLE PROCESS (memobase-app) ──────────────────────┐
                    │                                                          │
Client HTTP ──────→ │  [Gateway REST :8080]                                    │
                    │    ├─ Auth (JWT/API Key)                                 │
                    │    ├─ Rate Limiting (Redis)                              │
                    │    ├─ CORS, Circuit Breaker                              │
                    │    │                                                     │
                    │    ▼ gRPC localhost                                       │
                    │  [memobase-ingestion :9041]  ← Phase 0 (Data)           │
                    │    │ Blob insert, Buffer zone                            │
                    │    │ NATS → memobase.buffer.ready                        │
                    │    ▼                                                     │
                    │  [memobase-engine :9042]     ← Phase 1 (Intelligence)   │
                    │    │ Profile extraction, YOLO merge                      │
                    │    │ LLM calls: entry_summary + extract + merge          │
                    │    │ NATS → memobase.engine.completed                    │
                    │    ▼                                                     │
                    │  [memobase-context :9043]    ← Phase 2 (Application)    │
                    │    │ Context assembly, Profile caching (Redis)           │
                    │                                                          │
                    │  [memobase-pipeline :9044]   ← Phase 2 (Application)    │
                    │    │ Pipeline orchestration                               │
                    │                                                          │
MCP Clients ──────→ │  [MCP SSE :8082]                                        │
                    │                                                          │
K8s Probes ───────→ │  [Health :9090]  /healthz /readyz /status               │
                    │                                                          │
                    └──────────────────────────────────────────────────────────┘
                              │         │         │
                              ▼         ▼         ▼
                         PostgreSQL   Redis   NATS JetStream
                         (pgvector)
```

## 2. Supervisor Lifecycle

### 2.1 Phased Startup

```
Phase 0 (Data)         → memobase-ingestion   (blob storage, buffer zone)
    ↓ wait port 9041
Phase 1 (Intelligence) → memobase-engine       (LLM processing)
    ↓ wait port 9042
Phase 2 (Application)  → memobase-context      (read path, cache)
                       → memobase-pipeline     (orchestration)
    ↓ wait ports 9043, 9044
Phase 3 (Gateway)      → vnp-gateway           (REST + MCP)
    ↓ wait port 8080
ALL READY → /readyz returns 200
```

### 2.2 Ordered Shutdown (Reverse)

```
SIGTERM received →
  Phase 3: Stop gateway (drain HTTP connections)
  Phase 2: Stop context + pipeline
  Phase 1: Stop engine (finish LLM calls)
  Phase 0: Stop ingestion (flush buffers)
```

## 3. Data Flow

### 3.1 Insert Blob → Profile Extraction Pipeline

```
Client POST /api/v1/blobs/insert/{user_id}
  → Gateway (REST)
  → memobase-ingestion (gRPC InsertBlob)
    → Store blob in PostgreSQL
    → Add BufferZone entry (status: idle)
    → Check buffer capacity (token_sum > threshold?)
      → If full: NATS publish memobase.buffer.ready
  → memobase-engine (NATS subscriber)
    → Fetch blobs from DB
    → LLM Call #1: entry_chat_summary
    → LLM Call #2: extract_topics (parallel)
    → LLM Call #3: merge_yolo (parallel)
    → DB: Upsert profiles + events + embeddings
    → NATS: memobase.engine.completed
  → memobase-context (NATS subscriber)
    → Invalidate Redis profile cache
```

### 3.2 Get Context (Read Path)

```
Client GET /api/v1/users/context/{user_id}
  → Gateway (REST)
  → memobase-context (gRPC GetContext)
    → Redis cache check (profiles)
    → PostgreSQL fallback
    → Profile truncation algorithm
    → gRPC → memobase-event.SearchEventGists (if needed)
    → Assembly: profile_section + event_section
    → Return context string
```

## 4. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| gRPC localhost (not in-process calls) | Zero code changes to services; proven pattern |
| NATS for buffer pipeline | Same async semantics as microservice deployment |
| Phased startup | Dependencies: ingestion → engine → context → gateway |
| Supervisor per-goroutine panic recovery | Single service crash doesn't kill process |
| Redis profile cache (TTL 20min) | Low-latency read path for context assembly |

## 5. External Dependencies

| Dependency | Purpose | Required |
|-----------|---------|----------|
| PostgreSQL 17 + pgvector | Profile/event storage, vector search | ✅ |
| Redis 7+ | Profile caching, rate limiting | ✅ |
| NATS JetStream | Async pipeline events | ✅ |
| LLM Provider | Profile extraction, YOLO merge | ✅ (for engine) |

## 6. Known Limitations

- gRPC localhost adds ~0.1ms overhead per call vs in-process
- NATS server required as external dependency (cannot use in-process)
- Memory footprint higher than single microservice (all services in one process)
- All services share Go runtime (GC pauses affect all)
