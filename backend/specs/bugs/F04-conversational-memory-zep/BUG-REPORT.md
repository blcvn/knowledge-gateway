# Bug Report — F04: Conversational Memory (Zep)

> Feature: Zep user, session, memory, graph, search management
> Luồng: `apps/memory → gateway/adapter/handler/services.go (ZepHandler) → memory-service`

---

## BUG-F04-001: Zep Client Là NoopClient (Real Zep Integration Chưa Implement)

**Severity:** CRITICAL  
**File:** `services/memory-service/cmd/server/main.go:78-79`

**Mô tả:**  
Zep client được khởi tạo với `NoopClient{}` — một implementation rỗng trả về empty/nil cho mọi operation. Comment rõ ràng xác nhận đây là placeholder.

```go
zepClient := &zepnoop.NoopClient{} // Default: no-op
// TODO: wire real Zep SDK client when ZEP_API_KEY is set
```

**Impact:**  
- Tất cả Zep operations: CreateUser, GetUser, PutMemory, GetMemory, GraphSearch, SessionSearch, AddFact, SetOntology đều không thực sự tương tác với Zep backend.
- Conversational memory feature hoàn toàn không hoạt động.

---

## BUG-F04-002: ZepHandler Forward Tới `memory-service` Nhưng Service Này Không Có gRPC Forward Interface

**Severity:** HIGH  
**File:** `gateway/adapter/handler/services.go:171-205`

**Mô tả:**  
`ZepHandler` routes tới `memory-service`, nhưng `memory-service` sử dụng `forward.RegisterForwardService(grpcServer, router)` — cần kiểm tra router có đăng ký các paths `/v1/zep/*` hay không. Nếu không đăng ký, mọi request sẽ fail với gRPC status NOT_FOUND.

```go
func (h *ZepHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
}
```

**Impact:**  
- Nếu `memory-service` router chưa register Zep paths, tất cả Zep API calls fail.
- Cần verify `memgrpc.RegisterRoutes(router, handler)` có bao gồm Zep paths không.

---

## BUG-F04-003: `zep-go` Symlink Bị Broken

**Severity:** MEDIUM  
**File:** `services/zep-go`

**Mô tả:**  
`services/zep-go` là một symlink (31 bytes) không phải thư mục thực. Không rõ symlink trỏ tới đâu và có valid không.

**Impact:**  
- Build sẽ fail nếu có code import từ `zep-go`.
