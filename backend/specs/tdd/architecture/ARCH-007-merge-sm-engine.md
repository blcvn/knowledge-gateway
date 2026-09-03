---
id: ARCH-007
title: Merge sm-document + sm-memory + sm-profile → sm-engine
service: sm-engine
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

`sm-document` creates documents → emits to `sm-memory` for fact extraction → `sm-memory` updates `sm-profile` with dynamic traits. Linear chain qua NATS giữa 3 services trong cùng engine. All share PostgreSQL + pgvector.

## Kiến Trúc Mới

```
services/sm-engine/
├── internal/
│   ├── domain/
│   │   ├── document/   # Document, Chunk, ContentExtraction
│   │   ├── memory/     # Memory, Relation, ForgettingCurve
│   │   └── profile/    # Profile, StaticPreference, DynamicTrait
│   ├── usecase/
│   │   ├── document/   # CreateDocument, GetChunks (triggers memory locally)
│   │   ├── memory/     # CreateMemory, ForgetMemory, CreateRelation
│   │   └── profile/    # GetProfile, UpdateProfile, GetDynamicTraits
│   ├── adapter/grpc/
│   │   ├── document_handler.go   # SmDocumentService
│   │   ├── memory_handler.go     # SmMemoryService
│   │   └── profile_handler.go    # SmProfileService
│   └── infra/
```

**Key**: Document → memory → profile chain becomes local. External events: `sm.engine.document.created` (→ sm-search for indexing), `sm.engine.memory.created` (→ sm-search), `sm.engine.memory.forgotten` (→ sm-search).

## Acceptance Criteria

- [ ] AC-1: Document CRUD + chunking + content extraction functional
- [ ] AC-2: Memory engine with forgetting curve (Ebbinghaus decay) preserved
- [ ] AC-3: Profile updates from memory events (static + dynamic traits)
- [ ] AC-4: `sm.engine.*` events emitted to sm-search for index management
- [ ] AC-5: sm-connector can still trigger document creation via gRPC
