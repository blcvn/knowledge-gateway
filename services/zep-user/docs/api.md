---
id: DOC-S02
service: zep-user
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-user — API Reference

## gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.user.v1;

service UserService {
  rpc CreateUser(CreateUserRequest) returns (UserResponse);
  rpc GetUser(GetUserRequest) returns (UserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);
  rpc ListAllUsers(ListUsersRequest) returns (UserListResponse);
  rpc ListAllOrderedUsers(ListUsersRequest) returns (UserListResponse);
  rpc ListUserSessions(ListUserSessionsRequest) returns (SessionListResponse);
}
```

## Messages

### UserResponse

```protobuf
message UserResponse {
  string uuid = 1;
  string user_id = 2;
  string email = 3;
  string first_name = 4;
  string last_name = 5;
  string project_uuid = 6;
  google.protobuf.Struct metadata = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}
```

### CreateUserRequest

```protobuf
message CreateUserRequest {
  string user_id = 1;     // required, alphanumeric_with_underscores
  string email = 2;       // optional
  string first_name = 3;  // optional
  string last_name = 4;   // optional
  google.protobuf.Struct metadata = 5;
}
```

### UpdateUserRequest

```protobuf
message UpdateUserRequest {
  string user_id = 1;     // required
  optional string email = 2;
  optional string first_name = 3;
  optional string last_name = 4;
  google.protobuf.Struct metadata = 5;  // merge-patch: set null to remove keys
}
```

### ListUsersRequest

```protobuf
message ListUsersRequest {
  int32 limit = 1;
  int32 offset = 2;
  string order_by = 3;    // "created_at" | "user_id"
}
```

## RPC Details

### CreateUser

| Attribute | Value |
|-----------|-------|
| **Auth** | Required (JWT/API Key via gateway) |
| **Tenant** | `project_uuid` extracted from gRPC metadata `x-tenant-id` |
| **Validation** | `user_id` must be alphanumeric with underscores |

**Success**: `200 OK` → `UserResponse`
**Errors**:
- `ALREADY_EXISTS` — user_id already exists in project
- `INVALID_ARGUMENT` — invalid user_id format
- `UNAUTHENTICATED` — missing auth

### GetUser

| Attribute | Value |
|-----------|-------|
| **Request** | `GetUserRequest {user_id: string}` |
| **Scope** | Filtered by `project_uuid` |

**Success**: `200 OK` → `UserResponse`
**Errors**: `NOT_FOUND` — user not found or soft-deleted

### UpdateUser

| Attribute | Value |
|-----------|-------|
| **Concurrency** | Advisory lock on user_id (SHA-256 hash) |
| **Metadata** | JSONB merge-patch — set key to `null` to remove |

**Success**: `200 OK` → `UserResponse`
**Errors**: `NOT_FOUND`, `DEADLINE_EXCEEDED` (lock timeout)

### DeleteUser

| Attribute | Value |
|-----------|-------|
| **Strategy** | Soft delete (`deleted_at` timestamp) |
| **Cascade** | Publishes `zep.user.deleted` → zep-thread, zep-graph |

**Success**: `200 OK` → `Empty`
**Errors**: `NOT_FOUND`

### ListAllUsers / ListAllOrderedUsers

| Attribute | Value |
|-----------|-------|
| **Pagination** | Offset-based (`limit`, `offset`) |
| **Ordering** | `created_at DESC` (default), `user_id ASC` |
| **Filter** | `deleted_at IS NULL` |

**Success**: `200 OK` → `UserListResponse {users[], total_count, has_more}`

## NATS Events Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `zep.user.created` | `{user_id, project_uuid, timestamp}` | After successful CreateUser |
| `zep.user.updated` | `{user_id, project_uuid, fields[], timestamp}` | After successful UpdateUser |
| `zep.user.deleted` | `{user_id, project_uuid, timestamp}` | After successful DeleteUser |
