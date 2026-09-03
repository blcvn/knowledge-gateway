# Feature 27 — Organization & API SDK Manager

> **Loại:** Platform | **Priority:** P1 | **Status:** Implemented (UI)

## Mô tả

Organization & API SDK Manager cung cấp giao diện quản lý tổ chức và SDK integration cho Tenants — bao gồm: org settings, member management, role assignments, API key management, rate limit visualization, và webhook configuration.

---

## Business Logic

### Organization Settings

Tenant admin quản lý thông tin tổ chức:
- Organization name, slug, branding
- Subscription tier (free/pro/enterprise) — read-only, contact sales để upgrade
- Billing information (link to billing portal)
- SSO configuration (Google OAuth)
- Engine aliases: map custom names tới engines (e.g., `my-memory → sm-memory`)

### Member Management

- List members trong organization
- Member info: name, email, role, joined date, last active
- Invite new members (email invitation flow)
- Remove members
- Role assignment: admin/editor/viewer

### API Key Management (SDK)

Quản lý API keys dùng để integrate với SDK:
- List keys: prefix, created, last used, status, rate tier
- Create key: chọn rate tier (free/pro/enterprise)
- Revoke key (immediate effect)
- Key secret chỉ visible một lần khi tạo

### Rate Limits

Visualize rate limit status:
- Current usage vs limit per tier
- Time window: per-minute, per-hour, per-day
- Historical rate limit events (hit count)

### Webhooks

Configure webhooks cho events quan trọng:
- Events: memory.stored, memory.forgotten, session.completed, rate_limit.exceeded, health.degraded
- Delivery: HTTP POST với JSON payload
- Retry: Exponential backoff, max 3 retries
- Delivery history: Last N deliveries với status code

---

## Dataflow

### Organization Management

```
Console UI (Organization Settings)
        │
        ├── GET /v1/console/org/settings     → Org info, engine aliases, SSO config
        ├── PUT /v1/console/org/settings     → Update settings
        ├── GET /v1/console/org/members      → Member list
        └── GET /v1/console/org/roles        → Available roles
```

### API Key Lifecycle

```
Console UI (SDK Manager)
        │
        ├── GET /v1/console/sdk/keys
        │         └── List: [{prefix, created, last_used, status, tier}]
        │
        ├── POST /v1/console/sdk/keys
        │         ├── Input: {name, tier: "free|pro|enterprise"}
        │         ├── Generate: prefix (8 chars) + secret (32 chars)
        │         ├── SHA-256 hash secret → store
        │         └── Return: {prefix, secret} — last time visible
        │
        ├── DELETE /v1/console/sdk/keys/{id}
        │         └── Set status = revoked (immediate)
        │
        ├── GET /v1/console/sdk/rate-limits
        │         └── Current usage vs limits per tier
        │
        ├── GET /v1/console/sdk/webhooks
        │         └── List configured webhooks
        │
        ├── POST /v1/console/sdk/webhooks
        │         ├── Input: {url, events: [...], secret}
        │         └── Verify URL reachability
        │
        └── DELETE /v1/console/sdk/webhooks/{id}
                  └── Remove webhook


Webhook delivery:
        NATS event → gateway → HTTP POST to webhook URL
                └── Retry on failure (exponential backoff, max 3)
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/org/settings` | Org settings |
| `PUT` | `/v1/console/org/settings` | Update settings |
| `GET` | `/v1/console/org/members` | Member list |
| `GET` | `/v1/console/org/roles` | Available roles |
| `GET` | `/v1/console/sdk/keys` | List API keys |
| `POST` | `/v1/console/sdk/keys` | Create API key |
| `DELETE` | `/v1/console/sdk/keys/{id}` | Revoke key |
| `GET` | `/v1/console/sdk/rate-limits` | Rate limit status |
| `GET` | `/v1/console/sdk/webhooks` | List webhooks |
| `POST` | `/v1/console/sdk/webhooks` | Create webhook |
| `DELETE` | `/v1/console/sdk/webhooks/{id}` | Delete webhook |

---

## Business Value

### Pain Points được giải quyết

- **PP-P2-04 (No single API management)**
- **PP-P6-02 (No SDK)**

### Actors hưởng lợi

P2 Platform Engineer, P6 Framework Integrator

### Giải pháp tham chiếu

- [S9 — Enterprise Governance](../../bussiness/solutions/S9-governance-compliance.md)
- [S10 — Zero-config Infrastructure](../../bussiness/solutions/S10-infrastructure-simplicity.md)

### ROI / Kết quả đo được

> API key lifecycle central management | Engine aliases | SSO | SDK foundation

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
