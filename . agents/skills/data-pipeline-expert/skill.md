---
skill_id: SKILL-010
version: 1.0.0
status: active
priority: P1
group: AI & Data Engineering
created_at: 2026-04-24
---

# SKILL-010 · Data Pipeline Engineering

## Mô tả

Thiết kế và vận hành các pipeline xử lý dữ liệu multi-stage — đảm bảo data integrity, idempotency, error recovery, observability cho toàn bộ luồng từ PRD input → KG → UI Schema.

## Agents sử dụng

- `requirement-parser-agent`
- `semantic-extractor-agent`
- `knowledge-graph-agent`
- `ui-schema-generator-agent`

---

## Năng lực cốt lõi

### 1. Pipeline Architecture (DAG-based)

```
Document Processing DAG:
━━━━━━━━━━━━━━━━━━━━━━━━

[Ingest] ──► [NLP Preprocess] ──► [LLM Extract] ──► [KG Build] ──► [Schema Gen]
              │                    │                  │              │
              ▼ checkpoint         ▼ checkpoint       ▼ checkpoint   ▼ checkpoint
           [nlp_done]          [extract_done]     [kg_done]      [schema_done]

Dead Letter Queue:
  - After 3 retries → DLQ
  - DLQ consumer → alert + human review
```

```go
// Pipeline stage definition
type Stage struct {
    Name       string
    Checkpoint string  // unique identifier for restart
    Execute    func(ctx context.Context, input StageInput) (StageOutput, error)
    Retry      RetryConfig
}

type RetryConfig struct {
    MaxAttempts int
    Backoff     []time.Duration  // [1s, 2s, 4s]
    RetryOn     []error          // specific errors to retry
}

// Pipeline executor with checkpoint-based restart
func (p *Pipeline) Run(ctx context.Context, jobID string) error {
    lastCheckpoint := p.loadCheckpoint(jobID)
    
    for _, stage := range p.stages {
        // Skip completed stages (idempotent restart)
        if lastCheckpoint.IsCompleted(stage.Checkpoint) {
            continue
        }
        
        output, err := p.runWithRetry(ctx, stage, p.state)
        if err != nil {
            p.sendToDLQ(jobID, stage.Name, err)
            return fmt.Errorf("stage %s failed: %w", stage.Name, err)
        }
        
        p.saveCheckpoint(jobID, stage.Checkpoint, output)
        p.state = output
    }
    return nil
}
```

### 2. Idempotency

```go
// Idempotency key generation — dựa trên content hash
func GenerateIdempotencyKey(docID string, content []byte) string {
    h := sha256.New()
    h.Write([]byte(docID))
    h.Write(content)
    return hex.EncodeToString(h.Sum(nil))
}

// Idempotent job creation
func (s *PipelineService) CreateJob(ctx context.Context, req *CreateJobRequest) (*Job, error) {
    key := GenerateIdempotencyKey(req.DocumentID, req.Content)
    
    // Check if job with same content already exists
    existing, err := s.repo.FindByIdempotencyKey(ctx, key)
    if err == nil && existing != nil {
        return existing, nil  // Return existing job — idempotent
    }
    
    return s.repo.CreateJob(ctx, &Job{
        IdempotencyKey: key,
        DocumentID:     req.DocumentID,
        Status:         JobStatusPending,
    })
}
```

### 3. Data Validation at Stage Boundaries

```go
// Input/output contracts for each stage
type NLPStageInput struct {
    RawText    string `validate:"required,min=10"`
    DocumentID string `validate:"required,uuid"`
    Language   string `validate:"oneof=vi en mixed"`
}

type NLPStageOutput struct {
    Sentences  []Sentence `validate:"required,min=1"`
    Paragraphs []Paragraph
    WordCount  int        `validate:"min=1"`
}

// Validate at every stage boundary
func validateStageInput(input any) error {
    validate := validator.New()
    if err := validate.Struct(input); err != nil {
        return fmt.Errorf("stage input validation failed: %w", err)
    }
    return nil
}
```

### 4. Error Recovery

```go
// Retry với exponential backoff + jitter
func (p *Pipeline) runWithRetry(ctx context.Context, stage Stage, input StageInput) (StageOutput, error) {
    var lastErr error
    
    for attempt := 0; attempt < stage.Retry.MaxAttempts; attempt++ {
        output, err := stage.Execute(ctx, input)
        if err == nil {
            return output, nil
        }
        
        // Check if retryable
        if !isRetryable(err, stage.Retry.RetryOn) {
            return nil, err  // Non-retryable error → fail immediately
        }
        
        lastErr = err
        delay := stage.Retry.Backoff[min(attempt, len(stage.Retry.Backoff)-1)]
        delay += jitter(delay * 10 / 100)  // ±10% jitter
        
        select {
        case <-time.After(delay):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### 5. Observability

```go
// OpenTelemetry distributed tracing
func (s *NLPStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
    ctx, span := tracer.Start(ctx, "nlp.preprocess",
        trace.WithAttributes(
            attribute.String("doc_id", input.DocumentID),
            attribute.Int("text_length", len(input.RawText)),
        ),
    )
    defer span.End()
    
    // Metrics
    start := time.Now()
    defer func() {
        pipelineLatency.WithLabelValues("nlp_preprocess").
            Observe(time.Since(start).Seconds())
    }()
    
    output, err := s.process(ctx, input)
    if err != nil {
        span.RecordError(err)
        pipelineErrors.WithLabelValues("nlp_preprocess").Inc()
        return nil, err
    }
    
    span.SetStatus(codes.Ok, "")
    pipelineThroughput.WithLabelValues("nlp_preprocess").Inc()
    return output, nil
}

// Prometheus metrics
var (
    pipelineLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "pipeline_stage_duration_seconds",
        Buckets: []float64{0.1, 0.5, 1, 5, 10, 30},
    }, []string{"stage"})
    
    pipelineErrors = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "pipeline_stage_errors_total",
    }, []string{"stage"})
)
```

### 6. Message Queue Integration

```go
// Kafka producer for async stage communication
type KafkaProducer struct {
    writer *kafka.Writer
}

func (p *KafkaProducer) PublishStageComplete(ctx context.Context, event StageCompleteEvent) error {
    payload, _ := json.Marshal(event)
    return p.writer.WriteMessages(ctx, kafka.Message{
        Topic: fmt.Sprintf("pipeline.stage.%s.completed", event.StageName),
        Key:   []byte(event.JobID),
        Value: payload,
        Headers: []kafka.Header{
            {Key: "idempotency-key", Value: []byte(event.IdempotencyKey)},
            {Key: "trace-id", Value: []byte(event.TraceID)},
        },
    })
}

// Kafka consumer with at-least-once delivery
func (c *StageConsumer) Start(ctx context.Context) error {
    for {
        msg, err := c.reader.FetchMessage(ctx)
        if err != nil {
            return err
        }
        
        if err := c.processMessage(ctx, msg); err != nil {
            // Log but don't commit — message will be redelivered
            log.Error("failed to process message", "error", err)
            continue
        }
        
        // Commit only after successful processing
        c.reader.CommitMessages(ctx, msg)
    }
}
```

### 7. Data Lineage

```go
// Tracing data từ source (PRD line) → target (UI element)
type LineageRecord struct {
    SourceDocID   string
    SourceLine    int
    SourceText    string
    TransformChain []TransformStep
    TargetType    string  // "Actor" | "Action" | "Screen" | "Component"
    TargetID      string
    CreatedAt     time.Time
}

type TransformStep struct {
    Stage      string  // "nlp", "llm_extraction", "kg_build"
    InputHash  string  // SHA256 of input
    OutputHash string  // SHA256 of output
    Timestamp  time.Time
}
```

---

## Checklist

- [ ] Mỗi pipeline stage có checkpoint key duy nhất
- [ ] Idempotency key được generate từ content hash
- [ ] Validation schemas định nghĩa rõ input/output của mỗi stage
- [ ] Retry strategy đã config (max attempts, backoff, retryable errors)
- [ ] Dead Letter Queue đã setup và có consumer xử lý
- [ ] OpenTelemetry traces propagate qua toàn bộ pipeline
- [ ] Prometheus metrics cho latency, throughput, error rate
- [ ] Data lineage records được lưu cho audit trail
- [ ] Kafka topics follow naming convention: `pipeline.stage.{name}.{event}`
