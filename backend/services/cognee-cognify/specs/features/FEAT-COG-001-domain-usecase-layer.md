---
id: FEAT-COG-001
title: Cognify Service — Domain + Usecase Layer (8-Stage Pipeline)
service: cognee-cognify
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement Layer 1 (Domain) và Layer 2 (Usecase) cho cognee-cognify — 8-stage KG construction pipeline với LLM entity extraction, relationship extraction, deduplication, graph construction, embedding generation, và community summarization.

## Bối Cảnh Nghiệp Vụ

cognee-cognify là core LLM-intensive service. Nhận DataIngestedEvent từ cognee-ingestion, xử lý qua 8 stages, output là knowledge graph (Neo4j) + vector embeddings (Qdrant). Pipeline là state machine, mỗi stage persists progress, cho phép resume on failure.

## Scope

### In Scope
- Domain entities: `CognifyJob`, `PipelineStage`, `Chunk`, `Entity`, `Relationship`, `Community`, `Ontology`
- Domain value objects: `JobStatus`, `ChunkingStrategy`, `EntityType`, `StageType`
- Domain events: `PipelineCompletedEvent`, `StageAdvancedEvent`
- Domain errors: `JobNotFoundError`, `PipelineFailedError`, `LLMTimeoutError`
- 8 Usecase stages: classify, chunk, extract_entities, extract_relationships, deduplicate, build_graph, embed, summarize
- Usecase orchestrator: `CognifyOrchestrator` (runs all 8 stages sequentially)
- Port interfaces: `GraphRepository`, `VectorRepository`, `LLMClient`, `EmbedderClient`, `JobRepository`, `EventPublisher`

### Out of Scope
- gRPC handlers, NATS subscribers (FEAT-COG-002)
- Infrastructure implementations (FEAT-COG-003)

## Thiết Kế Kỹ Thuật

### Directory Structure

```
internal/
├── domain/
│   ├── entity.go           # CognifyJob, Chunk, Entity, Relationship, Community
│   ├── value_object.go     # JobStatus, ChunkingStrategy, EntityType, StageType
│   ├── event.go            # PipelineCompletedEvent, StageAdvancedEvent
│   └── errors.go           # Domain error types
├── usecase/
│   ├── cognify.go          # CognifyOrchestrator — runs all stages
│   ├── classify.go         # Stage 1: Content type → chunking strategy (LLM)
│   ├── chunk.go            # Stage 2: Text → Chunks (recursive/semantic)
│   ├── extract_entities.go # Stage 3: Chunks → Entities (LLM NER)
│   ├── extract_rels.go     # Stage 4: Chunks → Relationships (LLM)
│   ├── deduplicate.go      # Stage 5: Entity resolution (LLM + embedding)
│   ├── build_graph.go      # Stage 6: Entities + Rels → Neo4j
│   ├── embed.go            # Stage 7: Chunks + Entities → Qdrant
│   ├── summarize.go        # Stage 8: Community detection + LLM summary
│   ├── port/
│   │   ├── input.go        # CognifyUseCase, JobManager
│   │   └── output.go       # GraphRepo, VectorRepo, LLMClient, EmbedderClient, JobRepo
│   └── dto/
│       ├── request.go      # TriggerCognifyReq, CognifyConfig
│       └── response.go     # CognifyJobResult, PipelineMetrics
```

### Pipeline Orchestrator

```go
func (uc *CognifyOrchestrator) Execute(ctx context.Context, req dto.TriggerCognifyReq) (*dto.CognifyJobResult, error) {
    job := domain.NewCognifyJob(req.DatasetID, req.TenantID, req.Config)
    uc.jobRepo.Create(ctx, job)
    
    stages := []Stage{
        uc.classifyStage,
        uc.chunkStage,
        uc.extractEntitiesStage,
        uc.extractRelationshipsStage,
        uc.deduplicateStage,
        uc.buildGraphStage,
        uc.embedStage,
        uc.summarizeStage,
    }
    
    for i, stage := range stages {
        job.AdvanceStage(stage.Name())
        uc.jobRepo.Update(ctx, job)
        
        if err := stage.Execute(ctx, job); err != nil {
            job.Fail(err)
            uc.jobRepo.Update(ctx, job)
            return nil, err
        }
    }
    
    job.Complete()
    uc.jobRepo.Update(ctx, job)
    uc.eventPub.PublishPipelineCompleted(ctx, job.ToCompletedEvent())
    return job.ToResult(), nil
}
```

### LLM Integration Points

| Stage | LLM Call | Input | Output (JSON Schema) |
|-------|---------|-------|---------------------|
| classify | `CompleteStructured` | Full text sample | `{content_type, language, topics[], strategy}` |
| extract_entities | `CompleteStructured` | Each chunk | `{entities: [{name, type, description}]}` |
| extract_relationships | `CompleteStructured` | Each chunk + entities | `{relationships: [{source, target, relation, weight}]}` |
| deduplicate | `CompleteStructured` | Entity pairs | `{is_same: bool, confidence: float}` |
| summarize | `Complete` | Community nodes list | Community summary text |

### Pipeline Metrics

```go
type PipelineMetrics struct {
    ChunksCreated          int
    EntitiesExtracted      int
    RelationshipsExtracted int
    EntitiesDeduplicated   int
    CommunitiesFound       int
    EmbeddingsGenerated    int
    LLMCallsTotal          int
    LLMTokensUsed          int
    TotalDurationMs        int64
}
```

## Acceptance Criteria

- [ ] AC-1: Given a CognifyJob, When created, Then status is PENDING and all fields initialized
- [ ] AC-2: Given mock ports, When orchestrator executes all 8 stages, Then job transitions PENDING → RUNNING → COMPLETED
- [ ] AC-3: Given stage failure at stage N, When error occurs, Then job status is FAILED with error message and stage name
- [ ] AC-4: Given chunks, When extract_entities stage runs with mock LLM returning structured JSON, Then entities are correctly parsed
- [ ] AC-5: Given entity pairs, When deduplicate stage runs, Then duplicate entities are merged with provenance tracking
- [ ] AC-6: Given completed pipeline, When PipelineCompletedEvent is built, Then metrics accurately reflect all stage outputs
- [ ] AC-7: Given CognifyConfig with skip_dedup=true, When orchestrator runs, Then deduplicate stage is skipped
- [ ] AC-8: Port interfaces have no infrastructure dependencies

## Test Requirements

- **Unit tests**: Each stage with mock LLM responses (deterministic JSON fixtures)
- **Domain tests**: Job state machine transitions
- **Orchestrator test**: Full pipeline with all mock ports
- **Coverage**: ≥ 80%
