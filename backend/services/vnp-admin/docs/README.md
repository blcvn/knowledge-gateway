---
id: DOC-S01
service: vnp-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — Platform Team
---

# vnp-admin

> **Group**: Platform | **gRPC Port**: 9050 | **Health Port**: 9103 | **Origin**: Shared

## Purpose

Central administration service for the VNP Memory platform. Manages **tenant lifecycle**, **API key provisioning** (SHA-256 hashed), **user CRUD**, **aggregated health checks** across all domain services, **billing/quota management**, and **system configuration**.

### Business Capability

- **Tenant Management**: Create/update/delete tenants with configuration profiles
- **API Key Lifecycle**: Generate → hash (SHA-256) → store → validate → revoke API keys per tenant
- **User CRUD**: User profiles within tenants
- **Health Aggregation**: Fan-out health checks to all 35 domain services, aggregate status
- **Billing & Quotas**: Track API usage, enforce rate limits, manage subscription tiers
- **Configuration Management**: Per-tenant feature flags and service configuration overrides

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Database**: PostgreSQL (primary store for tenants, API keys, users, billing)
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-vnp-admin
make run-vnp-admin
docker compose up vnp-admin postgresql
```

## API Surface

### gRPC Service

```protobuf
service VNPAdminService {
  // Tenant management
  rpc CreateTenant(CreateTenantRequest) returns (Tenant);
  rpc GetTenant(GetTenantRequest) returns (Tenant);
  rpc UpdateTenant(UpdateTenantRequest) returns (Tenant);
  rpc ListTenants(ListTenantsRequest) returns (ListTenantsResponse);
  rpc DeleteTenant(DeleteTenantRequest) returns (google.protobuf.Empty);

  // API key management
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);

  // Health aggregation
  rpc GetAggregatedHealth(google.protobuf.Empty) returns (AggregatedHealthResponse);

  // User management
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/admin/tenants` | Create tenant |
| POST | `/v1/admin/tenants/{id}/keys` | Issue API key |
| GET | `/v1/admin/health` | Aggregated health |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Primary persistence |
| All 35 domain services | gRPC health | Health aggregation fan-out |
| NATS JetStream | Async | Publish `vnp.admin.tenant.created`, `vnp.admin.key.issued` |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/08-platform-services.md)

## Owner

- **Team**: VNP Memory — Platform
