# Technical Design Document: Zep Core Service

## 1. System Architecture

`zep-core` is the consolidated engine merging `zep-user`, `zep-thread`, and `zep-memory` to eliminate synchronous cross-service gRPC calls and achieve a sub-200ms critical path.

```text
zep-core/
├── internal/
│   ├── domain/
│   │   ├── user/        # User entity, JSONB metadata
│   │   ├── thread/      # Thread, Session logic
│   │   └── memory/      # Message, Context assembly
│   ├── usecase/
│   │   ├── user/        # CRUD, merge-patch metadata
│   │   ├── thread/      # Session lifecycles
│   │   └── memory/      # PutMemory, GetMemory, Fact Overlay
│   ├── adapter/
│   │   ├── grpc/        # Handlers: User, Thread, Memory
│   │   ├── search/      # gRPC Client to zep-search
│   │   └── broker/      # NATS Publisher
│   └── infra/
│       ├── postgres/    # Consolidated DB
│       └── metrics/     # Prometheus/OTel
```

## 2. Component Design

### 2.1 Domain Layer
- **User**: `uuid`, `project_uuid`, `metadata` (JSONB).
- **Thread/Session**: `uuid`, `user_uuid`, `project_uuid`, `ended_at`.
- **Message**: `uuid`, `thread_uuid`, `role`, `content`, `metadata`.

### 2.2 Usecase Layer
- **PutMemory**:
  - Automatically upserts the Thread/Session using the local Thread module.
  - Persists messages in PostgreSQL.
  - Fires `zep.memory.messages.ingested` to NATS.
- **GetMemory**:
  - Fetches last max(N,4) messages from PostgreSQL.
  - Computes `groupID = user_id ?? session_id`.
  - Triggers asynchronous fetches to `zep-search` (Graphiti) passing `groupID` and recent messages as query context.
  - Synthesizes the final context array containing `{ messages: [lastN], facts: [...] }`.
- **Metadata Patching**:
  - Uses JSONB merge-patch.
  - Implements PostgreSQL advisory locks for concurrent metadata updates to prevent race conditions.
  - Lock key = SHA-256 hash of session ID.
  - Retry policy: exponential backoff 200ms→30s, max 15 retries.

### 2.3 Adapter Layer
- **ZepSearchClient**: Dedicated internal gRPC client to call `zep-search`.
- **NATS Publisher**: Emits events (`zep.user.deleted`, `zep.thread.session.ended`, `zep.memory.messages.ingested`).

## 3. Data Models

```sql
CREATE TYPE role_type_enum AS ENUM (
  'norole', 'system', 'assistant', 'user', 'function', 'tool'
);

CREATE TABLE users (
    uuid UUID PRIMARY KEY,
    user_id VARCHAR(255) UNIQUE,
    project_uuid UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    deleted_at BIGINT
);

CREATE TABLE threads (
    uuid UUID PRIMARY KEY,
    session_id VARCHAR(255) UNIQUE,
    user_uuid UUID REFERENCES users(uuid),
    project_uuid UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    ended_at BIGINT,
    deleted_at BIGINT
);

CREATE TABLE messages (
    uuid UUID PRIMARY KEY,
    thread_uuid UUID REFERENCES threads(uuid),
    role VARCHAR(50) NOT NULL,
    role_type role_type_enum NOT NULL DEFAULT 'user',
    content TEXT NOT NULL,
    token_count INT,
    metadata JSONB DEFAULT '{}',
    created_at BIGINT NOT NULL,
    deleted_at BIGINT
);
```
