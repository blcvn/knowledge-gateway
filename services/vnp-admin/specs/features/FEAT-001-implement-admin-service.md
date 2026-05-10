---
id: FEAT-001
title: Implement vnp-admin — Tenant/APIKey/User Management Service
service: vnp-admin
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-002
linked_tdd: TDD-vnp-admin
---

## Mục Tiêu

Implement vnp-admin as a production-grade Go microservice providing tenant lifecycle, API key provisioning (SHA-256 hashed), user CRUD, health aggregation, and billing/quota management for the entire VNP Memory platform.

## Bối Cảnh Nghiệp Vụ

vnp-admin is the central administration service — every other service depends on it for tenant context and authentication validation. Gateway resolves API keys via vnp-admin. All downstream engines check tenant existence through vnp-admin.

## Scope

### In Scope (AI phải implement)
- Domain entities: Tenant, APIKey, User, BillingEntry (from tdd.md §2.1)
- Repository interfaces: TenantRepository, APIKeyRepository, UserRepository, BillingRepository
- Usecase services: TenantService, APIKeyService, UserService, HealthService
- gRPC handlers: AdminService (from tdd.md §3)
- PostgreSQL repository adapters (pgx/v5)
- NATS event publisher: TenantCreated, TenantDeleted, KeyRevoked
- Config loader (env vars)
- Health check server
- go.mod, cmd/server/main.go

### Out of Scope
- OPA policy integration (Phase 2)
- Billing payment processing (external)

## Thiết Kế Kỹ Thuật

### API Contract (from tdd.md §3)
```protobuf
service AdminService {
  rpc CreateTenant(CreateTenantRequest) returns (Tenant);
  rpc GetTenant(GetTenantRequest) returns (Tenant);
  rpc UpdateTenant(UpdateTenantRequest) returns (Tenant);
  rpc DeleteTenant(DeleteTenantRequest) returns (Empty);
  rpc ListTenants(ListTenantsRequest) returns (ListTenantsResponse);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (Empty);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc AggregateHealth(HealthRequest) returns (AggregatedHealthResponse);
}
```

### Data Model (from data-model.md)
- `tenants`: id, name, plan, config (JSONB), created_at, updated_at
- `api_keys`: id, tenant_id, key_hash (SHA-256), scope, rate_limit, active, created_at, revoked_at
- `users`: id, tenant_id, email, role, metadata (JSONB), created_at

### Internal Architecture
```
services/vnp-admin/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   ├── tenant.go
│   │   │   ├── api_key.go
│   │   │   ├── user.go
│   │   │   └── billing.go
│   │   ├── repository/
│   │   │   ├── tenant_repo.go
│   │   │   ├── api_key_repo.go
│   │   │   └── user_repo.go
│   │   ├── event.go
│   │   └── errors.go
│   ├── usecase/
│   │   ├── tenant_service.go
│   │   ├── apikey_service.go
│   │   ├── user_service.go
│   │   └── health_service.go
│   ├── adapter/
│   │   └── grpc/
│   │       └── admin_handler.go
│   └── infra/
│       ├── config/config.go
│       ├── persistence/
│       │   ├── pg_tenant.go
│       │   ├── pg_apikey.go
│       │   └── pg_user.go
│       └── nats/publisher.go
└── go.mod
```

## Acceptance Criteria

- [ ] AC-1: `go build ./cmd/server/` compiles without errors
- [ ] AC-2: CreateTenant → stores in PostgreSQL → publishes TenantCreated to NATS
- [ ] AC-3: CreateAPIKey → generates key → hashes with SHA-256 → stores hash only → returns plaintext once
- [ ] AC-4: ValidateAPIKey → hashes input → matches against stored hash → returns tenant context
- [ ] AC-5: AggregateHealth → fan-out health checks to configured services → returns aggregated status
- [ ] AC-6: All domain entities match tdd.md §2.1 field definitions
- [ ] AC-7: gRPC reflection enabled for service discovery

## Test Requirements
- **Unit tests:** TenantService.Create, APIKeyService.Validate (SHA-256 round-trip), HealthService.Aggregate
- **Integration tests:** Full gRPC round-trip with test PostgreSQL
- **Minimum coverage:** 80%

## Definition of Done
- [ ] Code implements all Acceptance Criteria
- [ ] Unit tests pass, coverage ≥ 80%
- [ ] `docs/changelog.md` updated with implementation entry
- [ ] No lint errors (`golangci-lint run`)
