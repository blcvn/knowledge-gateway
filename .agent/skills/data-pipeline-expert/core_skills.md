# Core Skills — Data Pipeline Engineering

## Pipeline Architecture Patterns

### Multi-Stage Pipeline Design (for this Platform)
```
[Stage 1: Ingest]        Input: Raw PRD document
        ↓                Output: Normalized document chunks
[Stage 2: Parse]         Input: Document chunks
        ↓                Output: Classified blocks (Actor/Flow/Rule)
[Stage 3: Extract]       Input: Classified blocks
        ↓                Output: Semantic entities JSON
[Stage 4: KG Write]      Input: Semantic entities
        ↓                Output: Knowledge Graph updated
[Stage 5: Schema Gen]    Input: KG queries
        ↓                Output: JSON UI Schema
[Stage 6: Validate]      Input: UI Schema + KG + PRD
                         Output: Traceability report
```

### Stage Contracts (Input/Output Schemas)
Every stage defines explicit schemas using JSON Schema or Go structs:
```go
// Stage 2 Output / Stage 3 Input
type ParsedBlock struct {
    ID          string    `json:"id"`
    Type        BlockType `json:"type"` // Actor | Flow | Rule | Constraint
    Content     string    `json:"content"`
    SourceLines []int     `json:"source_lines"` // For traceability
    Confidence  float64   `json:"confidence"`
}
```

## Idempotency Implementation

### Idempotency Keys
Every pipeline run is assigned a unique `run_id`. Each stage checks:
```go
func (s *ExtractStage) Run(ctx context.Context, input StageInput) (StageOutput, error) {
    // Check if this stage already completed for this run_id
    cached, err := s.store.GetStageResult(ctx, input.RunID, "extract")
    if err == nil && cached != nil {
        return cached, nil // Return cached result — idempotent
    }
    // ... execute stage ...
    s.store.SaveStageResult(ctx, input.RunID, "extract", result)
    return result, nil
}
```

### Checkpointing Strategy
- Persist stage outputs to durable storage (Redis with TTL / PostgreSQL) after each stage completes.
- On pipeline restart, read checkpoint and resume from the first failed stage.
- Checkpoint key format: `pipeline:{run_id}:stage:{stage_name}:status`

## Error Handling & Retry Strategy

### Per-Stage Error Classification
| Error Type | Retry? | Strategy |
|---|---|---|
| Transient (network timeout, LLM rate limit) | ✅ Yes | Exponential backoff with jitter (max 3 retries) |
| Validation (malformed output) | ✅ Yes (once) | Retry with correction prompt |
| Fatal (schema mismatch, auth failure) | ❌ No | Move to dead-letter, alert |
| Partial (some entities extracted, some failed) | ✅ Partial | Continue with successful items, log failures |

### Dead-Letter Handling
Failed stage inputs that exhaust retries are written to a dead-letter store:
```go
type DeadLetterEntry struct {
    RunID     string          `json:"run_id"`
    Stage     string          `json:"stage"`
    Input     json.RawMessage `json:"input"`
    Error     string          `json:"error"`
    Attempts  int             `json:"attempts"`
    CreatedAt time.Time       `json:"created_at"`
}
```

## Observability
- **Stage Metrics (Prometheus):** `pipeline_stage_duration_seconds`, `pipeline_stage_errors_total`, `pipeline_stage_items_processed_total`
- **Distributed Tracing (OpenTelemetry):** Each stage creates a child span with `run_id` and `stage_name` as attributes.
- **Data Lineage:** Every output entity carries `source_block_id` and `source_lines` pointing back to the original PRD text.
