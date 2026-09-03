---
id: DOC-S02
service: ov-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-admin — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9056

## gRPC Service Definition

```protobuf
// api/proto/openviking/admin/v1/service.proto
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

## Account Management

### CreateAccount

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Account display name |
| `namespace_policy` | NamespacePolicy | No | Isolation policy settings |

### CreateUser

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_id` | string | Yes | Parent account |
| `name` | string | Yes | User display name |
| `role` | Role | Yes | ADMIN / USER / AGENT |

## API Key Management

### CreateAPIKey

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `account_id` | string | Yes | Key scope: account |
| `user_id` | string | No | Key scope: user (if USER role) |
| `role` | Role | Yes | Key role (ROOT/ADMIN/USER) |
| `label` | string | No | Human-readable key label |
| `expires_at` | timestamp | No | Key expiration (optional) |

**Response**: `APIKey { key_id, raw_key (only returned once), role, created_at }`

**Note**: `raw_key` is hashed with Argon2id before storage. It is only returned at creation time.

### ValidateAPIKey

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `raw_key` | string | Yes | API key to validate |

**Response**: `ValidateAPIKeyResponse { valid, account_id, user_id, role, agent_id }`

## Role Hierarchy

| Role | Permissions |
|------|------------|
| `ROOT` | All operations, impersonate any account/user |
| `ADMIN` | Account-scoped CRUD, manage users within account |
| `USER` | User-scoped, cannot override identity headers |
| `AGENT` | Agent-scoped, limited namespace access |

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | Account, user, or key not found |
| `ALREADY_EXISTS` | 409 | Account or user already exists |
| `PERMISSION_DENIED` | 403 | Insufficient role permissions |
| `UNAUTHENTICATED` | 401 | Invalid API key |
| `INVALID_ARGUMENT` | 400 | Invalid parameters |
