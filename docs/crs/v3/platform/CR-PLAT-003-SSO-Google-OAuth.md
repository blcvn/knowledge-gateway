# Change Request: CR-PLAT-003 — SSO Google OAuth Integration

**CR ID:** CR-PLAT-003
**Component:** `backend/gateway`, `backend/services/vnp-platform`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F14](../../../features/14-authentication-multi-tenancy/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-01 | Platform Engineer | No SSO support → manual user management |
| PP-P4-02 | Enterprise Architect | Enterprise requires SSO/SAML |

---

## 2. OAuth Flow

```
1. Client → GET /v1/auth/sso/google/authorize
   → Redirect to Google OAuth consent
2. Google → Callback: /v1/auth/sso/google/callback?code=xxx
   → Exchange code for Google token
   → Validate Google token (tokeninfo API)
   → Lookup or create user in DB
   → Issue VNP JWT (30min) + refresh token (7d)
3. Client → GET /v1/auth/sso/google/callback → VNP JWT
```

---

## 3. User Provisioning

```
First login:
  - Create user record (name, email, avatar from Google)
  - Assign to default tenant (from org invite OR create new)
  - Default role: viewer

Subsequent logins:
  - Update last_login_at
  - Sync avatar_url from Google (optional)
```

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/auth/sso/google/authorize` | Start OAuth flow |
| `GET` | `/v1/auth/sso/google/callback` | OAuth callback |
| `POST` | `/v1/auth/sso/google` | Exchange Google token → VNP JWT (mobile flow) |

---

## 5. Acceptance Criteria

- [ ] Full OAuth2 PKCE flow implemented
- [ ] Google token validated server-side (not just decoded)
- [ ] New users auto-provisioned on first SSO login
- [ ] Existing email → linked to existing account
- [ ] State parameter: CSRF protection
- [ ] Refresh token rotation implemented
