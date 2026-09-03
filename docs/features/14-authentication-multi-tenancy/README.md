# Feature 14 — Authentication & Multi-tenancy

> **Loại:** Platform | **Priority:** P0 | **Status:** Implemented

## Mô tả

Authentication & Multi-tenancy là lớp bảo mật và isolation cấp enterprise của VNP Memory. Hỗ trợ hai phương thức auth (API Key và JWT RS256), ba subscription tiers, và đảm bảo **zero cross-tenant data leaks** thông qua TenantID injection trên mọi query.

---

## Business Logic

### Authentication Methods

**1. API Key**
- Format: `{prefix}.{secret}` — prefix là 8 chars visible, secret được hash SHA-256 trước khi lưu.
- Trong request header: `Authorization: Bearer vnp_{key}`
- Vòng đời: `Active → Revoked | Expired`
- `ExpiresAt` và `RevokedAt` fields để lifecycle management.
- Chỉ prefix được lưu trong logs/responses — secret không bao giờ exposed sau creation.

**2. JWT RS256**
- Standard Bearer token.
- RSA-256 signed — private key ở server, public key có thể distribute.
- Claims: `tenant_id`, `user_id`, `roles`, `exp`.

**3. Dev Mode**
- `AUTH_DEV_MODE=true`: Skip tất cả authentication checks.
- **Only accept localhost traffic** — production guard.
- Phù hợp cho local development.

### Multi-tenant Isolation

TenantID là "magic ingredient" cho tenant isolation:
- Mọi domain entity có `TenantID` field.
- Mọi query được inject `WHERE tenant_id = :current_tenant`.
- Không có global queries — không thể "accidentally" query cross-tenant.
- Integration tests verify: cross-tenant queries return 0 results.

### Subscription Tiers

| Tier | Rate Limit | Features |
|------|-----------|---------|
| `free` | Basic | Limited engines, shared resources |
| `pro` | Higher | All engines, dedicated resources |
| `enterprise` | Unlimited | All + custom ontology, SLA, dedicated infra |

Rate limits được enforce tại gateway bằng Redis (sliding window counter).

### User Roles per Tenant

Mỗi user có role trong context của từng tenant:
- `admin`: Full access, manage other users
- `editor`: Read + write memory operations
- `viewer`: Read-only access

### SSO (Google)

Gateway hỗ trợ SSO với Google OAuth2:
- `POST /v1/auth/sso/google` — exchange Google token → VNP JWT.
- Tạo user account tự động nếu chưa có.

---

## Dataflow

### API Key Authentication

```
Request: POST /v1/memory/store
         Authorization: Bearer vnp_abc12345.secret_part_here
        │
        ▼
Auth Middleware (gateway)
        │
        ├── 1. Extract token from header
        ├── 2. Parse prefix (first 8 chars after "vnp_")
        ├── 3. Lookup by prefix in APIKey table
        │         ├── Not found → 401 Unauthorized
        │         └── Found → get hashed_secret from DB
        │
        ├── 4. SHA-256 hash incoming secret
        ├── 5. Compare with stored hash
        │         ├── Mismatch → 401 Unauthorized
        │         └── Match → proceed
        │
        ├── 6. Check key status
        │         ├── Revoked → 401 Unauthorized
        │         └── Expired (ExpiresAt < now) → 401 Unauthorized
        │
        ├── 7. Load tenant from key.TenantID
        ├── 8. Set AuthContext {TenantID, UserID, RateTier} in request context
        │
        └── Proceed to handler
```

### Rate Limiting

```
Every request → Rate Limit Middleware
        │
        ├── Key: "ratelimit:{tenant_id}:{endpoint}"
        ├── Redis: INCR + EXPIRE (sliding window)
        │
        ├── Limit not exceeded → Allow
        └── Limit exceeded:
                  ├── Return 429 Too Many Requests
                  └── NATS publish: gateway.ratelimit.exceeded {tenant_id}
```

### Tenant Creation Flow

```
POST /v1/admin/tenants
        │
        ├── Input: {name, slug, tier: "free|pro|enterprise"}
        │
        ▼
vnp-admin service
        │
        ├── Create Tenant record
        │         {id, name, slug, tier, status: "active", created_at}
        │
        └── Return tenant ID

POST /v1/admin/tenants/{id}/keys
        │
        ├── Generate: prefix (8 chars) + secret (32 chars random)
        ├── SHA-256 hash secret
        ├── Store: {tenant_id, prefix, hashed_secret, created_at}
        └── Return: {prefix, secret} — last time secret is visible
```

### JWT Authentication Flow

```
Request: GET /v1/auth/me
         Authorization: Bearer eyJhbGciOiJSUzI1NiJ9...
        │
        ▼
Auth Middleware
        │
        ├── 1. Detect JWT (starts with eyJ)
        ├── 2. Verify RSA-256 signature with public key
        │         ├── Invalid signature → 401
        │         └── Valid → decode claims
        │
        ├── 3. Check expiry (exp claim)
        ├── 4. Extract {tenant_id, user_id, roles}
        └── Set AuthContext → proceed
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/auth/register` | User registration |
| `POST` | `/v1/auth/login` | Login → JWT |
| `POST` | `/v1/auth/logout` | Logout (invalidate token) |
| `POST` | `/v1/auth/refresh` | Refresh JWT |
| `POST` | `/v1/auth/sso/google` | Google SSO |
| `GET` | `/v1/auth/me` | Current user info |
| `POST` | `/v1/admin/tenants` | Create tenant |
| `POST` | `/v1/admin/tenants/{id}/keys` | Issue API key |
| `GET` | `/v1/console/governance/tenants` | List tenants |
| `POST` | `/v1/console/governance/tenants` | Create tenant (console) |
| `PUT` | `/v1/console/governance/tenants/{id}` | Update tenant |

---

## Services

| Service | Vai trò |
|---------|---------|
| `vnp-platform` / `vnp-admin` | Tenant lifecycle, user management, API key CRUD |
| `gateway` middleware | Auth validation, rate limiting per request |

---

## Business Value

### Pain Points được giải quyết

- **PP-P2-03 (Tenant isolation gap)**
- **PP-P4-03 (No policy enforcement)**

### Actors hưởng lợi

P2 Platform Engineer, P4 Enterprise Architect

### Giải pháp tham chiếu

- [S9 — Enterprise Governance](../../bussiness/solutions/S9-governance-compliance.md)

### ROI / Kết quả đo được

> Zero cross-tenant leaks (integration-tested) | API key lifecycle management

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
