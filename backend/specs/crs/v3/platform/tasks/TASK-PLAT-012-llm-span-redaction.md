# TASK-PLAT-012 — LLM Span Instrumentation & Secret Redaction

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-012 |
| **Wave** | 3 (Observability) |
| **Solution** | [SOL-PLAT-004](../solutions/SOL-PLAT-004-OpenTelemetry-Tracing.md) §2.3–2.4 |
| **Component** | `shared/pkg/telemetry/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-010 |
| **Estimated** | 2h |

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** LLM span redaction (PII removal from trace attributes) not implemented
---

## Mục tiêu

Tạo helper functions để instrument LLM calls và DB queries với OTel spans. Implement secret redaction để không leak API keys/tokens vào span attributes.

---

## Công việc cụ thể

### 1. Tạo `shared/pkg/telemetry/llm_span.go` [NEW]

```go
package telemetry

import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// StartLLMSpan creates a child span for an LLM call
// Returns context with span + the span (caller must defer span.End())
func StartLLMSpan(ctx context.Context, model, task string) (context.Context, trace.Span) {
    tr := Tracer("llm")
    return tr.Start(ctx, "llm.complete",
        trace.WithAttributes(
            attribute.String("llm.model", model),
            attribute.String("llm.task", task),
        ),
    )
}

// RecordLLMResult adds token usage and cost to an LLM span
func RecordLLMResult(span trace.Span, inputTokens, outputTokens int64, costUSD float64) {
    span.SetAttributes(
        attribute.Int64("llm.input_tokens", inputTokens),
        attribute.Int64("llm.output_tokens", outputTokens),
        attribute.Float64("llm.cost_usd", costUSD),
    )
}

// StartDBSpan creates a child span for a database operation
func StartDBSpan(ctx context.Context, dbSystem, operation string) (context.Context, trace.Span) {
    tr := Tracer("db")
    return tr.Start(ctx, fmt.Sprintf("db.%s.query", dbSystem),
        trace.WithAttributes(
            attribute.String("db.system", dbSystem),     // "postgresql", "neo4j", "redis"
            attribute.String("db.operation", operation), // "SELECT", "INSERT", "MATCH"
        ),
    )
}

// StartEngineSpan creates a span for an engine dispatch operation
func StartEngineSpan(ctx context.Context, engineName, operation string) (context.Context, trace.Span) {
    tr := Tracer("engine")
    return tr.Start(ctx, fmt.Sprintf("engine.%s.%s", engineName, operation),
        trace.WithAttributes(
            attribute.String("engine.name", engineName),
            attribute.String("engine.operation", operation),
        ),
    )
}

// RecordError marks a span as failed with error details
func RecordError(span trace.Span, err error) {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

### 2. Tạo `shared/pkg/telemetry/redaction.go` [NEW]

```go
package telemetry

// sensitiveHeaders/keys: NEVER add these to span attributes
var sensitiveKeys = map[string]bool{
    "authorization":                          true,
    "x-api-key":                              true,
    "http.request.header.authorization":      true,
    "http.request.header.x-api-key":          true,
    "http.response.header.set-cookie":        true,
    "vnp_token":                              true,
    "refresh_token":                          true,
    "password":                               true,
    "secret":                                 true,
    "api_key":                                true,
}

// SafeAttribute creates an OTel attribute, redacting sensitive keys
func SafeAttribute(key string, value string) attribute.KeyValue {
    if sensitiveKeys[strings.ToLower(key)] {
        return attribute.String(key, "[REDACTED]")
    }
    return attribute.String(key, value)
}

// SafeAttributes filters a map of headers/attributes, redacting sensitive ones
func SafeAttributes(attrs map[string]string) []attribute.KeyValue {
    result := make([]attribute.KeyValue, 0, len(attrs))
    for k, v := range attrs {
        result = append(result, SafeAttribute(k, v))
    }
    return result
}

// AddHTTPAttributes adds safe HTTP attributes to a span (no auth headers)
func AddHTTPAttributes(span trace.Span, r *http.Request) {
    span.SetAttributes(
        attribute.String("http.method", r.Method),
        attribute.String("http.url", r.URL.Path),
        attribute.String("http.user_agent", r.UserAgent()),
        // Do NOT add: Authorization, X-API-Key, Cookie
    )
}
```

### 3. Usage pattern — how engine services should instrument LLM calls

```go
// Example usage in a service (graphiti-ingestion):

func (s *IngestionService) IngestEpisode(ctx context.Context, episode *domain.Episode) error {
    ctx, span := telemetry.StartEngineSpan(ctx, "graphiti", "ingest")
    defer span.End()

    // LLM entity extraction
    llmCtx, llmSpan := telemetry.StartLLMSpan(ctx, "gpt-4o-mini", "entity_extraction")
    entities, tokens, cost, err := s.llm.ExtractEntities(llmCtx, episode.Content)
    telemetry.RecordLLMResult(llmSpan, tokens.Input, tokens.Output, cost)
    llmSpan.End()
    if err != nil {
        telemetry.RecordError(span, err)
        return err
    }

    // DB upsert
    dbCtx, dbSpan := telemetry.StartDBSpan(ctx, "neo4j", "MERGE")
    err = s.neo4j.UpsertEntities(dbCtx, entities)
    dbSpan.End()
    if err != nil {
        telemetry.RecordError(span, err)
        return err
    }

    return nil
}
```

---

## Acceptance Criteria

- [ ] `StartLLMSpan()` creates child span with `llm.model` + `llm.task` attributes
- [ ] `RecordLLMResult()` adds `llm.input_tokens`, `llm.output_tokens`, `llm.cost_usd`
- [ ] `StartDBSpan()` creates child span with `db.system` + `db.operation`
- [ ] `SafeAttribute()` returns `[REDACTED]` for sensitive keys (Authorization, X-API-Key, etc.)
- [ ] LLM span attributes: no API keys, passwords, or secrets
- [ ] `go test ./shared/pkg/telemetry/...` passes

## Files

```
shared/pkg/telemetry/llm_span.go     [NEW]
shared/pkg/telemetry/redaction.go    [NEW]
```
