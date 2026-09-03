# Bug Report — F21: Sessions Explorer

> Feature: List sessions, live sessions, session detail, timeline, diff, working memory, user summary
> Luồng: `apps/memory → gateway/console.go (SessionHandler) → memory-service / storage-service`

---

## BUG-F21-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:388-442`

---

## BUG-F21-002: `GetWorkingMemory` Forward Tới `storage-service` Thay Vì `memory-service`

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/console.go:420-426`

**Mô tả:**  
`GetWorkingMemory` forward tới `storage-service` trong khi tất cả session operations khác đều dùng `memory-service`.

```go
func (h *SessionHandler) GetWorkingMemory(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "storage-service", h.logger)(w, r)  // Inconsistent
}
```

**Impact:**  
- Working memory data có thể không có trong `storage-service` — wrong service.

---

## BUG-F21-003: `storage-service` Không Có Implementation

**Severity:** HIGH  
**File:** `services/storage-service/`

**Mô tả:**  
`storage-service` directory tồn tại nhưng cần verify có implementation code không.

---

## BUG-F21-004: Route Conflict — `GET /v1/console/sessions/live` vs `GET /v1/console/sessions/{id}`

**Severity:** HIGH  
**File:** `gateway/adapter/handler/router.go:213-214`

**Mô tả:**  
```go
mux.HandleFunc("GET /v1/console/sessions/live", session.ListLiveSessions)  // Line 213
mux.HandleFunc("GET /v1/console/sessions/{id}", session.GetSession)          // Line 214
```

Request `GET /v1/console/sessions/live` có thể match với `{id} = "live"`. Cần verify Go 1.22+ router ưu tiên literal path pattern over wildcard.

**Impact:**  
- `ListLiveSessions` có thể bị route tới `GetSession` với `id="live"`.
