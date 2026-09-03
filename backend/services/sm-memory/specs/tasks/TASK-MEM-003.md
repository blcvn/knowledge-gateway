---
id: TASK-MEM-003
title: Data Models & Repositories
service: sm-memory
status: Done
priority: P0
created: 2026-05-11
---

# Data Models & Repositories

## Objective
Implement the storage and persistence adapters.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-memory
title: Technical Design — sm-memory
service: sm-memory
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-memory

> **Group**: Supermemory | **gRPC Port**: 9072 | **Health Port**: 9117

> **🚨 DEPRECATION NOTICE**: This specification is obsolete. The service has been merged into `sm-engine` (Ref: [ARCH-007-merge-sm-engine]).


## 1. Service Overview

Memory engine: LLM-powered fact extraction from documents, knowledge graph construction with versioned memory chains, Ebbinghaus forgetting curve decay, and memory relation management (updates/extends/derives).

## 2. Clean Architecture Layers

### Domain Layer (Layer 1)
- **MemoryEntry**: id, memory (text), space_id, org_id, user_id, version, is_latest, parent_memory_id, root_memory_id, memory_relations (map[id]→relation_type), source_count, is_inference, is_forgotten, is_static, forget_after, forget_reason, memory_embedding, metadata, created_at, updated_at
- **MemoryRelation**: enum — `updates` | `extends` | `derives`
- **MemoryDocumentSource**: memory_entry_id, document_id, relevance_score (0-100), metadata, added_at
- **ForgettingCurve**: Ebbinghaus decay function — `ShouldForget(now) = age > decayThreshold(accessCount * relevanceScore)`
- **Events**: MemoryCreated, MemoryForgotten

### Usecase Layer (Layer 2)
- **ExtractMemoriesUseCase**: Receive document chunks → LLM extract facts → deduplicate → create memory entries with source links
- **CreateMemoryUseCase**: validate → embed → persist → check for existing similar memories → create relations if overlap
- **ForgetMemoryUseCase**: apply forgetting curve → mark is_forgotten=true → emit event
- **GetMemoryWithContextUseCase**: fetch memory + parent chain (up to 3 ancestors) + child chain (up to 3 descendants)

### Adapter Layer (Layer 3)
- **gRPC handler**: CreateMemory, GetMemory, UpdateMemory, ForgetMemory, ListMemories, CreateRelation, GetMemoryWithContext
- **PostgreSQL repos**: MemoryEntryRepository, MemoryDocumentSourceRepository
- **NATS**: subscribe `sm.document.created`, publish `sm.memory.created`, `sm.memory.forgotten`

### Infrastructure Layer (Layer 4)
- Config, Server, Wire, Telemetry, Bifrost client (LLM extraction + embedding)

## 3. gRPC API

```protobuf
service SmMemoryService {
  rpc CreateMemory(CreateMemoryRequest) returns (Memory);
  rpc GetMemory(GetMemoryRequest) returns (Memory);
  rpc UpdateMemory(UpdateMemoryRequest) returns (Memory);
  rpc ForgetMemory(ForgetMemoryRequest) returns (google.protobuf.Empty);
  rpc ListMemories(ListMemoriesRequest) returns (ListMemoriesResponse);
  rpc CreateRelation(CreateRelationRequest) returns (Relation);
  rpc GetMemoryWithContext(GetContextRequest) returns (MemoryWithContext);
}
```

## 4. Forgetting Curve

```go
func (m *MemoryEntry) ShouldForget(now time.Time) bool {
    if m.IsStatic { return false }
    if m.ForgetAfter != nil && now.After(*m.ForgetAfter) { return true }
    age := now.Sub(m.UpdatedAt)
    strength := float64(m.SourceCount) * relevanceAvg(m.Sources)
    return age > decayThreshold(strength)
}
// decayThreshold: higher strength → longer retention
// strength=1 → 24h, strength=5 → 7d, strength=10 → 30d
```

## 5. Memory Versioning

```
[v1] ←updates― [v2] ←updates― [v3 (is_latest=true)]
        ↖extends― [v2a] (branch)
```

- `root_memory_id` links all versions to the original
- `parent_memory_id` links to immediate predecessor
- `is_latest=true` only on the newest version in each chain

## 6. NATS Events

| Direction | Subject | Payload |
|-----------|---------|---------|
| Subscribe | `sm.document.created` | `{id, org_id, type, chunk_count}` |
| Publish | `sm.memory.created` | `{id, org_id, space_id, memory_text}` |
| Publish | `sm.memory.forgotten` | `{id, org_id, reason}` |

## 7. Data Model

### Tables
- `memory_entries`: id(PK), memory(TEXT), space_id(FK), org_id, user_id, version, is_latest, parent_memory_id, root_memory_id, memory_relations(JSONB), source_count, is_inference, is_forgotten, is_static, forget_after, forget_reason, memory_embedding(VECTOR(1536)), metadata(JSONB), created_at, updated_at
- `memory_document_sources`: memory_entry_id(FK), document_id(FK), relevance_score(INT 0-100), metadata(JSONB), added_at — composite PK

### Key Indexes
- `idx_memory_org_space` (org_id, space_id, is_latest, is_forgotten) — list queries
- `idx_memory_root` (root_memory_id) — version chain traversal
- `idx_memory_embedding` HNSW — similarity search for dedup
- `idx_memory_source` (document_id) — source lookups

## 8. Observability

- **Metrics**: memory_created_total, memory_forgotten_total, extraction_latency, forgetting_curve_evaluations
- **Traces**: OTel spans for extraction pipeline, embedding, relation creation
- **Health**: gRPC + HTTP /healthz on port 9117

## 9. Multi-Tenancy

`org_id` + `space_id` isolation. Container tags mapped to space_id for API compatibility.

---

> **Next Steps**: FEAT-001 (Memory Extraction Pipeline), FEAT-002 (Forgetting Curve), FEAT-003 (Version Chain Management), ARCH-001 (Memory Graph Relations)

```

## Acceptance Criteria
- [x] Database schema / migrations created.
- [x] Repository implementations accurately query the data models.
