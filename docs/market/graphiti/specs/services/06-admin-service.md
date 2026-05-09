# graphiti-admin — Admin Service

**Version:** 2.0 | **Date:** 2026-05-09  
**Origin:** Python L6 partial (maintenance operations, index management)  
**Architecture:** Clean Architecture | **Protocol:** gRPC

---

## 1. Service Overview

Admin Service quản lý operational concerns: tenant management, index operations, health aggregation, community rebuild triggers, và system configuration. Đây là service dành cho operators và automated pipelines, không cho end-users.

### Responsibilities

| Concern | Description |
|---------|-------------|
| **Tenant Management** | Create, configure, delete tenant (group_id) partitions |
| **Index Operations** | Build/delete vector indices, fulltext indices, constraints |
| **Community Rebuild** | Trigger full community detection for specified group_ids |
| **Health Aggregation** | Aggregate health status from all downstream services |
| **Data Cleanup** | Clear data per group or entire graph |
| **Migration** | Database schema migration management |
| **Metrics Export** | System-wide metrics aggregation |
| **Configuration** | Runtime configuration management |

---

## 2. Clean Architecture Layout

```
services/graphiti-admin/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── domain/                         # Layer 1: Entities
│   │   ├── tenant.go                   #   Tenant, TenantConfig, TenantQuota
│   │   ├── index.go                    #   IndexDefinition, IndexStatus
│   │   ├── health.go                   #   ServiceHealth, AggregateHealth
│   │   ├── migration.go                #   Migration, MigrationStatus
│   │   └── errors.go
│   ├── usecase/                        # Layer 2: Use Cases
│   │   ├── manage_tenant.go            #   Tenant CRUD + quota enforcement
│   │   ├── manage_indices.go           #   Build/delete indices
│   │   ├── rebuild_communities.go      #   Trigger community detection
│   │   ├── clear_data.go              #   Group/global data cleanup
│   │   ├── check_health.go            #   Aggregate health checks
│   │   ├── run_migration.go           #   Schema migrations
│   │   ├── export_metrics.go          #   System metrics
│   │   ├── port/
│   │   │   ├── input.go
│   │   │   └── output.go             #   StorePort, KnowledgePort, HealthPort
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/                        # Layer 3: Interface Adapters
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── client/
│   │   │   ├── store_client.go
│   │   │   ├── knowledge_client.go
│   │   │   ├── ingestion_client.go
│   │   │   └── search_client.go
│   │   └── event/
│   │       └── publisher.go           #   Tenant lifecycle events
│   └── infra/
│       ├── config/
│       │   └── config.go
│       ├── server/
│       │   └── grpc.go
│       ├── telemetry/
│       └── wire/
├── api/
│   └── proto/
│       └── admin/v1/
│           └── admin.proto
├── Dockerfile
└── Makefile
```

---

## 3. gRPC API (Protobuf)

```protobuf
syntax = "proto3";
package graphiti.admin.v1;

import "google/protobuf/timestamp.proto";

service AdminService {
  // Tenant management
  rpc CreateTenant(CreateTenantRequest) returns (CreateTenantResponse);
  rpc GetTenant(GetTenantRequest) returns (GetTenantResponse);
  rpc UpdateTenantConfig(UpdateTenantConfigRequest) returns (UpdateTenantConfigResponse);
  rpc DeleteTenant(DeleteTenantRequest) returns (DeleteTenantResponse);
  rpc ListTenants(ListTenantsRequest) returns (ListTenantsResponse);
  
  // Index management
  rpc BuildIndices(BuildIndicesRequest) returns (BuildIndicesResponse);
  rpc DeleteAllIndices(DeleteAllIndicesRequest) returns (DeleteAllIndicesResponse);
  rpc GetIndexStatus(GetIndexStatusRequest) returns (GetIndexStatusResponse);
  
  // Community management
  rpc RebuildCommunities(RebuildCommunitiesRequest) returns (RebuildCommunitiesResponse);
  rpc RemoveCommunities(RemoveCommunitiesRequest) returns (RemoveCommunitiesResponse);
  
  // Data management
  rpc ClearGroup(ClearGroupRequest) returns (ClearGroupResponse);
  rpc ClearAll(ClearAllRequest) returns (ClearAllResponse);
  
  // Health
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
  rpc DeepHealthCheck(DeepHealthCheckRequest) returns (DeepHealthCheckResponse);
  
  // Metrics
  rpc GetMetrics(GetMetricsRequest) returns (GetMetricsResponse);
  rpc GetTokenUsage(GetTokenUsageRequest) returns (GetTokenUsageResponse);
  
  // Migration
  rpc RunMigrations(RunMigrationsRequest) returns (RunMigrationsResponse);
  rpc GetMigrationStatus(GetMigrationStatusRequest) returns (GetMigrationStatusResponse);
}

message CreateTenantRequest {
  string tenant_id = 1;               // maps to group_id
  string display_name = 2;
  TenantConfig config = 3;
}

message TenantConfig {
  TenantQuota quota = 1;
  string llm_model_override = 2;       // tenant-specific model
  string embedder_model_override = 3;
  map<string, string> metadata = 4;
}

message TenantQuota {
  int64 max_episodes = 1;
  int64 max_entities = 2;
  int64 max_edges = 3;
  int64 max_llm_tokens_per_day = 4;
  int32 max_requests_per_minute = 5;
}

message HealthCheckResponse {
  ServiceStatus status = 1;            // HEALTHY, DEGRADED, UNHEALTHY
  map<string, ServiceHealth> services = 2;
  google.protobuf.Timestamp checked_at = 3;
}

message ServiceHealth {
  string name = 1;
  ServiceStatus status = 2;
  string version = 3;
  int64 uptime_seconds = 4;
  optional string error = 5;
}

enum ServiceStatus {
  SERVICE_STATUS_UNSPECIFIED = 0;
  SERVICE_STATUS_HEALTHY = 1;
  SERVICE_STATUS_DEGRADED = 2;
  SERVICE_STATUS_UNHEALTHY = 3;
}
```

---

## 4. Health Check Aggregation

```
Admin Service
  │
  ├─► gRPC Health Check → ingestion   (9001)
  ├─► gRPC Health Check → search      (9002)
  ├─► gRPC Health Check → knowledge   (9003)
  ├─► gRPC Health Check → store       (9004)
  ├─► Redis PING                       (6379)
  ├─► NATS health                      (4222)
  │
  ▼
Aggregate:
  ALL healthy → HEALTHY
  ANY degraded → DEGRADED
  ANY unhealthy → UNHEALTHY
```

---

## 5. Tenant Lifecycle Events

| Event | Payload | Subscribers |
|-------|---------|-------------|
| `tenant.created` | `{tenant_id, config}` | Store (init schema), Search (init cache) |
| `tenant.updated` | `{tenant_id, config_diff}` | Knowledge (model override), Gateway (rate limit) |
| `tenant.deleted` | `{tenant_id}` | Store (clear data), Search (clear cache) |

---

## 6. Configuration

```yaml
# config/admin.yaml
server:
  grpc_port: 9005

services:
  ingestion:
    address: "graphiti-ingestion:9001"
    health_check_interval: 30s
  search:
    address: "graphiti-search:9002"
    health_check_interval: 30s
  knowledge:
    address: "graphiti-knowledge:9003"
    health_check_interval: 30s
  store:
    address: "graphiti-store:9004"
    health_check_interval: 30s

health:
  check_interval: 30s
  timeout_per_service: 5s

events:
  nats_url: "nats://nats:4222"
  stream: "graphiti-admin"

telemetry:
  otel_endpoint: "otel-collector:4317"
  service_name: "graphiti-admin"
```

---

## 7. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Separate admin service** | Operational concerns isolated from data/processing path |
| **Tenant as first-class entity** | Enterprise multi-tenancy requires quota, config, lifecycle management |
| **Health aggregation** | Single endpoint for monitoring systems (K8s, Prometheus) |
| **Event-driven tenant lifecycle** | Services self-configure when tenants change; no polling |
| **Single replica sufficient** | Admin is low-traffic; HA via K8s restart policy |
