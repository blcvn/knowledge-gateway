---
id: TDD-ov-admin
title: Technical Design — ov-admin
service: ov-admin
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: OpenViking
---

# Technical Design — ov-admin

> **Group**: OpenViking | **gRPC Port**: 9056 | **Origin**: OpenViking (Auth + APIKeyManager + Identity)

## 1. Service Overview

Account/User/Agent CRUD with RBAC (4 roles: ROOT > ADMIN > USER > AGENT), API key lifecycle (Argon2id hashing), namespace isolation (`viking://{account}/{user}/{agent}/`), 3 auth modes (dev/trusted/api_key), and health aggregation across 5 OV services.

**Origin mapping**: `openviking/server/auth.py` (433 lines) + `openviking/server/api_keys/api_key_manager.py` + `openviking/server/identity.py`.

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── account.go               # Account, AccountConfig, NamespacePolicy
│   ├── user.go                  # User, Role enum {ROOT, ADMIN, USER, AGENT}
│   ├── agent.go                 # Agent, AgentConfig
│   ├── api_key.go              # APIKey, KeyStatus, ValidateResult
│   └── namespace.go            # NamespaceURI, NamespacePolicy
├── repository/
│   ├── account_repo.go          # AccountRepository
│   ├── user_repo.go             # UserRepository
│   └── api_key_repo.go          # APIKeyRepository
├── event.go                     # AccountCreated, UserDeleted
└── errors.go                    # AccountNotFound, DuplicateUser, InvalidKey, PermissionDenied
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── account_ops.go              # Account CRUD
├── user_ops.go                 # User CRUD within accounts
├── api_key_ops.go             # Create (hash) → Validate (Argon2id) → Revoke
├── health_ops.go              # Fan-out health checks to 5 OV services
├── port/
│   ├── input.go               # AccountUseCase, UserUseCase, APIKeyUseCase, HealthUseCase
│   └── output.go             # HashPort, HealthCheckerPort
└── dto/
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/handler.go             # OvAdminService gRPC
├── client/
│   └── health_client.go        # gRPC health fan-out (errgroup)
└── hasher/
    └── argon2_hasher.go        # Argon2id implementation (or delegate to ov-crypto)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/
│   ├── account_repo.go         # PostgreSQL
│   ├── user_repo.go
│   └── api_key_repo.go
├── config/config.go
└── wire/wire.go
```

## 3. gRPC API

```protobuf
service OvAdminService {
  rpc CreateAccount(CreateAccountRequest) returns (Account);
  rpc GetAccount(GetAccountRequest) returns (Account);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
  rpc DeleteAccount(DeleteAccountRequest) returns (google.protobuf.Empty);
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (APIKey);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeRequest) returns (google.protobuf.Empty);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc GetHealth(google.protobuf.Empty) returns (HealthResponse);
}
```

## 4. NATS Events

None published directly. ov-admin is a query-only service for auth. Account/user lifecycle events could be added in future.

## 5. Data Model

- **ov_accounts**: Account metadata + namespace policy
- **ov_users**: Users within accounts + role assignment
- **ov_agents**: Agents within users + config
- **ov_api_key_hashes** (in ov-crypto): Argon2id hashes + prefix + role

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Purpose |
|---------|-----------|----------|---------|
| PostgreSQL | Outbound | SQL | Account, user, agent persistence |
| ov-crypto | Outbound | gRPC | API key hashing (or direct Argon2id) |
| All 5 OV services | Outbound | gRPC Health | Health aggregation |
| vnp-admin | Peer | gRPC | Tenant-level coordination |

## 7. Observability

- **Metrics**: Account/user CRUD counts, API key validations, health check duration
- **Traces**: OTel spans for CRUD + health aggregation
- **Health**: gRPC Health v1 + HTTP `/healthz` on port 9109

## 8. Multi-Tenancy

- **Isolation**: `account_id` → all queries scoped
- **RBAC**: Enforced at usecase layer per operation

---

> **Next Steps**: Decompose into FEAT specs: FEAT-001 (Account CRUD), FEAT-002 (User CRUD + RBAC), FEAT-003 (API Key Lifecycle), FEAT-004 (Health Aggregation), FEAT-005 (Auth Mode Resolution).
