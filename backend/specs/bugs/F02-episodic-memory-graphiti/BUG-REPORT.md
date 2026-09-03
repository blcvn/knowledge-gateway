# Bug Report — F02: Episodic Memory (Graphiti)

> Feature: `POST /v1/graphiti/episodes`, `POST /v1/graphiti/search`, `GET /v1/graphiti/nodes/{id}`, `GET /v1/graphiti/edges/{id}`
> Luồng: `apps/memory → gateway/adapter/handler/services.go (GraphitiHandler) → kg-service`

---

## BUG-F02-001: Graphiti Handler Route Tới `kg-service` Thay Vì Đúng Service (Sai Routing)

**Severity:** HIGH  
**File:** `gateway/adapter/handler/services.go:65-79`

**Mô tả:**  
Tất cả `GraphitiHandler` methods đều forward tới `kg-service`. Theo feature spec, Graphiti yêu cầu các microservices riêng biệt: `graphiti-ingestion`, `graphiti-search`, `graphiti-knowledge`.

```go
func (h *GraphitiHandler) IngestEpisode(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "kg-service", h.logger)(w, r)  // Sai: nên là "graphiti-ingestion"
}
func (h *GraphitiHandler) Search(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "graphiti-search", h.logger)(w, r)  // Sai: nên là "graphiti-search"
}
func (h *GraphitiHandler) GetNode(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "kg-service", h.logger)(w, r)  // Sai: nên là "graphiti-knowledge"
}
func (h *GraphitiHandler) GetEdge(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "kg-service", h.logger)(w, r)  // Sai: nên là "graphiti-knowledge"
}
```

**Impact:**  
- Nếu `kg-service` không handle `/v1/graphiti/*` paths, tất cả Graphiti requests sẽ fail.
- Tách biệt giữa ingestion và search bị phá vỡ (routing lộn xộn).

---

## BUG-F02-002: gRPC ForwardService Dùng Raw Bytes Không Typed Proto (Architectural)

**Severity:** HIGH  
**File:** `gateway/adapter/client/registry.go:133`

**Mô tả:**  
Gateway forward requests xuống service bằng cách invoke `/vnp.gateway.forward.v1.ForwardService/Forward` với raw bytes. Không có typed gRPC proto contract nào được sử dụng. `conn.Invoke` expects protobuf-encoded message nhưng nhận raw JSON bytes.

```go
err := conn.Invoke(ctx, "/vnp.gateway.forward.v1.ForwardService/Forward", req.Body, &resp)
```

**Impact:**  
- `req.Body` là JSON bytes, không phải protobuf-encoded. gRPC sẽ fail decode.
- Service-side cũng nhận raw bytes và phải tự parse — không có type safety.
- Không tương thích với standard gRPC tooling.

**Note:** Đây là architectural bug ảnh hưởng tất cả services, không chỉ Graphiti.

---

## BUG-F02-003: Thiếu Services `graphiti-*` Trong Codebase

**Severity:** HIGH  
**File:** `services/` directory

**Mô tả:**  
Thư mục `services/graphiti-ingestion`, `services/graphiti-search`, `services/graphiti-knowledge` tồn tại như thư mục nhưng không có implementation code. Services được liệt kê trong `allServices` và `engineServices` của gateway nhưng không có server nào thực sự running.

**Services không có main.go:**
- `services/graphiti-ingestion`
- `services/graphiti-search`  
- `services/graphiti-knowledge`
- `services/graphiti-store`
- `services/graphiti-pipeline`
- `services/graphiti-admin`

**Impact:**  
- Mọi Graphiti request từ gateway sẽ fail với `connection refused` hoặc `circuit open`.

---

## BUG-F02-004: Auth Middleware Không Được Apply (Xem BUG-F01-006)

**Severity:** HIGH  
**File:** `gateway/adapter/handler/router.go`

**Mô tả:**  
Giống F01-006: Auth middleware thiếu trong chain. Graphiti endpoints không được protect.
