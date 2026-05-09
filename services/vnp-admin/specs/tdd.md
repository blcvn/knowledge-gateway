---
id: TDD-vnp-admin
title: Technical Design — vnp-admin
service: vnp-admin
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: Platform
---

# Technical Design — vnp-admin

> **Group**: Platform | **gRPC Port**: 9050 | **Origin**: Shared

## 1. Service Overview

Central administration for the entire VNP Memory platform. Tenant lifecycle, API key provisioning (SHA-256 hashed), user CRUD, health aggregation across all 35 domain services, billing/quota management, and configuration overrides.

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── tenant.go                # Tenant, TenantConfig, Plan enum
│   ├── api_key.go              # APIKey, KeyScope, RateLimit
│   ├── user.go                 # User, UserRole
│   └── billing.go              # BillingEntry, UsageMetrics
├── repository/
│   ├── tenant_repo.go           # TenantRepository
│   ├── api_key_repo.go          # APIKeyRepository
│   ├── user_repo.go             # UserRepository
│   └── billing_repo.go          # BillingRepository
├── event.go                     # TenantCreated, TenantDeleted, UserDeleted
└── errors.go                    # TenantNotFound, DuplicateTenant, KeyRevoked
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── tenant_ops.go               # Tenant CRUD with config profiles
├── api_key_ops.go             # Key create (SHA-256 hash) → validate → revoke
├── user_ops.go                # User CRUD within tenants
├── health_ops.go              # Fan-out gRPC Health to all 35 services
├── billing_ops.go             # Usage tracking, quota enforcement
├── config_ops.go              # Per-tenant config overrides
├── port/
│   ├── input.go               # TenantUseCase, APIKeyUseCase, HealthUseCase
│   └── output.go             # EventPublisherPort, HealthCheckerPort
└── dto/
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/handler.go             # VNPAdminService gRPC
├── event/publisher.go          # NATS: admin.tenant.created/deleted, admin.user.deleted
└── client/
    └── health_client.go        # Fan-out gRPC health to 35 services (errgroup)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/                # PostgreSQL repositories
├── config/config.go
└── wire/wire.go
```

## 3. gRPC API

```protobuf
service VNPAdminService {
  rpc CreateTenant(CreateTenantRequest) returns (Tenant);
  rpc GetTenant(GetTenantRequest) returns (Tenant);
  rpc UpdateTenant(UpdateTenantRequest) returns (Tenant);
  rpc ListTenants(ListTenantsRequest) returns (ListTenantsResponse);
  rpc DeleteTenant(DeleteTenantRequest) returns (google.protobuf.Empty);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc GetAggregatedHealth(google.protobuf.Empty) returns (AggregatedHealthResponse);
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
}
```

## 4. NATS Events

| Subject | Subscriber |
|---------|------------|
| `admin.tenant.created` | All services (init schema/cache) |
| `admin.tenant.deleted` | All services (cascade delete) |
| `admin.user.deleted` | memobase-*, vnp-event (cascade) |

## 5. Multi-Tenancy

- vnp-admin IS the tenant authority — all other services derive tenant_id from it
- Tenant config stored in JSONB for flexible feature flags

---

> **Next Steps**: FEAT-001 (Tenant CRUD), FEAT-002 (API Key Lifecycle), FEAT-003 (Health Aggregation), FEAT-004 (Billing/Quota).
