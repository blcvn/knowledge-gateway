---
id: DOC-S01
service: ov-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — OpenViking Team
---

# ov-admin

> **Group**: OpenViking | **gRPC Port**: 9056 | **Health Port**: 9109 | **Origin**: OpenViking

## Purpose

Account/User/Agent CRUD, **API key management** (Argon2id hashed), **RBAC enforcement**, **namespace isolation**, and **health aggregation** across OpenViking services. Replaces Python `openviking/server/auth.py`, `openviking/server/api_keys/`, and `openviking/server/identity.py`.

### Business Capability

- **Account Management**: Create/update/delete accounts (top-level tenant isolation)
- **User Management**: Users within accounts, role assignment
- **Agent Management**: Agents within users (namespace isolation: `account/user/agent`)
- **API Key Lifecycle**: Generate → hash (Argon2id) → store → validate → revoke
- **RBAC Model**: 4 roles — ROOT, ADMIN, USER, AGENT with cascading permissions
- **Auth Modes**: `api_key` (production), `trusted` (internal), `dev` (development)
- **Health Aggregation**: Fan-out health checks to ov-fs, ov-search, ov-session, ov-resource, ov-crypto

## Auth Modes (from OpenViking reference)

| Mode | Authentication | Identity Resolution |
|------|---------------|---------------------|
| `dev` | None | Implicit ROOT with default account/user |
| `trusted` | Optional root API key | Trust X-OpenViking-Account/User headers |
| `api_key` | Required | Resolve via APIKeyManager (root → admin → user) |

## RBAC Model

```
Account → User → Agent (namespace isolation)
Roles: ROOT > ADMIN > USER > AGENT

ROOT:  All operations, can impersonate any account/user
ADMIN: Account-scoped, can manage users within account
USER:  User-scoped, cannot override X-OpenViking-User
AGENT: Agent-scoped, limited to assigned namespace
```

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Database**: PostgreSQL (accounts, users, API keys)
- **Crypto**: Argon2id for API key hashing
- **Architecture**: 4-layer Clean Architecture
- **DI**: Google Wire

## Quick Start

```bash
make build-ov-admin
make run-ov-admin
docker compose up ov-admin postgresql
```

## API Surface

### gRPC Service

```protobuf
service OvAdminService {
  // Account management
  rpc CreateAccount(CreateAccountRequest) returns (Account);
  rpc GetAccount(GetAccountRequest) returns (Account);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
  rpc DeleteAccount(DeleteAccountRequest) returns (google.protobuf.Empty);

  // User management
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);

  // API Key management
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (APIKey);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeRequest) returns (google.protobuf.Empty);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);

  // Health
  rpc GetHealth(google.protobuf.Empty) returns (HealthResponse);
}
```

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Account, user, API key persistence |
| ov-fs, ov-search, ov-session, ov-resource, ov-crypto | gRPC Health | Health aggregation |
| vnp-admin | gRPC | Tenant-level admin coordination |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)

## Owner

- **Team**: VNP Memory — OpenViking
