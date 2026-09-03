# Change Request: CR-ENT-005 — Unified Observability (Metrics, Traces, LLM Cost)

**CR ID:** CR-ENT-005
**Component:** `backend/services/vnp-observability`, `backend/shared/pkg/telemetry`
**Priority:** 🟡 High
**Status:** Open
**Version:** v5 / Enterprise & Operations
**Solution:** [S10 — Infrastructure Simplicity](../../../bussiness/solutions/S10-infrastructure-simplicity.md)
**Features:** [F24](../../../features/24-infrastructure-health/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-02 | Platform Engineer | Không monitor được latency, error rate per engine |
| PP-P8-02 | Product Manager | Không track được LLM cost — không optimize budget |

**Before:** 35+ services, metrics phân tán.
**After:** Single Prometheus endpoint + Grafana dashboard + LLM cost tracking.

---

## 2. Key Metrics

```prometheus
# Memory operations
vnp_memory_store_duration_ms{engine, type, tenant}     histogram
vnp_memory_recall_duration_ms{engine, tenant}           histogram
vnp_memory_store_total{engine, type, status}            counter
vnp_memory_recall_total{engine, status}                 counter

# LLM cost tracking
vnp_llm_calls_total{provider, model, task}              counter
vnp_llm_tokens_total{provider, model, type}             counter  # input/output
vnp_llm_cost_usd{provider, model, task}                 counter

# Agent observe
vnp_observe_hooks_total{hook_type, agent}               counter
vnp_observe_session_duration_seconds{agent}             histogram

# Infrastructure
vnp_engine_health{engine}                               gauge   # 1=up, 0=down
vnp_db_connection_pool_used{db}                         gauge
```

---

## 3. API & Endpoints

```http
# Prometheus metrics scrape
GET :8083/metrics
→ Prometheus text format

# LLM cost summary (Console UI)
GET /v1/console/analytics/llm-cost?from=2026-09-01&to=2026-09-03
→ {
    "total_usd": 12.45,
    "by_provider": {"openai": 8.20, "anthropic": 4.25},
    "by_task": {"extraction": 3.10, "consolidation": 9.35}
  }

# Memory usage analytics
GET /v1/console/analytics/memory?tenant_id=t_456
→ {
    "total_memories": 15847,
    "by_engine": {"graphiti": 4200, "cognee": 3100, ...},
    "storage_bytes": 28483920
  }
```

---

## 4. Thay đổi đề xuất

### 4.1 `backend/shared/pkg/telemetry/metrics.go` [MODIFY]

```go
var (
    MemoryStoreDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "vnp_memory_store_duration_ms",
        Help:    "Memory store operation duration",
        Buckets: []float64{10, 25, 50, 100, 250, 500, 1000},
    }, []string{"engine", "type", "tenant"})

    LLMCostUSD = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "vnp_llm_cost_usd",
        Help: "Cumulative LLM cost in USD",
    }, []string{"provider", "model", "task"})
)
```

---

## 5. Acceptance Criteria

- [ ] Prometheus scrape từ `:8083/metrics` hoạt động
- [ ] Grafana dashboard: latency p50/p95/p99 per engine
- [ ] LLM cost tracking: per provider, model, task type
- [ ] Alerts: p95 latency > 500ms → PagerDuty
- [ ] LLM cost budget alert: > $100/day → Slack notification
