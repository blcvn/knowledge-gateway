# TASK-ZEP-009 — services/memory-service: GetMemory & GetUserContext

**Task ID:** TASK-ZEP-009  
**Wave:** 3 (Memory Core)  
**Solution:** [SOL-ZEP-002](../solutions/SOL-ZEP-002-Memory-Message-Context-Assembly.md)  
**Depends on:** TASK-ZEP-008 (PutMemory + message schema)  
**Ước tính:** 3h  
**Priority:** Critical — LLM context injection

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-memory: 16 .go - GetContext + assembly  
---

## Mục tiêu

Implement 3 read use cases cho Memory Service:
1. **GetMemory** — messages + facts với graceful degradation (graph có thể down)
2. **GetUserContext** — pre-formatted string để inject vào LLM system prompt
3. **UpdateMessageMetadata** — JSONB merge-patch với advisory lock
4. **DeleteSessionMemory** — xóa toàn bộ messages của session

---

## Công việc cụ thể

### 1. Tạo `GetMemory` Use Case

**`services/memory-service/internal/usecase/zep/get_memory.go`**

```go
// GetMemoryRequest params
type GetMemoryRequest struct {
    SessionID   string
    ProjectUUID string
    LastN       int    // số messages gần nhất, default 10, min 4
}

// GetMemoryUseCase assembly logic:
// 1. Fetch last N messages từ PostgreSQL (luôn thành công)
// 2. Resolve groupID: userID nếu có, else sessionID
// 3. Get relevant facts từ Search Service (timeout 100ms)
//    → Nếu search down: log warning, facts = [] (KHÔNG return error)
// 4. Trả về Memory{Messages, RelevantFacts, Metadata}
//
// CRITICAL: Graph/Search service down → vẫn trả về messages bình thường
func (uc *GetMemoryUseCase) Execute(ctx context.Context, req GetMemoryRequest) (*Memory, error) { ... }

// Memory là composite response (assembled at read time, NOT persisted)
type Memory struct {
    Messages      []Message
    RelevantFacts []Fact      // empty slice (NOT nil) nếu graph down
    Metadata      map[string]any
}
```

**Graceful degradation pattern:**
```go
// Timeout 100ms để không block main response
factsCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
defer cancel()
facts, err := uc.searchClient.GetRelevantFacts(factsCtx, ...)
if err != nil {
    slog.Warn("graph unavailable, returning messages only", "error", err)
    facts = []Fact{}  // empty, NOT nil
}
```

### 2. Tạo `GetUserContext` Use Case

**`services/memory-service/internal/usecase/zep/get_user_context.go`**

```go
// GetUserContextUseCase:
// 1. Get user's facts từ Search Service (up to 20 facts)
// 2. Get recent messages (last 5) từ thread nếu thread_id provided
// 3. Format thành string cho LLM injection
//
// Output format:
// "FACTS about this user:
//  - Alice works at Beta Inc (valid from: 2023-07)
//  - Alice lives in Hanoi
//
// RECENT MESSAGES:
// [user]: What's the weather?
// [assistant]: ..."

func (uc *GetUserContextUseCase) Execute(ctx context.Context, req GetUserContextRequest) (*UserContextResponse, error) { ... }

type UserContextResponse struct {
    Context string  // formatted string, inject vào system prompt
    Facts   []Fact  // raw facts cho programmatic access
}

// buildContextBlock format context từ facts + messages
func buildContextBlock(facts []Fact, messages []Message) string { ... }
```

### 3. Tạo `UpdateMessageMetadata` Use Case

```go
// Dùng pkg/metadata.MergeJSONBMetadata với advisory lock
// Không thể replace entire metadata, chỉ merge-patch
func (uc *UpdateMessageMetadataUseCase) Execute(ctx context.Context, messageUUID string, patch map[string]any) error { ... }
```

### 4. Tạo `DeleteSessionMemory` Use Case

```go
// Soft delete tất cả messages của session (set deleted_at = now())
// Không xóa session entity (chỉ xóa messages)
func (uc *DeleteSessionMemoryUseCase) Execute(ctx context.Context, sessionID, projectUUID string) error { ... }
```

### 5. Mở rộng `MessageRepository`

```go
// services/memory-service/internal/infra/postgres/message_repo.go
// Thêm:
func (r *MessageRepo) GetLastN(ctx context.Context, sessionID, projectUUID string, n int) ([]Message, error) {
    // SELECT * FROM messages WHERE session_id=$1 AND project_uuid=$2 AND deleted_at IS NULL
    // ORDER BY created_at DESC LIMIT $3
}
```

### 6. Tests

- `TestGetMemory_SearchDown_ReturnsMessages`: mock search client returns error → Memory.RelevantFacts = []
- `TestGetMemory_LastN_Minimum4`: lastN=1 → trả về 4 messages (minimum)
- `TestGetUserContext_FormatOutput`: facts + messages → formatted string đúng format
- `TestGetUserContext_SearchDown_EmptyContext`: graceful degradation
- `TestGetMemory_LastN_Correct`: lastN=10 → 10 messages (DESC order)
- `TestBuildContextBlock_WithTemporalFacts`: fact với valid_at → includes "(valid from: 2023-07)"

---

## API Endpoints (Gateway)

```
GET    /api/v2/sessions/{id}/memory              → GetMemory (messages + facts)
DELETE /api/v2/sessions/{id}/memory              → DeleteSessionMemory
GET    /api/v2/sessions/{id}/messages            → ListMessages (paginated)
GET    /api/v2/sessions/{id}/messages/{uuid}     → GetMessage
PATCH  /api/v2/sessions/{id}/messages/{uuid}     → UpdateMessageMetadata
GET    /api/v2/users/{id}/context                → GetUserContext
```

**Query params cho GetMemory:**
```
GET /api/v2/sessions/{id}/memory?lastN=10
```

---

## Acceptance Criteria

- [ ] `go build ./services/memory-service/...` không có lỗi
- [ ] Search service down → GetMemory trả về messages với `relevant_facts: []` (không crash)
- [ ] lastN=1 → 4 messages (minimum enforced)
- [ ] `GetUserContext` output bắt đầu bằng "FACTS about this user:" nếu có facts
- [ ] Fact có valid_at → output chứa "(valid from: YYYY-MM)"
- [ ] UpdateMessageMetadata dùng advisory lock (no race condition)
- [ ] `go test ./services/memory-service/...` pass

---

## Files tạo ra

```
services/memory-service/internal/usecase/zep/
├── get_memory.go
├── get_memory_test.go
├── get_user_context.go
├── get_user_context_test.go
├── update_message_metadata.go
└── delete_session_memory.go
```

## Sau khi hoàn thành

Chạy: `go build ./services/memory-service/... && go test ./services/memory-service/...`
