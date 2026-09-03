# Bug Report — F24: Infrastructure Health

> Feature: Topology graph, service list/detail, databases, resources, deployments
> Luồng: `apps/memory → gateway/console.go (InfraHandler) → obs-service`

---

## BUG-F24-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:608-654`

---

## BUG-F24-002: `InfraUseCase.Topology()` Không Được Wire Vào InfraHandler

**Severity:** HIGH  
**File:** `gateway/cmd/main.go:204`

**Mô tả:**  
```go
_ = usecase.NewInfraUseCase(registry, logger)
```

`InfraUseCase.Topology()` thực hiện parallel health check trực tiếp với gRPC registry — không phụ thuộc vào external service. Nhưng `InfraHandler` forward tới `obs-service` thay vì dùng built-in topology logic.

**Impact:**  
- Topology data không được aggregate từ actual gRPC health checks.
- `obs-service` cần implement topology endpoint riêng — duplicate logic.

---

## BUG-F24-003: `allServices` List Trong InfraUseCase Bao Gồm `zep-core` Không Tồn Tại

**Severity:** LOW  
**File:** `gateway/usecase/console.go:175`

**Mô tả:**  
```go
var allServices = []string{
    ...
    "zep-user", "zep-thread", "zep-memory", "zep-graph", "zep-search", "zep-admin", "zep-core",
    ...
}
```

`zep-core` không có directory trong `services/`. Health check sẽ luôn fail cho service này.

**Impact:**  
- Topology report luôn hiển thị `zep-core` là unhealthy — misleading.
