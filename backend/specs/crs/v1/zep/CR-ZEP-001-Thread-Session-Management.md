# Change Request: CR-ZEP-001 — Conversation Thread & Session Management

**CR ID:** CR-ZEP-001  
**Component:** `services/thread-service` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** Zep PRD §6.1 F1, SRS §3.1 Session, specs/services/03-thread-service.md  
**Target latency:** < 200ms (p95)

---

## 1. Mô tả

Xây dựng **Thread Service (zep-thread)** — quản lý vòng đời conversation sessions (threads) trong VNP Memory:

1. **Thread CRUD**: Tạo, đọc, cập nhật thread với metadata JSONB.
2. **Session Lifecycle**: Hỗ trợ `ended_at` — khi thread kết thúc, không thể thêm messages mới.
3. **User-Thread Association**: Liên kết thread với user để context được cá nhân hóa.
4. **Multi-Tenant Isolation**: `project_uuid` trên tất cả entities.
5. **Session Search**: Tìm kiếm theo session để tra cứu lịch sử conversation.
6. **Soft Delete**: `deleted_at` timestamp thay vì hard delete.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện chưa có khái niệm **Thread/Session** riêng biệt được quản lý với lifecycle rõ ràng.
- Thiếu cơ chế `ended_at` để đánh dấu session đã kết thúc (ngăn thêm messages sau khi kết thúc).
- Chưa có User-Thread association cho phép query cross-session theo user.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/zep-thread/` (Port gRPC: 9042)

### 3.2. Domain Model

```go
type Session struct {
    UUID        string
    SessionID   string          // user-provided unique ID
    UserID      *string         // optional: link to a user
    ProjectUUID string          // multi-tenant scope
    Metadata    map[string]any  // JSONB free-form metadata
    EndedAt     *time.Time      // nil = active, non-nil = ended
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   *time.Time      // soft delete
}
```

### 3.3. PostgreSQL Schema

```sql
CREATE TABLE sessions (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   VARCHAR UNIQUE NOT NULL,
    user_id      VARCHAR REFERENCES users(user_id),
    project_uuid UUID NOT NULL,
    metadata     JSONB DEFAULT '{}',
    ended_at     TIMESTAMPTZ,   -- marks session closed
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX session_user_id_idx ON sessions(user_id);
CREATE UNIQUE INDEX ON sessions(session_id, project_uuid, deleted_at);
```

### 3.4. Use Cases

| Use Case | Mô tả |
|----------|-------|
| `CreateThread` | Tạo session mới với optional `user_id` |
| `GetThread` | Lấy session by ID |
| `UpdateThread` | Cập nhật metadata (JSONB merge-patch) với advisory lock |
| `EndThread` | Set `ended_at = now()` — session đóng lại |
| `DeleteThread` | Soft delete (`deleted_at`) |
| `ListThreads` | Liệt kê threads của user hoặc toàn bộ project |
| `ListUserThreads` | Liệt kê threads của một user cụ thể |
| `SearchThreads` | Tìm kiếm threads theo metadata |
| `UpsertSession` | Create-or-get session (dùng bởi Memory Service) |

### 3.5. Concurrency Control

PostgreSQL Advisory Lock cho concurrent metadata updates:
```go
// Lock key = SHA-256(sessionID)[:8] as int64
lock := sha256.Sum256([]byte(sessionID))
lockKey := binary.BigEndian.Int64(lock[:8])
// SELECT pg_advisory_lock($1); ... UPDATE; SELECT pg_advisory_unlock($1);
// Retry: exponential backoff 200ms → 30s, max 15 retries
```

### 3.6. API Endpoints (qua Gateway)

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v2/sessions` | List all sessions |
| `POST` | `/api/v2/sessions` | Create session |
| `GET` | `/api/v2/sessions-ordered` | Paginated ordered list |
| `POST` | `/api/v2/sessions/search` | Search sessions |
| `GET` | `/api/v2/sessions/:id` | Get session |
| `PATCH` | `/api/v2/sessions/:id` | Update session metadata |
| `GET` | `/api/v2/users/:id/sessions` | List user's sessions |

---

## 4. Acceptance Criteria

- [ ] Tạo session → thêm messages → gọi `EndThread` → thử thêm message nữa → bị từ chối với lỗi `SessionEnded`.
- [ ] Tạo session với `user_id` → `GET /users/:id/sessions` trả về session đó.
- [ ] Concurrent PATCH với cùng `session_id` → advisory lock đảm bảo không có race condition trên metadata.
- [ ] Soft delete session → `GET /sessions/:id` trả về 404 (not found).
- [ ] Metadata JSONB được merge (không replace toàn bộ) khi gọi PATCH nhiều lần.
