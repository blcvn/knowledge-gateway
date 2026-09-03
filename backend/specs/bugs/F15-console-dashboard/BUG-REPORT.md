# Bug Report — F15: Console Dashboard

> Feature: Health, Metrics, Throughput, Heatmap cho Console
> Luồng: `apps/memory → gateway/console.go (DashboardHandler) → vnp-platform`

---

## BUG-F15-001: Dashboard Endpoints Require `requireAdmin()` Nhưng Auth Middleware Không Apply (Xem BUG-F14-001)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:60-90`

**Mô tả:**  
Dashboard handlers dùng `requireAdmin()` nhưng vì Auth middleware không được áp dụng (BUG-F14-001), `AuthFromContext` luôn trả về `nil`, và `requireAdmin()` luôn trả về 401 Unauthorized.

```go
func (h *DashboardHandler) Health(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) {  // Luôn trả về 401 vì auth context là nil
        return
    }
    ...
}
```

**Impact:**  
- Console Dashboard không accessible ngay cả với admin user.

---

## BUG-F15-002: `vnp-platform` Không Có Implementation Cho Dashboard APIs

**Severity:** HIGH  
**File:** `services/vnp-platform/`, `services/vnp-dashboard/`

**Mô tả:**  
Dashboard handler forward tới `vnp-platform` service. Cần verify `vnp-platform` có implement:
- `GET /v1/console/dashboard/health`
- `GET /v1/console/dashboard/metrics`
- `GET /v1/console/dashboard/throughput`
- `GET /v1/console/dashboard/heatmap`

Nếu `vnp-platform` chỉ là generic platform service không biết dashboard paths, requests sẽ fail.

---

## BUG-F15-003: `PipelineUseCase.AggregateStatus` Và `InfraUseCase.Topology` Không Được Wire Vào Dashboard Handler

**Severity:** MEDIUM  
**File:** `gateway/cmd/main.go:202-206`

**Mô tả:**  
`PipelineUseCase` và `InfraUseCase` có `AggregateStatus()` và `Topology()` methods — implement health fan-out. Tuy nhiên chúng bị gán vào `_` trong main.go:

```go
_ = usecase.NewPipelineUseCase(registry, logger)
_ = usecase.NewInfraUseCase(registry, logger)
```

Dashboard handler không sử dụng các usecases này — forward tới `vnp-platform` thay vì dùng fan-out logic đã implement.

**Impact:**  
- Real-time health aggregation không được dùng.
- Dashboard phụ thuộc vào `vnp-platform` thay vì direct fan-out.
