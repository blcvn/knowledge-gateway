# TASK-SM-006 — services/memory-service: Fact Extraction & Knowledge Graph

**Task ID:** TASK-SM-006  
**Wave:** 2 (Core Memory)  
**Solution:** [SOL-SM-002](../solutions/SOL-SM-002-Memory-Engine-Knowledge-Graph.md)  
**Depends on:** TASK-SM-005 (document.processed NATS event)  
**Ước tính:** 5h  
**Priority:** Critical — LongMemEval 81.6% benchmark requires this

---

## Mục tiêu

Nâng cấp `services/memory-service/internal/domain/sm/` với Fact Extraction pipeline và Knowledge Graph:
1. Nâng cấp `MemoryEntry` entity (Version Chain, Relations, Auto-Forget fields)
2. LLM Fact Extraction via Bifrost (trigger từ NATS `document.processed`)
3. Relation classification: updates | extends | derives
4. Forget algorithm 3 phases (exact → semantic → ID)
5. Auto-Forget cron job (mỗi 1 giờ)
6. Memory Graph visualization API

---

## Công việc cụ thể

### 1. Nâng cấp `MemoryEntry` Domain Model

**`services/memory-service/internal/domain/sm/memory.go`** — MODIFY

```go
type RelationType string
const (
    RelationUpdates RelationType = "updates"  // Thay thế hoàn toàn
    RelationExtends RelationType = "extends"  // Bổ sung thêm
    RelationDerives RelationType = "derives"  // Suy luận từ
)

// Thêm vào MemoryEntry (KHÔNG xóa fields cũ):
type MemoryEntry struct {
    // ... existing fields ...

    // Version Chain
    Version        int
    IsLatest       bool
    ParentMemoryID *string
    RootMemoryID   *string

    // Knowledge Graph Relations
    MemoryRelations map[string]RelationType // {memoryID: relationType}

    // Classification
    IsStatic    bool
    IsInference bool
    IsForgotten bool
    ForgetAfter *time.Time
    ForgetReason *string

    // Embedding (replace existing if any)
    MemoryEmbedding []float32

    SpaceID     string
    SourceCount int
}
```

### 2. Tạo DB Migration

**`services/memory-service/migrations/003_upgrade_memory_entries.sql`**

```sql
ALTER TABLE sm_memories RENAME TO memory_entries;

ALTER TABLE memory_entries
    ADD COLUMN IF NOT EXISTS space_id         TEXT NOT NULL DEFAULT 'sm_project_default',
    ADD COLUMN IF NOT EXISTS version          INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS is_latest        BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS parent_memory_id UUID REFERENCES memory_entries(id),
    ADD COLUMN IF NOT EXISTS root_memory_id   UUID REFERENCES memory_entries(id),
    ADD COLUMN IF NOT EXISTS memory_relations JSONB DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS is_static        BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_inference     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_forgotten     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS forget_after     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS forget_reason    TEXT,
    ADD COLUMN IF NOT EXISTS memory_embedding vector(1536),
    ADD COLUMN IF NOT EXISTS source_count     INT DEFAULT 1;

CREATE INDEX idx_mem_org_space_latest ON memory_entries(org_id, space_id, is_latest) WHERE is_forgotten = false;
CREATE INDEX idx_mem_forget_after ON memory_entries(forget_after) WHERE forget_after IS NOT NULL AND is_forgotten = false;
CREATE INDEX ON memory_entries USING hnsw (memory_embedding vector_cosine_ops) WITH (m=16, ef_construction=128);
CREATE INDEX idx_mem_root ON memory_entries(root_memory_id);
```

### 3. Implement Fact Extraction Use Case

**`services/memory-service/internal/usecase/sm/fact_extraction.go`**

```go
// Triggered by NATS "document.processed"
type FactExtractionUseCase struct {
    llm       LLMPort     // Bifrost (claude-3-haiku hoặc gpt-4o-mini)
    embedder  EmbedderPort
    memRepo   MemoryRepository
    publisher EventPublisher
}

// Execute: extract facts → find similar → classify relation → create/update
// LLM prompt: extract structured JSON facts with {text, classification, isStatic, forgetAfter}
// Similarity threshold for existing memory check: 0.80 cosine
// classifyRelation: "updates" | "extends" | "derives"
// If "updates": mark existing as IsLatest=false
func (uc *FactExtractionUseCase) Execute(ctx, docID, spaceID, orgID string) error
```

### 4. Implement Forget Algorithm (3 Phases)

**`services/memory-service/internal/usecase/sm/forget.go`**

```go
// Phase 1: exact content match
// Phase 2: semantic search threshold 0.85
// Phase 3: forget by ID
// Mark IsForgotten=true (soft delete)
// Publish NATS "memory.forgotten" per memory
func (uc *ForgetUseCase) Execute(ctx context.Context, req ForgetRequest) error
```

### 5. Implement Auto-Forget Cron

**`services/memory-service/internal/adapter/cron/auto_forget.go`**

```go
// Run every 1 hour: find all ForgetAfter < now AND IsForgotten=false
// Mark IsForgotten=true, ForgetReason="auto-forget: forgetAfter expired"
// Publish "memory.forgotten" events
func (j *AutoForgetJob) Run(ctx context.Context)
```

Bootstrap integration:
```go
// apps/memory/internal/bootstrap/memory.go
cron := cron.New()
cron.AddFunc("0 * * * *", autoForgetJob.Run)
cron.Start()
```

### 6. Implement Memory Graph API

**`services/memory-service/internal/usecase/sm/get_graph.go`**

```go
// Returns nodes (all memories including non-latest) + edges (relations)
// GET /api/v1/memories/graph?spaceId=xxx
func (uc *GetMemoryGraphUseCase) Execute(ctx, orgID, spaceID string) (*MemoryGraph, error)
```

### 7. REST Endpoints (thêm vào handler)

```
GET  /api/v1/memories             → ListMemories (spaceId, isLatest, isStatic filters)
POST /api/v1/memories             → CreateMemory (direct)
GET  /api/v1/memories/{id}        → GetMemory + version chain
POST /api/v1/memories/forget      → Forget (content/semantic/ID)
GET  /api/v1/memories/graph       → Knowledge graph (nodes + edges)
GET  /api/v1/memories/{id}/chain  → Full version chain
```

### 8. Tests

- `TestFactExtraction_SimilarExists_Relation`: existing memory → classify relation
- `TestFactExtraction_Updates_MarksOldNotLatest`: "updates" relation → old.IsLatest=false
- `TestForget_Phase1_ExactMatch`: exact content → found + forgotten
- `TestForget_Phase2_SemanticMatch`: approximate → semantic search threshold 0.85
- `TestForget_Phase3_ByID`: memory ID → forgotten directly
- `TestAutoForgetJob_ExpiredMemories`: forgetAfter = yesterday → IsForgotten=true
- `TestGetMemoryGraph_NodesAndEdges`: 3 memories with relations → graph structure

---

## Acceptance Criteria

- [ ] `go build ./services/memory-service/...` không lỗi
- [ ] Migration không drop existing data từ sm_memories
- [ ] "Thích Python và ghét Java" → 2 separate memory facts
- [ ] "Bây giờ thích Go hơn Python" → Python memory.IsLatest=false (RelationUpdates)
- [ ] Forget với approximate content → found via semantic search
- [ ] forgetAfter=now-1s → cron marks as forgotten
- [ ] GetMemoryGraph → {nodes: [...], edges: [{source, target, relation}]}
- [ ] `go test ./services/memory-service/...` pass

---

## Files tạo/sửa

```
services/memory-service/
├── internal/
│   ├── domain/sm/
│   │   └── memory.go              (MODIFY: thêm Version Chain fields)
│   ├── usecase/sm/
│   │   ├── fact_extraction.go     (NEW)
│   │   ├── fact_extraction_test.go (NEW)
│   │   ├── forget.go              (MODIFY: 3-phase algorithm)
│   │   ├── forget_test.go         (NEW)
│   │   └── get_graph.go           (NEW)
│   ├── adapter/
│   │   ├── cron/
│   │   │   └── auto_forget.go     (NEW)
│   │   └── subscriber/
│   │       └── document_events.go (NEW: listen document.processed)
│   └── infra/postgres/
│       └── memory_repo.go         (MODIFY: thêm FindSimilar, ListLatest)
└── migrations/
    └── 003_upgrade_memory_entries.sql (NEW)
```

## Sau khi hoàn thành

Chạy: `go build ./services/memory-service/... && go test ./services/memory-service/...`
