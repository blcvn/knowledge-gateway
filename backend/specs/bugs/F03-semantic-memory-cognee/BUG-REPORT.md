# Bug Report — F03: Semantic Memory (Cognee)

> Feature: `POST /v1/cognee/datasets`, `/upload`, `/cognify`, `/search`, `/memify`, `/datapoints`
> Luồng: `apps/memory → gateway/adapter/handler/services.go (CogneeHandler) → kg-service`

---

## BUG-F03-001: Tất Cả Cognee Endpoints Route Tới `kg-service` Không Phải Cognee Services

**Severity:** HIGH  
**File:** `gateway/adapter/handler/services.go:20-53`

**Mô tả:**  
`CogneeHandler` forward mọi request tới `kg-service`. Theo feature spec và kiến trúc, Cognee có các microservices riêng: `cognee-ingestion`, `cognee-cognify`, `cognee-search`.

```go
func (h *CogneeHandler) CreateDataset(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "kg-service", h.logger)(w, r)  // Nên là "cognee-ingestion"
}
func (h *CogneeHandler) Cognify(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "kg-service", h.logger)(w, r)  // Nên là "cognee-cognify"
}
func (h *CogneeHandler) Search(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "kg-service", h.logger)(w, r)  // Nên là "cognee-search"
}
```

**Impact:**  
- Nếu `kg-service` không implement Cognee protocol, tất cả requests fail.
- Cognify job và search sẽ trả về wrong response format.

---

## BUG-F03-002: Thiếu Services `cognee-*` Implementations

**Severity:** HIGH  
**File:** `services/cognee-*` directories

**Mô tả:**  
Các services `cognee-ingestion`, `cognee-cognify`, `cognee-search`, `cognee-pipeline` tồn tại như thư mục nhưng thiếu `main.go` và gRPC server implementation. Xem cấu trúc:

- `services/cognee-cognify/` — không có main.go
- `services/cognee-ingestion/` — không có main.go  
- `services/cognee-search/` — không có main.go
- `services/cognee-pipeline/` — không có main.go

**Impact:**  
- Semantic Memory không thể hoạt động end-to-end.

---

## BUG-F03-003: `NATS publish "memory.blob.inserted"` Không Được Implement

**Severity:** MEDIUM  
**File:** `gateway/usecase/route.go:61-79`

**Mô tả:**  
Sau khi forward request tới cognee-ingestion, feature spec yêu cầu publish NATS event `memory.blob.inserted`. Code hiện tại không publish bất kỳ event nào sau store.

```go
// route.go: Sau khi forward, không có publisher.Publish()
result := &domain.RouteResult{...}
// MISSING: uc.publisher.Publish(ctx, "memory.blob.inserted", ...)
return result, nil
```

**Impact:**  
- Pipeline downstream (consolidation trigger, event tracking) không nhận được signal.
- Timeline feature (`GET /v1/memory/timeline`) sẽ thiếu events.
