# Bug Report — F07: Adaptive Memory (Supermemory)

> Feature: Document CRUD, memory creation, search, RAG, profiles, connections, spaces
> Luồng: `apps/memory → gateway/adapter/handler/services.go (SMHandler) → memory-service / search-service / vnp-platform`

---

## BUG-F07-001: Inconsistent Service Routing Trong SMHandler

**Severity:** HIGH  
**File:** `gateway/adapter/handler/services.go:217-251`

**Mô tả:**  
`SMHandler` routes tới 3 services khác nhau một cách không nhất quán và không đúng với feature spec:

| Handler | Service Hiện Tại | Service Đúng |
|---------|-----------------|--------------|
| `CreateDocument` | `memory-service` | `sm-document` |
| `GetDocument` | `memory-service` | `sm-document` |
| `CreateMemory` | `memory-service` | `sm-memory` |
| `Search` | `search-service` | `sm-search` |
| `RAG` | `search-service` | `sm-engine` |
| `GetProfile` | `memory-service` | `sm-profile` |
| `CreateConnection` | `search-service` | `sm-connector` |
| `SyncConnection` | `search-service` | `sm-connector` |
| `CreateSpace` | `vnp-platform` | `sm-project` |

**Impact:**  
- Supermemory adaptive KG feature broken do routing sai.
- `memory-service` không implement Supermemory protocol.

---

## BUG-F07-002: Thiếu Tất Cả `sm-*` Service Implementations

**Severity:** CRITICAL  
**File:** `services/sm-*` directories

**Mô tả:**  
Tất cả 10 Supermemory microservices (`sm-analytics`, `sm-auth`, `sm-connector`, `sm-document`, `sm-engine`, `sm-mcp`, `sm-memory`, `sm-profile`, `sm-project`, `sm-search`) đều không có implementation code (chỉ có thư mục trống hoặc không có `main.go`).

**Impact:**  
- Adaptive Memory (Feature 07) hoàn toàn không hoạt động end-to-end.
- Mọi request tới `sm-*` endpoints sẽ fail với connection refused.

---

## BUG-F07-003: `CreateSpace` Route Tới `vnp-platform` Thay Vì `sm-project`

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/services.go:249-251`

**Mô tả:**  
```go
func (h *SMHandler) CreateSpace(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)  // Nên là "sm-project"
}
```

**Impact:**  
- Space/project creation trong Supermemory sẽ hit platform service thay vì Supermemory project service.
