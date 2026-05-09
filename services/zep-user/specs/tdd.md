---
id: TDD-zep-user
title: Technical Design — zep-user
service: zep-user
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Zep
---

# Technical Design — zep-user

> **Group**: Zep | **gRPC Port**: 9061 | **Health Port**: 12061

## 1. Service Overview

User CRUD, metadata management (JSONB merge-patch), project-level isolation via `project_uuid`. Foundation entity for all Zep session and memory operations. Provides user-to-graph association for Temporal Knowledge Graph.

## 2. Clean Architecture Layers

### Domain Layer (Layer 1)
- **User** entity: UUID, UserID (alphanumeric + underscores), Email, FirstName, LastName, ProjectUUID, Metadata (JSONB map), CreatedAt, UpdatedAt, DeletedAt
- **UserID** value object: validation `[a-zA-Z0-9_]+`
- **Metadata** value object: JSONB merge-patch (set key=null to delete)
- **Domain Events**: UserCreated, UserUpdated, UserDeleted
- **Domain Errors**: ErrUserNotFound, ErrUserAlreadyExists, ErrInvalidUserID, ErrProjectRequired

### Usecase Layer (Layer 2)
- **CreateUser**: Validate user_id → extract tenant → build entity → persist → publish event
- **GetUser**: Lookup by user_id + project_uuid scope
- **UpdateUser**: Advisory lock → get existing → merge metadata → persist
- **DeleteUser**: Soft delete (set deleted_at) → publish cascade event
- **ListUsers / ListOrderedUsers**: Paginated listing with ordering
- **Port interfaces**: UserService (input), UserRepository + UserEventPublisher (output)

### Adapter Layer (Layer 3)
- **gRPC handler**: UserServiceServer implementation with proto ↔ domain mappers
- **PostgreSQL repo**: User CRUD via uptrace/bun ORM with advisory locks
- **NATS publisher**: Publish user lifecycle events to JetStream

### Infrastructure Layer (Layer 4)
- Config (Viper), Server (gRPC), Wire (DI), Telemetry (OTel + Prometheus)

## 3. gRPC API

```protobuf
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

## 4. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.user.created` | `{user_id, project_uuid, timestamp}` | zep-graph (init user node) |
| `zep.user.updated` | `{user_id, project_uuid, fields[], timestamp}` | zep-graph (update user node) |
| `zep.user.deleted` | `{user_id, project_uuid, timestamp}` | zep-thread, zep-graph (cascade) |

## 5. Storage

PostgreSQL: users table with JSONB metadata, advisory locks, soft deletes, partial indexes.

```sql
CREATE TABLE users (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    email TEXT,
    first_name TEXT,
    last_name TEXT,
    project_uuid UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (user_id, project_uuid)
);
```

## 6. Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | User persistence with JSONB metadata, advisory locks |
| NATS JetStream | Pub | Publish user lifecycle events |

## 7. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs, advisory lock contention rate
- **Traces**: OTel spans for every usecase method
- **Logs**: Structured JSON via slog with request_id, tenant_id, user_id
- **Health**: gRPC health check + HTTP /healthz, /readyz, /livez on port 12061

## 8. Multi-Tenancy

Project isolation via `project_uuid`. All queries filtered by project scope. Propagated via gRPC metadata `x-tenant-id`.

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.
