---
id: FEAT-PIP-002
title: Usecase Layer — Saga Orchestrator + Knowledge Processing
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement usecase layer chứa business logic cho saga orchestration (6-step pipeline) và knowledge processing (entity/edge extraction, resolution, embedding, community). Định nghĩa port interfaces cho adapter layer.

## Scope

### In Scope
- Saga orchestrator state machine with compensating actions
- IngestEpisode usecase: dedup → queue → saga pipeline
- BulkIngest usecase: batch processing with cross-episode dedup
- ExtractEntities usecase: content → LLM → parsed entities
- ResolveEntities usecase: search similar → LLM compare → merge/create
- ExtractEdges usecase: episode + entities → LLM → temporal fact triples
- ResolveEdges usecase: find contradictions → invalidate old edges
- GenerateEmbedding usecase: text → vector via embedder client
- UpdateCommunity usecase: label propagation → LLM summarization
- Port interfaces (input + output) for all adapter dependencies
- DTO structs for request/response mapping

### Out of Scope
- Adapter implementations (separate FEAT specs)

## Thiết Kế Kỹ Thuật

### Saga Orchestrator Logic

```go
func (s *SagaOrchestrator) Execute(ctx context.Context, episode Episode) error {
    steps := []SagaStep{
        {Name: StepExtractEntities, Execute: s.knowledge.ExtractEntities, Compensate: nil},
        {Name: StepResolveEntities, Execute: s.knowledge.ResolveEntities, Compensate: nil},
        {Name: StepExtractEdges, Execute: s.knowledge.ExtractEdges, Compensate: nil},
        {Name: StepResolveEdges, Execute: s.knowledge.ResolveEdges, Compensate: nil},
        {Name: StepGenerateEmbeddings, Execute: s.knowledge.GenerateEmbeddings, Compensate: nil},
        {Name: StepSaveBulk, Execute: s.store.SaveBulk, Compensate: s.store.RollbackBulk},
        {Name: StepUpdateCommunity, Execute: s.knowledge.UpdateCommunity, Compensate: nil},
    }
    return s.executeSteps(ctx, episode, steps)
}
```

### Per-Group Serialization

```go
// GroupLock ensures only one saga per group_id runs at a time
type GroupLock interface {
    Acquire(ctx context.Context, groupID GroupID) (unlock func(), err error)
}
```

## Acceptance Criteria

- [ ] AC-1: IngestEpisode deduplicates by (name, group_id, reference_time) hash
- [ ] AC-2: Saga state transitions follow defined state machine (QUEUED → PROCESSING → COMPLETED/FAILED)
- [ ] AC-3: Compensation executes on SaveBulk failure (RollbackBulk called)
- [ ] AC-4: Per-group lock prevents concurrent saga execution within same group_id
- [ ] AC-5: BulkIngest processes episodes in streaming fashion with dedup across batch
- [ ] AC-6: All output ports are interfaces (not concrete types)
- [ ] AC-7: ExtractEntities returns parsed entities from LLM response
- [ ] AC-8: ResolveEntities merges duplicates with >0.85 similarity threshold

## Test Requirements

- **Unit tests**: Saga state machine, dedup logic, group lock, LLM response parsing (mock LLM client)
- **Minimum coverage**: 80%

## Definition of Done

- [ ] All usecase methods compile against port interfaces only
- [ ] Unit tests pass with mocked adapters, coverage ≥ 80%
- [ ] Saga state machine handles all transitions including compensation
- [ ] No direct infrastructure imports in usecase/
