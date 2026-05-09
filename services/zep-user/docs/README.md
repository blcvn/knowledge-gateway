---
id: DOC-S01
service: zep-user
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Zep Team
---

# zep-user

> **Group**: Zep (Context Engineering) | **gRPC Port**: 9061 | **Health Port**: 12061 | **Origin**: Zep

## Purpose

Quản lý toàn bộ lifecycle của User entities trong Zep context engineering platform. Service này cung cấp User CRUD với metadata management (JSONB merge-patch), multi-tenant isolation qua `project_uuid`, soft delete pattern cho audit trail preservation, và user-to-graph association cho Temporal Knowledge Graph.

### Business Capability

- **User CRUD**: Create/Read/Update/Delete users với human-readable `user_id` (alphanumeric + underscores)
- **Metadata Merge-Patch**: JSONB metadata hỗ trợ partial update (set `null` để xóa key) với advisory lock protection
- **Project Isolation**: Mọi user scoped theo `project_uuid` cho multi-tenant separation
- **Soft Deletes**: Users marked với `deleted_at` cho audit trails, recovery, và temporal analysis
- **Graph Association**: User entity liên kết tới Temporal KG — mỗi user là singleton node trong knowledge graph

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream publisher
- **Database**: PostgreSQL (users table với JSONB metadata, advisory locks)
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire
- **Observability**: OTel traces + Prometheus metrics + slog structured logging

## Quick Start

```bash
# Build
make build-zep-user

# Run locally
make run-zep-user

# Docker Compose
docker compose up zep-user

# Test
make test-zep-user
```

## API Surface

### gRPC Service

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

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v2/users` | Create user |
| `GET` | `/api/v2/users/{user_id}` | Get user |
| `PATCH` | `/api/v2/users/{user_id}` | Update user (metadata merge-patch) |
| `DELETE` | `/api/v2/users/{user_id}` | Soft delete user |
| `GET` | `/api/v2/users` | List users (paginated) |
| `GET` | `/api/v2/users/{user_id}/sessions` | List user's sessions |

## NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.user.created` | `{user_id, project_uuid, timestamp}` | zep-graph (init user node) |
| `zep.user.updated` | `{user_id, project_uuid, fields[], timestamp}` | zep-graph (update user node) |
| `zep.user.deleted` | `{user_id, project_uuid, timestamp}` | zep-thread (cascade soft delete sessions), zep-graph (delete user graph data) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | User persistence with JSONB metadata, advisory locks |
| NATS JetStream | Pub | Publish user lifecycle events |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/06-zep-services.md)
- [Reference Design](../../../references/zep/specs/services/02-user-service.md)

## Owner

- **Team**: VNP Memory — Zep
