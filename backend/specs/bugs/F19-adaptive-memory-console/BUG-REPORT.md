# Bug Report — F19: Adaptive Memory Console

> Feature: Memories list, versions, connectors management, analytics, forget rules
> Luồng: `apps/memory → gateway/console.go (AdaptiveHandler) → search-service / obs-service`

---

## BUG-F19-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:276-338`

---

## BUG-F19-002: Adaptive Memories và Connectors Forward Tới `search-service` Thay Vì `sm-*` Services

**Severity:** HIGH  
**File:** `gateway/adapter/handler/console.go:281-314`

**Mô tả:**  
`ListMemories`, `GetVersions`, `ListConnectors`, `CreateConnector`, `SyncConnector` tất cả forward tới `search-service`. Theo feature spec và F07 analysis, các operations này nên đến `sm-memory`, `sm-connector` services.

**Impact:**  
- `search-service` không implement Supermemory protocol → wrong data hoặc errors.

---

## BUG-F19-003: `obs-service` Không Có Implementation

**Severity:** HIGH  
**File:** `services/obs-service/`

**Mô tả:**  
`GetAnalytics`, `GetForgetRules`, `UpdateForgetRules` forward tới `obs-service`. Service này tồn tại trong `services/` nhưng cần verify có implementation code không.

**Impact:**  
- Adaptive analytics và forget rules management không hoạt động nếu `obs-service` không implement.
