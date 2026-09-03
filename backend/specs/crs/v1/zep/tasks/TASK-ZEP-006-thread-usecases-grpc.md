# TASK-ZEP-006 — services/zep-thread: Use Cases & gRPC Server

**Task ID:** TASK-ZEP-006  
**Wave:** 2 (Core CRUD)  
**Solution:** [SOL-ZEP-001](../solutions/SOL-ZEP-001-Thread-Session-Management.md)  
**Depends on:** TASK-ZEP-005 (thread domain + schema)  
**Ước tính:** 3h  
**Priority:** Critical

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-thread: 6 .go - thread usecases + gRPC server  
---

## Mục tiêu

Implement tất cả 9 use cases cho Thread Service và wire gRPC server:
- CreateThread, GetThread, UpdateThread, EndThread, DeleteThread
- ListThreads, ListUserThreads, SearchThreads, UpsertSession

---

## Công việc cụ thể

### 1. Tạo 9 Use Case files trong `services/zep-thread/internal/usecase/`

#### `create_thread.go`
```go
// CreateThreadUseCase: tạo session mới
// Input: SessionID, ProjectUUID, UserID (optional), Metadata
// Returns: *Session
// Error: ErrSessionAlreadyExists nếu sessionID+projectUUID đã tồn tại
```

#### `get_thread.go`
```go
// GetThreadUseCase: lấy session theo sessionID
// Input: SessionID, ProjectUUID
// Returns: *Session
// Error: ErrSessionNotFound nếu không tồn tại hoặc đã bị soft delete
```

#### `update_thread.go`
```go
// UpdateThreadUseCase: JSONB merge-patch metadata (advisory lock)
// Input: SessionID, ProjectUUID, MetadataPatch map[string]any
// Sử dụng MergePatchMetadata (không replace toàn bộ)
// Error: ErrSessionNotFound, ErrSessionDeleted
```

#### `end_thread.go`
```go
// EndThreadUseCase: set ended_at = now()
// Input: SessionID, ProjectUUID
// Error: ErrSessionNotFound, ErrSessionAlreadyEnded, ErrSessionDeleted
// Guard: nếu IsEnded() → return ErrSessionAlreadyEnded
```

#### `delete_thread.go`
```go
// DeleteThreadUseCase: soft delete (set deleted_at = now())
// Input: SessionID, ProjectUUID
// Error: ErrSessionNotFound
// Note: đây là soft delete, data vẫn còn trong DB
```

#### `list_threads.go`
```go
// ListThreadsUseCase: list sessions với pagination
// Input: ProjectUUID, Limit (default 50, max 200), Offset
// Returns: []*Session, total int
```

#### `list_user_threads.go`
```go
// ListUserThreadsUseCase: list sessions của một user
// Input: UserID, ProjectUUID
// Returns: []*Session (ordered by created_at DESC)
```

#### `search_threads.go`
```go
// SearchThreadsUseCase: full-text search trên metadata
// Input: ProjectUUID, Query string
// Returns: []*Session (relevance-ordered)
```

#### `upsert_session.go`
```go
// UpsertSessionUseCase: create-or-get (used by Memory Service)
// Input: SessionID, ProjectUUID, UserID (optional)
// Returns: *Session (created or existing)
// Note: atomic via INSERT ... ON CONFLICT DO UPDATE
```

### 2. Tạo proto file `proto/zep/thread/v1/thread.proto`

```protobuf
syntax = "proto3";
package zep.thread.v1;

service ThreadService {
    rpc CreateThread(CreateThreadRequest) returns (Session);
    rpc GetThread(GetThreadRequest) returns (Session);
    rpc UpdateThread(UpdateThreadRequest) returns (Session);
    rpc EndThread(EndThreadRequest) returns (google.protobuf.Empty);
    rpc DeleteThread(DeleteThreadRequest) returns (google.protobuf.Empty);
    rpc ListThreads(ListThreadsRequest) returns (ListThreadsResponse);
    rpc ListUserThreads(ListUserThreadsRequest) returns (ListThreadsResponse);
    rpc SearchThreads(SearchThreadsRequest) returns (ListThreadsResponse);
    rpc UpsertSession(UpsertSessionRequest) returns (Session);
}

message Session {
    string uuid = 1;
    string session_id = 2;
    optional string user_id = 3;
    string project_uuid = 4;
    google.protobuf.Struct metadata = 5;
    optional google.protobuf.Timestamp ended_at = 6;
    google.protobuf.Timestamp created_at = 7;
    google.protobuf.Timestamp updated_at = 8;
    optional google.protobuf.Timestamp deleted_at = 9;
}
```

### 3. Tạo `services/zep-thread/internal/adapter/grpc/thread_server.go`

Implement `ThreadServiceServer` interface từ generated protobuf:
- Ánh xạ domain → protobuf messages
- Error mapping: `ErrSessionNotFound` → `codes.NotFound`, `ErrSessionAlreadyEnded` → `codes.FailedPrecondition`

### 4. Tạo `services/zep-thread/internal/usecase/end_thread_test.go`

Test cases:
- `TestEndThread_Success`: active session → set ended_at
- `TestEndThread_AlreadyEnded`: ended session → ErrSessionAlreadyEnded
- `TestEndThread_NotFound`: nonexistent → ErrSessionNotFound
- `TestUpsertSession_Idempotent`: upsert twice → same session returned

---

## Acceptance Criteria

- [ ] `go build ./services/zep-thread/...` không có lỗi
- [ ] `protoc` generate code không có lỗi
- [ ] EndThread trên ended session → `codes.FailedPrecondition`
- [ ] UpsertSession 2 lần → same UUID
- [ ] UpdateThread dùng advisory lock (không race condition)
- [ ] `go test ./services/zep-thread/...` pass

---

## Files tạo ra

```
services/zep-thread/
├── internal/
│   ├── usecase/
│   │   ├── create_thread.go
│   │   ├── get_thread.go
│   │   ├── update_thread.go
│   │   ├── end_thread.go
│   │   ├── end_thread_test.go
│   │   ├── delete_thread.go
│   │   ├── list_threads.go
│   │   ├── list_user_threads.go
│   │   ├── search_threads.go
│   │   └── upsert_session.go
│   └── adapter/
│       └── grpc/
│           └── thread_server.go
└── proto/
    └── thread.proto
```

## Sau khi hoàn thành

Chạy: `go build ./services/zep-thread/... && go test ./services/zep-thread/...`
