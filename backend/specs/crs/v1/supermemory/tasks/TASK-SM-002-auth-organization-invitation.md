# TASK-SM-002 — Auth: Organization Model & Invitation System

**Task ID:** TASK-SM-002  
**Wave:** 1 (Foundation)  
**Solution:** [SOL-SM-007](../solutions/SOL-SM-007-Auth-Organization-RBAC.md)  
**Depends on:** TASK-SM-001 (RBAC roles)  
**Ước tính:** 3h  
**Priority:** Critical — multi-user org management

**Trạng thái:** 🔄 Partial  
**Ghi chú:** sm-auth: 8 .go - organization scaffold; invitation flow incomplete  
---

## Mục tiêu

Tạo Organization model trong `services/vnp-platform/`:
1. `Organization` entity (id, name, slug, owner, plan, settings)
2. `OrgMember` entity với 4 roles
3. `OrgInvitation` entity (token, expires 7 days)
4. PostgreSQL schema migration
5. REST API: create org, add/remove member, invite, accept invitation

---

## Công việc cụ thể

### 1. Tạo Domain Models

**`services/vnp-platform/internal/domain/admin/organization.go`**

```go
type Organization struct {
    ID          string
    Name        string
    Slug        string       // URL-friendly, unique
    OwnerUserID string
    Plan        Plan         // "free" | "pro" | "enterprise"
    Settings    OrgSettings
    CreatedAt   time.Time
}

type OrgSettings struct {
    MaxMembers     int
    MaxAPIKeys     int
    MaxConnections int
    CustomOAuth    bool  // Enterprise only
}

type OrgMember struct {
    OrgID     string
    UserID    string
    Role      auth.Role  // owner | admin | editor | viewer
    InvitedBy string
    JoinedAt  time.Time
}

type OrgInvitation struct {
    ID         string
    OrgID      string
    Email      string
    Role       auth.Role
    Token      string     // 16 random bytes hex
    ExpiresAt  time.Time  // CreatedAt + 7 days
    CreatedBy  string
    AcceptedAt *time.Time
}
```

### 2. Tạo SQL Migration

**`services/vnp-platform/migrations/003_create_organizations.sql`**

```sql
CREATE TABLE organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    owner_id   UUID NOT NULL REFERENCES users(id),
    plan       TEXT NOT NULL DEFAULT 'free',
    settings   JSONB DEFAULT '{"max_members":5,"max_api_keys":10,"max_connections":3,"custom_oauth":false}',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE org_members (
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'viewer',
    invited_by UUID,
    joined_at  TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (org_id, user_id),
    CONSTRAINT org_members_role_check CHECK (role IN ('owner','admin','editor','viewer'))
);

CREATE TABLE org_invitations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'viewer',
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_by  UUID NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE oauth2_clients (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    secret_hash   TEXT,
    redirect_uris TEXT[] NOT NULL,
    scopes        TEXT[] DEFAULT '{}',
    is_public     BOOLEAN DEFAULT true,
    org_id        UUID REFERENCES organizations(id),
    created_at    TIMESTAMPTZ DEFAULT now()
);

-- Nâng cấp api_keys table
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organizations(id),
    ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'editor';
```

### 3. Implement Use Cases

**`services/vnp-platform/internal/usecase/admin/create_org.go`**:
- Validate slug uniqueness
- Create org + add owner as member với role "owner"

**`services/vnp-platform/internal/usecase/admin/invite_member.go`**:
- Require `member:manage` permission
- Generate secure token (16 random bytes hex)
- ExpiresAt = now + 7 days
- Email notification (fire-and-forget goroutine)

**`services/vnp-platform/internal/usecase/admin/accept_invitation.go`**:
- Validate token + expiry
- Add user as org member với invitation.Role
- Set AcceptedAt = now

**`services/vnp-platform/internal/usecase/admin/remove_member.go`**:
- Require `member:manage` permission
- Cannot remove org owner (role=owner)

### 4. Tạo REST Handlers

**`gateway/adapter/handler/org_handler.go`**

```
POST   /api/v1/auth/organizations               → CreateOrg
GET    /api/v1/auth/organizations/{id}          → GetOrg
POST   /api/v1/auth/organizations/{id}/members  → AddMember (requires member:manage)
DELETE /api/v1/auth/organizations/{id}/members/{userId} → RemoveMember
POST   /api/v1/auth/organizations/{id}/invitations → InviteMember
POST   /api/v1/auth/invitations/{token}/accept  → AcceptInvitation
```

### 5. Tests

- `TestCreateOrg_SlugUnique`: duplicate slug → 409 Conflict
- `TestInviteMember_TokenExpiry`: expired token → 410 Gone
- `TestAcceptInvitation_AddsAsMember`: accept → org_members row created
- `TestRemoveMember_CannotRemoveOwner`: owner → 403
- `TestAddMember_RequiresPermission`: viewer adds member → 403
- `TestOrgSlug_URLFriendly`: "My Org!" → slug = "my-org"

---

## Acceptance Criteria

- [ ] `go build ./services/vnp-platform/... ./gateway/...` không lỗi
- [ ] Migration SQL chạy không lỗi
- [ ] CreateOrg + add owner as "owner" member (single tx)
- [ ] InviteMember → token valid for 7 days, expired after 7d+1s
- [ ] AcceptInvitation with expired token → 410 Gone
- [ ] RemoveMember(owner) → 403 Forbidden
- [ ] Viewer POST /members → 403 (RequirePermission middleware)
- [ ] `go test ./services/vnp-platform/...` pass

---

## Files tạo/sửa

```
services/vnp-platform/internal/domain/admin/
└── organization.go         (NEW)

services/vnp-platform/internal/usecase/admin/
├── create_org.go           (NEW)
├── invite_member.go        (NEW)
├── accept_invitation.go    (NEW)
├── remove_member.go        (NEW)
└── org_test.go             (NEW)

services/vnp-platform/internal/infra/postgres/
└── org_repo.go             (NEW)

services/vnp-platform/migrations/
└── 003_create_organizations.sql (NEW)

gateway/adapter/handler/
└── org_handler.go          (NEW)
```

## Sau khi hoàn thành

Chạy: `go build ./... && go test ./services/vnp-platform/...`
