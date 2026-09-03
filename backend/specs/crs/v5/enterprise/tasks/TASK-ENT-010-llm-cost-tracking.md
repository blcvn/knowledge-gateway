# TASK-ENT-010 — LLM Cost Tracking Interceptor

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-010 |
| **Wave** | 3 |
| **Solution** | [SOL-ENT-005](../solutions/SOL-ENT-005-Unified-Observability.md) §1.2 |
| **Component** | `shared/pkg/telemetry/bifrost.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-ENT-009 |
| **Estimated** | 3h |

---

## Mục tiêu

Wrap LLM client to auto-track token usage and estimated cost per provider/model/task.

---

## Công việc cụ thể

### `shared/pkg/telemetry/bifrost.go` [NEW]

```go
package telemetry

import "github.com/prometheus/client_golang/prometheus"

var (
    LLMCallsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_llm_calls_total", Help: "Total LLM API calls"},
        []string{"provider", "model", "task"},
    )
    LLMTokensTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_llm_tokens_total", Help: "Total LLM tokens consumed"},
        []string{"provider", "model", "token_type"}, // input|output
    )
    LLMCostUSD = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "vnp_llm_cost_usd_total", Help: "Estimated LLM cost in USD"},
        []string{"provider", "model", "task"},
    )
)

func init() {
    prometheus.MustRegister(LLMCallsTotal, LLMTokensTotal, LLMCostUSD)
}

// InstrumentedLLM wraps any LLMClient to add metrics
type InstrumentedLLM struct {
    inner port.LLMClient
}

func WrapLLMClient(client port.LLMClient) port.LLMClient {
    return &InstrumentedLLM{inner: client}
}

func (l *InstrumentedLLM) Complete(ctx context.Context, req *port.CompletionRequest) (*port.CompletionResponse, error) {
    resp, err := l.inner.Complete(ctx, req)
    if err != nil { return nil, err }

    provider := resp.Provider; if provider == "" { provider = "bifrost" }
    model    := resp.Model;    if model == "" { model = "unknown" }
    task     := req.Task;     if task == "" { task = "general" }

    LLMCallsTotal.WithLabelValues(provider, model, task).Inc()
    LLMTokensTotal.WithLabelValues(provider, model, "input").Add(float64(resp.InputTokens))
    LLMTokensTotal.WithLabelValues(provider, model, "output").Add(float64(resp.OutputTokens))

    inputCost  := float64(resp.InputTokens) * costPerInputToken(model) / 1000.0
    outputCost := float64(resp.OutputTokens) * costPerOutputToken(model) / 1000.0
    LLMCostUSD.WithLabelValues(provider, model, task).Add(inputCost + outputCost)

    return resp, nil
}

// Cost table (per 1K tokens)
func costPerInputToken(model string) float64 {
    costs := map[string]float64{
        "gpt-4o":        0.005,
        "gpt-4o-mini":   0.00015,
        "claude-3-5-sonnet": 0.003,
    }
    if c, ok := costs[model]; ok { return c }
    return 0.001 // default estimate
}
func costPerOutputToken(model string) float64 { /* similar */ return 0.002 }
```

---

## Acceptance Criteria

- [ ] LLMCallsTotal incremented per call
- [ ] LLMTokensTotal tracks input and output separately
- [ ] LLMCostUSD estimated correctly for gpt-4o, claude-3-5-sonnet
- [ ] WrapLLMClient transparent (same interface)
- [ ] Unit test: verify metrics incremented after Complete()

## Files

```
shared/pkg/telemetry/bifrost.go       [NEW]
shared/pkg/telemetry/bifrost_test.go  [NEW]
```
