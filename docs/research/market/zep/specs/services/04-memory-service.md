# 04 — Memory Service (zep-memory)

> **gRPC**: 9043 | **Health**: 9143  
> **Origin**: L3 — Business Logic Layer (Memory DAO + Message DAO)

---

## 1. Purpose

Core orchestrator của Zep — quản lý message ingestion và context assembly. Cung cấp:
- **PutMemory**: Ingest messages → PostgreSQL → trigger async graph extraction
- **GetMemory**: Assemble context = last N messages + relevant facts from KG
- **Message CRUD**: Store, retrieve, update messages per session
- **User Context**: Pre-formatted context blocks optimized for LLMs

Đây là service phức tạp nhất, orchestrate giữa zep-thread (session state) + zep-graph (async extraction) + zep-search (fact retrieval).

---

## 2. Clean Architecture Layout

```
services/zep-memory/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── memory.go              # Memory entity (messages + facts overlay)
│   │   ├── message.go             # Message entity
│   │   ├── role_type.go           # RoleType enum (norole|system|assistant|user|function|tool)
│   │   ├── fact.go                # Fact value object (from Graphiti)
│   │   ├── event.go               # MessagesIngested, MemoryDeleted events
│   │   └── errors.go              # ErrSessionEnded, ErrEmptyMessages
│   │
│   ├── usecase/
│   │   ├── put_memory.go          # PutMemory — ingest messages + trigger extraction
│   │   ├── get_memory.go          # GetMemory — assemble context (messages + facts)
│   │   ├── delete_memory.go       # DeleteMemory — soft delete all messages in session
│   │   ├── get_messages.go        # GetMessagesForSession — paginated
│   │   ├── get_message.go         # GetMessage by UUID
│   │   ├── update_message.go      # UpdateMessageMetadata (JSONB merge)
│   │   ├── get_user_context.go    # GetUserContext — formatted context for LLMs
│   │   ├── port/
│   │   │   ├── input.go           # MemoryService interface
│   │   │   └── output.go          # MessageRepository, ThreadClient, SearchClient, EventPublisher
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── message_repo.go    # Message CRUD (batch insert, GetLastN)
│   │   │       └── model.go
│   │   ├── client/
│   │   │   ├── thread_client.go       # gRPC client → zep-thread
│   │   │   └── search_client.go       # gRPC client → zep-search
│   │   └── event/
│   │       └── publisher.go           # NATS publisher for async graph extraction
│   │
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Layer

### 3.1 Memory Entity (API Overlay)

```go
package domain

// Memory is the composite response assembling messages + graph facts
type Memory struct {
    Messages      []Message
    RelevantFacts []Fact
    Metadata      map[string]any
}
```

### 3.2 Message Entity

```go
type Message struct {
    UUID        string
    SessionID   string
    ProjectUUID string
    Role        string        // "user" | "assistant" | "system"
    RoleType    RoleType      // enum
    Content     string
    TokenCount  int
    Metadata    Metadata      // JSONB
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time
}

type RoleType string

const (
    RoleTypeNone      RoleType = "norole"
    RoleTypeSystem    RoleType = "system"
    RoleTypeAssistant RoleType = "assistant"
    RoleTypeUser      RoleType = "user"
    RoleTypeFunction  RoleType = "function"
    RoleTypeTool      RoleType = "tool"
)
```

### 3.3 Fact Value Object (from Graphiti)

```go
// Fact represents a temporal edge from the knowledge graph
type Fact struct {
    UUID       string
    Name       string         // relationship label
    Fact       string         // human-readable fact statement
    CreatedAt  time.Time
    ValidAt    *time.Time     // when fact became true
    InvalidAt  *time.Time     // when fact ceased to be true
    ExpiredAt  *time.Time     // when fact was superseded
}

// IsCurrentlyValid returns true if fact is active at given time
func (f *Fact) IsCurrentlyValid(at time.Time) bool {
    if f.ValidAt != nil && at.Before(*f.ValidAt) {
        return false
    }
    if f.InvalidAt != nil && at.After(*f.InvalidAt) {
        return false
    }
    return true
}
```

### 3.4 Domain Errors

```go
var (
    ErrSessionEnded    = errors.New("session has been ended; cannot add messages")
    ErrEmptyMessages   = errors.New("messages list cannot be empty")
    ErrMessageNotFound = errors.New("message not found")
    ErrInvalidRole     = errors.New("invalid message role")
)
```

---

## 4. Use Case Layer

### 4.1 Port Interfaces

```go
package port

type MemoryService interface {
    PutMemory(ctx context.Context, req dto.PutMemoryRequest) error
    GetMemory(ctx context.Context, req dto.GetMemoryRequest) (*dto.MemoryResponse, error)
    DeleteMemory(ctx context.Context, sessionID string) error
    GetMessagesForSession(ctx context.Context, sessionID string, limit, offset int) (*dto.MessageListResponse, error)
    GetMessage(ctx context.Context, sessionID string, messageUUID string) (*dto.MessageResponse, error)
    UpdateMessageMetadata(ctx context.Context, sessionID string, messageUUID string, metadata map[string]any) (*dto.MessageResponse, error)
    GetUserContext(ctx context.Context, threadID string, templateID *string) (*dto.UserContextResponse, error)
}

type MessageRepository interface {
    CreateMany(ctx context.Context, messages []*domain.Message) error
    GetLastN(ctx context.Context, sessionID string, projectUUID string, n int) ([]*domain.Message, error)
    GetByUUID(ctx context.Context, uuid string, projectUUID string) (*domain.Message, error)
    UpdateMetadata(ctx context.Context, uuid string, projectUUID string, metadata domain.Metadata) error
    ListBySession(ctx context.Context, sessionID string, projectUUID string, limit, offset int) ([]*domain.Message, int, error)
    SoftDeleteBySession(ctx context.Context, sessionID string, projectUUID string) error
}

// Inter-service clients
type ThreadClient interface {
    UpsertSession(ctx context.Context, sessionID string, userID *string) (*dto.SessionInfo, error)
    GetSession(ctx context.Context, sessionID string) (*dto.SessionInfo, error)
}

type SearchClient interface {
    GetRelevantFacts(ctx context.Context, groupID string, queryMessages []string, maxFacts int) ([]domain.Fact, error)
}

type GraphEventPublisher interface {
    PublishMessagesIngested(ctx context.Context, event domain.MessagesIngested) error
}
```

### 4.2 PutMemory Use Case (Critical Path)

```go
func (uc *PutMemoryUseCase) Execute(ctx context.Context, req dto.PutMemoryRequest) error {
    projectUUID := tenant.FromContext(ctx).ProjectUUID
    
    // 1. Upsert session (create if not exists)
    sessionInfo, err := uc.threadClient.UpsertSession(ctx, req.SessionID, nil)
    if err != nil {
        return err
    }
    
    // 2. Check session.EndedAt → reject if ended
    if sessionInfo.IsEnded {
        return domain.ErrSessionEnded
    }
    
    // 3. Build message entities
    messages := make([]*domain.Message, len(req.Messages))
    for i, m := range req.Messages {
        messages[i] = &domain.Message{
            UUID:        uuid.NewString(),
            SessionID:   req.SessionID,
            ProjectUUID: projectUUID,
            Role:        m.Role,
            RoleType:    domain.RoleType(m.RoleType),
            Content:     m.Content,
            Metadata:    domain.Metadata(m.Metadata),
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        }
    }
    
    // 4. Batch INSERT messages → PostgreSQL
    if err := uc.messageRepo.CreateMany(ctx, messages); err != nil {
        return err
    }
    
    // 5. Publish async event → NATS → zep-graph (10-20s processing)
    uc.publisher.PublishMessagesIngested(ctx, domain.MessagesIngested{
        SessionID:   req.SessionID,
        UserID:      sessionInfo.UserID,
        ProjectUUID: projectUUID,
        Messages:    messages,
        AddPrefix:   true,
        Timestamp:   time.Now(),
    })
    
    return nil
}
```

### 4.3 GetMemory Use Case (Context Assembly)

```go
func (uc *GetMemoryUseCase) Execute(ctx context.Context, req dto.GetMemoryRequest) (*dto.MemoryResponse, error) {
    projectUUID := tenant.FromContext(ctx).ProjectUUID
    
    // 1. Fetch last max(N, 4) messages
    fetchN := max(req.LastN, 4)
    messages, err := uc.messageRepo.GetLastN(ctx, req.SessionID, projectUUID, fetchN)
    if err != nil {
        return nil, err
    }
    
    // 2. Determine groupID = user_id ?? session_id
    session, err := uc.threadClient.GetSession(ctx, req.SessionID)
    if err != nil {
        return nil, err
    }
    groupID := req.SessionID
    if session.UserID != nil {
        groupID = *session.UserID
    }
    
    // 3. Get relevant facts from knowledge graph (via zep-search)
    last4Contents := extractLast4Contents(messages)
    facts, err := uc.searchClient.GetRelevantFacts(ctx, groupID, last4Contents, 5)
    if err != nil {
        // Graceful degradation: return messages without facts
        facts = nil
    }
    
    // 4. Assemble Memory response
    return &dto.MemoryResponse{
        Messages:      dto.FromMessages(messages[:min(len(messages), req.LastN)]),
        RelevantFacts: dto.FromFacts(facts),
    }, nil
}
```

---

## 5. gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.memory.v1;

service MemoryService {
  rpc PutMemory(PutMemoryRequest) returns (google.protobuf.Empty);
  rpc GetMemory(GetMemoryRequest) returns (MemoryResponse);
  rpc DeleteMemory(DeleteMemoryRequest) returns (google.protobuf.Empty);
  rpc GetMessagesForSession(GetMessagesRequest) returns (MessageListResponse);
  rpc GetMessage(GetMessageRequest) returns (MessageResponse);
  rpc UpdateMessageMetadata(UpdateMessageMetadataRequest) returns (MessageResponse);
  rpc GetUserContext(GetUserContextRequest) returns (UserContextResponse);
}

message PutMemoryRequest {
  string session_id = 1;
  repeated MessageInput messages = 2;
}

message MessageInput {
  string role = 1;          // "user" | "assistant" | "system"
  string role_type = 2;     // enum
  string content = 3;
  google.protobuf.Struct metadata = 4;
}

message GetMemoryRequest {
  string session_id = 1;
  int32 last_n = 2;        // number of recent messages to return
}

message MemoryResponse {
  repeated MessageResponse messages = 1;
  repeated FactResponse relevant_facts = 2;
  google.protobuf.Struct metadata = 3;
}

message FactResponse {
  string uuid = 1;
  string name = 2;
  string fact = 3;
  google.protobuf.Timestamp created_at = 4;
  optional google.protobuf.Timestamp valid_at = 5;
  optional google.protobuf.Timestamp invalid_at = 6;
  optional google.protobuf.Timestamp expired_at = 7;
}

message MessageResponse {
  string uuid = 1;
  string session_id = 2;
  string role = 3;
  string role_type = 4;
  string content = 5;
  int32 token_count = 6;
  google.protobuf.Struct metadata = 7;
  google.protobuf.Timestamp created_at = 8;
}

message GetUserContextRequest {
  string thread_id = 1;
  optional string template_id = 2;  // custom context template
}

message UserContextResponse {
  string context = 1;              // pre-formatted context string for LLM
  repeated FactResponse facts = 2;
}
```

---

## 6. PostgreSQL Schema

```sql
CREATE TYPE role_type_enum AS ENUM (
    'norole', 'system', 'assistant', 'user', 'function', 'tool'
);

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
    deleted_at   TIMESTAMPTZ,
    
    FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE
);

CREATE INDEX memstore_session_id_idx ON messages(session_id) WHERE deleted_at IS NULL;
CREATE INDEX memstore_id_idx ON messages(uuid) WHERE deleted_at IS NULL;
CREATE INDEX memstore_composite_idx ON messages(session_id, project_uuid, deleted_at);
CREATE INDEX memstore_created_at_idx ON messages(created_at DESC) WHERE deleted_at IS NULL;
```

---

## 7. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.memory.messages.ingested` | `{session_id, user_id, project_uuid, messages[], add_prefix}` | zep-graph (async entity extraction, 10-20s) |
| `zep.memory.deleted` | `{session_id, project_uuid}` | zep-graph (cleanup related episodes) |

### Event Payload: MessagesIngested

```go
type MessagesIngested struct {
    SessionID   string
    UserID      *string        // if user linked, also extract to user's graph
    ProjectUUID string
    Messages    []*Message
    AddPrefix   bool           // prefix episode UUIDs with groupID
    Timestamp   time.Time
}
```

---

## 8. Inter-Service Dependencies

```
zep-memory
  ├── → zep-thread (gRPC)    # UpsertSession, GetSession
  ├── → zep-search (gRPC)    # GetRelevantFacts (for GetMemory)
  └── → zep-graph  (NATS)    # PublishMessagesIngested (async)
```

---

## 9. Configuration

```yaml
memory:
  grpc:
    port: 9043
  health:
    port: 9143
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 15
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  clients:
    thread: "zep-thread:9042"
    search: "zep-search:9045"
  context:
    default_last_n: 10
    max_facts: 5
    min_messages_for_facts: 4   # minimum messages to use as search context
  telemetry:
    service_name: "zep-memory"
    otel_endpoint: "otel-collector:4317"
```
