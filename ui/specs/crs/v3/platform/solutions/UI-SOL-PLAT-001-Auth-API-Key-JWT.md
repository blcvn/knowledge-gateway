# UI Solution: UI-SOL-PLAT-001 — Auth & API Key Lifecycle UI

**Solution ID:** UI-SOL-PLAT-001  
**CR References:** [CR-PLAT-001](../../../../docs/crs/v3/platform/CR-PLAT-001-Auth-API-Key-JWT.md)  
**Feature:** Auth Backend — API Key Lifecycle + JWT RS256  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/pages/login/`, `ui/src/pages/api-sdk/`

---

## 1. Mục Đích

Align frontend Auth & API Key UI với backend PLAT-001:
- Login form với JWT RS256 validation
- API Key lifecycle: create → rotate → revoke
- Key rotation flow (create new + invalidate old in one action)
- JWK discovery UI (dev info panel)

---

## 2. Backend API Contract

```http
POST /v1/auth/login → LoginResponse (JWT RS256)
POST /v1/auth/refresh → { access_token, expires_in }
POST /v1/auth/logout → void
GET  /.well-known/jwks.json → JWK Set

GET    /v1/console/sdk/keys           → APIKey[]
POST   /v1/console/sdk/keys           → CreateKeyResponse (raw_key shown ONCE)
DELETE /v1/console/sdk/keys/{id}      → void (revoke)
POST   /v1/console/sdk/keys/{id}/rotate → CreateKeyResponse (new key)
```

---

## 3. Components

### 3.1 API Key Rotation Flow

```
APIKeyRotateModal
├── WarningBanner   ← "Old key will be immediately invalidated"
├── ExpirySelector  ← 7d | 30d | 90d | 1 year | no expiry
└── RotateButton    → POST /sdk/keys/{id}/rotate
    ↓ on success:
RawKeyDisplay       ← show new raw_key ONCE + "Old key is now revoked"
```

### 3.2 Key Status States

```typescript
// Key lifecycle visual states
const KEY_STATUS_STYLES = {
  active:  'border-green-200 bg-green-50',
  revoked: 'border-gray-200 bg-gray-50 opacity-60',
  expired: 'border-red-200 bg-red-50 opacity-60',
};

// Expiry warning: < 7 days → amber badge "Expires in 5 days"
function ExpiryBadge({ expiresAt }: { expiresAt: string | null }) {
  if (!expiresAt) return <span>No expiry</span>;
  const daysLeft = differenceInDays(new Date(expiresAt), new Date());
  if (daysLeft < 0)  return <span className="text-red-600">Expired</span>;
  if (daysLeft < 7)  return <span className="text-amber-600">⚠️ {daysLeft}d left</span>;
  return <span>{formatDate(expiresAt)}</span>;
}
```

### 3.3 JWT Debug Panel (Dev Mode)

```
JWTDebugPanel (collapsible, only in dev/staging)
├── CurrentTokenDisplay     ← encoded JWT (masked after 20 chars)
├── DecodedPayload          ← JSON: sub, tenant_id, roles, exp, iat
├── JWKSEndpoint            ← "GET /.well-known/jwks.json"
└── TokenValidUntil         ← countdown timer to expiry
```

---

## 4. Acceptance Criteria (Frontend)

- [ ] Login returns JWT → decoded to extract `tenant_id` + `role`
- [ ] Key rotation: new `raw_key` shown immediately after rotate
- [ ] Revoke: requires typed confirmation ("revoke" + key name)
- [ ] Expiry warning for keys < 7 days from expiry
- [ ] JWT debug panel in dev mode only (`VITE_ENV !== 'production'`)
- [ ] `/.well-known/jwks.json` link in SDK settings (dev info)
