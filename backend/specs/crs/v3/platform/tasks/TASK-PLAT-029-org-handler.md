# TASK-PLAT-029 — Org Member Management HTTP Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-029 |
| **Wave** | 3 |
| **Solution** | [SOL-PLAT-009](../solutions/SOL-PLAT-009-Org-Member-Management.md) §4 |
| **Component** | `gateway/adapter/handler/org_handler.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-028 |
| **Estimated** | 2h |

---

## Mục tiêu

Tạo `OrgHandler` với đầy đủ member management endpoints.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/org_handler.go` [NEW]

```go
package handler

type OrgHandler struct {
    memberUC *usecase.MemberUseCase
}

func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    members, err := h.memberUC.ListMembers(r.Context(), tenantID)
    if err != nil { writeError(w, 500, "list_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"members": members, "total": len(members)})
}

func (h *OrgHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
    var req struct{ Email string `json:"email"`; Role string `json:"role"` }
    json.NewDecoder(r.Body).Decode(&req)
    if req.Role == "" { req.Role = "viewer" }
    tenantID := tenant.FromContext(r.Context())
    inv, err := h.memberUC.InviteMember(r.Context(), tenantID, req.Email, req.Role)
    if err == usecase.ErrAlreadyMember { writeError(w, 409, "already_member", ""); return }
    if err != nil { writeError(w, 500, "invite_failed", err.Error()); return }
    writeJSON(w, 202, map[string]string{"invitation_id": inv.ID, "status": "email_sent"})
}

func (h *OrgHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
    userID   := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())
    if err := h.memberUC.RemoveMember(r.Context(), tenantID, userID); err != nil {
        writeError(w, 500, "remove_failed", err.Error()); return
    }
    writeJSON(w, 200, map[string]bool{"removed": true})
}

func (h *OrgHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
    userID   := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())
    var req struct{ Role string `json:"role"` }
    json.NewDecoder(r.Body).Decode(&req)
    if err := h.memberUC.ChangeRole(r.Context(), tenantID, userID, req.Role); err != nil {
        writeError(w, 400, "role_change_failed", err.Error()); return
    }
    writeJSON(w, 200, map[string]bool{"updated": true})
}

func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    member, err := h.memberUC.AcceptInvite(r.Context(), token)
    if err == usecase.ErrInvitationExpired { writeError(w, 400, "invite_expired", ""); return }
    if err != nil { writeError(w, 500, "accept_failed", err.Error()); return }
    jwt := h.authUC.IssueJWT(r.Context(), member.UserID)
    writeJSON(w, 200, map[string]string{"access_token": jwt, "role": member.Role})
}
```

### 2. Routes trong `gateway/adapter/handler/router.go` [MODIFY]

```go
r.Get("/v1/console/org/members", orgHandler.ListMembers)
r.Post("/v1/console/org/members/invite", orgHandler.InviteMember)
r.Delete("/v1/console/org/members/{id}", orgHandler.RemoveMember)
r.Put("/v1/console/org/members/{id}/role", orgHandler.ChangeRole)
r.Get("/v1/auth/accept-invite", authHandler.AcceptInvite)
```

---

## Acceptance Criteria

- [ ] `GET /v1/console/org/members` → list with total
- [ ] `POST /v1/console/org/members/invite` → 202 + invitation_id
- [ ] Duplicate invite → 409
- [ ] `DELETE` → removes + revokes keys
- [ ] `GET /v1/auth/accept-invite?token=xxx` → issues JWT

## Files

```
gateway/adapter/handler/org_handler.go  [NEW]
gateway/adapter/handler/router.go       [MODIFY]
```
