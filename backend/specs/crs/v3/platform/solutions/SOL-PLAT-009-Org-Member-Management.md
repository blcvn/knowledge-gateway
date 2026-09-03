# Solution: SOL-PLAT-009 — Organization Member Management

**CR:** CR-PLAT-009
**TDD refs:** `models/vnp-platform.md §admin`, `architecture/01-gateway.md §4`
**Version:** v3/platform

---

## 1. TDD Model Reference

From `models/vnp-platform.md`:
```go
type User struct {
    ID, Email, Name, PasswordHash, AuthProvider, Role, TenantID, CreatedAt, UpdatedAt
}
type TenantMember struct {
    UserID, TenantID, Role, JoinedAt, Status
}
```

---

## 2. Member Domain

```go
// services/vnp-platform/internal/domain/member.go [NEW]
type Member struct {
    UserID       string    `json:"user_id"`
    Email        string    `json:"email"`
    Name         string    `json:"name"`
    Role         string    `json:"role"`        // admin|editor|viewer
    Status       string    `json:"status"`      // active|invited|suspended
    JoinedAt     time.Time `json:"joined_at"`
    LastActiveAt time.Time `json:"last_active_at,omitempty"`
}

type Invitation struct {
    ID        string    `json:"id"`
    TenantID  string
    Email     string
    Role      string
    Token     string    // secure random, not exposed in response
    ExpiresAt time.Time // 48h from creation
    CreatedAt time.Time
}
```

---

## 3. Member Management Usecase

```go
// services/vnp-platform/internal/usecase/member.go [NEW]
type MemberUseCase struct {
    memberRepo port.MemberRepository
    inviteRepo port.InvitationRepository
    userRepo   port.UserRepository
    apiKeyRepo port.APIKeyRepository
    emailSvc   port.EmailService
}

// InviteMember — send email, create invitation record
func (u *MemberUseCase) InviteMember(ctx context.Context, tenantID, inviterID, email, role string) (*Invitation, error) {
    // Check if already member
    if existing, _ := u.memberRepo.GetByEmail(ctx, tenantID, email); existing != nil {
        // Resend invitation if pending
        if existing.Status == "invited" {
            inv, _ := u.inviteRepo.GetLatestForEmail(ctx, tenantID, email)
            go u.emailSvc.SendInvitation(email, inv.Token, tenantID)
            return inv, nil
        }
        return nil, ErrAlreadyMember
    }

    token := generateSecureToken() // 32-byte random, hex encoded
    inv := &Invitation{
        TenantID: tenantID, Email: email, Role: role,
        Token: token, ExpiresAt: time.Now().Add(48 * time.Hour),
    }
    if err := u.inviteRepo.Create(ctx, inv); err != nil { return nil, err }
    go u.emailSvc.SendInvitation(email, token, tenantID)
    return inv, nil
}

// AcceptInvite — called when invitee clicks link
func (u *MemberUseCase) AcceptInvite(ctx context.Context, token string) (*Member, error) {
    inv, err := u.inviteRepo.GetByToken(ctx, token)
    if err != nil || time.Now().After(inv.ExpiresAt) {
        return nil, ErrInvitationExpired
    }

    // Create user if not exists
    user, err := u.userRepo.GetByEmail(ctx, inv.Email)
    if err != nil {
        user, err = u.userRepo.Create(ctx, &User{Email: inv.Email, Role: inv.Role})
        if err != nil { return nil, err }
    }

    // Add to tenant
    member := &Member{UserID: user.ID, Email: user.Email, Role: inv.Role, Status: "active"}
    if err := u.memberRepo.Add(ctx, inv.TenantID, member); err != nil { return nil, err }
    u.inviteRepo.Delete(ctx, token) // consume token
    return member, nil
}

// RemoveMember — revoke API keys, remove from tenant
func (u *MemberUseCase) RemoveMember(ctx context.Context, tenantID, userID string) error {
    // Revoke all API keys for this user in this tenant
    if err := u.apiKeyRepo.RevokeAllForUser(ctx, tenantID, userID); err != nil {
        slog.Warn("failed to revoke API keys", "user_id", userID, "err", err)
    }
    return u.memberRepo.Remove(ctx, tenantID, userID)
}

// ChangeRole — immediate effect (no caching)
func (u *MemberUseCase) ChangeRole(ctx context.Context, tenantID, userID, newRole string) error {
    if newRole != "admin" && newRole != "editor" && newRole != "viewer" {
        return fmt.Errorf("invalid role: %s", newRole)
    }
    return u.memberRepo.UpdateRole(ctx, tenantID, userID, newRole)
}
```

---

## 4. HTTP Handlers

```go
// gateway/adapter/handler/org_handler.go [NEW]
type OrgHandler struct {
    memberUC MemberUseCase
    orgRepo  port.OrgRepository
}

// GET /v1/console/org/members
func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    members, err := h.memberUC.ListMembers(r.Context(), tenantID)
    if err != nil { writeError(w, 500, "list_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"members": members, "total": len(members)})
}

// POST /v1/console/org/members/invite
func (h *OrgHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
    var req struct{ Email string `json:"email"`; Role string `json:"role"` }
    json.NewDecoder(r.Body).Decode(&req)
    tenantID := tenant.FromContext(r.Context())
    inviterID := auth.UserIDFromContext(r.Context())
    inv, err := h.memberUC.InviteMember(r.Context(), tenantID, inviterID, req.Email, req.Role)
    if err == ErrAlreadyMember { writeError(w, 409, "already_member", ""); return }
    if err != nil { writeError(w, 500, "invite_failed", err.Error()); return }
    writeJSON(w, 202, map[string]string{"invitation_id": inv.ID, "status": "email_sent"})
}

// DELETE /v1/console/org/members/{id}
func (h *OrgHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    userID   := chi.URLParam(r, "id")
    if err := h.memberUC.RemoveMember(r.Context(), tenantID, userID); err != nil {
        writeError(w, 500, "remove_failed", err.Error())
        return
    }
    writeJSON(w, 200, map[string]bool{"removed": true})
}

// PUT /v1/console/org/members/{id}/role
func (h *OrgHandler) ChangeRole(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    userID   := chi.URLParam(r, "id")
    var req struct{ Role string `json:"role"` }
    json.NewDecoder(r.Body).Decode(&req)
    if err := h.memberUC.ChangeRole(r.Context(), tenantID, userID, req.Role); err != nil {
        writeError(w, 400, "role_change_failed", err.Error())
        return
    }
    writeJSON(w, 200, map[string]bool{"updated": true})
}

// GET /v1/auth/accept-invite?token=xxx
func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    member, err := h.memberUC.AcceptInvite(r.Context(), token)
    if err == ErrInvitationExpired { writeError(w, 400, "invite_expired", ""); return }
    if err != nil { writeError(w, 500, "accept_failed", err.Error()); return }
    // Issue JWT for the newly joined member
    jwt := h.authUC.IssueJWT(r.Context(), member.UserID)
    writeJSON(w, 200, map[string]string{"access_token": jwt, "role": member.Role})
}
```

---

## 5. DB Migration

```sql
-- deployment/dev/migrations/0046_invitations.sql
CREATE TABLE IF NOT EXISTS tenant_members (
    user_id    UUID NOT NULL,
    tenant_id  UUID NOT NULL,
    role       TEXT NOT NULL DEFAULT 'viewer',
    status     TEXT NOT NULL DEFAULT 'active',
    joined_at  TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, tenant_id)
);

CREATE TABLE IF NOT EXISTS member_invitations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'viewer',
    token      TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 6. File Changes

| File | Action |
|---|---|
| `services/vnp-platform/internal/domain/member.go` | **[NEW]** |
| `services/vnp-platform/internal/usecase/member.go` | **[NEW]** |
| `services/vnp-platform/internal/usecase/member_test.go` | **[NEW]** |
| `gateway/adapter/handler/org_handler.go` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** org routes |
| `deployment/dev/migrations/0046_invitations.sql` | **[NEW]** |
