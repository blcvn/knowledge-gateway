# TASK-GR-021 — OTel Instrumentation (Ingestion Pipeline)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-021 |
| **Wave** | 4 (Admin & Observability) |
| **Component** | `services/graphiti-ingestion/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-007 §5, §6 |
| **Priority** | Medium |
| **Depends On** | TASK-GR-011 |
| **Estimated** | 3h |

---

## Context

Instrument `graphiti-ingestion` với OpenTelemetry traces + Prometheus metrics. Mỗi bước của 9-step pipeline có span riêng. Metrics: episode count, entity rate, edge rate, LLM latency, token usage.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-ingestion/internal/infra/otel/tracer.go` |
| CREATE | `services/graphiti-ingestion/internal/infra/metrics/prometheus.go` |
| MODIFY | `services/graphiti-ingestion/internal/usecase/ingest_episode.go` |

---

## Implementation

### File 1: `services/graphiti-ingestion/internal/infra/otel/tracer.go`

```go
package otel

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

const TracerName = "graphiti.ingestion"

func Tracer() trace.Tracer { return otel.Tracer(TracerName) }

// StartSpan starts a child span for a pipeline step
func StartSpan(ctx context.Context, stepName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
    ctx, span := Tracer().Start(ctx, "graphiti.ingestion."+stepName)
    if len(attrs) > 0 { span.SetAttributes(attrs...) }
    return ctx, span
}

// Common attributes
func GroupIDAttr(groupID string) attribute.KeyValue {
    return attribute.String("graphiti.group_id", groupID)
}

func SourceAttr(source string) attribute.KeyValue {
    return attribute.String("graphiti.episode.source", source)
}

func EpisodeUUIDAttr(uuid string) attribute.KeyValue {
    return attribute.String("graphiti.episode.uuid", uuid)
}
```

### File 2: `services/graphiti-ingestion/internal/infra/metrics/prometheus.go`

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    EpisodesIngested = promauto.NewCounterVec(prometheus.CounterOpts{
        Namespace: "graphiti",
        Subsystem: "ingestion",
        Name:      "episodes_total",
        Help:      "Total number of episodes ingested",
    }, []string{"group_id", "source", "status"})

    EntitiesExtracted = promauto.NewCounterVec(prometheus.CounterOpts{
        Namespace: "graphiti",
        Subsystem: "ingestion",
        Name:      "entities_extracted_total",
        Help:      "Total entities extracted from episodes",
    }, []string{"group_id", "resolution"}) // resolution: new|merged

    EdgesExtracted = promauto.NewCounterVec(prometheus.CounterOpts{
        Namespace: "graphiti",
        Subsystem: "ingestion",
        Name:      "edges_extracted_total",
        Help:      "Total edges extracted",
    }, []string{"group_id", "resolution"}) // resolution: new|duplicate|contradiction|update

    PipelineStepDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Namespace: "graphiti",
        Subsystem: "ingestion",
        Name:      "step_duration_seconds",
        Help:      "Duration of each pipeline step",
        Buckets:   prometheus.DefBuckets,
    }, []string{"step"})

    LLMCallDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Namespace: "graphiti",
        Subsystem: "llm",
        Name:      "call_duration_seconds",
        Help:      "LLM API call duration per prompt type",
        Buckets:   []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
    }, []string{"prompt_name", "provider", "model_size"})

    LLMTokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Namespace: "graphiti",
        Subsystem: "llm",
        Name:      "tokens_total",
        Help:      "Total LLM tokens consumed",
    }, []string{"prompt_name", "type"}) // type: prompt|completion

    WorkerQueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Namespace: "graphiti",
        Subsystem: "ingestion",
        Name:      "worker_queue_depth",
        Help:      "Current episode queue depth per group worker",
    }, []string{"group_id"})
)
```

### MODIFY `ingest_episode.go` — wrap each step with spans + metrics

```go
// At the top of Execute():
ctx, rootSpan := otel.StartSpan(ctx, "ingest_episode",
    otel.GroupIDAttr(req.GroupID),
    otel.SourceAttr(string(req.Source)),
)
defer func() {
    if err != nil { rootSpan.RecordError(err) }
    rootSpan.End()
}()

// Wrap Step 4 (extract entities):
{
    stepCtx, span := otel.StartSpan(ctx, "step4_extract_entities")
    stepStart := time.Now()
    extractEntitiesResp, err = uc.knowledge.ExtractEntities(stepCtx, ...)
    metrics.PipelineStepDuration.
        WithLabelValues("extract_entities").
        Observe(time.Since(stepStart).Seconds())
    span.End()
}

// After successful persist (Step 9):
metrics.EpisodesIngested.
    WithLabelValues(req.GroupID, string(req.Source), "success").Inc()
metrics.EntitiesExtracted.
    WithLabelValues(req.GroupID, "new").Add(float64(stats.EntitiesNew))
metrics.EntitiesExtracted.
    WithLabelValues(req.GroupID, "merged").Add(float64(stats.EntitiesExtracted - stats.EntitiesNew))
metrics.EdgesExtracted.
    WithLabelValues(req.GroupID, "new").Add(float64(stats.EdgesNew))

// Worker pool queue depth update (called from GroupWorkerPool.Stats periodically):
for groupID, depth := range pool.Stats() {
    metrics.WorkerQueueDepth.WithLabelValues(groupID).Set(float64(depth))
}
```

---

## Prometheus Alert Rules (to add to `deploy/dev/configs/prometheus.yml`)

```yaml
groups:
  - name: graphiti_ingestion
    rules:
      - alert: GraphitiHighPipelineLatency
        expr: histogram_quantile(0.95, graphiti_ingestion_step_duration_seconds_bucket{step="extract_entities"}) > 15
        for: 5m
        labels: { severity: warning }
        annotations: { summary: "Graphiti extract_entities p95 > 15s" }

      - alert: GraphitiWorkerQueueBacklog
        expr: graphiti_ingestion_worker_queue_depth > 50
        for: 2m
        labels: { severity: warning }
        annotations: { summary: "Graphiti worker queue backlog > 50 episodes" }

      - alert: GraphitiHighLLMTokenUsage
        expr: rate(graphiti_llm_tokens_total{type="completion"}[5m]) > 10000
        labels: { severity: warning }
        annotations: { summary: "High LLM completion token rate" }
```

---

## Verification

```bash
cd services/graphiti-ingestion
go build ./internal/infra/otel/...
go build ./internal/infra/metrics/...

# Check metrics endpoint
curl http://localhost:2112/metrics | grep graphiti_
```

**Expected metrics visible:**
- `graphiti_ingestion_episodes_total{group_id="test",source="text",status="success"}`
- `graphiti_ingestion_step_duration_seconds_bucket{step="extract_entities",...}`
- `graphiti_llm_tokens_total{prompt_name="extract_nodes",type="completion"}`
