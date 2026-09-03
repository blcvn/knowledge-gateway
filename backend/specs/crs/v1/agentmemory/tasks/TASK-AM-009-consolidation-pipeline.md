# TASK-AM-009 — Consolidation Pipeline (4-Tier)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-009 |
| **Wave** | 2 (Integration) |
| **Component** | `services/memory-service/internal/consolidation/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-006 §2.1 → §2.9 |
| **Priority** | High |
| **Depends On** | TASK-AM-006 |
| **Estimated** | 6h |

---

## Context

Thêm 4-tier consolidation pipeline vào `services/memory-service/`. Chạy như background goroutine: mỗi 2h trigger consolidation cycle. NATS consumer từ `agentmemory.session.ended` trigger immediate summarization.

**4 tiers:**
1. Working → Episodic: compress raw obs without summaries
2. Episodic → Semantic: summarize completed sessions → SessionSummary
3. Semantic → Procedural: extract repeated workflows + lessons + insights
4. Decay + Eviction

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/memory-service/internal/consolidation/pipeline.go` |
| CREATE | `services/memory-service/internal/consolidation/compressor.go` |
| CREATE | `services/memory-service/internal/consolidation/circuit_breaker.go` |
| CREATE | `services/memory-service/internal/consolidation/summarize.go` |
| CREATE | `services/memory-service/internal/consolidation/procedural.go` |
| CREATE | `services/memory-service/internal/consolidation/lessons.go` |
| CREATE | `services/memory-service/internal/domain/agentmemory/consolidation_entities.go` |
| CREATE | `services/memory-service/internal/adapter/event/consumer.go` |
| CREATE | `services/memory-service/internal/adapter/repository/postgres/consolidation_repo.go` |
| MODIFY | `apps/memory/internal/bootstrap/memory.go` |

---

## Implementation

### `internal/domain/agentmemory/consolidation_entities.go`

```go
package agentmemory

import "time"

type SessionSummary struct {
    SessionID        string
    TenantID         string
    Title            string
    Narrative        string
    KeyDecisions     []string
    FilesModified    []string
    Concepts         []string
    ObservationCount int
    CreatedAt        time.Time
}

type ProceduralMemory struct {
    ID               string
    TenantID         string
    Project          string
    Name             string
    Steps            []string
    StepHash         string    // for dedup
    TriggerCondition string
    ExpectedOutcome  string
    Frequency        int
    Confidence       float64
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type Lesson struct {
    ID          string
    TenantID    string
    Project     string
    Content     string
    Confidence  float64
    Source      string
    Categories  []string
    AccessCount int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Insight struct {
    ID         string
    TenantID   string
    Content    string
    LessonIDs  []string
    Confidence float64
    CreatedAt  time.Time
}
```

### `internal/consolidation/circuit_breaker.go`

```go
package consolidation

import (
    "sync"
    "time"
)

type CircuitBreaker struct {
    mu           sync.Mutex
    state        string  // "closed" | "open" | "half-open"
    failureCount int
    lastFailure  time.Time
    threshold    int
    cooldown     time.Duration
}

func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
    return &CircuitBreaker{state: "closed", threshold: threshold, cooldown: cooldown}
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    switch cb.state {
    case "closed": return true
    case "open":
        if time.Since(cb.lastFailure) > cb.cooldown {
            cb.state = "half-open"
            return true
        }
        return false
    case "half-open": return true
    }
    return true
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failureCount = 0
    cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failureCount++
    cb.lastFailure = time.Now()
    if cb.failureCount >= cb.threshold { cb.state = "open" }
}
```

### `internal/consolidation/compressor.go`

```go
package consolidation

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

const compressionSystemPrompt = `You are a memory system. Compress this AI agent observation.
Return ONLY valid JSON:
{
  "title": "one-line action summary (max 80 chars)",
  "subtitle": "key detail or null",
  "facts": ["key fact 1", "key fact 2", "key fact 3"],
  "narrative": "2-3 sentence description",
  "concepts": ["entity1", "entity2"],
  "files": ["/path/to/file"],
  "importance": 0.7
}`

type Compressor struct {
    llm            port.ILLMProvider
    cb             *CircuitBreaker
    autoCompress   bool
}

func NewCompressor(llm port.ILLMProvider, autoCompress bool) *Compressor {
    return &Compressor{
        llm:          llm,
        cb:           NewCircuitBreaker(3, 5*time.Minute),
        autoCompress: autoCompress,
    }
}

func (c *Compressor) Compress(ctx context.Context, raw agentmemory.RawObs) agentmemory.CompressedObs {
    if c.autoCompress && c.llm != nil && c.cb.Allow() {
        comp, err := c.compressWithLLM(ctx, raw)
        if err == nil { c.cb.RecordSuccess(); return comp }
        c.cb.RecordFailure()
    }
    return syntheticCompress(raw)
}

func (c *Compressor) compressWithLLM(ctx context.Context, raw agentmemory.RawObs) (agentmemory.CompressedObs, error) {
    userMsg := fmt.Sprintf("HookType: %s\nToolName: %s\nOutput: %s",
        raw.HookType, raw.ToolName, string(raw.ToolOutput))

    resp, err := c.llm.Chat(ctx, compressionSystemPrompt, userMsg)
    if err != nil { return agentmemory.CompressedObs{}, err }

    var result struct {
        Title     string   `json:"title"`
        Subtitle  string   `json:"subtitle"`
        Facts     []string `json:"facts"`
        Narrative string   `json:"narrative"`
        Concepts  []string `json:"concepts"`
        Files     []string `json:"files"`
        Importance float64 `json:"importance"`
    }
    if err := json.Unmarshal([]byte(resp), &result); err != nil {
        return agentmemory.CompressedObs{}, err
    }
    return agentmemory.CompressedObs{
        SessionID: raw.SessionID, Title: result.Title, Subtitle: result.Subtitle,
        Facts: result.Facts, Narrative: result.Narrative, Concepts: result.Concepts,
        Files: result.Files, Importance: result.Importance,
    }, nil
}

// syntheticCompress — same as observe-service pipeline, zero LLM
func syntheticCompress(raw agentmemory.RawObs) agentmemory.CompressedObs { ... }
```

### `internal/consolidation/pipeline.go`

```go
package consolidation

import (
    "context"
    "log"
    "time"

    "github.com/vnp-memory/services/memory-service/internal/consolidation/port"
)

type ConsolidationConfig struct {
    IntervalHours      int     // default 2
    DecayDays          int     // default 30
    MaxMemoriesProject int     // default 1000
    MinProcedureFreq   int     // default 3
    LessonHalfLifeDays int     // default 90
    AutoCompress       bool
}

type ConsolidationPipeline struct {
    observeRepo   port.IObservationRepo
    memRepo       port.IAgentMemoryRepo
    summaryRepo   port.ISessionSummaryRepo
    proceduralRepo port.IProceduralRepo
    lessonRepo    port.ILessonRepo
    compressor    *Compressor
    summarizer    *Summarizer
    procedural    *ProceduralExtractor
    lessons       *LessonExtractor
    config        ConsolidationConfig
}

func (p *ConsolidationPipeline) Start(ctx context.Context) {
    interval := time.Duration(p.config.IntervalHours) * time.Hour
    if interval == 0 { interval = 2 * time.Hour }
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C: p.run(ctx)
        case <-ctx.Done(): return
        }
    }
}

func (p *ConsolidationPipeline) run(ctx context.Context) {
    log.Println("[consolidation] pipeline starting")
    p.step1WorkingToEpisodic(ctx)
    p.step2EpisodicToSemantic(ctx)
    p.step3SemanticToProcedural(ctx)
    p.step4DecayAndEvict(ctx)
    log.Println("[consolidation] pipeline completed")
}

func (p *ConsolidationPipeline) step1WorkingToEpisodic(ctx context.Context) {
    sessions, _ := p.observeRepo.ListSessionsNeedingCompression(ctx)
    for _, sess := range sessions {
        rawObs, _ := p.observeRepo.ListRawUncompressed(ctx, sess.ID)
        for _, raw := range rawObs {
            comp := p.compressor.Compress(ctx, raw)
            comp.ID = uuid.New().String()
            p.observeRepo.SaveCompressed(ctx, comp)
        }
    }
}

func (p *ConsolidationPipeline) step2EpisodicToSemantic(ctx context.Context) {
    sessions, _ := p.observeRepo.ListCompletedSessionsWithoutSummary(ctx)
    for _, sess := range sessions {
        summary := p.summarizer.Summarize(ctx, sess.ID)
        if summary != nil { p.summaryRepo.Save(ctx, *summary) }
    }
}

func (p *ConsolidationPipeline) step3SemanticToProcedural(ctx context.Context) {
    p.procedural.ExtractAll(ctx)
    p.lessons.ExtractAll(ctx)
    p.lessons.SynthesizeInsights(ctx)
}

func (p *ConsolidationPipeline) step4DecayAndEvict(ctx context.Context) {
    p.lessons.ApplyDecay(ctx)
    // Evict memories over max limit
    p.evictIfNeeded(ctx)
}

// SummarizeNow is called immediately from NATS consumer
func (p *ConsolidationPipeline) SummarizeNow(ctx context.Context, sessionID string) {
    summary := p.summarizer.Summarize(ctx, sessionID)
    if summary != nil { p.summaryRepo.Save(ctx, *summary) }
}
```

### `internal/adapter/event/consumer.go`

```go
package event

import (
    "context"
    "encoding/json"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/vnp-memory/services/memory-service/internal/consolidation"
)

type EventConsumer struct {
    pipeline *consolidation.ConsolidationPipeline
}

func (c *EventConsumer) Subscribe(nc *nats.Conn) error {
    _, err := nc.Subscribe("agentmemory.session.ended", c.handleSessionEnded)
    return err
}

func (c *EventConsumer) handleSessionEnded(msg *nats.Msg) {
    var evt struct {
        SessionID        string `json:"session_id"`
        ObservationCount int    `json:"observation_count"`
    }
    json.Unmarshal(msg.Data, &evt)

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        c.pipeline.SummarizeNow(ctx, evt.SessionID)
    }()
}
```

---

## Verification

```bash
cd services/memory-service
go build ./...
go test ./internal/consolidation/... -v
```

**Tests:**
```go
func TestCompressor_FallbackToSynthetic(t *testing.T) {
    // LLM returns error → synthetic compress used
}

func TestCircuitBreaker_OpenAfter3Failures(t *testing.T) {
    cb := NewCircuitBreaker(3, 5*time.Minute)
    cb.RecordFailure(); cb.RecordFailure(); cb.RecordFailure()
    assert.False(t, cb.Allow())  // circuit is open
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| Background pipeline runs every 2h | ✅ |
| `session.ended` NATS → immediate summarization | ✅ |
| AUTO_COMPRESS=true → LLM compression with facts[] | ✅ |
| LLM unavailable → synthetic fallback | ✅ |
| 3 LLM failures → circuit breaker opens | ✅ |
| Procedural memory extracted from ≥ 3 repeated workflows | ✅ |
| Memory strength decay after 30 days ≈ 50% | ✅ |
