# Solution: SOL-006 — Memory Consolidation Pipeline (4-Tier)

**CR ID:** CR-AM-006  
**Solution ID:** SOL-006  
**Priority:** High (Wave 2)  
**Architecture:** EXTEND `services/memory-service/` + Background scheduler + PostgreSQL

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §5.1`:
- `services/memory-service/` đã có `IngestUseCase` (Memobase pipeline): Blob → Buffer → TokenCount → Auto-flush.
- Memobase buffer flush (`FlushThreshold=20`) là closest thing to consolidation, nhưng chỉ trong session.
- VNP Memory có `Bifrost` LLM gateway — dùng cho LLM compression.
- NATS `agentmemory.session.ended` (từ CR-AM-001) sẽ trigger session summarization.

**Chiến lược:** Thêm `internal/consolidation/` vào `services/memory-service/` — chạy như background goroutine trong monolith.

---

## 2. Giải pháp

### 2.1. [NEW] `internal/consolidation/` trong `services/memory-service/`

```
services/memory-service/internal/consolidation/
├── pipeline.go         # ConsolidationPipeline — 4-tier orchestrator
├── compressor.go       # LLM + Synthetic compression
├── summarize.go        # Session summarization (Tier 3)
├── procedural.go       # Procedural memory extraction (Tier 4)
├── lessons.go          # Lessons & Insights extraction
└── decay.go            # Strength decay + lesson decay sweeps
```

### 2.2. ConsolidationPipeline

```go
// services/memory-service/internal/consolidation/pipeline.go

type ConsolidationPipeline struct {
    observeRepo   port.IObservationRepo     // read from observe-service's tables
    memRepo       port.IAgentMemoryRepo     // write AgentMemory
    summaryRepo   port.ISessionSummaryRepo  // write SessionSummary
    proceduralRepo port.IProceduralRepo
    lessonRepo    port.ILessonRepo
    insightRepo   port.IInsightRepo
    compressor    *Compressor
    summarizer    *Summarizer
    procedural    *ProceduralExtractor
    lessons       *LessonExtractor
    decay         *DecayService
    searchClient  port.ISearchNotifier
    publisher     port.IEventPublisher
    config        ConsolidationConfig
}

type ConsolidationConfig struct {
    IntervalHours       int
    DecayDays           int
    MaxMemoriesProject  int
    MinProcedureFreq    int
    LessonHalfLifeDays  int
}

func (p *ConsolidationPipeline) Start(ctx context.Context) {
    ticker := time.NewTicker(time.Duration(p.config.IntervalHours) * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            p.run(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (p *ConsolidationPipeline) run(ctx context.Context) {
    log.Info("consolidation pipeline starting")
    
    // Tier 1 → Tier 2: Compress uncapped raw observations
    p.step1WorkingToEpisodic(ctx)
    
    // Tier 2 → Tier 3: Summarize completed sessions
    p.step2EpisodicToSemantic(ctx)
    
    // Tier 3 → Tier 4: Distill patterns → procedural + lessons
    p.step3SemanticToProcedural(ctx)
    
    // Decay + Eviction
    p.step4DecayAndEvict(ctx)
    
    log.Info("consolidation pipeline completed")
}

func (p *ConsolidationPipeline) step1WorkingToEpisodic(ctx context.Context) {
    // Find sessions with raw observations but no compressed observations
    sessions, _ := p.observeRepo.ListSessionsNeedingCompression(ctx)
    for _, sess := range sessions {
        rawObs, _ := p.observeRepo.ListRawUncompressed(ctx, sess.ID)
        for _, raw := range rawObs {
            comp := p.compressor.Compress(ctx, raw)
            p.observeRepo.SaveCompressed(ctx, comp)
        }
    }
}

func (p *ConsolidationPipeline) step2EpisodicToSemantic(ctx context.Context) {
    // Find completed sessions without summaries
    sessions, _ := p.observeRepo.ListCompletedSessionsWithoutSummary(ctx)
    for _, sess := range sessions {
        summary := p.summarizer.Summarize(ctx, sess.ID)
        if summary != nil { p.summaryRepo.Save(ctx, *summary) }
    }
}

func (p *ConsolidationPipeline) step3SemanticToProcedural(ctx context.Context) {
    // Detect repeated workflows → ProceduralMemory
    p.procedural.ExtractAll(ctx)
    // Extract lessons from session summaries
    p.lessons.ExtractAll(ctx)
    // Synthesize insights from cross-lesson patterns
    p.lessons.SynthesizeInsights(ctx)
}

func (p *ConsolidationPipeline) step4DecayAndEvict(ctx context.Context) {
    p.decay.ApplyStrengthDecay(ctx)
    p.decay.ApplyLessonDecay(ctx)
    // Evict if over MaxMemoriesProject
    p.evictIfNeeded(ctx)
}
```

### 2.3. Compressor (LLM + Synthetic)

```go
// services/memory-service/internal/consolidation/compressor.go

type Compressor struct {
    llm           port.ILLMProvider
    circuitBreaker *CircuitBreaker
    autoCompress  bool
}

const compressionSystemPrompt = `You are a memory system. Compress this AI agent observation.
Return ONLY valid JSON:
{
  "title": "one-line action summary (max 80 chars)",
  "subtitle": "key detail or null",
  "facts": ["key fact 1", "key fact 2", "key fact 3"],
  "narrative": "2-3 sentence description of what happened and why",
  "concepts": ["entity1", "entity2"],
  "files": ["/path/to/affected/file"],
  "importance": 0.7
}`

func (c *Compressor) Compress(ctx context.Context, raw RawObservation) CompressedObservation {
    if c.autoCompress && c.llm != nil && c.circuitBreaker.Allow() {
        comp, err := c.compressWithLLM(ctx, raw)
        if err == nil { return comp }
        c.circuitBreaker.RecordFailure()
        // Fall through to synthetic
    }
    return c.syntheticCompress(raw)
}

func (c *Compressor) compressWithLLM(ctx context.Context, raw RawObservation) (CompressedObservation, error) {
    userMsg := fmt.Sprintf("HookType: %s\nToolName: %s\nToolOutput: %v",
        raw.HookType, raw.ToolName, raw.ToolOutput)
    
    resp, err := c.llm.Chat(ctx, compressionSystemPrompt, userMsg)
    if err != nil { return CompressedObservation{}, err }
    c.circuitBreaker.RecordSuccess()
    
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
        return CompressedObservation{}, err
    }
    return CompressedObservation{
        ID: newID(), SessionID: raw.SessionID, Timestamp: raw.Timestamp,
        Title: result.Title, Subtitle: result.Subtitle, Facts: result.Facts,
        Narrative: result.Narrative, Concepts: result.Concepts, Files: result.Files,
        Importance: result.Importance, SourceRawID: raw.ID,
    }, nil
}
```

### 2.4. Circuit Breaker for LLM

```go
// services/memory-service/internal/consolidation/circuit_breaker.go

type CircuitBreaker struct {
    mu           sync.Mutex
    state        string    // "closed" | "open" | "half-open"
    failureCount int
    lastFailure  time.Time
    threshold    int
    cooldown     time.Duration
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    switch cb.state {
    case "closed":   return true
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

### 2.5. Session Summarizer

```go
// services/memory-service/internal/consolidation/summarize.go

type Summarizer struct {
    obsRepo port.IObservationRepo
    llm     port.ILLMProvider
    cb      *CircuitBreaker
}

const summarizeSystemPrompt = `Summarize this AI agent coding session.
Return JSON:
{
  "title": "1 sentence session title",
  "narrative": "2-3 paragraph description",
  "key_decisions": ["decision 1", "decision 2"],
  "files_modified": ["/path/to/file"],
  "concepts": ["concept1", "concept2"]
}`

func (s *Summarizer) Summarize(ctx context.Context, sessionID string) *SessionSummary {
    compObs, _ := s.obsRepo.ListCompressed(ctx, sessionID)
    if len(compObs) == 0 { return nil }
    
    // Build input for LLM
    var lines []string
    for _, obs := range compObs {
        lines = append(lines, fmt.Sprintf("- %s: %s", obs.HookType, obs.Title))
        lines = append(lines, "  Facts: "+strings.Join(obs.Facts, "; "))
    }
    input := strings.Join(lines, "\n")
    
    summary := &SessionSummary{SessionID: sessionID, ObservationCount: len(compObs)}
    
    if s.llm != nil && s.cb.Allow() {
        resp, err := s.llm.Chat(ctx, summarizeSystemPrompt, input)
        if err == nil {
            var result struct {
                Title        string   `json:"title"`
                Narrative    string   `json:"narrative"`
                KeyDecisions []string `json:"key_decisions"`
                FilesModified []string `json:"files_modified"`
                Concepts     []string `json:"concepts"`
            }
            if json.Unmarshal([]byte(resp), &result) == nil {
                summary.Title = result.Title
                summary.Narrative = result.Narrative
                summary.KeyDecisions = result.KeyDecisions
                summary.FilesModified = result.FilesModified
                summary.Concepts = result.Concepts
                s.cb.RecordSuccess()
                return summary
            }
        }
        s.cb.RecordFailure()
    }
    
    // Synthetic summary: top-importance observations
    sortByImportance(compObs)
    summary.Title = compObs[0].Title
    summary.Narrative = strings.Join(extractNarratives(compObs[:min(3, len(compObs))]), " ")
    return summary
}
```

### 2.6. Procedural Memory Extraction

```go
// services/memory-service/internal/consolidation/procedural.go

type ProceduralExtractor struct {
    summaryRepo    port.ISessionSummaryRepo
    proceduralRepo port.IProceduralRepo
    llm            port.ILLMProvider
    minFrequency   int
}

// Algorithm:
// 1. Load all session summaries (key_decisions[])
// 2. Build step sequences from key_decisions
// 3. Find sequences that appear >= MinFrequency times (using LCS/shingling)
// 4. Create/update ProceduralMemory for repeated patterns

func (e *ProceduralExtractor) ExtractAll(ctx context.Context) {
    summaries, _ := e.summaryRepo.ListAll(ctx)
    sequences := extractSequences(summaries) // map[normalized_seq] → []sessionIDs
    
    for seq, sessionIDs := range sequences {
        if len(sessionIDs) < e.minFrequency { continue }
        
        existing, _ := e.proceduralRepo.FindByStepHash(ctx, hashSequence(seq))
        if existing != nil {
            // Update frequency
            e.proceduralRepo.IncrementFrequency(ctx, existing.ID)
            continue
        }
        
        proc := ProceduralMemory{
            ID:        newID(),
            Steps:     seq,
            Frequency: len(sessionIDs),
            Confidence: float64(len(sessionIDs)) / 10.0,
        }
        
        // LLM generate name + trigger + outcome
        if e.llm != nil {
            e.enrichWithLLM(ctx, &proc, seq)
        } else {
            proc.Name = fmt.Sprintf("Workflow: %s → %s", seq[0], seq[len(seq)-1])
        }
        
        e.proceduralRepo.Save(ctx, proc)
    }
}
```

### 2.7. NATS Consumer — Session Ended → Trigger Summarize

```go
// services/memory-service/internal/adapter/event/consumer.go

// Subscribe to agentmemory.session.ended from observe-service
// When received: trigger immediate session summarization (don't wait for 2h pipeline)

func (c *EventConsumer) handleSessionEnded(msg *nats.Msg) {
    var evt struct {
        SessionID        string `json:"session_id"`
        ObservationCount int    `json:"observation_count"`
    }
    json.Unmarshal(msg.Data, &evt)
    
    // Async summarize immediately after session end
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        summary := c.summarizer.Summarize(ctx, evt.SessionID)
        if summary != nil { c.summaryRepo.Save(ctx, *summary) }
    }()
}
```

### 2.8. PostgreSQL Schema (thêm vào migration)

```sql
-- Migration: 0013_consolidation.up.sql

CREATE TABLE session_summaries (
    session_id          TEXT PRIMARY KEY,
    tenant_id           TEXT NOT NULL,
    title               TEXT,
    narrative           TEXT,
    key_decisions       TEXT[] DEFAULT '{}',
    files_modified      TEXT[] DEFAULT '{}',
    concepts            TEXT[] DEFAULT '{}',
    observation_count   INT DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE procedural_memories (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           TEXT NOT NULL,
    project             TEXT,
    name                TEXT NOT NULL,
    steps               TEXT[] NOT NULL,
    step_hash           TEXT NOT NULL UNIQUE,  -- for dedup
    trigger_condition   TEXT,
    expected_outcome    TEXT,
    frequency           INT NOT NULL DEFAULT 1,
    confidence          FLOAT8 NOT NULL DEFAULT 0.5,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lessons (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT,
    content     TEXT NOT NULL,
    confidence  FLOAT8 NOT NULL DEFAULT 0.7,
    source      TEXT,
    categories  TEXT[] DEFAULT '{}',
    access_count INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE insights (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    content     TEXT NOT NULL,
    lesson_ids  UUID[] DEFAULT '{}',
    confidence  FLOAT8 NOT NULL DEFAULT 0.6,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_session_summaries_tenant ON session_summaries(tenant_id);
CREATE INDEX idx_procedural_memories_tenant ON procedural_memories(tenant_id, project);
CREATE INDEX idx_lessons_confidence ON lessons(tenant_id, confidence DESC);
```

### 2.9. Gateway Routes

```go
r.Post("/v1/memory/compress",            h.ForwardTo("memory-service", "ConsolidationService/CompressObservation"))
r.Post("/v1/memory/summarize",           h.ForwardTo("memory-service", "ConsolidationService/SummarizeSession"))
r.Post("/v1/memory/consolidate",         h.ForwardTo("memory-service", "ConsolidationService/RunPipeline"))
r.Get("/v1/memory/procedural",           h.ForwardTo("memory-service", "ConsolidationService/ListProcedural"))
r.Get("/v1/memory/procedural/{id}",      h.ForwardTo("memory-service", "ConsolidationService/GetProcedural"))
r.Get("/v1/memory/lessons",              h.ForwardTo("memory-service", "ConsolidationService/ListLessons"))
r.Get("/v1/memory/lessons/{id}",         h.ForwardTo("memory-service", "ConsolidationService/GetLesson"))
r.Post("/v1/memory/lessons/decay-sweep", h.ForwardTo("memory-service", "ConsolidationService/LessonDecaySweep"))
r.Get("/v1/memory/insights",             h.ForwardTo("memory-service", "ConsolidationService/ListInsights"))
```

### 2.10. Bootstrap Integration

```go
// apps/memory/internal/bootstrap/memory.go — MODIFY thêm consolidation

// NATS consumer for session.ended
natsConsumer.Subscribe("agentmemory.session.ended", eventConsumer.handleSessionEnded)

// Consolidation pipeline goroutine
consolidationPipeline := consolidation.NewPipeline(cfg.Consolidation, ...)
go consolidationPipeline.Start(context.Background())
```

---

## 3. Acceptance Criteria Mapping

| AC từ CR-AM-006 | Covered by |
|-----------------|------------|
| Background consolidation mỗi 2h | pipeline.Start() ticker |
| Session có summary sau kết thúc | NATS consumer handleSessionEnded() |
| Compress với AUTO_COMPRESS=true → facts[] từ LLM | compressor.compressWithLLM() |
| LLM không available → synthetic | circuit breaker + fallback |
| 3 LLM failures → circuit opens | CircuitBreaker.RecordFailure() |
| Memory strength sau 30 ngày = 50% | decay.ApplyStrengthDecay() |
| MaxMemories exceeded → eviction | step4DecayAndEvict() |
| 4-step workflow 3 lần → ProceduralMemory | procedural.ExtractAll() |
| lessons/decay-sweep → confidence giảm | lessons.ApplyDecay() |
