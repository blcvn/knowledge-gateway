# Bug Report — F18: User Profiles Console

> Feature: List profiles, get profile, events, context, buffers, config management
> Luồng: `apps/memory → gateway/console.go (ProfileHandler) → memory-service / vnp-platform`

---

## BUG-F18-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:209-262`

---

## BUG-F18-002: Route Conflict — `GET /v1/console/profiles/config` vs `GET /v1/console/profiles/{user_id}`

**Severity:** HIGH  
**File:** `gateway/adapter/handler/router.go:188-191`

**Mô tả:**  
```go
mux.HandleFunc("GET /v1/console/profiles", profile.ListProfiles)
mux.HandleFunc("GET /v1/console/profiles/config", profile.GetConfig)       // Line 189
mux.HandleFunc("PUT /v1/console/profiles/config", profile.UpdateConfig)
mux.HandleFunc("GET /v1/console/profiles/{user_id}", profile.GetProfile)   // Line 191
```

Request `GET /v1/console/profiles/config` có thể match với `{user_id} = "config"` pattern thay vì literal `/config` path — cần kiểm tra Go 1.22 routing specificity.

**Impact:**  
- `GET /v1/console/profiles/config` có thể route sai handler.

---

## BUG-F18-003: `GetEvents` Forward Tới `vnp-platform` (Inconsistency)

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/console.go:224-230`

**Mô tả:**  
`ProfileHandler.GetEvents` forward tới `vnp-platform` trong khi profile data (GetProfile, GetContext, GetBuffers) forward tới `memory-service`. Events nên đến từ `vnp-event` service theo feature spec.

```go
func (h *ProfileHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)  // Nên là "vnp-event"
}
```
