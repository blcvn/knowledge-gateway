# Bug Report — F16: Memory Explorer

> Feature: Search, get, neighbors, versions trong memory explorer console
> Luồng: `apps/memory → gateway/console.go (ExplorerHandler) → search-service`

---

## BUG-F16-001: `search-service` Không Có Implementation

**Severity:** CRITICAL  
**File:** `services/search-service/`

**Mô tả:**  
`ExplorerHandler` forward tới `search-service` cho tất cả operations (Search, GetMemory, GetNeighbors, GetVersions). `search-service` directory tồn tại nhưng không có implementation code.

**Impact:**  
- Memory Explorer console không hoạt động.
- Tất cả search requests fail với connection refused.

---

## BUG-F16-002: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:105-134`

**Mô tả:**  
Explorer handlers đều dùng `requireAdmin()`, luôn fail vì auth context nil.

---

## BUG-F16-003: `GetMemory` và `GetNeighbors` Không Extract Path Parameter `{id}`

**Severity:** HIGH  
**File:** `gateway/adapter/handler/handler.go:96-101`

**Mô tả:**  
`extractPathParams()` trả về empty map (BUG-F01-004). `GET /v1/console/memory/{id}` và `GET /v1/console/memory/{id}/neighbors` sẽ forward request tới search-service mà không có `id` trong params.

**Impact:**  
- search-service không biết memory nào cần query.
