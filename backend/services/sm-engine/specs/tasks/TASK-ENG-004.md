---
id: TASK-ENG-004
title: gRPC Handlers & Events
service: sm-engine
status: Done
priority: P0
created: 2026-05-11
---

# gRPC Handlers & Events

## Objective
Implement the external communication interfaces via gRPC and NATS.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-engine
title: Technical Design Document — sm-engine
service: sm-engine
version: 1.0.0
status: Ready
created: 2026-05-10
updated: 2026-05-11
linked_sol: SOL-001
linked_adr: ADR-0001
---

# sm-engine — Technical Design Document

## Origin

Consolidated service created per SOL-001 (Service Consolidation 35 → 18).
Combines `sm-document`, `sm-memory`, and `sm-profile` into a single in-process workflow.

## 1. Service Overview

`sm-engine` is the core of the Supermemory adaptive knowledge graph. It handles:
- **Document Management**: Document CRUD, format-aware chunking (PDF, HTML, Text, Image), and content extraction.
- **Memory Engine**: Fact extraction from documents, relationship tracking, and memory retention via an **Ebbinghaus Forgetting Curve**.
- **Profile Management**: User static preferences and dynamic traits inferred from memories.

The linear flow (`document created` → `extract memories` → `update profile traits`) is executed synchronously in-process instead of relying on NATS.

## 2. Clean Architecture Layers

### Domain Layer
- **Document Model**: Document, Chunk, SourceType, ContentExtraction.
- **Memory Model**: Memory, Relation, ForgettingCurve (Ebbinghaus algorithm), DecayFunction.
- **Profile Model**: Profile, StaticPreference, DynamicTrait, TraitCategory.

### Usecase Layer
- **DocumentUseCase**: CreateDocument (triggers local memory extraction), GetChunks, DeleteDocument, UpdateDocument.
- **MemoryUseCase**: CreateMemory (triggers local profile update), ForgetMemory (Ebbinghaus decay), CreateRelation.
- **ProfileUseCase**: GetProfile, UpdateProfile, GetDynamicTraits.

### Adapter Layer
- **gRPC Handlers**: SmDocumentService, SmMemoryService, SmProfileService.
- **Repositories**: PostgreSQL + pgvector for persistence.
- **Event Publisher**: Emits `sm.engine.document.created`, `sm.engine.memory.created`, and `sm.engine.memory.forgotten` to NATS for search indexing.

### Infrastructure Layer
- **Bifrost LLM**: Memory extraction, profile trait inference.
- **Decay Engine**: Background worker for forgetting curve recalculations.
- **Observability**: OTel, Prometheus, Zap structured logging.

## 3. The Forgetting Curve Algorithm

Based on the Ebbinghaus model, memories decay over time unless reinforced.
- **Strength ($S$)**: Base strength determined by memory relevance and access count.
- **Time ($t$)**: Time elapsed since last access.
- **Decay Factor ($D$)**: Exponent dictating decay rate.
- **Retention ($R$)**: $R = e^{-t / (S * D)}$.
Memories falling below a retention threshold are forgotten (soft-deleted) and an `sm.engine.memory.forgotten` event is emitted.

## 4. Feature Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| FEAT-ENG-001 | In-Process Document-Memory-Profile Pipeline | Ready | P0 |
| FEAT-ENG-002 | Ebbinghaus Forgetting Curve Worker | Ready | P0 |
| FEAT-ENG-003 | Format-Aware Content Extraction | Ready | P1 |
| FEAT-ENG-004 | Dynamic Profile Trait Inference | Ready | P1 |

## 5. Task Specs Registry

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| TASK-ENG-001 | Implement Domain Models (Document, Memory, Profile) | Pending | P0 |
| TASK-ENG-002 | Implement Usecases & Local Orchestration | Pending | P0 |
| TASK-ENG-003 | Implement Ebbinghaus Decay Worker | Pending | P0 |
| TASK-ENG-004 | Implement Repositories (Postgres + pgvector) | Pending | P0 |
| TASK-ENG-005 | Implement gRPC Handlers & NATS Publisher | Pending | P1 |
| TASK-ENG-006 | Configure OTel, Wire, and Server Bootstrap | Pending | P1 |

```

## Acceptance Criteria
- [x] Proto files defined/matched and gRPC handlers implemented.
- [x] NATS publishers and subscribers correctly configured.
