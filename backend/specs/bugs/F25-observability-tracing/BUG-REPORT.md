# Bug Report — F25: Observability & Tracing

> Feature: Metrics, traces, errors, cost analytics
> Luồng: `apps/memory → gateway/console.go (ObservabilityHandler) → obs-service`

---

## BUG-F25-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:668-706`

---

## BUG-F25-002: `obs-service` Và `vnp-observability` Chưa Có Unified Implementation

**Severity:** HIGH  
**File:** `services/obs-service/`, `services/vnp-observability/`

**Mô tả:**  
Observability handler forward tới `obs-service`. Cần verify service này implement:
- `/v1/console/observability/metrics` — LLM cost, token usage, latency
- `/v1/console/observability/traces` — distributed trace viewer
- `/v1/console/observability/errors` — error aggregation
- `/v1/console/observability/costs` — cost analytics

**Impact:**  
- Nếu `obs-service` chưa implement các paths này, observability console không hoạt động.

---

## BUG-F25-003: MigrateObservabilitySchema Được Gọi Nhưng Gateway Không Serve Observability Data Trực Tiếp

**Severity:** LOW  
**File:** `gateway/cmd/main.go:120-125`

**Mô tả:**  
```go
persistence.MigrateObservabilitySchema(migrateCtx, pool)
// Creates: error_aggregates, llm_cost_events
```

Schema được tạo nhưng không có gateway-level handlers để write/read từ các tables này directly. Data được expected từ downstream services.

**Impact:**  
- `error_aggregates` và `llm_cost_events` tables tồn tại trong gateway DB nhưng không bao giờ được populated.
