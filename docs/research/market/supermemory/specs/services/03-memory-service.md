# 03 — Memory Service

> **gRPC**: 9002 | **Health**: 9082

---

## 1. Purpose

Memory Engine core: fact extraction từ documents, knowledge graph management (updates/extends/derives), version chain tracking, automatic forgetting, và memory lifecycle management.

---

## 2. Clean Architecture

```
services/memory-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # MemoryEntry, MemoryRelation, VersionChain
│   │   ├── value_object.go     # RelationType, MemoryLifecycle, MemoryClassification
│   │   ├── event.go            # MemoryCreated, MemoryForgotten, RelationCreated
│   │   └── errors.go           # ErrMemoryNotFound, ErrAlreadyForgotten
│   ├── usecase/
│   │   ├── extract_memories.go # AI fact extraction from document content
│   │   ├── create_memory.go    # Manual memory creation
│   │   ├── forget_memory.go    # By ID, by content (exact+semantic fallback)
│   │   ├── auto_forget.go      # Cron: check forgetAfter < now()
│   │   ├── resolve_relations.go# Detect updates/extends/derives against existing
│   │   ├── get_memory_graph.go # Graph data for visualization
│   │   ├── get_version_chain.go# Walk root→parent→current chain
│   │   ├── port/
│   │   │   ├── input.go        # ExtractMemoriesUC, ForgetMemoryUC, etc.
│   │   │   └── output.go       # MemoryRepo, RelationRepo, EmbeddingSearcher,
│   │   │                       # LLMClient, EventPublisher
│   │   └── dto/
│   │       └── memory.go       # ExtractInput, MemoryOutput, GraphOutput
│   ├── adapter/
│   │   ├── grpc/handler.go     # MemoryServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── memory.go          # MemoryEntry CRUD + version chain queries
│   │   │       ├── relation.go        # Memory relations management
│   │   │       └── document_source.go # M:M memory↔document links
│   │   ├── llm/
│   │   │   ├── fact_extractor.go      # LLM prompt for fact extraction
│   │   │   └── relation_detector.go   # LLM prompt for relation classification
│   │   ├── embedding/
│   │   │   └── similarity.go          # Cosine similarity for semantic forget
│   │   ├── event/
│   │   │   ├── publisher.go    # NATS: memory.created, memory.forgotten
│   │   │   └── subscriber.go  # NATS: document.processed → extract facts
│   │   └── scheduler/
│   │       └── auto_forget.go  # Cron job: periodic forgetAfter check
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   ├── 001_create_memory_entries.up.sql
│   └── 002_create_memory_relations.up.sql
└── Dockerfile
```

---

## 3. Domain Model

```go
type MemoryEntry struct {
    ID              string
    Memory          string                      // The fact text
    Content         *string                     // Extended content
    SpaceID         string                      // Space/project FK
    OrgID           string                      // Tenant scope

    // Version chain
    Version         int                         // Default 1
    IsLatest        bool                        // Default true
    ParentMemoryID  *string                     // Direct parent
    RootMemoryID    *string                     // Chain root

    // Relations to other memories
    MemoryRelations map[string]RelationType     // {memoryId: updates|extends|derives}

    // Classification
    IsStatic        bool                        // Long-term stable fact
    IsInference     bool                        // AI-derived fact
    IsForgotten     bool                        // Soft-deleted
    ForgetAfter     *time.Time                  // Auto-forget timestamp
    ForgetReason    *string

    // Embeddings
    MemoryEmbedding []float32

    // Source tracking
    SourceCount     int
    Metadata        map[string]any
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type RelationType string
const (
    RelationUpdates  RelationType = "updates"   // New info contradicts old
    RelationExtends  RelationType = "extends"   // New info enriches old
    RelationDerives  RelationType = "derives"   // System infers new connection
)
```

---

## 4. Fact Extraction Pipeline

```
document.processed event
       │
       ▼
┌── ExtractMemoriesUseCase ──────────────────────────────────┐
│                                                             │
│  1. Fetch document content + existing memories in space     │
│                                                             │
│  2. LLM Fact Extraction:                                    │
│     Prompt: "Extract facts from this content. For each:     │
│       - Classify: fact | preference | episode               │
│       - Determine if static or dynamic                      │
│       - Set forgetAfter if temporal                          │
│       - Provide entityContext if available"                  │
│     → Returns: []ExtractedFact                              │
│                                                             │
│  3. For each extracted fact:                                 │
│     a. Generate memory embedding                            │
│     b. Search existing memories (cosine similarity > 0.8)   │
│     c. If match found → LLM relation classification:        │
│        "Does new fact UPDATE, EXTEND, or DERIVE from old?"  │
│     d. Create MemoryEntry with relations                    │
│     e. If "updates": mark old memory isLatest=false         │
│     f. Create MemoryDocumentSource link                     │
│                                                             │
│  4. Publish memory.created events                           │
│  5. Publish memory.relation.created events                  │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Forget Algorithm (2-Phase)

```go
func (uc *ForgetMemoryUseCase) Execute(ctx context.Context, input ForgetInput) error {
    // Phase 1: Exact match
    if input.Content != "" {
        memory, err := uc.repo.FindByContent(ctx, input.OrgID, input.SpaceID, input.Content)
        if err == nil {
            return uc.markForgotten(ctx, memory, input.Reason)
        }
    }

    // Phase 2: Semantic search fallback (threshold 0.85)
    if input.Content != "" {
        embedding := uc.embedder.Generate(ctx, input.Content)
        matches := uc.repo.FindBySimilarity(ctx, input.OrgID, input.SpaceID, embedding, 0.85, 5)
        if len(matches) > 0 {
            return uc.markForgotten(ctx, matches[0], input.Reason)
        }
    }

    // Phase 3: By ID
    if input.MemoryID != "" {
        return uc.markForgotten(ctx, input.MemoryID, input.Reason)
    }

    return domain.ErrNotFound
}
```

---

## 6. gRPC Interface

```protobuf
service MemoryService {
  rpc ExtractMemories(ExtractMemoriesRequest) returns (ExtractMemoriesResponse);
  rpc ForgetMemory(ForgetMemoryRequest) returns (ForgetMemoryResponse);
  rpc GetMemoryGraph(GetMemoryGraphRequest) returns (MemoryGraphResponse);
  rpc GetVersionChain(GetVersionChainRequest) returns (VersionChainResponse);
  rpc ListMemories(ListMemoriesRequest) returns (ListMemoriesResponse);
  rpc GetMemory(GetMemoryRequest) returns (MemoryResponse);
}
```
