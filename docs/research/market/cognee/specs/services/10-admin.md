# 10 — Unified Admin Service (vnp-admin)

> **gRPC**: 9030 | **Health**: 9099

---

## 1. Purpose

Shared administrative service for both Cognee + Graphiti: user management, tenant lifecycle, API key management, dataset/group ACLs, aggregated health, and system configuration.

---

## 2. Clean Architecture

```
services/vnp-admin/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Tenant, User, APIKey, Dataset, Group, Permission
│   │   ├── value_object.go     # Role, PermissionLevel, APIKeyScope
│   │   ├── event.go            # TenantCreatedEvent, DatasetDeletedEvent
│   │   └── errors.go
│   ├── usecase/
│   │   ├── tenant/
│   │   │   ├── create_tenant.go
│   │   │   ├── delete_tenant.go    # Cascade to all services
│   │   │   └── list_tenants.go
│   │   ├── user/
│   │   │   ├── create_user.go
│   │   │   ├── update_user.go
│   │   │   └── list_users.go
│   │   ├── apikey/
│   │   │   ├── create_apikey.go
│   │   │   ├── revoke_apikey.go
│   │   │   └── validate_apikey.go
│   │   ├── dataset/
│   │   │   ├── manage_acl.go       # Cognee dataset permissions
│   │   │   └── delete_dataset.go   # Emit cascade event
│   │   ├── group/
│   │   │   └── manage_group.go     # Graphiti group management
│   │   ├── health/
│   │   │   └── aggregated_health.go # Fan-out health to all services
│   │   ├── port/
│   │   │   └── output.go
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/handler.go     # admin.v1.AdminService impl
│   │   ├── repository/
│   │   │   └── postgres/       # Users, tenants, API keys, ACLs
│   │   ├── client/
│   │   │   └── service_health.go  # gRPC health fan-out to all services
│   │   └── event/publisher.go     # NATS: admin.tenant.*, admin.dataset.*
│   └── infra/
│       └── wire/wire.go
```

---

## 3. Tenant Lifecycle

```
CreateTenant(name, config)
  1. Insert tenant row in PostgreSQL
  2. Emit admin.tenant.created → All services
  3. Each service initializes tenant-specific resources:
     - Cognee: RLS policy, dataset defaults
     - Graphiti: group_id namespace, indexes

DeleteTenant(tenant_id)
  1. Emit admin.tenant.deleted → All services
  2. Each service cascades cleanup:
     - Cognee: delete datasets, chunks, graph nodes, embeddings
     - Graphiti: delete all nodes/edges for group_id
  3. Soft-delete tenant row (30-day retention)
```

---

## 4. RBAC Model

```go
type Role string
const (
    RoleAdmin    Role = "admin"     // Full access
    RoleUser     Role = "user"      // Own data only
    RoleReader   Role = "reader"    // Read-only
    RoleService  Role = "service"   // Service-to-service
)

type Permission struct {
    TenantID   string
    UserID     string
    Resource   string    // "cognee:dataset:*", "graphiti:group:*"
    Action     string    // "read", "write", "delete", "admin"
}
```

---

## 5. Aggregated Health Check

```go
func (uc *AggregatedHealthUseCase) Execute(ctx context.Context) (*HealthReport, error) {
    // Fan-out gRPC health checks to all 8 services concurrently
    services := []string{
        "cognee-ingestion", "cognee-cognify", "cognee-search", "cognee-memory",
        "graphiti-ingestion", "graphiti-search", "graphiti-knowledge", "graphiti-store",
    }
    
    results := make(chan ServiceHealth, len(services))
    for _, svc := range services {
        go func(name string) {
            results <- uc.healthClient.Check(ctx, name)
        }(svc)
    }
    // Aggregate: all SERVING → healthy, any NOT_SERVING → degraded
}
```
