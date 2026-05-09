---
id: DOC-S03
service: vnp-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-admin — Service Architecture

> **Group**: Platform | **Pattern**: 4-layer Clean Architecture

## Layer Structure

```
services/vnp-admin/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go              # Tenant, APIKey, User, HealthStatus
│   │   ├── value_object.go        # TenantID, KeyHash, SubscriptionTier
│   │   ├── event.go               # TenantCreated, KeyIssued, KeyRevoked
│   │   └── errors.go              # ErrTenantNotFound, ErrKeyRevoked, ErrQuotaExceeded
│   ├── usecase/
│   │   ├── tenant.go              # Tenant CRUD + validation
│   │   ├── api_key.go             # Key generation, hashing, validation, revocation
│   │   ├── health.go              # Fan-out health aggregation
│   │   ├── user.go                # User CRUD within tenant
│   │   └── port/
│   │       ├── input.go           # TenantUseCase, APIKeyUseCase, HealthUseCase
│   │       └── output.go          # TenantRepo, KeyRepo, ServiceRegistry, EventPublisher
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC server (12 RPCs)
│   │   │   └── mapper.go          # Proto ↔ Domain
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── tenant_repo.go # Tenant persistence
│   │   │       ├── key_repo.go    # API key storage (SHA-256 hashed)
│   │   │       └── user_repo.go   # User persistence
│   │   ├── client/
│   │   │   └── health_client.go   # gRPC health check to all services
│   │   └── event/
│   │       └── nats_publisher.go  # Admin domain events
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

## Design Decisions

- **SHA-256 API key hashing**: Keys stored as `sha256(key)` — raw key returned only once at creation
- **Health fan-out**: Parallel gRPC health checks with 5s timeout and circuit breakers per service
- **Tenant-scoped everything**: All operations validate tenant context from gRPC metadata
- **Event-driven notifications**: NATS events for tenant/key lifecycle changes enable downstream caching invalidation

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| PostgreSQL | Primary storage (tenants, keys, users) |
| All 35 services (gRPC health) | Health aggregation |
| NATS JetStream | Admin domain events |
