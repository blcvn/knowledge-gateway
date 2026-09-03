---
id: MERGE-P1-T2
title: "vnp-platform: Absorb vnp-admin (Tenant + API Key Management)"
phase: P1
service: vnp-platform
priority: P0
status: Done
estimated: 6h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P1-T1]
---

## Mục Tiêu

Merge toàn bộ domain + usecase của `vnp-admin` (Admin CRUD, Tenant management, API Key issuance) vào `vnp-platform`. vnp-admin hiện là stub (forward.NewRouter, no routes) nhưng có **full domain/usecase/adapter structure** từ SOL-002.

## Context

`vnp-admin` theo SOL-002 đã được implement với 14 Go files (domain + usecase + adapter). Cần move các layer này vào vnp-platform và wire vào single gRPC server.

**Services bị absorb:** `vnp-admin` (1,925 lines)

## Scope

### Nguồn — `services/vnp-admin/`

```
services/vnp-admin/
├── domain/
│   ├── entity.go      → Tenant, APIKey, TenantPlan
│   └── errors.go      → ErrTenantNotFound, ErrDuplicateKey
├── usecase/
│   ├── service.go     → AdminService: CreateTenant, IssueAPIKey, RevokeTenant
│   └── port/          → TenantRepository, EventPublisher interfaces
├── adapter/
│   └── grpc/          → AdminHandler (gRPC server)
└── infra/
    ├── persistence/   → PGTenantRepository
    └── nats/          → EventPublisher impl
```

### Đích — `services/vnp-platform/internal/domain/admin/`

```
services/vnp-platform/internal/
├── domain/admin/
│   ├── entity.go          # Tenant, APIKey, TenantPlan, TenantStatus
│   └── errors.go          # domain errors
├── usecase/admin/
│   └── service.go         # CreateTenant, IssueAPIKey, GetTenant, ListTenants, RevokeTenant
├── adapter/grpc/
│   └── admin_handler.go   # ForwardService routes + optional gRPC service
└── infra/persistence/
    └── pg_tenant_repo.go  # PostgreSQL TenantRepository (likely already exists from SOL-002)
```

## Thay Đổi Cần Thực Hiện

### 1. Domain Entities

```go
// domain/admin/entity.go
type Tenant struct {
    ID        string
    Name      string
    Plan      TenantPlan   // "free" | "pro" | "enterprise"
    Status    TenantStatus // "active" | "suspended"
    CreatedAt time.Time
}

type APIKey struct {
    Key       string
    TenantID  string
    Scopes    []string
    ExpiresAt *time.Time
    CreatedAt time.Time
}

type TenantPlan string
const (
    PlanFree       TenantPlan = "free"
    PlanPro        TenantPlan = "pro"
    PlanEnterprise TenantPlan = "enterprise"
)
```

### 2. Usecase Interface

```go
type AdminUseCase interface {
    CreateTenant(ctx context.Context, name string, plan TenantPlan) (*Tenant, error)
    GetTenant(ctx context.Context, id string) (*Tenant, error)
    ListTenants(ctx context.Context) ([]*Tenant, error)
    UpdateTenant(ctx context.Context, id string, updates TenantUpdate) (*Tenant, error)
    IssueAPIKey(ctx context.Context, tenantID string, scopes []string) (*APIKey, error)
    RevokeAPIKey(ctx context.Context, key string) error
    Health(ctx context.Context) (*HealthStatus, error)
}
```

### 3. ForwardService Routes

```go
router.Handle("POST", "/v1/admin/tenants",          adminForward.CreateTenant)
router.Handle("GET",  "/v1/admin/tenants",          adminForward.ListTenants)
router.Handle("GET",  "/v1/admin/tenants/*",        adminForward.GetTenant)
router.Handle("PUT",  "/v1/admin/tenants/*",        adminForward.UpdateTenant)
router.Handle("POST", "/v1/admin/tenants/*/keys",   adminForward.IssueAPIKey)
router.Handle("GET",  "/v1/admin/health",           adminForward.Health)
router.Handle("GET",  "/v1/admin/metrics",          adminForward.Metrics)

// Governance console routes
router.Handle("GET",  "/v1/console/governance/tenants",        adminForward.ListTenants)
router.Handle("POST", "/v1/console/governance/tenants",        adminForward.CreateTenant)
router.Handle("PUT",  "/v1/console/governance/tenants/*",      adminForward.UpdateTenant)
router.Handle("GET",  "/v1/console/governance/policies",       adminForward.ListPolicies)
router.Handle("POST", "/v1/console/governance/policies",       adminForward.CreatePolicy)
router.Handle("PUT",  "/v1/console/governance/policies/*",     adminForward.UpdatePolicy)
router.Handle("GET",  "/v1/console/governance/audit",          adminForward.SearchAudit)
router.Handle("POST", "/v1/console/governance/gdpr/forget",    adminForward.GDPRForget)
router.Handle("POST", "/v1/console/governance/gdpr/forget/preview", adminForward.GDPRForgetPreview)
```

### 4. Database Migrations

```sql
-- migrations/003_admin_tenants.sql
CREATE TABLE IF NOT EXISTS vnp_tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    plan       TEXT NOT NULL DEFAULT 'free',
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vnp_api_keys (
    key        TEXT PRIMARY KEY,
    tenant_id  UUID NOT NULL REFERENCES vnp_tenants(id),
    scopes     TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
CREATE INDEX idx_api_keys_tenant ON vnp_api_keys(tenant_id);

CREATE TABLE IF NOT EXISTS vnp_policies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES vnp_tenants(id),
    name        TEXT NOT NULL,
    rules       JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vnp_audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID,
    actor       TEXT NOT NULL,
    action      TEXT NOT NULL,
    resource    TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_log_tenant ON vnp_audit_log(tenant_id);
CREATE INDEX idx_audit_log_created ON vnp_audit_log(created_at DESC);
```

### 5. NATS Events

```go
// Publish events sau mỗi tenant operation
publisher.Publish(ctx, "vnp.admin.tenant.created", TenantCreatedEvent{...})
publisher.Publish(ctx, "vnp.admin.apikey.issued", APIKeyIssuedEvent{...})
```

## Acceptance Criteria

- [ ] `POST /v1/admin/tenants` tạo tenant và persist vào PostgreSQL
- [ ] `POST /v1/admin/tenants/{id}/keys` issue API key với scopes
- [ ] `GET /v1/admin/health` returns `{"status":"ok"}`
- [ ] `GET /v1/console/governance/tenants` returns tenant list
- [ ] NATS event `vnp.admin.tenant.created` published sau khi tạo tenant
- [ ] `go build ./services/vnp-platform/...` passes
- [ ] Unit tests cho AdminUseCase pass

## Ghi Chú

- Tham chiếu `gateway/infra/persistence/pg_repos.go` — gateway đã có PGTenantStore, cần align schema
- Governance routes (policy, audit, GDPR) cần stub handlers trước, implement logic sau
- `vnp-admin` service giữ nguyên cho đến P4 cleanup
