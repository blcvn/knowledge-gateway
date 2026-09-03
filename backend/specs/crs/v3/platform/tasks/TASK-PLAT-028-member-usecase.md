# TASK-PLAT-028 — Member Management Usecase

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-028 |
| **Wave** | 2 |
| **Solution** | [SOL-PLAT-009](../solutions/SOL-PLAT-009-Org-Member-Management.md) §3 |
| **Component** | `services/vnp-platform/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-027 |
| **Estimated** | 3h |

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** Member usecase (InviteMember/RemoveMember/AcceptInvitation) not implemented
---

## Mục tiêu

Implement `MemberUseCase` với invite/accept/remove/changeRole.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/usecase/member.go` [NEW]

```go
package usecase

var ErrAlreadyMember     = errors.New("user is already a member")
var ErrInvitationExpired = errors.New("invitation has expired")

type MemberUseCase struct {
    memberRepo port.MemberRepository
    inviteRepo port.InvitationRepository
    userRepo   port.UserRepository
    apiKeyRepo port.APIKeyRepository
    emailSvc   port.EmailService
}

func (u *MemberUseCase) InviteMember(ctx context.Context, tenantID, email, role string) (*domain.Invitation, error) {
    if !domain.ValidRole(role) { return nil, fmt.Errorf("invalid role: %s", role) }
    if existing, _ := u.memberRepo.GetByEmail(ctx, tenantID, email); existing != nil && existing.Status == "active" {
        return nil, ErrAlreadyMember
    }
    token := generateSecureToken() // crypto/rand 32 bytes, hex encoded
    inv := &domain.Invitation{
        ID: uuid.NewString(), TenantID: tenantID, Email: email,
        Role: role, Token: token, ExpiresAt: time.Now().Add(48 * time.Hour),
    }
    if err := u.inviteRepo.Create(ctx, inv); err != nil { return nil, err }
    go u.emailSvc.SendInvitation(email, token, tenantID)
    return inv, nil
}

func (u *MemberUseCase) AcceptInvite(ctx context.Context, token string) (*domain.Member, error) {
    inv, err := u.inviteRepo.GetByToken(ctx, token)
    if err != nil { return nil, fmt.Errorf("invitation not found") }
    if inv.IsExpired() { return nil, ErrInvitationExpired }

    user, err := u.userRepo.GetByEmail(ctx, inv.Email)
    if err != nil {
        user, err = u.userRepo.Create(ctx, &domain.User{
            ID: uuid.NewString(), Email: inv.Email, Role: inv.Role, TenantID: inv.TenantID,
        })
        if err != nil { return nil, err }
    }

    member := &domain.Member{
        UserID: user.ID, TenantID: inv.TenantID,
        Email: inv.Email, Role: inv.Role, Status: "active",
    }
    if err := u.memberRepo.Add(ctx, inv.TenantID, member); err != nil { return nil, err }
    u.inviteRepo.Delete(ctx, token)
    return member, nil
}

func (u *MemberUseCase) RemoveMember(ctx context.Context, tenantID, userID string) error {
    // Cascade: revoke all API keys
    if err := u.apiKeyRepo.RevokeAllForUser(ctx, tenantID, userID); err != nil {
        slog.Warn("failed revoking keys on member remove", "user_id", userID)
    }
    return u.memberRepo.Remove(ctx, tenantID, userID)
}

func (u *MemberUseCase) ChangeRole(ctx context.Context, tenantID, userID, newRole string) error {
    if !domain.ValidRole(newRole) { return fmt.Errorf("invalid role: %s", newRole) }
    return u.memberRepo.UpdateRole(ctx, tenantID, userID, newRole)
}
```

### 2. Tạo `services/vnp-platform/internal/usecase/member_test.go` [NEW]

```go
func TestInviteMember_Success(t *testing.T) { ... }
func TestInviteMember_AlreadyMember(t *testing.T) { ... }
func TestAcceptInvite_Expired(t *testing.T) { ... }
func TestRemoveMember_RevokesKeys(t *testing.T) { ... }
func TestChangeRole_InvalidRole(t *testing.T) { ... }
```

---

## Acceptance Criteria

- [ ] `InviteMember` → creates invitation + sends email async
- [ ] `AcceptInvite` → creates user if not exists, adds to tenant
- [ ] `RemoveMember` → revokes API keys before removing
- [ ] `ChangeRole` → validates role before updating
- [ ] Tests pass

## Files

```
services/vnp-platform/internal/usecase/member.go       [NEW]
services/vnp-platform/internal/usecase/member_test.go  [NEW]
```
