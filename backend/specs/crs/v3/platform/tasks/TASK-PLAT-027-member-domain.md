# TASK-PLAT-027 — Member Domain & Repository Port

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-027 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-PLAT-009](../solutions/SOL-PLAT-009-Org-Member-Management.md) §2 |
| **Component** | `services/vnp-platform/internal/domain/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Mục tiêu

Tạo domain models `Member`, `Invitation` và DB migration cho member management.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/domain/member.go` [NEW]

```go
package domain

import "time"

type Member struct {
    UserID       string    `json:"user_id"`
    TenantID     string    `json:"tenant_id"`
    Email        string    `json:"email"`
    Name         string    `json:"name"`
    Role         string    `json:"role"`   // admin|editor|viewer
    Status       string    `json:"status"` // active|invited|suspended
    JoinedAt     time.Time `json:"joined_at"`
    LastActiveAt time.Time `json:"last_active_at,omitempty"`
}

type Invitation struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    Email     string    `json:"email"`
    Role      string    `json:"role"`
    Token     string    `json:"-"`        // never exposed
    ExpiresAt time.Time `json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
}

func (i *Invitation) IsExpired() bool {
    return time.Now().After(i.ExpiresAt)
}

func ValidRole(role string) bool {
    return role == "admin" || role == "editor" || role == "viewer"
}
```

### 2. Tạo `services/vnp-platform/internal/port/member_repository.go` [NEW]

```go
package port

type MemberRepository interface {
    Add(ctx context.Context, tenantID string, m *domain.Member) error
    Remove(ctx context.Context, tenantID, userID string) error
    UpdateRole(ctx context.Context, tenantID, userID, role string) error
    ListByTenant(ctx context.Context, tenantID string) ([]*domain.Member, error)
    GetByEmail(ctx context.Context, tenantID, email string) (*domain.Member, error)
}

type InvitationRepository interface {
    Create(ctx context.Context, inv *domain.Invitation) error
    GetByToken(ctx context.Context, token string) (*domain.Invitation, error)
    GetLatestForEmail(ctx context.Context, tenantID, email string) (*domain.Invitation, error)
    Delete(ctx context.Context, token string) error
}
```

### 3. DB Migration `deployment/dev/migrations/0046_invitations.sql` [NEW]

```sql
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
CREATE INDEX idx_invitations_token ON member_invitations(token);
```

---

## Acceptance Criteria

- [ ] `domain.Member` struct có đủ fields
- [ ] `Invitation.IsExpired()` returns true khi past ExpiresAt
- [ ] `ValidRole()` chỉ cho phép admin|editor|viewer
- [ ] Migration chạy không lỗi

## Files

```
services/vnp-platform/internal/domain/member.go             [NEW]
services/vnp-platform/internal/port/member_repository.go    [NEW]
deployment/dev/migrations/0046_invitations.sql               [NEW]
```
