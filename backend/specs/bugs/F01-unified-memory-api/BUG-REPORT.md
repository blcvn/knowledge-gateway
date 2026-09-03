# Bug Report — F01: Unified Memory API

> Feature: `POST /v1/memory/store`, `POST /v1/memory/recall`, `POST /v1/memory/forget`, `GET /v1/memory/timeline`
> Luồng: `apps/memory → gateway/adapter/handler/handler.go → usecase/route.go → services/*`

---

## BUG-F01-001: Forget Handler Chưa Được Implement (Blocking)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/handler.go:151-153`

**Mô tả:**  
`Forget` handler (`POST /v1/memory/forget`) hiện tại chỉ trả về hardcoded `202 Accepted` mà không thực sự xóa bất kỳ data nào ở bất kỳ engine nào. Đây là feature cốt lõi — xóa cascading trên 6 engines.

```go
// handler.go:150-153
func (h *MemoryHandler) Forget(w http.ResponseWriter, r *http.Request) {
    // TODO: Fan-out delete to all engines
    WriteJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
```

**Impact:**  
- GDPR forget flow KHÔNG hoạt động qua unified API endpoint này.
- Data không bao giờ bị xóa khi gọi `POST /v1/memory/forget`.
- Mặc dù `ForgetUseCase` đã được implement trong `usecase/console.go`, nó chưa bao giờ được wired vào handler này.

**Fix cần thiết:**
- Inject `ForgetUseCase` vào `MemoryHandler`.
- Parse `user_id` từ request body.
- Gọi `forgetUC.Execute(ctx, userID, requestedBy)` thay vì hardcoded response.

---

## BUG-F01-002: Routing Map Thiếu `adaptive` (sm-memory) Type (Blocking)

**Severity:** HIGH  
**File:** `gateway/usecase/route.go:30-36`

**Mô tả:**  
`serviceForType` mapping trong `RouteUseCase` thiếu `MemoryTypeAdaptive` (Supermemory). Theo feature spec, `adaptive` type → `sm-memory` service.

```go
var serviceForType = map[string]string{
    domain.MemoryTypeSemantic:       "cognee-ingestion",
    domain.MemoryTypeEpisodic:       "graphiti-ingestion",
    domain.MemoryTypeConversational: "memobase-ingestion",
    domain.MemoryTypeProfile:        "memobase-ingestion",
    domain.MemoryTypeProcedural:     "ov-resource",
    // MISSING: domain.MemoryTypeAdaptive → "sm-memory"
}
```

**Impact:**  
- Gọi `POST /v1/memory/store` với `type: "adaptive"` sẽ trả về lỗi `unknown memory type: adaptive`.
- Supermemory integration bị broken hoàn toàn qua Unified Memory API.

---

## BUG-F01-003: Classifier Luôn Trả Về `semantic` (Logic Bug)

**Severity:** MEDIUM  
**File:** `gateway/usecase/route.go:84-90`

**Mô tả:**  
`Classify()` — hàm phân loại nội dung để tự động chọn engine — hiện tại chỉ là stub, luôn trả về `MemoryTypeSemantic` bất kể nội dung gì. Không có keyword-based heuristics nào được implement.

```go
func (uc *RouteUseCase) Classify(_ context.Context, data []byte) (string, error) {
    // TODO: Replace with LLM-based classification via Bifrost
    content := string(data)
    _ = content  // content không được sử dụng
    return domain.MemoryTypeSemantic, nil  // Always semantic!
}
```

**Impact:**  
- `POST /v1/memory/store` với `type: "auto"` sẽ luôn route sang Cognee, bỏ qua tất cả engines khác.
- Auto-routing feature không hoạt động như thiết kế.

---

## BUG-F01-004: `extractPathParams` Không Extract Được Path Parameters (Logic Bug)

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/handler.go:96-101`

**Mô tả:**  
Hàm `extractPathParams()` chỉ trả về empty map, không extract path parameters từ URL. Go 1.22+ có `r.PathValue()` nhưng hàm này không sử dụng.

```go
func extractPathParams(r *http.Request) map[string]string {
    params := make(map[string]string)
    // Không có code extract nào!
    return params
}
```

**Impact:**  
- Path parameters (ví dụ `{id}` trong route) không được forward tới downstream services.
- Routes như `/v1/cognee/datasets/{id}/cognify` sẽ không chuyển `id` xuống service.

---

## BUG-F01-005: `Store` Response Trả Về 200 Thay Vì 202 (API Contract)

**Severity:** LOW  
**File:** `gateway/adapter/handler/handler.go:141`

**Mô tả:**  
Feature spec yêu cầu `Store` là non-blocking, trả về `202 Accepted`. Tuy nhiên handler trả về `200 OK`.

```go
WriteJSON(w, http.StatusOK, result)  // Phải là http.StatusAccepted (202)
```

**Impact:**  
- VI phạm API contract với AI Agent clients.
- Agent không biết request được xử lý async hay sync.

---

## BUG-F01-006: Auth Middleware Không Được Áp Dụng cho `/v1/memory/*` Routes

**Severity:** HIGH  
**File:** `gateway/adapter/handler/router.go:50-57`

**Mô tả:**  
Auth middleware (`middleware.Auth`) không được include trong `chain` function của router. Middleware chain chỉ có Logger, CORS, RequestID, Recovery — nhưng **không có Auth**. Mọi request đều có thể gọi `/v1/memory/store` mà không cần authentication.

```go
chain := func(h http.Handler) http.Handler {
    h = middleware.Logger(logger)(h)
    h = middleware.CORS("*", "true")(h)
    h = middleware.RequestID()(h)
    h = middleware.Recovery(logger)(h)
    // MISSING: h = middleware.Auth(authUC, logger)(h)
    // MISSING: h = middleware.RateLimit(rateLimitUC, logger)(h)
    return h
}
```

**Impact:**  
- Tất cả API endpoints đều publicly accessible mà không cần auth.
- `requireAdmin()` trong console handlers sẽ luôn thất bại vì `AuthFromContext` trả về `nil`.
- Multi-tenancy isolation bị phá vỡ hoàn toàn.
