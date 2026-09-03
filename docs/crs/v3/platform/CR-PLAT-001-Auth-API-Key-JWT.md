# Change Request: CR-PLAT-001 — Auth Backend: API Key Lifecycle & JWT RS256

**CR ID:** CR-PLAT-001
**Component:** `backend/gateway`, `backend/services/vnp-platform`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F14](../../../features/14-authentication-multi-tenancy/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P2-04 | Platform Engineer | Không có central API key management |
| PP-P4-03 | Enterprise Architect | No audit trail cho key operations |

**Before:** API key auth đã implement nhưng thiếu full lifecycle (rotation, expiry, audit).
**After:** Complete API key lifecycle: create → activate → rotate → revoke → expire + immutable audit.

---

## 2. API Key Specification

```
Format: vnp_{prefix}.{secret}
  prefix:  8-char alphanumeric (public, safe to log)
  secret:  32-char random (SHA-256 hashed before storage)

Header: Authorization: Bearer vnp_{key}

Lifecycle:
  Active → Revoked (manual)
  Active → Expired (TTL-based)
  Active → Rotated (create new + invalidate old)
```

---

## 3. JWT RS256 Specification

```json
{
  "sub":       "user_id",
  "tenant_id": "tenant_abc",
  "roles":     ["admin"],
  "exp":       1234567890,
  "iat":       1234567800
}
```

- RSA-2048 keypair: private key signs, public key verifies
- Key rotation: support multiple active public keys via JWK endpoint
- `GET /.well-known/jwks.json` — JWK Set for public key discovery

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/auth/login` | Email/password → JWT |
| `POST` | `/v1/auth/refresh` | Refresh token → new JWT |
| `POST` | `/v1/auth/logout` | Revoke refresh token |
| `GET`  | `/.well-known/jwks.json` | JWK Set (public key) |
| `GET`  | `/v1/console/sdk/keys` | List API keys |
| `POST` | `/v1/console/sdk/keys` | Create API key |
| `DELETE` | `/v1/console/sdk/keys/{id}` | Revoke key |
| `POST` | `/v1/console/sdk/keys/{id}/rotate` | Rotate key |

---

## 5. Dev Mode

- `AUTH_DEV_MODE=true`: Skip all auth checks
- Only accept localhost traffic (production guard)
- Auto-injects mock tenant_id: `dev-tenant`

---

## 6. Acceptance Criteria

- [ ] API key: prefix visible, secret hashed, never re-exposed
- [ ] JWT RS256: signed, verified at gateway middleware
- [ ] JWK endpoint serves current public keys
- [ ] Key rotation: creates new key, invalidates old
- [ ] Audit log: all key create/revoke/rotate operations
- [ ] Dev mode bypasses auth only for localhost
