# 09 — Data Models & Protobuf Definitions

> **Location**: `api/proto/`  
> **Format**: Protobuf v3 with Google well-known types

---

## 1. Proto Package Structure

```
api/proto/
├── common/v1/
│   ├── pagination.proto        # PageRequest, PageResponse
│   ├── temporal.proto          # TemporalAnnotation, Timestamp helpers
│   ├── errors.proto            # ErrorDetail, ErrorCode enum
│   └── health.proto            # HealthCheckRequest/Response
├── user/v1/
│   └── user.proto              # UserService + messages
├── thread/v1/
│   └── thread.proto            # ThreadService + messages
├── memory/v1/
│   └── memory.proto            # MemoryService + messages
├── graph/v1/
│   ├── graph.proto             # GraphService + messages
│   ├── fact.proto              # Fact messages
│   └── ontology.proto          # OntologyConfig messages
├── search/v1/
│   └── search.proto            # SearchService + messages
└── admin/v1/
    └── admin.proto             # AdminService + messages
```

---

## 2. Common Types

### 2.1 Pagination

```protobuf
syntax = "proto3";
package zep.common.v1;

message PageRequest {
  int32 limit = 1;              // default 20, max 100
  int32 offset = 2;             // 0-based
  string order_by = 3;          // field name
  string order = 4;             // "asc" | "desc"
}

message PageResponse {
  int32 total = 1;
  int32 limit = 2;
  int32 offset = 3;
  bool has_more = 4;
}
```

### 2.2 Temporal

```protobuf
message TemporalAnnotation {
  optional google.protobuf.Timestamp valid_at = 1;     // when became true
  optional google.protobuf.Timestamp invalid_at = 2;   // when ceased to be true
  optional google.protobuf.Timestamp expired_at = 3;   // when superseded
}
```

### 2.3 Error Detail

```protobuf
message ErrorDetail {
  string code = 1;              // ErrorCode string
  string message = 2;
  map<string, string> details = 3;
}

enum ErrorCode {
  ERROR_CODE_UNSPECIFIED = 0;
  ERROR_CODE_NOT_FOUND = 1;
  ERROR_CODE_ALREADY_EXISTS = 2;
  ERROR_CODE_PERMISSION_DENIED = 3;
  ERROR_CODE_UNAUTHENTICATED = 4;
  ERROR_CODE_INVALID_INPUT = 5;
  ERROR_CODE_RATE_LIMITED = 6;
  ERROR_CODE_SESSION_ENDED = 7;
  ERROR_CODE_LOCK_TIMEOUT = 8;
  ERROR_CODE_INTERNAL = 9;
}
```

---

## 3. Domain Models — Go ↔ PostgreSQL ↔ Proto Mapping

### 3.1 User

| Go Domain | PostgreSQL Column | Proto Field | Type |
|-----------|------------------|-------------|------|
| `UUID` | `uuid` | `uuid` | `string` (UUID) |
| `UserID` | `user_id` | `user_id` | `string` (unique) |
| `Email` | `email` | `email` | `string` |
| `FirstName` | `first_name` | `first_name` | `string` |
| `LastName` | `last_name` | `last_name` | `string` |
| `ProjectUUID` | `project_uuid` | `project_uuid` | `string` (UUID) |
| `Metadata` | `metadata` (JSONB) | `metadata` | `google.protobuf.Struct` |
| `CreatedAt` | `created_at` | `created_at` | `google.protobuf.Timestamp` |
| `UpdatedAt` | `updated_at` | `updated_at` | `google.protobuf.Timestamp` |
| `DeletedAt` | `deleted_at` | — | `*time.Time` (not exposed) |

### 3.2 Session/Thread

| Go Domain | PostgreSQL Column | Proto Field | Type |
|-----------|------------------|-------------|------|
| `UUID` | `uuid` | `uuid` | `string` (UUID) |
| `SessionID` | `session_id` | `session_id` | `string` (unique) |
| `UserID` | `user_id` | `user_id` | `optional string` |
| `ProjectUUID` | `project_uuid` | `project_uuid` | `string` (UUID) |
| `Metadata` | `metadata` (JSONB) | `metadata` | `google.protobuf.Struct` |
| `EndedAt` | `ended_at` | `ended_at` | `optional Timestamp` |
| `CreatedAt` | `created_at` | `created_at` | `Timestamp` |
| `UpdatedAt` | `updated_at` | `updated_at` | `Timestamp` |

### 3.3 Message

| Go Domain | PostgreSQL Column | Proto Field | Type |
|-----------|------------------|-------------|------|
| `UUID` | `uuid` | `uuid` | `string` (UUID) |
| `SessionID` | `session_id` | `session_id` | `string` |
| `ProjectUUID` | `project_uuid` | — | `string` (internal) |
| `Role` | `role` | `role` | `string` |
| `RoleType` | `role_type` | `role_type` | `string` (enum) |
| `Content` | `content` | `content` | `string` |
| `TokenCount` | `token_count` | `token_count` | `int32` |
| `Metadata` | `metadata` (JSONB) | `metadata` | `google.protobuf.Struct` |
| `CreatedAt` | `created_at` | `created_at` | `Timestamp` |

### 3.4 Fact (Graphiti Edge)

| Go Domain | Neo4j Property | Proto Field | Type |
|-----------|---------------|-------------|------|
| `UUID` | `uuid` | `uuid` | `string` |
| `Name` | `name` | `name` | `string` |
| `Fact` | `fact` | `fact` | `string` |
| `SourceID` | (relationship) | `source_id` | `string` |
| `TargetID` | (relationship) | `target_id` | `string` |
| `EdgeType` | `type` | `edge_type` | `string` |
| `GroupID` | `group_id` | `group_id` | `string` |
| `ValidAt` | `valid_at` | `valid_at` | `optional Timestamp` |
| `InvalidAt` | `invalid_at` | `invalid_at` | `optional Timestamp` |
| `ExpiredAt` | `expired_at` | `expired_at` | `optional Timestamp` |
| `CreatedAt` | `created_at` | `created_at` | `Timestamp` |

### 3.5 Entity Node

| Go Domain | Neo4j Property | Proto Field | Type |
|-----------|---------------|-------------|------|
| `UUID` | `uuid` | `uuid` | `string` |
| `Name` | `name` | `name` | `string` |
| `NodeType` | label | `node_type` | `string` |
| `GroupID` | `group_id` | `group_id` | `string` |
| `Summary` | `summary` | `summary` | `string` |
| `Labels` | labels | `labels` | `repeated string` |
| `Properties` | properties | `properties` | `google.protobuf.Struct` |
| `CreatedAt` | `created_at` | `created_at` | `Timestamp` |

### 3.6 Episode

| Go Domain | Neo4j Property | Proto Field | Type |
|-----------|---------------|-------------|------|
| `UUID` | `uuid` | `uuid` | `string` |
| `Name` | `name` | `name` | `string` |
| `Content` | `content` | `content` | `string` |
| `GroupID` | `group_id` | `group_id` | `string` |
| `SourceID` | `source_id` | `source_id` | `string` |
| `CreatedAt` | `created_at` | `created_at` | `Timestamp` |

---

## 4. Role Type Enum

```protobuf
enum RoleType {
  ROLE_TYPE_UNSPECIFIED = 0;
  ROLE_TYPE_NOROLE = 1;
  ROLE_TYPE_SYSTEM = 2;
  ROLE_TYPE_ASSISTANT = 3;
  ROLE_TYPE_USER = 4;
  ROLE_TYPE_FUNCTION = 5;
  ROLE_TYPE_TOOL = 6;
}
```

```sql
-- PostgreSQL enum
CREATE TYPE role_type_enum AS ENUM (
    'norole', 'system', 'assistant', 'user', 'function', 'tool'
);
```

---

## 5. Complete PostgreSQL Schema (All Tables)

```sql
-- Project/Tenant
CREATE TABLE projects (
    uuid            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    description     TEXT,
    organization_id TEXT,
    settings        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

-- API Keys
CREATE TABLE api_keys (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash     TEXT NOT NULL UNIQUE,
    key_prefix   TEXT NOT NULL,
    project_uuid UUID NOT NULL REFERENCES projects(uuid),
    name         TEXT NOT NULL,
    scopes       TEXT[] DEFAULT '{"read","write"}',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ
);

-- Users
CREATE TABLE users (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL,
    email        TEXT,
    first_name   TEXT,
    last_name    TEXT,
    project_uuid UUID NOT NULL,
    metadata     JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    UNIQUE (user_id, project_uuid)
);

-- Sessions/Threads
CREATE TABLE sessions (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   TEXT NOT NULL,
    user_id      TEXT,
    project_uuid UUID NOT NULL,
    metadata     JSONB DEFAULT '{}',
    ended_at     TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ,
    UNIQUE (session_id, project_uuid)
);

-- Role Type Enum
CREATE TYPE role_type_enum AS ENUM (
    'norole', 'system', 'assistant', 'user', 'function', 'tool'
);

-- Messages
CREATE TABLE messages (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   TEXT NOT NULL,
    project_uuid UUID NOT NULL,
    role         TEXT NOT NULL,
    role_type    role_type_enum NOT NULL DEFAULT 'norole',
    content      TEXT NOT NULL,
    token_count  INTEGER DEFAULT 0,
    metadata     JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

-- Indexes
CREATE INDEX user_user_id_idx ON users(user_id) WHERE deleted_at IS NULL;
CREATE INDEX user_email_idx ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX user_project_uuid_idx ON users(project_uuid) WHERE deleted_at IS NULL;

CREATE INDEX session_user_id_idx ON sessions(user_id) WHERE deleted_at IS NULL;
CREATE INDEX session_project_uuid_idx ON sessions(project_uuid) WHERE deleted_at IS NULL;
CREATE INDEX session_composite_idx ON sessions(session_id, project_uuid, deleted_at);
CREATE INDEX session_created_at_idx ON sessions(created_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX memstore_session_id_idx ON messages(session_id) WHERE deleted_at IS NULL;
CREATE INDEX memstore_id_idx ON messages(uuid) WHERE deleted_at IS NULL;
CREATE INDEX memstore_composite_idx ON messages(session_id, project_uuid, deleted_at);
CREATE INDEX memstore_created_at_idx ON messages(created_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX api_key_hash_idx ON api_keys(key_hash) WHERE revoked_at IS NULL;
CREATE INDEX api_key_project_idx ON api_keys(project_uuid) WHERE revoked_at IS NULL;
```

---

## 6. Neo4j Schema (Constraints & Indexes)

```cypher
-- Node constraints
CREATE CONSTRAINT entity_node_uuid IF NOT EXISTS
FOR (n:EntityNode) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT episode_uuid IF NOT EXISTS
FOR (n:Episode) REQUIRE n.uuid IS UNIQUE;

-- Node indexes
CREATE INDEX entity_node_group_idx IF NOT EXISTS
FOR (n:EntityNode) ON (n.group_id);

CREATE INDEX entity_node_type_idx IF NOT EXISTS
FOR (n:EntityNode) ON (n.node_type);

CREATE INDEX episode_group_idx IF NOT EXISTS
FOR (n:Episode) ON (n.group_id);

-- Fulltext indexes for search
CREATE FULLTEXT INDEX entity_node_search IF NOT EXISTS
FOR (n:EntityNode) ON EACH [n.name, n.summary];

-- Edge property indexes
CREATE INDEX entity_edge_valid_at_idx IF NOT EXISTS
FOR ()-[r:RELATES_TO]-() ON (r.valid_at);

CREATE INDEX entity_edge_group_idx IF NOT EXISTS
FOR ()-[r:RELATES_TO]-() ON (r.group_id);
```
