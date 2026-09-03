---
id: ARCH-005
title: Merge zep-user + zep-thread + zep-memory → zep-core
service: zep-core
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_adr: ADR-0001
behavior_change: false
---

## Vấn Đề Kiến Trúc Hiện Tại

`zep-memory.PutMemory` makes synchronous gRPC call to `zep-thread.UpsertSession` trên mỗi request. `zep-user` và `zep-thread` share cùng PostgreSQL schema. 3 services có tight coupling nhưng chạy 3 binaries riêng, 3 gRPC ports.

## Kiến Trúc Mới

```
services/zep-core/
├── internal/
│   ├── domain/
│   │   ├── user/       # User entity, metadata (JSONB)
│   │   ├── thread/     # Thread, Session, ended_at
│   │   └── memory/     # Message, ContextAssembly
│   ├── usecase/
│   │   ├── user/       # CreateUser, UpdateUser, DeleteUser, ListUsers
│   │   ├── thread/     # CreateThread, EndThread, ListThreads
│   │   └── memory/     # PutMemory (calls thread locally), GetMemory (sub-200ms)
│   ├── adapter/grpc/
│   │   ├── user_handler.go      # ZepUserService
│   │   ├── thread_handler.go    # ZepThreadService
│   │   └── memory_handler.go    # ZepMemoryService
│   └── infra/
```

**Critical**: PutMemory sub-200ms path must NOT be degraded. Thread UpsertSession becomes local function call → should improve latency.

## Acceptance Criteria

- [ ] AC-1: `zep-core` registers ZepUserService + ZepThreadService + ZepMemoryService
- [ ] AC-2: PutMemory latency ≤ 200ms p95 (improved from cross-service)
- [ ] AC-3: GetMemory context assembly with fact overlay functional
- [ ] AC-4: `zep.memory.messages.ingested` NATS event still emitted → zep-graph
- [ ] AC-5: `zep.user.deleted` cascade event still emitted
- [ ] AC-6: JSONB metadata merge-patch with advisory locks preserved
