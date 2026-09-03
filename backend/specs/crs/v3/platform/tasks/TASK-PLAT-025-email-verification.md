# TASK-PLAT-025 — Email Verification Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-025 |
| **Wave** | 2 |
| **Solution** | [SOL-PLAT-008](../solutions/SOL-PLAT-008-Tenant-Creation-Onboarding.md) §3 |
| **Component** | `gateway/adapter/handler/auth_handler.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-024 |
| **Estimated** | 2h |

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** Email verification flow not implemented in sm-auth
---

## Mục tiêu

Thêm `POST /v1/auth/signup` và `GET /v1/auth/verify-email` vào gateway auth handler.

---

## Công việc cụ thể

### 1. Sửa `gateway/adapter/handler/auth_handler.go` [MODIFY]

```go
// POST /v1/auth/signup
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
    var req SignupRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, 400, "invalid_request", err.Error()); return
    }
    resp, err := h.signupUC.Signup(r.Context(), &req)
    if err == ErrEmailAlreadyExists {
        writeError(w, 409, "email_exists", "email already registered"); return
    }
    if err != nil { writeError(w, 500, "signup_failed", err.Error()); return }
    writeJSON(w, 201, resp)
}

// GET /v1/auth/verify-email?token=xxx
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    if token == "" {
        writeError(w, 400, "missing_token", "token query param required"); return
    }
    userID, err := h.tokenRepo.Get(r.Context(), token)
    if err != nil || userID == "" {
        writeError(w, 400, "invalid_token", "verification link invalid or expired"); return
    }
    if err := h.userRepo.MarkEmailVerified(r.Context(), userID); err != nil {
        writeError(w, 500, "verify_failed", err.Error()); return
    }
    h.tokenRepo.Delete(r.Context(), token)
    writeJSON(w, 200, map[string]bool{"email_verified": true})
}
```

### 2. Thêm routes vào `gateway/adapter/handler/router.go` [MODIFY]

```go
r.Post("/v1/auth/signup", authHandler.Signup)
r.Get("/v1/auth/verify-email", authHandler.VerifyEmail)
```

---

## Acceptance Criteria

- [ ] `POST /v1/auth/signup` với valid body → 201 + user_id, tenant_id
- [ ] Duplicate email → 409
- [ ] `GET /v1/auth/verify-email?token=xxx` → marks email_verified=true
- [ ] Invalid/expired token → 400
- [ ] Token consumed after verification (single-use)

## Files

```
gateway/adapter/handler/auth_handler.go  [MODIFY]
gateway/adapter/handler/router.go        [MODIFY]
```
