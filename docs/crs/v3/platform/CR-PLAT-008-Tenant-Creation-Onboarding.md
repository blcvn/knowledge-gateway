# Change Request: CR-PLAT-008 — Tenant Creation & Onboarding Flow

**CR ID:** CR-PLAT-008
**Component:** `backend/gateway`, `backend/services/vnp-platform`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F14](../../../features/14-authentication-multi-tenancy/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-01 | Platform Engineer | No self-service tenant creation |

**Before:** Tenant phải được tạo thủ công bởi admin.
**After:** Self-service signup → auto-provision tenant → onboarding checklist.

---

## 2. Onboarding Flow

```
1. POST /v1/auth/signup
   Input: {email, password, org_name, tier: "free"}
   → Create user record
   → Create tenant record (slug = slugify(org_name))
   → Assign user as admin of tenant
   → Send welcome email with verification link

2. GET /v1/auth/verify-email?token=xxx
   → Mark email as verified
   → Provision default resources (rate limit counters)

3. POST /v1/auth/login
   → JWT issued with tenant_id + roles

4. GET /v1/console/onboarding
   → Checklist: {
       email_verified, api_key_created,
       first_memory_stored, mcp_connected
     }
```

---

## 3. Tenant Model

```go
type Tenant struct {
    ID           uuid.UUID
    Name         string
    Slug         string    // unique, URL-safe
    Tier         string    // free|pro|enterprise
    CreatedAt    time.Time
    OwnerID      uuid.UUID
    IsActive     bool
    // Engine aliases
    EngineAliases map[string]string // {"my-graph" → "graphiti"}
}
```

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/auth/signup` | Create account + tenant |
| `GET` | `/v1/auth/verify-email` | Email verification |
| `POST` | `/v1/admin/tenants` | Admin: create tenant |
| `GET` | `/v1/admin/tenants` | Admin: list tenants |
| `PUT` | `/v1/admin/tenants/{id}/tier` | Admin: change tier |
| `GET` | `/v1/console/onboarding` | Onboarding checklist |

---

## 5. Acceptance Criteria

- [ ] Signup creates user + tenant atomically (transaction)
- [ ] Email verification required before full access
- [ ] Slug: unique, alphanumeric + hyphens
- [ ] Admin can upgrade tier
- [ ] Onboarding checklist tracks completion
- [ ] Duplicate email → 409 Conflict
