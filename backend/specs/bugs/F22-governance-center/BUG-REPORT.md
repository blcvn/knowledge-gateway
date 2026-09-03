# Bug Report — F22: Governance Center

> Feature: Tenant management, policy CRUD (OPA), audit search, GDPR forget
> Luồng: `apps/memory → gateway/console.go (GovernanceHandler) → vnp-platform`

---

## BUG-F22-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:456-526`

---

## BUG-F22-002: `GDPRForget` Forward Tới `vnp-platform` Không Dùng `ForgetUseCase`

**Severity:** HIGH  
**File:** `gateway/adapter/handler/console.go:512-518`

**Mô tả:**  
`GDPRForget` và `GDPRForgetPreview` forward tới `vnp-platform`. Tuy nhiên `ForgetUseCase` trong `gateway/usecase/console.go` đã implement cascading delete trên tất cả engines. Chức năng này bị bypass hoàn toàn.

```go
func (h *GovernanceHandler) GDPRForget(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)  // Nên dùng ForgetUseCase
}
```

**Impact:**  
- GDPR forget chỉ xóa trên `vnp-platform`, không cascade sang các memory engines (Cognee, Graphiti, Memobase, etc.).
- `ForgetUseCase` được tạo trong `main.go` và gán vào `_`.

---

## BUG-F22-003: `PolicyUseCase` Không Được Wire Vào Governance Handler

**Severity:** HIGH  
**File:** `gateway/cmd/main.go:202`

**Mô tả:**  
```go
_ = usecase.NewPolicyUseCase(policyStore, publisher, logger)
```

`PolicyUseCase` (OPA policy CRUD) được tạo nhưng không inject vào `GovernanceHandler`. Handler forward tới `vnp-platform` thay vì dùng built-in policy store.

**Impact:**  
- OPA policy management không dùng gateway's own policy store.
- `ListPolicies`, `CreatePolicy`, `UpdatePolicy` sẽ fail nếu `vnp-platform` không implement.

---

## BUG-F22-004: `AuditUseCase` Không Được Wire Vào Governance Handler

**Severity:** HIGH  
**File:** `gateway/cmd/main.go:201`

**Mô tả:**  
`auditUC` được tạo và chỉ được dùng trong `ForgetUseCase`. Không được inject vào `GovernanceHandler` cho `SearchAudit` endpoint.

```go
auditUC := usecase.NewAuditUseCase(auditStore, publisher, logger)
// Forward tới vnp-platform thay vì dùng auditUC.Search()
```

**Impact:**  
- `GET /v1/console/governance/audit` không dùng gateway's PostgreSQL audit store.
