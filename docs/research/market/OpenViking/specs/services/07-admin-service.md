# 07 — OpenViking Admin Service

> **Service**: `openviking-admin`  
> **Port**: 9030 (gRPC) · 9099 (Health/Metrics)  
> **Origin**: L1 Admin Router + L2 TaskTracker + Maintenance  
> **Role**: Account/User/API Key CRUD, health aggregation, task tracking, system maintenance

---

## 1. Responsibilities

| Capability | Description |
|-----------|-------------|
| **Account Management** | Create, list, delete accounts (tenants) |
| **User Management** | Create, list, delete users within accounts |
| **API Key Management** | Create, list, revoke API keys with role assignment |
| **Health Aggregation** | Aggregate health from all downstream services |
| **Task Tracking** | Track background tasks across all services |
| **System Info** | Server status, version, configuration diagnostics |
| **Maintenance** | Storage maintenance, vector index cleanup |
| **Usage Stats** | Aggregated usage statistics per tenant |

---

## 2. Clean Architecture Layout

```
services/openviking-admin/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── account.go                  # Account, AccountConfig
│   │   ├── user.go                     # User within account
│   │   ├── api_key.go                  # APIKey, KeyRole, KeyMetadata
│   │   ├── task.go                     # BackgroundTask, TaskState
│   │   ├── health.go                   # ServiceHealth, AggregatedHealth
│   │   ├── stats.go                    # UsageStats, TenantStats
│   │   └── errors.go
│   ├── usecase/
│   │   ├── manage_accounts.go          # Account CRUD
│   │   ├── manage_users.go             # User CRUD
│   │   ├── manage_api_keys.go          # API key CRUD + role assignment
│   │   ├── aggregate_health.go         # Health check all services
│   │   ├── track_tasks.go              # Task status aggregation
│   │   ├── system_info.go              # Server status, diagnostics
│   │   ├── run_maintenance.go          # Storage cleanup, index maintenance
│   │   ├── collect_stats.go            # Usage statistics
│   │   ├── port/
│   │   │   ├── input.go               # AdminUseCase interfaces
│   │   │   └── output.go             # AccountStore, KeyStore, ServiceClients
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── redis/                 # API key store, task store
│   │   │   └── local/                 # File-based fallback
│   │   ├── client/                    # Health check clients to all services
│   │   │   ├── fs_client.go
│   │   │   ├── search_client.go
│   │   │   ├── session_client.go
│   │   │   ├── resource_client.go
│   │   │   └── crypto_client.go
│   │   └── event/
│   │       └── publisher.go            # NATS: admin.account.created/deleted
│   └── infra/
```

---

## 3. gRPC Service Definition

```protobuf
service AdminService {
  // Accounts
  rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
  rpc DeleteAccount(DeleteAccountRequest) returns (DeleteAccountResponse);

  // Users
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);

  // API Keys
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (RevokeAPIKeyResponse);
  rpc ResolveAPIKey(ResolveAPIKeyRequest) returns (ResolveAPIKeyResponse);

  // Health
  rpc GetHealth(GetHealthRequest) returns (GetHealthResponse);
  rpc GetAggregatedHealth(GetAggregatedHealthRequest) returns (AggregatedHealthResponse);

  // Tasks
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);

  // System
  rpc GetSystemInfo(GetSystemInfoRequest) returns (GetSystemInfoResponse);
  rpc WaitReady(WaitReadyRequest) returns (WaitReadyResponse);

  // Maintenance
  rpc RunMaintenance(RunMaintenanceRequest) returns (RunMaintenanceResponse);

  // Stats
  rpc GetUsageStats(GetUsageStatsRequest) returns (GetUsageStatsResponse);
}
```

---

## 4. API Key Model

```go
type APIKey struct {
    ID         string    // Unique key ID
    Key        string    // Hashed key value (bcrypt)
    AccountID  string    // Bound to account
    UserID     string    // Bound to user (empty for admin keys)
    Role       Role      // ROOT / ADMIN / USER
    Name       string    // Human-readable name
    CreatedAt  time.Time
    ExpiresAt  *time.Time // Optional expiry
    LastUsedAt *time.Time
    IsActive   bool
}
```

---

## 5. Health Aggregation

```
GetAggregatedHealth():
  1. Parallel gRPC health check to all 6 services
  2. Aggregate results:
     - ALL healthy → SERVING
     - ANY unhealthy → DEGRADED
     - ALL unhealthy → NOT_SERVING
  3. Include per-service details + latency
```

---

## 6. Account Lifecycle Events

```
CreateAccount → Publish admin.account.created
  → FS service: initialize root directories for account
  → Search service: create vector collection for account
  → Crypto service: derive account encryption key

DeleteAccount → Publish admin.account.deleted
  → FS service: cascade delete all account data
  → Search service: drop vector collection
  → Session service: delete all account sessions
```
