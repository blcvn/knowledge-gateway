# Bug Report — F23: Pipeline Monitor

> Feature: Pipeline status, queues, workers, templates, engine jobs
> Luồng: `apps/memory → gateway/console.go (PipelineHandler) → pipeline-service`

---

## BUG-F23-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:540-594`

---

## BUG-F23-002: `pipeline-service` Không Có Implementation

**Severity:** CRITICAL  
**File:** `services/pipeline-service/`

**Mô tả:**  
`pipeline-service` directory tồn tại nhưng không có implementation code. Tất cả pipeline monitor endpoints forward tới service này.

**Impact:**  
- Pipeline Monitor console không hoạt động end-to-end.
- Không có job scheduling, queue monitoring, worker management.

---

## BUG-F23-003: `PipelineUseCase` Không Được Wire Vào PipelineHandler

**Severity:** HIGH  
**File:** `gateway/cmd/main.go:203`

**Mô tả:**  
```go
_ = usecase.NewPipelineUseCase(registry, logger)
```

`PipelineUseCase.AggregateStatus()` có fan-out health check logic nhưng không được inject vào `PipelineHandler`. Handler forward tới non-existent `pipeline-service` thay vì dùng built-in aggregation.

---

## BUG-F23-004: Route Ambiguity — `GET /v1/console/pipelines/status` vs `GET /v1/console/pipelines/{engine}`

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/router.go:232,236`

**Mô tả:**  
```go
mux.HandleFunc("GET /v1/console/pipelines/status", pipeline.Status)      // literal
mux.HandleFunc("GET /v1/console/pipelines/{engine}", pipeline.GetEngine) // wildcard
```

Request `/v1/console/pipelines/status` có thể match với `{engine} = "status"`.

**Impact:**  
- `Status` handler có thể bị bypass, `GetEngine` được gọi với `engine="status"`.
