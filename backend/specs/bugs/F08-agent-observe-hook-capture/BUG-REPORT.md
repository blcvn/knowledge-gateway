# Bug Report — F08: Agent Observe & Hook Capture

> Feature: 14-step observe pipeline, session management, SSE stream
> Luồng: `apps/memory → gateway/adapter/handler/agentmemory.go (AgentMemoryHandler) → observe-service`

---

## BUG-F08-001: `import "github.com/vnp-memory/pkg/privacy"` — Sai Module Path

**Severity:** CRITICAL  
**File:** `services/observe-service/internal/observe/pipeline.go:12`

**Mô tả:**  
Import path `github.com/vnp-memory/pkg/privacy` không tồn tại. Theo folder architecture rules, shared packages phải nằm ở `shared/pkg/privacy` với module name `vnp-memory/shared/pkg/privacy`. Trong `go.mod` của observe-service, replace directive trỏ đúng tới `../../shared/pkg/privacy` nhưng module name trong go.mod là `github.com/vnp-memory/pkg/privacy` — sai với canonical naming.

```go
// pipeline.go
import "github.com/vnp-memory/pkg/privacy"  // Wrong path!

// go.mod
replace github.com/vnp-memory/pkg/privacy => ../../shared/pkg/privacy  // Replace đúng dir nhưng module name sai convention
```

**Impact:**  
- Build của `observe-service` sẽ fail với package not found error (nếu go.mod replace không hoạt động).
- Inconsistency với module naming convention.

---

## BUG-F08-002: Observe Pipeline Thiếu Step 3 (Authentication Session/Agent)

**Severity:** HIGH  
**File:** `services/observe-service/internal/observe/pipeline.go:66-110`

**Mô tả:**  
Feature spec mô tả 14-step pipeline với Step 3 là "Authenticate session/agent identity". Code hiện tại bỏ qua hoàn toàn authentication step — bất kỳ ai có session_id đều có thể observe.

**Impact:**  
- Security risk: Unauthorized observations có thể inject vào bất kỳ session nào.
- Cross-tenant observation injection là có thể.

---

## BUG-F08-003: Thiếu Step 10 (BM25 Index) và Step 11 (Embedding)

**Severity:** HIGH  
**File:** `services/observe-service/internal/observe/pipeline.go`

**Mô tả:**  
Feature spec mô tả 14 steps. Code chỉ implement 14 steps nhưng thiếu:
- **Step 10: BM25 index update** — `search.IndexObservation` được gọi async nhưng chỉ index compressed observation, không phải raw.
- **Step 11: Vector embedding** — Không có embedding generation code.

**Impact:**  
- Observations không searchable qua BM25 full-text search.
- Observations không searchable qua vector similarity search.
- `POST /v1/observe/sessions/{id}/observations` search sẽ trả về empty results.

---

## BUG-F08-004: SSE `StreamEvents` Trong Gateway Forward Tới gRPC Nhưng observe-service Có HTTP SSE Endpoint Riêng

**Severity:** MEDIUM  
**File:**
- `gateway/adapter/handler/agentmemory.go:60-62`
- `services/observe-service/internal/adapter/http/sse_handler.go`

**Mô tả:**  
Gateway `StreamEvents` forward sang gRPC `observe-service`. Tuy nhiên `observe-service` có một HTTP SSE handler (`sse_handler.go`) riêng biệt. Có thể tạo ra conflict hoặc duplicate streaming paths.

```go
// gateway/agentmemory.go
func (h *AgentMemoryHandler) StreamEvents(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "observe-service", h.logger)(w, r)  // gRPC forward
}

// observe-service có HTTP SSE handler riêng
// ServeSSE handles GET /v1/stream
```

**Impact:**  
- Gateway attempt gRPC forward tới observe-service SSE endpoint → incompatible protocol.
- Real-time streaming sẽ không hoạt động.

---

## BUG-F08-005: Dedup Hash Có Thể Collision Khi `ToolInput` Là Empty

**Severity:** LOW  
**File:** `services/observe-service/internal/observe/pipeline.go:73`

**Mô tả:**  
Dedup hash dùng `fmt.Sprint(req.ToolInput)` trên byte slice, không phải hex encoding. Nếu `ToolInput` là nil, hash sẽ là `SHA256(sessionID + toolName + "[]")` — có thể tạo false duplicates.

```go
hash := sha256.Sum256([]byte(req.SessionID + req.ToolName + fmt.Sprint(req.ToolInput)))
```
