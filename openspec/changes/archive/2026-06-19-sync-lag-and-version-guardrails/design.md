# Design: Sync Lag Classification và Version Guardrails

## Current Behavior

| Concern | Current State | Gap |
|---|---|---|
| Lag classification | Mọi version mismatch đều là "drift" | Không phân biệt in-flight vs stuck |
| Graph/vector lag independence | `projectNode` gọi cả hai tuần tự, một thất bại → retry cả hai | Không có per-replica lag timestamp |
| `SyncETAMs` | Field tồn tại trong type, luôn trả `0` | Caller không biết khi nào có thể đọc nhất quán |
| Reconcile granularity | `GraphDriftCount` + `VectorDriftCount` gộp tất cả dạng lỗi | Không phân biệt payload mismatch vs version lag |

## Architecture

```
Write Path
  PostgreSQL (source of truth)
      │  outbox events
      ▼
  workers.Runtime.PollOnce()
      ├─► projectNode()
      │       ├─► GraphAdapter.UpsertNode   → cập nhật LastGraphSyncedAt
      │       └─► VectorAdapter.Upsert     → cập nhật LastVectorSyncedAt
      └─► updateProjectionVersion()
              └─► UpsertProjectionVersion(record với timestamps)

Reconcile Path
  workers.Runtime.Reconcile()
      ├─► compare sourceVersion vs GraphVersion
      │       └─► classify: IN_FLIGHT | LAGGING | STUCK | MISMATCH
      └─► compare sourceVersion vs VectorVersion
              └─► classify: IN_FLIGHT | LAGGING | STUCK | MISMATCH

  Chỉ STUCK + MISMATCH → DriftCount
  IN_FLIGHT → bỏ qua (không phải lỗi)
  LAGGING → LaggingCount (cảnh báo)
```

## Key Design Decisions

### 1. `SyncLagClass` — phân loại trạng thái lag

```go
type SyncLagClass string

const (
    SyncLagClassInFlight SyncLagClass = "IN_FLIGHT" // expected, within tolerance
    SyncLagClassLagging  SyncLagClass = "LAGGING"   // older than tolerance, not yet stuck
    SyncLagClassStuck    SyncLagClass = "STUCK"      // retry exhausted or deadline missed
    SyncLagClassSynced   SyncLagClass = "SYNCED"     // version matches
)
```

**Classification logic** (applied per-replica per-entity):

```
if graphVersion == sourceVersion → SYNCED
else if event.RetryCount >= maxRetries → STUCK
else if now - event.CreatedAt > lagToleranceWindow → LAGGING
else → IN_FLIGHT
```

`lagToleranceWindow` defaults to `30s`; configurable via `SYNC_LAG_TOLERANCE_MS` env var.

### 2. Mở rộng `ProjectionVersionRecord`

```go
type ProjectionVersionRecord struct {
    EntityID           string    `json:"entity_id"`
    EntityKind         string    `json:"entity_kind"`       // "kg_node" | "kg_relationship"
    SourceVersion      int64     `json:"source_version"`
    SourceEventID      string    `json:"source_event_id"`
    SourceUpdatedAt    time.Time `json:"source_updated_at"`
    GraphBackend       string    `json:"graph_backend"`
    GraphVersion       int64     `json:"graph_version"`
    LastGraphSyncedAt  time.Time `json:"last_graph_synced_at"`  // NEW
    VectorBackend      string    `json:"vector_backend"`
    VectorVersion      int64     `json:"vector_version"`
    LastVectorSyncedAt time.Time `json:"last_vector_synced_at"` // NEW
}
```

`LastGraphSyncedAt` được cập nhật chỉ khi `GraphAdapter.UpsertNode` thành công.
`LastVectorSyncedAt` được cập nhật chỉ khi `VectorAdapter.Upsert` thành công.

Điều này cho phép `Reconcile` tính:
```
graphLagSeconds = now - record.LastGraphSyncedAt
```

### 3. Mở rộng `ReconciliationReport`

```go
type ReconciliationReport struct {
    GraphDriftCount              int                   `json:"graph_drift_count"`
    VectorDriftCount             int                   `json:"vector_drift_count"`
    ProjectionVersionDriftCount  int                   `json:"projection_version_drift_count"`
    GraphLaggingCount            int                   `json:"graph_lagging_count"`   // NEW
    VectorLaggingCount           int                   `json:"vector_lagging_count"`  // NEW
    GraphInFlightCount           int                   `json:"graph_in_flight_count"` // NEW
    VectorInFlightCount          int                   `json:"vector_in_flight_count"`// NEW
    Issues                       []ReconciliationIssue `json:"issues"`
    Overall                      string                `json:"overall"` // "pass" | "warn" | "fail"
}
```

`Overall` mở rộng để có `"warn"` khi có LaggingCount > 0 nhưng không có DriftCount.

`ReconciliationIssue.Kind` mở rộng thêm:
- `"graph_lag_lagging"` — graph version lag nhưng trong ngưỡng cảnh báo
- `"vector_lag_lagging"` — vector version lag nhưng trong ngưỡng cảnh báo
- `"graph_lag_stuck"` — graph version lag và đã vượt threshold (đây là drift thực)
- `"vector_lag_stuck"` — vector version lag và đã vượt threshold

### 4. Phân loại trong `Reconcile()`

Reconcile cần biết trạng thái outbox event để phân loại. Luồng hiện tại:

1. `Reconcile()` gọi `r.store.ListProjectionVersions()` để lấy ledger
2. Với mỗi entity có `GraphVersion < SourceVersion`:
   - Tìm outbox event liên quan qua `SourceEventID` trong ledger
   - Nếu event không tồn tại → `STUCK` (event bị mất)
   - Nếu event.RetryCount >= maxRetries → `STUCK`
   - Nếu `now - event.CreatedAt > lagToleranceWindow` → `LAGGING`
   - Ngược lại → `IN_FLIGHT`

Repository interface cần thêm:
```go
type Repository interface {
    // ... existing methods
    GetOutboxEventByID(id string) (write.OutboxEvent, bool) // NEW — dùng để classify lag
}
```

### 5. `SyncETAMs` dựa trên domain lag median

```go
// Sau khi write node, service tính SyncETAMs:
func estimateSyncETA(domainID string, ledger []ProjectionVersionRecord, defaultEtaMs int) int {
    var lags []float64
    for _, r := range ledger {
        if r.EntityKind == "kg_node" {
            // ... filter by domain from node record if available
            if !r.LastGraphSyncedAt.IsZero() && !r.SourceUpdatedAt.IsZero() {
                lag := r.LastGraphSyncedAt.Sub(r.SourceUpdatedAt).Milliseconds()
                if lag > 0 {
                    lags = append(lags, float64(lag))
                }
            }
        }
    }
    if len(lags) == 0 {
        return defaultEtaMs
    }
    return int(median(lags) * 1.5) // p50 * 1.5 safety margin
}
```

Giá trị này được điền vào `NodeCreateResponse.SyncETAMs` sau write thành công.
`defaultEtaMs` lấy từ `SYNC_ETA_DEFAULT_MS` env var (default: 5000ms).

### 6. Per-entity sync status

Thêm method vào `workers.Runtime`:
```go
func (r *Runtime) EntitySyncStatus(entityID, entityKind string) EntitySyncStatus
```

```go
type EntitySyncStatus struct {
    EntityID         string       `json:"entity_id"`
    EntityKind       string       `json:"entity_kind"`
    SourceVersion    int64        `json:"source_version"`
    GraphVersion     int64        `json:"graph_version"`
    GraphLagClass    SyncLagClass `json:"graph_lag_class"`
    LastGraphSyncedAt time.Time   `json:"last_graph_synced_at,omitempty"`
    VectorVersion    int64        `json:"vector_version"`
    VectorLagClass   SyncLagClass `json:"vector_lag_class"`
    LastVectorSyncedAt time.Time  `json:"last_vector_synced_at,omitempty"`
}
```

Admin MCP tool có thể gọi `EntitySyncStatus` để kiểm tra trạng thái sync của một entity cụ thể.

## Configuration

| Env Var | Default | Description |
|---|---|---|
| `SYNC_LAG_TOLERANCE_MS` | `30000` | Window (ms) trong đó version lag được coi là IN_FLIGHT |
| `SYNC_LAG_STUCK_RETRIES` | `3` | Số retry tối đa trước khi classify là STUCK (bằng `maxRetries`) |
| `SYNC_ETA_DEFAULT_MS` | `5000` | Default SyncETAMs khi domain chưa có lịch sử |

## Migration Notes

- `ProjectionVersionRecord` thêm hai field nullable: `LastGraphSyncedAt`, `LastVectorSyncedAt`
- Các record cũ sẽ có zero value — reconcile coi zero timestamp là "never synced" (STUCK nếu version mismatch)
- Không cần migration data; worker sẽ populate timestamps dần khi xử lý event mới
- `MemoryStore.UpsertProjectionVersion` cần lưu hai timestamp mới

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Lag tolerance window quá rộng che giấu lỗi thực | Expose `SYNC_LAG_TOLERANCE_MS` để operator tune; default 30s là an toàn cho outbox polling 1s |
| `GetOutboxEventByID` thêm query load vào reconcile | Cache event lookup trong Reconcile bằng map from `ListOutboxEvents()` — không cần query riêng |
| `SyncETAMs` median không chính xác khi domain ít data | Fallback về `SYNC_ETA_DEFAULT_MS` khi sample size < 5 |
| Reconcile phân loại sai do event bị purge khỏi outbox | Event với status DONE bị purge → không tìm thấy → dùng `LastGraphSyncedAt` để check: nếu gần đây → không phải STUCK |
