# Change Request: CR-PLAT-009 — Organization Member Management

**CR ID:** CR-PLAT-009
**Component:** `backend/gateway`, `backend/services/vnp-platform`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F27](../../../features/27-organization-api-sdk-manager/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-04 | Platform Engineer | Không quản lý được team members trong org |

**Before:** Members phải được admin tạo thủ công với hardcoded roles.
**After:** Email invitation flow, role assignment, member removal.

---

## 2. Member Invitation Flow

```
1. Admin → POST /v1/console/org/members/invite
   Input: {email, role: "editor"}
   → Create invitation record (token, 48h TTL)
   → Send email: "You've been invited to {org_name}"

2. Invitee → GET /v1/auth/accept-invite?token=xxx
   → Create user if not exists
   → Add to tenant with assigned role
   → Redirect to Console

3. Admin → DELETE /v1/console/org/members/{user_id}
   → Remove from tenant
   → Revoke all their API keys for this tenant
```

---

## 3. Member Record

```go
type TenantMember struct {
    UserID     uuid.UUID
    TenantID   uuid.UUID
    Role       string    // admin|editor|viewer
    JoinedAt   time.Time
    LastActiveAt time.Time
    Status     string    // active|invited|suspended
}
```

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET`    | `/v1/console/org/members` | List members |
| `POST`   | `/v1/console/org/members/invite` | Send invitation |
| `DELETE` | `/v1/console/org/members/{id}` | Remove member |
| `PUT`    | `/v1/console/org/members/{id}/role` | Change role |
| `GET`    | `/v1/console/org/settings` | Org settings |
| `PUT`    | `/v1/console/org/settings` | Update settings |
| `GET`    | `/v1/console/org/roles` | Available roles |
| `GET`    | `/v1/auth/accept-invite` | Accept invitation |

---

## 5. Acceptance Criteria

- [ ] Invitation email sent within 30s
- [ ] Invitation expires after 48h
- [ ] Member removal revokes their API keys immediately
- [ ] Role change takes effect within 1 request (no caching)
- [ ] Only admin can invite/remove members
- [ ] Duplicate invitation → resend email (not error)
