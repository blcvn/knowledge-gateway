# sync-lag-guard

## Requirements

### Requirement: Phân loại trạng thái lag của projection

Hệ thống SHALL phân loại trạng thái đồng bộ của mỗi entity theo bốn trạng thái: `SYNCED`, `IN_FLIGHT`, `LAGGING`, `STUCK`.

- `SYNCED` — `GraphVersion == SourceVersion` (hoặc `VectorVersion == SourceVersion`)
- `IN_FLIGHT` — version chênh lệch nhưng outbox event còn trong tolerance window (chưa quá `SYNC_LAG_TOLERANCE_MS` kể từ event creation)
- `LAGGING` — version chênh lệch, event cũ hơn tolerance window, retry count < maxRetries
- `STUCK` — version chênh lệch và: event không tồn tại, hoặc retry count >= maxRetries, hoặc `LastGraphSyncedAt` / `LastVectorSyncedAt` là zero và không có event pending

#### Scenario: Worker đang xử lý event trong tolerance window

- GIVEN một node được write với `SourceVersion=5`
- AND outbox event được tạo cách đây 10 giây
- AND `SYNC_LAG_TOLERANCE_MS=30000`
- WHEN `Reconcile()` chạy
- THEN graph projection với `GraphVersion=4` SHALL được classify là `IN_FLIGHT`
- AND `GraphDriftCount` SHALL không tăng
- AND `GraphInFlightCount` SHALL tăng 1
- AND `Overall` SHALL là `"pass"` nếu không có STUCK/MISMATCH khác

#### Scenario: Event quá cũ nhưng chưa exhaust retry

- GIVEN một node có `SourceVersion=5` và `GraphVersion=4`
- AND outbox event được tạo cách đây 60 giây
- AND `SYNC_LAG_TOLERANCE_MS=30000`
- AND `event.RetryCount=1` và `maxRetries=3`
- WHEN `Reconcile()` chạy
- THEN graph lag SHALL được classify là `LAGGING`
- AND `GraphLaggingCount` SHALL tăng 1
- AND `GraphDriftCount` SHALL không tăng
- AND `Overall` SHALL là `"warn"`
- AND `issues` SHALL chứa entry với Kind `"graph_lag_lagging"` cho node đó

#### Scenario: Event đã exhaust retry — STUCK

- GIVEN một node có `SourceVersion=5` và `GraphVersion=4`
- AND outbox event có `RetryCount=3` và `maxRetries=3`
- WHEN `Reconcile()` chạy
- THEN graph lag SHALL được classify là `STUCK`
- AND `GraphDriftCount` SHALL tăng 1
- AND `issues` SHALL chứa entry với Kind `"graph_lag_stuck"` cho node đó
- AND `Overall` SHALL là `"fail"`

#### Scenario: Không tìm được outbox event — STUCK

- GIVEN một node có `SourceVersion=7` và `GraphVersion=6`
- AND không có outbox event nào với `SourceEventID` tương ứng trong store
- AND `LastGraphSyncedAt` là zero value
- WHEN `Reconcile()` chạy
- THEN graph lag SHALL được classify là `STUCK`
- AND `GraphDriftCount` SHALL tăng 1

#### Scenario: Version lag nhưng tìm thấy timestamp gần đây

- GIVEN một node có `SourceVersion=8` và `GraphVersion=7`
- AND không tìm thấy outbox event (đã purge sau khi DONE)
- AND `LastGraphSyncedAt` là 5 giây trước
- WHEN `Reconcile()` chạy
- THEN graph lag SHALL được classify là `IN_FLIGHT` (vừa sync nhưng version chưa advance)
- AND `GraphDriftCount` SHALL không tăng

---

### Requirement: `ProjectionVersionRecord` lưu timestamp per-replica

Hệ thống SHALL mở rộng `ProjectionVersionRecord` với `LastGraphSyncedAt` và `LastVectorSyncedAt`.

#### Scenario: Cập nhật `LastGraphSyncedAt` khi graph projection thành công

- WHEN `GraphAdapter.UpsertNode` trả về `nil` error
- THEN `updateProjectionVersion` SHALL ghi `LastGraphSyncedAt = time.Now().UTC()`
- AND field này SHALL độc lập với `LastVectorSyncedAt`

#### Scenario: `LastVectorSyncedAt` không cập nhật khi vector thất bại

- GIVEN `GraphAdapter.UpsertNode` thành công
- AND `VectorAdapter.Upsert` trả về error
- WHEN `projectNode` xử lý
- THEN `ProjectionVersionRecord` SHALL có `LastGraphSyncedAt` được set
- AND `LastVectorSyncedAt` SHALL vẫn là giá trị cũ (không bị cập nhật)

#### Scenario: Vector lag độc lập được phát hiện

- GIVEN một node đã sync hoàn tất lên graph (`GraphVersion == SourceVersion`)
- AND vector projection bị thất bại nhiều lần, `VectorVersion < SourceVersion`
- WHEN `Reconcile()` chạy
- THEN `GraphDriftCount` SHALL là 0
- AND `VectorDriftCount` hoặc `VectorLaggingCount` SHALL tăng theo classification
- AND `issues` SHALL chứa entry với Kind `"vector_lag_stuck"` hoặc `"vector_lag_lagging"`

---

### Requirement: `ReconciliationReport` có counters phân tách

Hệ thống SHALL báo riêng `GraphLaggingCount`, `VectorLaggingCount`, `GraphInFlightCount`, `VectorInFlightCount` trong `ReconciliationReport`.

`Overall` SHALL có ba giá trị: `"pass"` | `"warn"` | `"fail"`.

- `"pass"` — không có DriftCount nào > 0 và không có LaggingCount nào > 0
- `"warn"` — ít nhất một LaggingCount > 0 nhưng không có DriftCount > 0
- `"fail"` — ít nhất một DriftCount > 0

#### Scenario: Không có lag → Overall pass

- GIVEN mọi entity đều có `GraphVersion == SourceVersion` và `VectorVersion == SourceVersion`
- WHEN `Reconcile()` chạy
- THEN `Overall` SHALL là `"pass"`
- AND tất cả counters SHALL là 0

#### Scenario: Chỉ có LAGGING → Overall warn

- GIVEN một số entity có version lag > tolerance nhưng retry chưa exhaust
- AND không có payload mismatch
- WHEN `Reconcile()` chạy
- THEN `Overall` SHALL là `"warn"`
- AND `GraphDriftCount` SHALL là 0
- AND `GraphLaggingCount` SHALL > 0

#### Scenario: Có STUCK → Overall fail

- GIVEN ít nhất một entity có graph lag classify là STUCK
- WHEN `Reconcile()` chạy
- THEN `Overall` SHALL là `"fail"`

---

### Requirement: `NodeCreateResponse.SyncETAMs` được tính từ lag thực

Hệ thống SHALL điền `SyncETAMs` trong `NodeCreateResponse` dựa trên median lag của domain.

#### Scenario: Domain có lịch sử sync

- GIVEN domain có ít nhất 5 node đã được sync
- AND median lag của `LastGraphSyncedAt - SourceUpdatedAt` là 800ms
- WHEN một node mới được tạo
- THEN `SyncETAMs` SHALL trả về `1200` (median * 1.5, rounded up)

#### Scenario: Domain chưa có lịch sử

- GIVEN domain không có projection version records nào với timestamp đầy đủ
- WHEN một node được tạo
- THEN `SyncETAMs` SHALL trả về giá trị `SYNC_ETA_DEFAULT_MS` (default 5000)

#### Scenario: Sample size quá nhỏ (< 5)

- GIVEN domain có 3 projection version records với timestamps
- WHEN một node được tạo
- THEN `SyncETAMs` SHALL fallback về `SYNC_ETA_DEFAULT_MS`

---

### Requirement: Per-entity sync status API qua Admin MCP

Hệ thống SHALL cho phép query trạng thái sync của một entity cụ thể qua Admin MCP tool.

#### Scenario: Query entity đã sync đầy đủ

- GIVEN node `node-abc` có `SourceVersion=5`, `GraphVersion=5`, `VectorVersion=5`
- WHEN admin query `EntitySyncStatus("node-abc", "kg_node")`
- THEN response SHALL có `GraphLagClass="SYNCED"` và `VectorLagClass="SYNCED"`

#### Scenario: Query entity đang in-flight

- GIVEN node `node-xyz` có `SourceVersion=6`, `GraphVersion=5`
- AND outbox event cho node này còn trong tolerance window
- WHEN admin query `EntitySyncStatus("node-xyz", "kg_node")`
- THEN response SHALL có `GraphLagClass="IN_FLIGHT"`
- AND `LastGraphSyncedAt` SHALL không phải zero value

#### Scenario: Query entity bị stuck

- GIVEN node `node-zzz` có `SourceVersion=9`, `GraphVersion=8`
- AND outbox event có `RetryCount >= maxRetries`
- WHEN admin query `EntitySyncStatus("node-zzz", "kg_node")`
- THEN response SHALL có `GraphLagClass="STUCK"`
