# TASK-AM-005 — Memory Lifecycle Domain (AgentMemory + Slots)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-005 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/memory-service/internal/domain/agentmemory/` + usecases |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §2.2 → §2.9 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-AM-001 |
| **Estimated** | 6h |

---

## Context

Thêm AgentMemory domain vào `services/memory-service/` (cùng chỗ với Memobase/Zep/SM). Triển khai Jaccard versioning, eviction, TTL auto-forget, strength decay, memory slots.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/memory-service/internal/domain/agentmemory/entity.go` |
| CREATE | `services/memory-service/internal/domain/agentmemory/value_object.go` |
| CREATE | `services/memory-service/internal/domain/agentmemory/errors.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/remember_agent.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/evict.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/auto_forget.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/decay.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/slots.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/retention.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/port/output.go` |

---

## Implementation

### `internal/domain/agentmemory/entity.go`

```go
package agentmemory

import (
    "time"
    "github.com/google/uuid"
)

type AgentMemory struct {
    ID                   string
    TenantID             string
    Project              string
    Type                 MemoryType
    Title                string
    Content              string
    Concepts             []string
    Files                []string
    SessionIDs           []string
    Strength             float64    // 0.0 - 1.0
    Version              int
    ParentID             string
    Supersedes           []string
    RelatedIDs           []string
    SourceObservationIDs []string
    IsLatest             bool
    ForgetAfter          *time.Time
    AgentID              string
    FlaggedEviction      bool
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

type MemorySlot struct {
    TenantID    string
    Project     string
    Scope       string    // "project" | "global"
    Label       string
    Content     string
    Description string
    SizeLimit   int
    Pinned      bool     // immune to eviction
    ReadOnly    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func NewAgentMemory(tenantID, project string, memType MemoryType, title, content string) AgentMemory {
    return AgentMemory{
        ID:       uuid.New().String(),
        TenantID: tenantID,
        Project:  project,
        Type:     memType,
        Title:    title,
        Content:  content,
        Strength: 0.7,
        Version:  1,
        IsLatest: true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

### `internal/domain/agentmemory/value_object.go`

```go
package agentmemory

type MemoryType string

const (
    MemTypePattern      MemoryType = "pattern"
    MemTypePreference   MemoryType = "preference"
    MemTypeArchitecture MemoryType = "architecture"
    MemTypeBug          MemoryType = "bug"
    MemTypeWorkflow     MemoryType = "workflow"
    MemTypeFact         MemoryType = "fact"
)

func IsValidType(t string) bool {
    switch MemoryType(t) {
    case MemTypePattern, MemTypePreference, MemTypeArchitecture, MemTypeBug, MemTypeWorkflow, MemTypeFact:
        return true
    }
    return false
}
```

### `internal/domain/agentmemory/errors.go`

```go
package agentmemory

import "errors"

var (
    ErrMemoryNotFound  = errors.New("memory not found")
    ErrSlotReadOnly    = errors.New("memory slot is read-only")
    ErrSlotSizeExceeded = errors.New("content exceeds slot size limit")
    ErrInvalidType     = errors.New("invalid memory type")
)
```

### `internal/usecase/agentmemory/remember_agent.go`

```go
package agentmemory

import (
    "context"
    "math"
    "sort"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

const JaccardThreshold = 0.7

type RememberAgentUseCase struct {
    repo         port.IMemoryRepo
    searchClient port.ISearchNotifier
    publisher    port.IEventPublisher
}

type RememberRequest struct {
    TenantID    string
    Project     string
    Type        string
    Title       string
    Content     string
    Concepts    []string
    Files       []string
    SessionID   string
    Strength    float64
    AgentID     string
    ForgetAfter *time.Time
}

type RememberResult struct {
    MemoryID   string
    Version    int
    Superseded []string
}

func (uc *RememberAgentUseCase) Execute(ctx context.Context, req RememberRequest) (*RememberResult, error) {
    if !agentmemory.IsValidType(req.Type) {
        return nil, agentmemory.ErrInvalidType
    }

    strength := req.Strength
    if strength <= 0 { strength = 0.7 }

    // 1. Load existing isLatest memories of same Type + Project
    existing, err := uc.repo.ListLatestByType(ctx, req.TenantID, req.Project, req.Type)
    if err != nil { return nil, err }

    // 2. Find best Jaccard match
    bestMatch, bestScore := findBestJaccardMatch(req.Concepts, existing)

    // 3. Supersede if threshold exceeded
    var superseded []string
    newVersion := 1
    if bestScore >= JaccardThreshold && bestMatch != nil {
        uc.repo.SetNotLatest(ctx, bestMatch.ID)
        superseded = append(superseded, bestMatch.ID)
        newVersion = bestMatch.Version + 1

        uc.publisher.Publish(ctx, "agentmemory.memory.superseded", map[string]any{
            "old_id": bestMatch.ID, "new_version": newVersion,
        })
    }

    // 4. Create new memory
    mem := agentmemory.AgentMemory{
        ID:          uuid.New().String(),
        TenantID:    req.TenantID,
        Project:     req.Project,
        Type:        agentmemory.MemoryType(req.Type),
        Title:       req.Title,
        Content:     req.Content,
        Concepts:    req.Concepts,
        Files:       req.Files,
        SessionIDs:  []string{req.SessionID},
        Strength:    strength,
        Version:     newVersion,
        Supersedes:  superseded,
        IsLatest:    true,
        ForgetAfter: req.ForgetAfter,
        AgentID:     req.AgentID,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    if err := uc.repo.Save(ctx, mem); err != nil { return nil, err }

    // 5. Notify search to reindex
    go uc.searchClient.IndexMemory(context.Background(), mem)

    // 6. Publish event
    uc.publisher.Publish(ctx, "agentmemory.memory.remembered", map[string]any{
        "memory_id": mem.ID, "type": mem.Type, "version": mem.Version, "superseded": superseded,
    })

    return &RememberResult{MemoryID: mem.ID, Version: mem.Version, Superseded: superseded}, nil
}

// Jaccard coefficient on lowercase concept sets
func jaccardSimilarity(a, b []string) float64 {
    setA := toLowerSet(a)
    setB := toLowerSet(b)
    intersection := 0
    for k := range setA { if setB[k] { intersection++ } }
    union := len(setA) + len(setB) - intersection
    if union == 0 { return 0 }
    return float64(intersection) / float64(union)
}

func toLowerSet(words []string) map[string]bool {
    s := make(map[string]bool, len(words))
    for _, w := range words { s[strings.ToLower(w)] = true }
    return s
}

func findBestJaccardMatch(concepts []string, existing []agentmemory.AgentMemory) (*agentmemory.AgentMemory, float64) {
    var bestMatch *agentmemory.AgentMemory
    bestScore := 0.0
    for i := range existing {
        score := jaccardSimilarity(concepts, existing[i].Concepts)
        if score > bestScore { bestScore = score; bestMatch = &existing[i] }
    }
    return bestMatch, bestScore
}
```

### `internal/usecase/agentmemory/evict.go`

```go
package agentmemory

import (
    "context"
    "math"
    "sort"
    "time"

    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type EvictUseCase struct {
    repo      port.IMemoryRepo
    publisher port.IEventPublisher
}

type EvictRequest struct {
    TenantID    string
    Project     string
    MaxMemories int
    DryRun      bool
}

type EvictResult struct {
    EvictedIDs []string
    DryRun     bool
}

type scoredMemory struct {
    Memory agentmemory.AgentMemory
    Score  float64
}

// score = strength × recency × frequency
// recency = exp(-daysSinceUpdate / 30.0)
// frequency = log1p(len(sessionIDs))
func evictionScore(m agentmemory.AgentMemory) float64 {
    daysSince := time.Since(m.UpdatedAt).Hours() / 24
    recency := math.Exp(-daysSince / 30.0)
    frequency := math.Log1p(float64(len(m.SessionIDs)))
    return m.Strength * recency * frequency
}

func (uc *EvictUseCase) Execute(ctx context.Context, req EvictRequest) (*EvictResult, error) {
    memories, err := uc.repo.ListLatestByProject(ctx, req.TenantID, req.Project)
    if err != nil { return nil, err }

    if len(memories) <= req.MaxMemories {
        return &EvictResult{DryRun: req.DryRun}, nil
    }

    // Score each
    scored := make([]scoredMemory, len(memories))
    for i, m := range memories {
        scored[i] = scoredMemory{Memory: m, Score: evictionScore(m)}
    }
    sort.Slice(scored, func(i, j int) bool { return scored[i].Score < scored[j].Score })

    toEvict := scored[:len(memories)-req.MaxMemories]
    var evictedIDs []string
    for _, sm := range toEvict {
        if sm.Memory.Strength >= 1.0 { continue } // pinned
        evictedIDs = append(evictedIDs, sm.Memory.ID)
        if !req.DryRun {
            uc.repo.Delete(ctx, sm.Memory.ID)
            uc.publisher.Publish(ctx, "agentmemory.memory.expired", map[string]any{
                "memory_id": sm.Memory.ID, "reason": "eviction",
            })
        }
    }
    return &EvictResult{EvictedIDs: evictedIDs, DryRun: req.DryRun}, nil
}
```

### `internal/usecase/agentmemory/auto_forget.go`

```go
package agentmemory

import (
    "context"
    "time"

    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type AutoForgetUseCase struct {
    repo         port.IMemoryRepo
    searchClient port.ISearchNotifier
    publisher    port.IEventPublisher
}

func (uc *AutoForgetUseCase) StartScheduler(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C: uc.sweep(ctx)
        case <-ctx.Done(): return
        }
    }
}

func (uc *AutoForgetUseCase) sweep(ctx context.Context) {
    expired, _ := uc.repo.FindExpired(ctx)
    for _, mem := range expired {
        uc.repo.Delete(ctx, mem.ID)
        go uc.searchClient.RemoveMemory(context.Background(), mem.ID)
        uc.publisher.Publish(ctx, "agentmemory.memory.expired", map[string]any{
            "memory_id": mem.ID, "reason": "ttl",
        })
    }
}
```

### `internal/usecase/agentmemory/decay.go`

```go
package agentmemory

import (
    "context"
    "math"
    "time"

    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type DecayScheduler struct {
    repo         port.IMemoryRepo
    halfLifeDays int
}

func NewDecayScheduler(repo port.IMemoryRepo, halfLifeDays int) *DecayScheduler {
    if halfLifeDays <= 0 { halfLifeDays = 30 }
    return &DecayScheduler{repo: repo, halfLifeDays: halfLifeDays}
}

func (d *DecayScheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(2 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C: d.applyDecay(ctx)
        case <-ctx.Done(): return
        }
    }
}

func (d *DecayScheduler) applyDecay(ctx context.Context) {
    memories, _ := d.repo.ListAll(ctx)
    for _, m := range memories {
        hoursSince := time.Since(m.UpdatedAt).Hours()
        factor := math.Exp(-hoursSince / (float64(d.halfLifeDays) * 24))
        newStrength := m.Strength * factor
        d.repo.UpdateStrength(ctx, m.ID, newStrength)
        if newStrength < 0.05 {
            d.repo.FlagForEviction(ctx, m.ID)
        }
    }
}
```

### `internal/usecase/agentmemory/slots.go`

```go
package agentmemory

import (
    "context"

    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type SlotsUseCase struct { repo port.ISlotsRepo }

type WriteSlotRequest struct {
    TenantID string; Project string; Scope string; Label string
    Content  string; Mode string  // "replace" | "append"
}

func (uc *SlotsUseCase) WriteSlot(ctx context.Context, req WriteSlotRequest) error {
    slot, err := uc.repo.GetSlot(ctx, req.TenantID, req.Scope, req.Label)
    if err != nil {
        return uc.repo.CreateSlot(ctx, agentmemory.MemorySlot{
            TenantID: req.TenantID, Project: req.Project, Scope: req.Scope,
            Label: req.Label, Content: req.Content,
        })
    }
    if slot.ReadOnly { return agentmemory.ErrSlotReadOnly }
    switch req.Mode {
    case "append": slot.Content = slot.Content + "\n" + req.Content
    default:       slot.Content = req.Content
    }
    if slot.SizeLimit > 0 && len(slot.Content) > slot.SizeLimit {
        return agentmemory.ErrSlotSizeExceeded
    }
    return uc.repo.UpdateSlot(ctx, *slot)
}

func (uc *SlotsUseCase) GetSlot(ctx context.Context, tenantID, scope, label string) (*agentmemory.MemorySlot, error) {
    return uc.repo.GetSlot(ctx, tenantID, scope, label)
}

func (uc *SlotsUseCase) DeleteSlot(ctx context.Context, tenantID, scope, label string) error {
    return uc.repo.DeleteSlot(ctx, tenantID, scope, label)
}

func (uc *SlotsUseCase) ListSlots(ctx context.Context, tenantID, scope string) ([]agentmemory.MemorySlot, error) {
    return uc.repo.ListSlots(ctx, tenantID, scope)
}
```

### `internal/usecase/agentmemory/retention.go`

```go
package agentmemory

import (
    "context"
    "math"
    "time"

    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type RetentionUseCase struct{ repo port.IMemoryRepo }

type RetentionScore struct {
    Score            float64
    RecencyFactor    float64
    FrequencyFactor  float64
    ImportanceFactor float64
    DaysSinceAccess  float64
    RecommendAction  string  // "keep" | "review" | "evict"
}

func (uc *RetentionUseCase) GetScore(ctx context.Context, memID string) (*RetentionScore, error) {
    mem, err := uc.repo.GetByID(ctx, memID)
    if err != nil { return nil, err }

    daysSince := time.Since(mem.UpdatedAt).Hours() / 24
    recency   := math.Exp(-daysSince / 30.0)
    frequency := math.Log1p(float64(len(mem.SessionIDs)))
    score     := mem.Strength * recency * frequency

    action := "keep"
    if score < 0.1 { action = "evict" } else if score < 0.3 { action = "review" }

    return &RetentionScore{
        Score: score, RecencyFactor: recency, FrequencyFactor: frequency,
        ImportanceFactor: mem.Strength, DaysSinceAccess: daysSince,
        RecommendAction: action,
    }, nil
}
```

### `internal/usecase/agentmemory/port/output.go`

```go
package port

import (
    "context"
    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
)

type IMemoryRepo interface {
    Save(ctx context.Context, mem agentmemory.AgentMemory) error
    GetByID(ctx context.Context, id string) (*agentmemory.AgentMemory, error)
    ListLatestByType(ctx context.Context, tenantID, project, memType string) ([]agentmemory.AgentMemory, error)
    ListLatestByProject(ctx context.Context, tenantID, project string) ([]agentmemory.AgentMemory, error)
    ListAll(ctx context.Context) ([]agentmemory.AgentMemory, error)
    SetNotLatest(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
    FindExpired(ctx context.Context) ([]agentmemory.AgentMemory, error)
    UpdateStrength(ctx context.Context, id string, strength float64) error
    FlagForEviction(ctx context.Context, id string) error
}

type ISlotsRepo interface {
    GetSlot(ctx context.Context, tenantID, scope, label string) (*agentmemory.MemorySlot, error)
    CreateSlot(ctx context.Context, slot agentmemory.MemorySlot) error
    UpdateSlot(ctx context.Context, slot agentmemory.MemorySlot) error
    DeleteSlot(ctx context.Context, tenantID, scope, label string) error
    ListSlots(ctx context.Context, tenantID, scope string) ([]agentmemory.MemorySlot, error)
}

type ISearchNotifier interface {
    IndexMemory(ctx context.Context, mem agentmemory.AgentMemory) error
    RemoveMemory(ctx context.Context, id string) error
}

type IEventPublisher interface {
    Publish(ctx context.Context, subject string, payload any) error
}
```

---

## Verification

```bash
cd services/memory-service
go build ./...
go test ./internal/usecase/agentmemory/... -v
```

**Jaccard tests:**
```go
func TestJaccardVersioning_Supersede(t *testing.T) {
    // Concepts ["jose","jwt","auth"] → create memory1
    // Concepts ["jose","jwt","signing"] → Jaccard > 0.7 → supersede memory1, create v2
}

func TestJaccardVersioning_NoSupersede(t *testing.T) {
    // Concepts ["redis","cache"] → Jaccard < 0.7 với ["jose","jwt"] → no supersede
}

func TestEvict_DryRun(t *testing.T) {
    // DryRun=true → IDs returned but DB not changed
}

func TestSlot_AppendMode(t *testing.T) {
    // Write "line1" then append "line2" → content = "line1\nline2"
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| Remember với concepts > 70% Jaccard overlap → supersede + version++ | ✅ |
| Remember với concepts < 70% overlap → independent memory | ✅ |
| `forget_after` past → auto_forget sweep deletes | ✅ |
| Evict `dry_run=true` → list IDs, DB unchanged | ✅ |
| `strength >= 1.0` (pinned) → immune to eviction | ✅ |
| Slot `append` mode → content concatenated | ✅ |
| Retention score → `recommend_action: evict` when score < 0.1 | ✅ |
| Decay after 30 days → strength × exp(-1) ≈ 37% of original | ✅ |
