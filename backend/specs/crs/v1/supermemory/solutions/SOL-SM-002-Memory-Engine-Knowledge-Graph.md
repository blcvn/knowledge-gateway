# Solution: SOL-SM-002 — Memory Engine & Knowledge Graph

**CR ID:** CR-SM-002  
**Solution ID:** SOL-SM-002  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp `services/memory-service/` để thêm lõi Knowledge Graph với Fact Extraction, Version Chain, và Automatic Forgetting. Đây là component tạo ra benchmark #1 LongMemEval (81.6%) của Supermemory — nhờ LLM phân loại quan hệ giữa các memories thay vì lưu raw content.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `SMMemory` entity | `services/memory-service/internal/domain/sm/` | Tồn tại: ID, TenantID, Content, Tags[], Embedding |
| `sm-memory` gRPC service | `apps/memory/internal/bootstrap/` | Có: CRUD cơ bản |
| `MemoryService` usecase | `services/memory-service/internal/usecase/sm/` | Có: Store, Search, Forget |
| NATS JetStream | `apps/memory/internal/bus/` | Event `sm.engine.*` đang tồn tại |

### Gap phân tích

- `SMMemory` thiếu: `Version`, `IsLatest`, `ParentMemoryID`, `RootMemoryID`, `MemoryRelations`, `IsStatic`, `IsInference`, `IsForgotten`, `ForgetAfter`
- Không có LLM Fact Extraction pipeline
- Không có Knowledge Graph (quan hệ `updates/extends/derives`)
- Không có Auto-Forget cron job
- Không có Version Chain tracking

---

## 3. Thiết kế Giải pháp

### 3.1. Nâng cấp Domain Model

```go
// services/memory-service/internal/domain/sm/memory.go
// [MODIFY] Thêm các fields vào SMMemory

package sm

import "time"

type RelationType string

const (
    RelationUpdates RelationType = "updates"  // Thay thế hoàn toàn
    RelationExtends RelationType = "extends"  // Bổ sung thêm thông tin
    RelationDerives RelationType = "derives"  // Suy luận từ
)

type MemoryEntry struct {
    // --- Existing fields (giữ nguyên) ---
    ID      string
    OrgID   string  // Thay TenantID → OrgID cho đồng nhất
    Content string  // Fact text (thay vì raw content)
    Tags    []string

    // --- New: Scope ---
    SpaceID string   // Container tag / project space

    // --- New: Version Chain ---
    Version        int      // Tăng từ 1
    IsLatest       bool     // false khi bị update bởi memory mới
    ParentMemoryID *string  // ID memory trực tiếp trước trong chain
    RootMemoryID   *string  // ID memory gốc của chain

    // --- New: Relations (Knowledge Graph) ---
    // Key: memory ID, Value: relation type
    MemoryRelations map[string]RelationType

    // --- New: Classification ---
    IsStatic    bool       // true = long-term fact (e.g., "User is a Go developer")
    IsInference bool       // true = AI-derived (không từ user input trực tiếp)
    IsForgotten bool       // Soft delete flag
    ForgetAfter *time.Time // Auto-forget date
    ForgetReason *string

    // --- New: Embedding ---
    MemoryEmbedding []float32 // Dense vector

    SourceCount int
    Metadata    map[string]any
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 3.2. Fact Extraction Pipeline

```go
// services/memory-service/internal/usecase/sm/fact_extraction.go

// Được trigger bởi NATS event "document.processed"
type FactExtractionUseCase struct {
    llm        LLMPort          // Qua Bifrost
    embedder   EmbedderPort
    memRepo    MemoryRepository
    publisher  EventPublisher
}

func (uc *FactExtractionUseCase) Execute(ctx context.Context, docID, spaceID, orgID string) error {
    // Step 1: Fetch document content
    content := uc.docClient.GetContent(ctx, docID)

    // Step 2: Fetch existing memories trong space (để so sánh)
    existingMems, _ := uc.memRepo.ListLatest(ctx, orgID, spaceID)

    // Step 3: LLM Extract facts
    prompt := buildExtractionPrompt(content, existingMems)
    facts, _ := uc.llm.Extract(ctx, prompt)
    // Response: [{text, classification: fact|preference|episode, isStatic: bool, forgetAfter: ?time}]

    for _, fact := range facts {
        // Step 4: Generate embedding
        embedding, _ := uc.embedder.Embed(ctx, fact.Text)

        // Step 5: Cosine similarity search vs existing memories
        similar, _ := uc.memRepo.FindSimilar(ctx, orgID, spaceID, embedding, 0.80)

        var newMem *MemoryEntry
        if len(similar) > 0 {
            // Step 6a: LLM classify relation
            relation := uc.classifyRelation(ctx, fact.Text, similar[0].Content)

            // Step 6b: Create new memory with relation
            newMem = &MemoryEntry{
                Content:         fact.Text,
                MemoryRelations: map[string]RelationType{similar[0].ID: relation},
                Version:         similar[0].Version + 1,
                ParentMemoryID:  &similar[0].ID,
                RootMemoryID:    rootID(similar[0]),
                IsLatest:        true,
                IsStatic:        fact.IsStatic,
                ForgetAfter:     fact.ForgetAfter,
            }

            // Step 6c: Nếu "updates" → mark cũ là isLatest=false
            if relation == RelationUpdates {
                similar[0].IsLatest = false
                uc.memRepo.Update(ctx, &similar[0])
            }
        } else {
            // New memory (no existing match)
            newMem = &MemoryEntry{
                Content:  fact.Text,
                Version:  1,
                IsLatest: true,
                IsStatic: fact.IsStatic,
                ForgetAfter: fact.ForgetAfter,
            }
        }

        newMem.MemoryEmbedding = embedding
        newMem.OrgID = orgID
        newMem.SpaceID = spaceID
        uc.memRepo.Create(ctx, newMem)
        uc.publisher.Publish(ctx, "memory.created", MemoryCreatedEvent{MemoryID: newMem.ID})
    }
    return nil
}

// LLM prompt để phân loại relation
func (uc *FactExtractionUseCase) classifyRelation(ctx context.Context, newFact, existingFact string) RelationType {
    prompt := fmt.Sprintf(`
New fact: "%s"
Existing fact: "%s"

How does the new fact relate to the existing fact?
- "updates": New fact completely replaces/contradicts existing fact
- "extends": New fact adds information without contradicting
- "derives": New fact is logically derived from existing fact

Answer with one word: updates|extends|derives`, newFact, existingFact)

    result, _ := uc.llm.Complete(ctx, prompt)
    return RelationType(strings.TrimSpace(result))
}
```

### 3.3. Forget Algorithm (3 Phases)

```go
// services/memory-service/internal/usecase/sm/forget.go

type ForgetUseCase struct {
    memRepo  MemoryRepository
    embedder EmbedderPort
    publisher EventPublisher
}

func (uc *ForgetUseCase) Execute(ctx context.Context, req ForgetRequest) error {
    var targets []*MemoryEntry

    // Phase 1: Exact content match
    if req.Content != "" {
        targets, _ = uc.memRepo.FindByExactContent(ctx, req.OrgID, req.SpaceID, req.Content)
    }

    // Phase 2: Semantic search (threshold 0.85) nếu Phase 1 không tìm được
    if len(targets) == 0 && req.Content != "" {
        embedding, _ := uc.embedder.Embed(ctx, req.Content)
        targets, _ = uc.memRepo.FindSimilar(ctx, req.OrgID, req.SpaceID, embedding, 0.85)
    }

    // Phase 3: Forget by ID nếu được cung cấp
    if len(targets) == 0 && req.MemoryID != "" {
        mem, _ := uc.memRepo.Get(ctx, req.MemoryID)
        targets = []*MemoryEntry{mem}
    }

    // Mark isForgotten=true (soft delete)
    for _, mem := range targets {
        mem.IsForgotten = true
        mem.ForgetReason = req.Reason
        uc.memRepo.Update(ctx, mem)
        uc.publisher.Publish(ctx, "memory.forgotten", MemoryForgottenEvent{MemoryID: mem.ID})
    }
    return nil
}
```

### 3.4. Auto-Forget Cron Job

```go
// services/memory-service/internal/adapter/cron/auto_forget.go

type AutoForgetJob struct {
    memRepo  MemoryRepository
    publisher EventPublisher
}

// Chạy mỗi 1 giờ
func (j *AutoForgetJob) Run(ctx context.Context) {
    expired, _ := j.memRepo.FindExpired(ctx, time.Now())
    for _, mem := range expired {
        mem.IsForgotten = true
        mem.ForgetReason = ptr("auto-forget: forgetAfter expired")
        j.memRepo.Update(ctx, mem)
        j.publisher.Publish(ctx, "memory.forgotten", MemoryForgottenEvent{MemoryID: mem.ID})
    }
    slog.Info("auto-forget completed", "count", len(expired))
}
```

Đăng ký cron trong bootstrap:
```go
// apps/memory/internal/bootstrap/memory.go
cron := cron.New()
cron.AddFunc("0 * * * *", autoForgetJob.Run)  // Mỗi 1 giờ
cron.Start()
```

### 3.5. Memory Graph Visualization Data

```go
// services/memory-service/internal/usecase/sm/get_graph.go

type MemoryGraph struct {
    Nodes []MemoryNode
    Edges []MemoryEdge
}

type MemoryNode struct {
    ID        string
    Type      string // "memory" | "document"
    Content   string
    IsLatest  bool
    IsStatic  bool
    Version   int
}

type MemoryEdge struct {
    Source   string
    Target   string
    Relation RelationType // updates | extends | derives
}

func (uc *GetMemoryGraphUseCase) Execute(ctx context.Context, orgID, spaceID string) (*MemoryGraph, error) {
    memories, _ := uc.memRepo.ListAll(ctx, orgID, spaceID) // Bao gồm cả non-latest

    nodes := make([]MemoryNode, 0, len(memories))
    edges := make([]MemoryEdge, 0)

    for _, m := range memories {
        nodes = append(nodes, MemoryNode{
            ID: m.ID, Type: "memory",
            Content: m.Content, IsLatest: m.IsLatest,
            IsStatic: m.IsStatic, Version: m.Version,
        })
        for targetID, rel := range m.MemoryRelations {
            edges = append(edges, MemoryEdge{
                Source: m.ID, Target: targetID, Relation: rel,
            })
        }
    }
    return &MemoryGraph{Nodes: nodes, Edges: edges}, nil
}
```

---

## 4. Database Schema

```sql
-- memory_entries table (nâng cấp từ sm_memories)
CREATE TABLE memory_entries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL,
    space_id         TEXT NOT NULL DEFAULT 'sm_project_default',
    content          TEXT NOT NULL,
    tags             TEXT[] DEFAULT '{}',

    -- Version chain
    version          INT NOT NULL DEFAULT 1,
    is_latest        BOOLEAN NOT NULL DEFAULT true,
    parent_memory_id UUID REFERENCES memory_entries(id),
    root_memory_id   UUID REFERENCES memory_entries(id),

    -- Relations (JSONB: {memory_id: "updates|extends|derives"})
    memory_relations JSONB DEFAULT '{}',

    -- Classification
    is_static        BOOLEAN NOT NULL DEFAULT false,
    is_inference     BOOLEAN NOT NULL DEFAULT false,
    is_forgotten     BOOLEAN NOT NULL DEFAULT false,
    forget_after     TIMESTAMPTZ,
    forget_reason    TEXT,

    -- Vector
    memory_embedding vector(1536),

    source_count     INT DEFAULT 1,
    metadata         JSONB DEFAULT '{}',
    created_at       TIMESTAMPTZ DEFAULT now(),
    updated_at       TIMESTAMPTZ DEFAULT now()
);

-- Indexes
CREATE INDEX idx_mem_org_space_latest ON memory_entries(org_id, space_id, is_latest) WHERE is_forgotten = false;
CREATE INDEX idx_mem_forget_after ON memory_entries(forget_after) WHERE forget_after IS NOT NULL AND is_forgotten = false;
CREATE INDEX ON memory_entries USING hnsw (memory_embedding vector_cosine_ops);
CREATE INDEX idx_mem_root ON memory_entries(root_memory_id);
```

---

## 5. NATS Event Flow

```
document.processed
    → FactExtractionUseCase
        → memory.created  (mỗi fact)
            → Profile Service (rebuild profile)
            → Analytics Service (update memory count)
        → memory.relation.created (mỗi relation)

memory.forgotten
    → Profile Service (rebuild profile)
    → Analytics Service (update count)
```

---

## 6. LLM Integration (Bifrost)

Tất cả LLM calls đi qua **Bifrost** (multi-provider gateway đã có sẵn trong VNP Memory):

```go
// Fact extraction: ~500-1000 tokens/call
// Relation classification: ~100-200 tokens/call
// Provider: anthropic/claude-3-haiku (nhanh + rẻ) hoặc openai/gpt-4o-mini
```

Prompt engineering:
- Fact extraction: System prompt hướng dẫn trích xuất facts structured JSON
- Relation classification: Binary classification, trả về 1 trong 3 giá trị
- Batch processing: Gộp nhiều facts vào 1 LLM call để giảm API costs

---

## 7. API Endpoints (Gateway)

```
// Thêm vào /v1/sm/* routes

GET  /api/v1/memories                  - List memories (filter: spaceID, isLatest, isStatic)
POST /api/v1/memories                  - Create memory (direct, not via document)
GET  /api/v1/memories/:id              - Get memory + version chain
POST /api/v1/memories/forget           - Forget (content/semantic/ID)
GET  /api/v1/memories/graph            - Knowledge graph data (nodes + edges)
GET  /api/v1/memories/:id/chain        - Full version chain
```

---

## 8. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model migration (thêm fields vào MemoryEntry) | 1 ngày |
| **P2** | Database schema migration | 1 ngày |
| **P3** | Fact Extraction pipeline + LLM integration | 3 ngày |
| **P4** | Relation classification (updates/extends/derives) | 2 ngày |
| **P5** | Forget algorithm 3-phase | 1 ngày |
| **P6** | Auto-forget cron job | 1 ngày |
| **P7** | Memory Graph visualization API | 1 ngày |
| **P8** | NATS event subscriptions + publisher | 1 ngày |
| **P9** | Tests + Acceptance Criteria | 2 ngày |

**Tổng:** ~13 ngày (Wave 2 — song song với CR-SM-001)

---

## 9. Rủi ro và Mitigation

| Rủi ro | Mức độ | Mitigation |
|--------|--------|-----------|
| LLM fact extraction không chính xác 100% | High | Prompt engineering + confidence threshold, human review |
| Relation classification edge cases | Medium | 3 categories đơn giản, fallback về "extends" nếu không rõ |
| Version chain gây N+1 queries | Medium | Fetch toàn bộ chain trong 1 query với recursive CTE |
| Auto-forget false positives | Low | Require explicit forgetAfter từ LLM, không tự suy luận |

---

## 10. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| "Thích Python và ghét Java" → 2 facts riêng | LLM extraction tách facts theo nghĩa |
| "Bây giờ thích Go hơn Python" → Python fact isLatest=false | Relation=updates, old memory marked |
| forgetAfter=now+1s → cron đánh dấu forgotten | Auto-forget cron mỗi 1 giờ |
| Forget gần đúng → semantic search tìm được | Phase 2: semantic search threshold 0.85 |
| GetMemoryGraph → nodes + edges | Graph API: list all + relations |
