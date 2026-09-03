# Bug Report — F26: Session Replay

> Feature: Session replay — re-play agent session observations step by step
> Luồng: Observe-service replay module

---

## BUG-F26-001: Session Replay Module Tồn Tại Nhưng Không Có Gateway Routes

**Severity:** CRITICAL  
**File:**
- `services/observe-service/internal/replay/` — có code  
- `gateway/adapter/handler/router.go` — không có replay routes

**Mô tả:**  
`observe-service` có `internal/replay/` directory với implementation, nhưng không có gateway routes cho session replay. Không có `ReplayHandler` trong gateway và không có endpoints được register.

**Impact:**  
- Session Replay feature không accessible từ apps/memory.
- Replay code trong observe-service chưa được exposed qua API.

---

## BUG-F26-002: Replay API Endpoint Không Được Định Nghĩa

**Severity:** HIGH  
**File:** `gateway/adapter/handler/router.go`

**Mô tả:**  
Không có replay-related endpoint trong gateway router. Cần thêm:
- `GET /v1/observe/sessions/{id}/replay` — start replay
- `GET /v1/observe/sessions/{id}/replay/step` — step through
- `POST /v1/observe/sessions/{id}/replay/seek` — seek to position
