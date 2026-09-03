# SOL-ENT-005 — Solution: Unified Observability (Metrics, Traces, LLM Cost)

| Field | Value |
|---|---|
| **Solution ID** | SOL-ENT-005 |
| **CR** | [CR-ENT-005](../../../../docs/crs/v5/enterprise/CR-ENT-005-Unified-Observability.md) |
| **TDD ref** | [09-shared-packages.md](../../../tdd/architecture/09-shared-packages.md) §telemetry |
| **Status** | Open |
| **Priority** | 🟡 High |

---

## 1. Giải pháp

Dùng `shared/pkg/telemetry` (OpenTelemetry) để instrument tất cả services.

### 1.1 `shared/pkg/telemetry/metrics.go` [MODIFY]

```go
var (
    // Memory operation metrics
    MemoryStoreDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "vnp_memory_store_duration_ms",
            Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500},
        },
        []string{"engine", "type", "tenant"},
    )
    MemoryStoreTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_memory_store_total"},
        []string{"engine", "type", "status"},
    )
    MemoryRecallDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "vnp_memory_recall_duration_ms",
            Buckets: []float64{50, 100, 250, 500, 1000},
        },
        []string{"engine", "tenant"},
    )

    // LLM cost tracking
    LLMCallsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_llm_calls_total"},
        []string{"provider", "model", "task"},
    )
    LLMTokensTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_llm_tokens_total"},
        []string{"provider", "model", "token_type"},  // input/output
    )
    LLMCostUSD = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_llm_cost_usd_total"},
        []string{"provider", "model", "task"},
    )

    // Observe/AgentMemory
    ObserveHooksTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_observe_hooks_total"},
        []string{"hook_type", "agent"},
    )
    SessionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "vnp_observe_session_duration_seconds",
            Buckets: prometheus.ExponentialBuckets(1, 2, 10),
        },
        []string{"agent"},
    )
)

func Init() {
    prometheus.MustRegister(MemoryStoreDuration, MemoryStoreTotal,
        MemoryRecallDuration, LLMCallsTotal, LLMTokensTotal,
        LLMCostUSD, ObserveHooksTotal, SessionDuration)
}
```

### 1.2 Bifrost LLM cost interceptor

```go
// shared/pkg/telemetry/bifrost.go [NEW]
// Wrap LLM client to auto-track token usage and cost
func WrapLLMClient(client port.LLMClient) port.LLMClient {
    return &instrumentedLLM{inner: client}
}

func (l *instrumentedLLM) Complete(ctx context.Context, req *port.CompletionRequest) (*port.CompletionResponse, error) {
    start := time.Now()
    resp, err := l.inner.Complete(ctx, req)
    if err != nil { return nil, err }

    // Track tokens
    LLMTokensTotal.WithLabelValues(resp.Provider, resp.Model, "input").Add(float64(resp.InputTokens))
    LLMTokensTotal.WithLabelValues(resp.Provider, resp.Model, "output").Add(float64(resp.OutputTokens))

    // Estimate cost (per 1k tokens)
    inputCost := float64(resp.InputTokens) * costPerInputToken(resp.Model) / 1000
    outputCost := float64(resp.OutputTokens) * costPerOutputToken(resp.Model) / 1000
    LLMCostUSD.WithLabelValues(resp.Provider, resp.Model, req.Task).Add(inputCost + outputCost)
    LLMCallsTotal.WithLabelValues(resp.Provider, resp.Model, req.Task).Inc()

    return resp, nil
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `shared/pkg/telemetry/metrics.go` | MODIFY — add all metric vars |
| `shared/pkg/telemetry/bifrost.go` | NEW — LLM cost tracking interceptor |
| `gateway/adapter/handler/metrics.go` | VERIFY — GET /metrics exposed |
| `deployment/dev/grafana/dashboards/vnp-memory.json` | NEW — Grafana dashboard |

---

## 3. Acceptance Criteria

- [ ] `GET :8083/metrics` returns all Prometheus metrics
- [ ] LLM cost tracked per provider/model/task in real-time
- [ ] Grafana dashboard shows: store/recall latency, error rates, LLM cost per day
- [ ] Alert: `vnp_engine_health{engine} == 0` fires PagerDuty
