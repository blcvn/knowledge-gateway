# Tasks: Sync Lag Classification và Version Guardrails

## Task 1 — Mở rộng `ProjectionVersionRecord` với per-replica timestamps

**File:** `internal/write/types.go`

Thêm hai field vào `ProjectionVersionRecord`:
```go
LastGraphSyncedAt  time.Time `json:"last_graph_synced_at"`
LastVectorSyncedAt time.Time `json:"last_vector_synced_at"`
```

**File:** `internal/write/store.go` (`MemoryStore`)

`UpsertProjectionVersion` và `GetProjectionVersion` không cần thay đổi — struct tự động mang field mới.

---

## Task 2 — Thêm `SyncLagClass` và `EntitySyncStatus` vào `workers/types.go`

**File:** `internal/workers/types.go`

Thêm:
```go
type SyncLagClass string

const (
    SyncLagClassSynced   SyncLagClass = "SYNCED"
    SyncLagClassInFlight SyncLagClass = "IN_FLIGHT"
    SyncLagClassLagging  SyncLagClass = "LAGGING"
    SyncLagClassStuck    SyncLagClass = "STUCK"
)

type EntitySyncStatus struct {
    EntityID            string       `json:"entity_id"`
    EntityKind          string       `json:"entity_kind"`
    SourceVersion       int64        `json:"source_version"`
    GraphVersion        int64        `json:"graph_version"`
    GraphLagClass       SyncLagClass `json:"graph_lag_class"`
    LastGraphSyncedAt   time.Time    `json:"last_graph_synced_at,omitempty"`
    VectorVersion       int64        `json:"vector_version"`
    VectorLagClass      SyncLagClass `json:"vector_lag_class"`
    LastVectorSyncedAt  time.Time    `json:"last_vector_synced_at,omitempty"`
}
```

Mở rộng `ReconciliationReport`:
```go
GraphLaggingCount   int `json:"graph_lagging_count"`
VectorLaggingCount  int `json:"vector_lagging_count"`
GraphInFlightCount  int `json:"graph_in_flight_count"`
VectorInFlightCount int `json:"vector_in_flight_count"`
```

`Overall` mở rộng: `"pass"` | `"warn"` | `"fail"`.

---

## Task 3 — Thêm `GetOutboxEventByID` vào `Repository` interface và `MemoryStore`

**File:** `internal/workers/runtime.go` (interface `Repository`)

```go
GetOutboxEventByID(id string) (write.OutboxEvent, bool)
```

**File:** `internal/write/store.go` (`MemoryStore`)

```go
func (s *MemoryStore) GetOutboxEventByID(id string) (write.OutboxEvent, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, e := range s.outbox {
        if e.ID == id {
            return e, true
        }
    }
    return write.OutboxEvent{}, false
}
```

---

## Task 4 — Thêm `SYNC_LAG_TOLERANCE_MS` và `SYNC_ETA_DEFAULT_MS` vào config

**File:** `internal/config/config.go` (hoặc file config hiện tại)

Thêm hai config field với giá trị default:
- `SyncLagToleranceMs int` (default: 30000)
- `SyncEtaDefaultMs   int` (default: 5000)

Truyền vào `Runtime` khi khởi tạo.

---

## Task 5 — Implement lag classification helper

**File:** `internal/workers/runtime.go`

Thêm hàm private:
```go
func classifyLag(
    replicaVersion, sourceVersion int64,
    sourceEventID string,
    lastSyncedAt time.Time,
    getEvent func(id string) (write.OutboxEvent, bool),
    maxRetries int,
    lagToleranceWindow time.Duration,
) SyncLagClass
```

Logic:
1. `replicaVersion == sourceVersion` → `SYNCED`
2. Lookup event bằng `sourceEventID` qua `getEvent`
3. Nếu event tìm thấy:
   - `event.RetryCount >= maxRetries` → `STUCK`
   - `time.Since(event.CreatedAt) > lagToleranceWindow` → `LAGGING`
   - Ngược lại → `IN_FLIGHT`
4. Nếu event không tìm thấy (đã purge):
   - `lastSyncedAt` là zero → `STUCK`
   - `time.Since(lastSyncedAt) <= lagToleranceWindow` → `IN_FLIGHT`
   - Ngược lại → `STUCK`

---

## Task 6 — Cập nhật `updateProjectionVersion` để ghi timestamps

**File:** `internal/workers/runtime.go`

Hàm `updateProjectionVersion` hiện nhận `graphVersion` và `vectorVersion`. Thêm logic:

Tách thành hai lần ghi — hoặc extend record với timestamps:
```go
func (r *Runtime) updateProjectionVersionWithTimestamps(
    event write.OutboxEvent,
    sourceVersion int,
    entityID string,
    sourceUpdatedAt time.Time,
    graphVersion, vectorVersion int64,
    graphSynced, vectorSynced bool, // NEW: liệu adapter call có thành công không
) {
    now := time.Now().UTC()
    record := write.ProjectionVersionRecord{...}
    if graphSynced {
        record.LastGraphSyncedAt = now
    }
    if vectorSynced {
        record.LastVectorSyncedAt = now
    }
    _ = writer.UpsertProjectionVersion(context.Background(), record)
}
```

Cập nhật `handleEvent` để truyền success flags từ kết quả của `projectNode`.

---

## Task 7 — Refactor `projectNode` để track per-replica success

**File:** `internal/workers/runtime.go`

`projectNode` hiện return một error duy nhất. Refactor để return hai flags:
```go
func (r *Runtime) projectNode(node write.NodeRecord) (graphSynced, vectorSynced bool, err error)
```

- `graphSynced = true` nếu `GraphAdapter.UpsertNode` thành công
- `vectorSynced = true` nếu `VectorAdapter.Upsert` thành công
- `err` vẫn là error đầu tiên gặp phải (để quyết định retry)

Truyền `graphSynced`, `vectorSynced` vào `updateProjectionVersionWithTimestamps`.

---

## Task 8 — Cập nhật `Reconcile()` để dùng lag classification

**File:** `internal/workers/runtime.go`

Trong vòng lặp `for id, source := range sourceNodes`:

Thay:
```go
if expectedVersion != 0 && graphNode.SyncVersion != expectedVersion {
    report.GraphDriftCount++
    report.Issues = append(...)
}
```

Bằng:
```go
lagClass := classifyLag(
    graphNode.SyncVersion, expectedVersion,
    ledger.SourceEventID, ledger.LastGraphSyncedAt,
    r.store.GetOutboxEventByID,
    r.maxRetries, r.lagToleranceWindow,
)
switch lagClass {
case SyncLagClassStuck:
    report.GraphDriftCount++
    report.Issues = append(report.Issues, ReconciliationIssue{..., Kind: "graph_lag_stuck"})
case SyncLagClassLagging:
    report.GraphLaggingCount++
    report.Issues = append(report.Issues, ReconciliationIssue{..., Kind: "graph_lag_lagging"})
case SyncLagClassInFlight:
    report.GraphInFlightCount++
}
```

Làm tương tự cho `VectorVersion`.

Cập nhật `Overall` logic:
```go
if report.GraphDriftCount == 0 && report.VectorDriftCount == 0 && report.ProjectionVersionDriftCount == 0 {
    if report.GraphLaggingCount > 0 || report.VectorLaggingCount > 0 {
        report.Overall = "warn"
    } else {
        report.Overall = "pass"
    }
} else {
    report.Overall = "fail"
}
```

---

## Task 9 — Implement `EntitySyncStatus()` trên `Runtime`

**File:** `internal/workers/runtime.go`

```go
func (r *Runtime) EntitySyncStatus(entityID, entityKind string) EntitySyncStatus
```

Lookup từ `r.store.GetProjectionVersion(entityID, entityKind)` + `classifyLag` cho cả graph và vector.

---

## Task 10 — Điền `SyncETAMs` trong write service

**File:** `internal/write/service.go`

Sau khi node được tạo thành công, tính `SyncETAMs`:

```go
func estimateSyncETA(store SyncETAReader, domainID string, defaultMs int) int
```

`SyncETAReader` là interface nhỏ trên `store` để lấy projection version records.

Trả về median của `LastGraphSyncedAt - SourceUpdatedAt` trong 30 records gần nhất của domain. Fallback về `defaultMs` nếu sample < 5.

---

## Task 11 — Tests

**File:** `internal/workers/runtime_test.go`

Thêm test cases:
- `TestReconcile_InFlightLagNotDrift` — entity trong tolerance window không tăng DriftCount
- `TestReconcile_LaggingRaisesWarn` — entity cũ hơn tolerance, overall = "warn"
- `TestReconcile_StuckRetryExhausted` — retry >= maxRetries → overall = "fail"
- `TestReconcile_GraphSyncedVectorLagging` — graph synced, vector lagging → GraphDriftCount=0, VectorLaggingCount>0
- `TestEntitySyncStatus_Synced` — entity đã sync đầy đủ
- `TestEntitySyncStatus_InFlight` — entity đang sync
- `TestEstimateSyncETA_WithHistory` — median tính đúng
- `TestEstimateSyncETA_FallbackDefault` — ít data → default
