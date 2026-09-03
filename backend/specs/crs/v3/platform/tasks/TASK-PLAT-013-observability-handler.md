# TASK-PLAT-013 — Console Observability API Handlers

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-013 |
| **Wave** | 3 (Observability) |
| **Solution** | [SOL-PLAT-004](../solutions/SOL-PLAT-004-OpenTelemetry-Tracing.md) §2.5 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-010 |
| **Estimated** | 3h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** ObservabilityHandler: Metrics, Errors, Traces endpoints + Prometheus /metrics server (observability.go)
---

## Mục tiêu

Implement console observability API handlers: traces list, trace detail, error aggregation, LLM cost breakdown, metrics summary. Route qua `vnp-observability` gRPC service hoặc query Jaeger trực tiếp.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/observability.go` [NEW]

```go
package handler

type ObservabilityHandler struct {
    // Option A: proxy to vnp-observability gRPC service
    obsSvc port.ObservabilityServiceClient
    // Option B: query Jaeger HTTP API directly (simpler for v3)
    jaegerURL string
    httpClient *http.Client
}

// GET /v1/console/observability/traces?limit=100
func (h *ObservabilityHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    limit := queryIntDefault(r, "limit", 100)
    service := r.URL.Query().Get("service")  // optional filter

    // Query Jaeger API: GET /api/traces?service=vnp-gateway&limit=100
    jaegerURL := fmt.Sprintf("%s/api/traces?service=%s&limit=%d",
        h.jaegerURL, url.QueryEscape(service), limit)

    resp, err := h.httpClient.Get(jaegerURL)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "jaeger_error", err.Error())
        return
    }
    defer resp.Body.Close()

    // Forward Jaeger response (transformed to VNP format)
    var jaegerResp struct {
        Data []map[string]interface{} `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&jaegerResp)

    // Transform: pick traceID, rootSpan, duration, services, error flag
    traces := transformTraces(jaegerResp.Data, auth.TenantID)
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "traces": traces,
        "count":  len(traces),
    })
}

// GET /v1/console/observability/traces/{id}
func (h *ObservabilityHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
    traceID := chi.URLParam(r, "id")

    resp, err := h.httpClient.Get(fmt.Sprintf("%s/api/traces/%s", h.jaegerURL, traceID))
    if err != nil || resp.StatusCode == http.StatusNotFound {
        writeError(w, http.StatusNotFound, "trace_not_found", traceID)
        return
    }
    defer resp.Body.Close()

    var raw map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&raw)

    // Transform to waterfall format
    waterfall := transformToWaterfall(raw)
    writeJSON(w, http.StatusOK, waterfall)
}

// GET /v1/console/observability/errors?window=1h
func (h *ObservabilityHandler) ListErrors(w http.ResponseWriter, r *http.Request) {
    // Query Jaeger for traces with error tags
    // GET /api/traces?tags={"error":"true"}&limit=50
    // ... implementation ...
    writeJSON(w, http.StatusOK, map[string]interface{}{"errors": []interface{}{}})
}

// GET /v1/console/observability/costs?window=24h
func (h *ObservabilityHandler) GetCosts(w http.ResponseWriter, r *http.Request) {
    // Aggregate LLM cost from Prometheus metrics:
    // vnp_llm_cost_usd_total{tenant_id}
    // ... query Prometheus API ...
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "total_usd":  0.0,
        "by_model":   map[string]float64{},
        "by_service": map[string]float64{},
    })
}

// GET /v1/console/observability/metrics
func (h *ObservabilityHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    // Aggregate from Prometheus:
    // - vnp_memory_store_total, vnp_memory_recall_duration_ms
    // - vnp_llm_calls_total, vnp_llm_cost_usd_total
    writeJSON(w, http.StatusOK, map[string]interface{}{
        "memory_store_total":   0,
        "memory_recall_p95_ms": 0,
        "llm_calls_total":      0,
        "llm_cost_usd_total":   0.0,
    })
}
```

### 2. Modify `gateway/adapter/handler/router.go` [MODIFY] — register observability routes

```go
r.Route("/v1/console/observability", func(r chi.Router) {
    r.Use(requireAdmin)
    r.Get("/traces",      obsH.ListTraces)
    r.Get("/traces/{id}", obsH.GetTrace)
    r.Get("/errors",      obsH.ListErrors)
    r.Get("/costs",       obsH.GetCosts)
    r.Get("/metrics",     obsH.GetMetrics)
})
```

### 3. Add Jaeger config to `gateway/internal/infra/config/config.go` [MODIFY]

```go
type OTELConfig struct {
    Endpoint   string  // OTLP exporter endpoint
    SampleRate float64
    JaegerURL  string  // Jaeger HTTP API for query (e.g. http://jaeger:16686)
}
```

---

## Acceptance Criteria

- [ ] `GET /v1/console/observability/traces` returns last 100 traces from Jaeger
- [ ] `GET /v1/console/observability/traces/{id}` returns waterfall with parent → child spans
- [ ] `GET /v1/console/observability/errors` returns recent error traces
- [ ] `GET /v1/console/observability/costs` returns LLM cost breakdown
- [ ] `GET /v1/console/observability/metrics` returns aggregated metrics
- [ ] All endpoints require `admin` role
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/adapter/handler/observability.go   [NEW]
gateway/adapter/handler/router.go          [MODIFY — register /v1/console/observability/* routes]
gateway/internal/infra/config/config.go   [MODIFY — add JaegerURL]
```
