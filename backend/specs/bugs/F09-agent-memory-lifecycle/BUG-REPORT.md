# Bug Report — F09: Agent Memory Lifecycle

> Feature: `POST /v1/memory/agent/remember`, list, get, delete, retention score, evict, auto-forget
> Luồng: `apps/memory → gateway/adapter/handler/agentmemory.go → memory-service`

---

## BUG-F09-001: AgentMemory Handler Route Tất Cả Tới `memory-service` Không Có Router Registration Tương Ứng

**Severity:** HIGH  
**File:** `gateway/adapter/handler/agentmemory.go:67-99`, `services/memory-service/cmd/server/main.go`

**Mô tả:**  
AgentMemory handler (`RememberAgent`, `ListAgentMemories`, etc.) forward tới `memory-service`. Nhưng trong `memory-service/main.go`, handler registration chỉ bao gồm:
- Memobase operations
- Zep operations
- SM operations

Không có registration cho AgentMemory paths (`/v1/memory/agent/*`, `/v1/memory/slots/*`).

```go
// memory-service/main.go
handler := memgrpc.NewMemoryHandler(mbIngest, mbContext, zUser, zMem, zGraph, smMem, smDoc)
memgrpc.RegisterRoutes(router, handler)
// Không có AgentMemory handler registration!
```

**Impact:**  
- Tất cả `/v1/memory/agent/*` requests sẽ trả về gRPC NOT_FOUND.
- Agent Memory Lifecycle feature hoàn toàn không hoạt động end-to-end.

---

## BUG-F09-002: `GetRetentionScore` Không Có Service Implementation

**Severity:** HIGH  
**File:** `gateway/adapter/handler/agentmemory.go:86-89`

**Mô tả:**  
`GET /v1/memory/agent/{id}/retention` được route tới `memory-service`. Retention score calculation (ebbinghaus forgetting curve) là complex business logic không được implement trong `memory-service`. Không có usecase file nào cho retention score calculation.

**Impact:**  
- Retention score endpoint trả về gRPC NOT_FOUND hoặc empty data.

---

## BUG-F09-003: `AutoForgetSweep` Không Có Scheduler/Trigger

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/agentmemory.go:96-99`

**Mô tả:**  
`POST /v1/memory/agent/auto-forget` là manual trigger cho auto-forget sweep. Theo feature spec, auto-forget cũng nên chạy theo schedule. Không có background scheduler nào được setup trong `memory-service/main.go`.

**Impact:**  
- Auto-forget phụ thuộc hoàn toàn vào manual API calls — không có scheduled cleanup.

---

## BUG-F09-004: Memory Slots Route Conflict — `/v1/memory/agent/list` vs `/v1/memory/agent/{id}`

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/router.go:154-155`

**Mô tả:**  
```go
mux.HandleFunc("GET /v1/memory/agent/list", agentmemH.ListAgentMemories)
mux.HandleFunc("GET /v1/memory/agent/{id}", agentmemH.GetAgentMemory)
```

Khi request `GET /v1/memory/agent/list` được gửi, Go 1.22+ router có thể ambiguous match với `{id} = "list"`. Cần kiểm tra thứ tự đăng ký và specificity rules.

**Impact:**  
- `GET /v1/memory/agent/list` có thể được xử lý bởi `GetAgentMemory` với `id="list"` thay vì `ListAgentMemories`.
