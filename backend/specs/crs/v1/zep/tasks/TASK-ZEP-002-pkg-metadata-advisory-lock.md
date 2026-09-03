# TASK-ZEP-002 — pkg/metadata: PostgreSQL Advisory Lock (Shared)

**Task ID:** TASK-ZEP-002  
**Wave:** 1 (Foundation)  
**Solution:** [SOL-ZEP-009](../solutions/SOL-ZEP-009-Resilience-Observability.md)  
**Depends on:** TASK-ZEP-001 (pkg/resilience)  
**Ước tính:** 2h  
**Priority:** Critical — dùng bởi Thread Service, Memory Service, Admin Service

**Trạng thái:** ✅ Implemented  
**Ghi chú:** zep-thread: 6 .go - metadata + advisory lock implemented  
---

## Mục tiêu

Tạo `pkg/metadata/advisory_lock.go` — shared package cho PostgreSQL session-level advisory locks dùng trong concurrent JSONB metadata update. Không còn mỗi service phải tự implement.

---

## Input Context

- **Pattern:** `pg_advisory_lock(int64)` → `UPDATE SET metadata = metadata || $patch::jsonb` → `pg_advisory_unlock(int64)`
- **Lock key generation:** SHA-256 của string ID → first 8 bytes → int64
- **Target path:** `pkg/metadata/`
- **Used by:** Thread Service (session metadata), Memory Service (message metadata), Admin Service

---

## Công việc cụ thể

### 1. Tạo `pkg/metadata/advisory_lock.go`

```go
package metadata

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "encoding/binary"
    "encoding/json"
    "fmt"
)

// AdvisoryLockKey chuyển đổi bất kỳ string ID nào thành int64
// dùng làm PostgreSQL advisory lock key.
// Sử dụng SHA-256 first 8 bytes → collision probability: 1/(2^64)
//
// Example:
//   AdvisoryLockKey("session_abc123") → stable int64
func AdvisoryLockKey(id string) int64 {
    hash := sha256.Sum256([]byte(id))
    return int64(binary.BigEndian.Uint64(hash[:8]))
}

// WithAdvisoryLock acquires a PostgreSQL session-level advisory lock,
// executes fn trong một transaction, rồi release lock.
//
// Lock pattern:
//   SELECT pg_advisory_lock($1)     -- acquire
//   BEGIN
//     fn(tx)                        -- execute
//   COMMIT
//   SELECT pg_advisory_unlock($1)   -- release (deferred)
//
// NOTE: Lock là session-level, sẽ tự release khi connection đóng
func WithAdvisoryLock(ctx context.Context, db *sql.DB, lockID string, fn func(*sql.Tx) error) error { ... }

// MergeJSONBMetadata thực hiện JSONB merge-patch under advisory lock.
// SQL: UPDATE {table} SET metadata = metadata || $patch::jsonb, updated_at = NOW()
//      WHERE {idColumn} = $id AND deleted_at IS NULL
//
// Parameters:
//   table    — tên bảng (e.g. "sessions", "messages")
//   idColumn — tên cột ID (e.g. "session_id", "uuid")
//   id       — giá trị ID (dùng làm lock key)
//   patch    — JSONB patch map (sẽ merge, không replace toàn bộ)
func MergeJSONBMetadata(ctx context.Context, db *sql.DB, table, idColumn, id string, patch map[string]any) error { ... }
```

### 2. Tạo `pkg/metadata/advisory_lock_test.go`

Test cases (dùng `testcontainers/postgres` hoặc mock `*sql.DB`):
- `TestAdvisoryLockKey_DeterministicOutput`: cùng input → cùng output
- `TestAdvisoryLockKey_DifferentInputsDifferentKeys`: "session_a" ≠ "session_b"
- `TestWithAdvisoryLock_SerializesAccess`: 10 goroutines concurrent update → không race condition
- `TestWithAdvisoryLock_RollbackOnError`: fn returns error → transaction rolled back
- `TestMergeJSONBMetadata_MergesNotReplaces`: PATCH {"a":1} + PATCH {"b":2} → {"a":1,"b":2}
- `TestMergeJSONBMetadata_OverwritesExistingKey`: PATCH {"a":2} overwrites {"a":1}

---

## Acceptance Criteria

- [ ] `go build ./pkg/metadata/...` không có lỗi
- [ ] `go test ./pkg/metadata/...` 100% pass
- [ ] `AdvisoryLockKey("x")` luôn trả về cùng giá trị (deterministic)
- [ ] Concurrent `MergeJSONBMetadata` với cùng lockID → chỉ 1 goroutine chạy tại một thời điểm
- [ ] JSONB merge: `{"a":1}` || `{"b":2}` → `{"a":1,"b":2}` (không mất key cũ)
- [ ] Lock bị release kể cả khi fn panic (defer)

---

## Files tạo ra

```
pkg/metadata/
├── advisory_lock.go
└── advisory_lock_test.go
```

## Sau khi hoàn thành

Chạy: `go build ./pkg/metadata/... && go test ./pkg/metadata/...`
