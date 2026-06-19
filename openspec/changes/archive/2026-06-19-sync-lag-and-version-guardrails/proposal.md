# Proposal: Sync Lag Classification và Version Guardrails

## Problem

Cơ chế sync hiện tại dùng outbox pattern để chiếu dữ liệu từ PostgreSQL (source of truth) sang graph DB và vector store. Tuy nhiên có ba vấn đề chưa được giải quyết:

### 1. Không phân biệt "lag đang tiến hành" vs "lag bị kẹt"

`Reconcile()` hiện tại coi mọi trường hợp `graphNode.SyncVersion != expectedVersion` đều là drift — không phân biệt giữa:

- **In-flight lag**: worker đang xử lý event, version chưa cập nhật là bình thường trong vài giây
- **Stuck lag**: worker đã retry nhiều lần và thất bại, hoặc không có event nào được gửi — entity bị kẹt tại version cũ

Kết quả: reconciliation báo "fail" ngay cả khi sync đang chạy đúng, tạo ra false alarm.

### 2. Graph version và vector version có thể bị kẹt độc lập

Trong `projectNode`, graph adapter và vector adapter được gọi tuần tự trong cùng một call. Nếu vector adapter thất bại sau khi graph adapter đã thành công, cả event sẽ bị retry — làm graph bị upsert lại không cần thiết. Không có cơ chế theo dõi riêng biệt xem graph hay vector đang bị lag.

`ProjectionVersionRecord` lưu cả `GraphVersion` lẫn `VectorVersion`, nhưng:
- Không có `LastGraphSyncedAt` / `LastVectorSyncedAt` — không thể tính lag theo thời gian
- `GraphVersion < SourceVersion` và `VectorVersion < SourceVersion` được gộp vào cùng một drift counter

### 3. `SyncETAMs` trong response chưa được tính

`NodeCreateResponse.SyncETAMs` tồn tại trong type nhưng luôn trả về `0`. Caller không biết khi nào dữ liệu có thể đọc nhất quán từ graph/vector store.

## Proposed Solution

### 1. Phân loại trạng thái lag (`SyncLagClass`)

Mở rộng `ReconciliationReport` và `ProjectionVersionRecord` để phân loại lag theo ba trạng thái:

- `IN_FLIGHT` — version sau source nhưng event chưa qua lag tolerance window (configurable, default 30s). Không phải drift.
- `LAGGING` — event cũ hơn tolerance window nhưng retry count < threshold. Đây là cảnh báo.
- `STUCK` — retry count vượt ngưỡng, hoặc không có event nào trong deadline window. Đây là lỗi thực sự.

Reconciliation chỉ tính `GraphDriftCount` / `VectorDriftCount` cho trạng thái `STUCK` và payload mismatch. `LAGGING` được báo qua `LaggingCount` riêng.

### 2. Mở rộng `ProjectionVersionRecord` với timestamp per-replica

Thêm `LastGraphSyncedAt` và `LastVectorSyncedAt` vào `ProjectionVersionRecord`. Mỗi lần projection thành công cho một replica, timestamp đó được cập nhật độc lập. Cho phép tính lag time riêng cho graph vs vector.

### 3. Điền `SyncETAMs` dựa trên lag metrics thực

Sau khi write node thành công, service query median lag (từ `projection_version` ledger) cho domain đó và điền vào `SyncETAMs`. Nếu domain chưa có history, trả về configured `sync_eta_default_ms` (default 5000ms).

### 4. Per-entity sync status

Thêm trường `sync_status` vào `NodeRecord` response (hoặc endpoint riêng) trả về trạng thái sync hiện tại của entity: version ở source, version đã sync ở graph và vector, lag classification.

## Out of Scope

- Thay đổi lịch chạy của outbox worker (polling interval, concurrency)
- Tách graph/vector projection thành hai event stream riêng
- Backfill lại toàn bộ projection khi guardrail được deploy

## Success Criteria

- `Reconcile()` không báo drift cho entity có lag trong tolerance window
- `LAGGING` và `STUCK` được đếm và báo riêng trong `ReconciliationReport`
- `ProjectionVersionRecord` có `LastGraphSyncedAt` và `LastVectorSyncedAt` cập nhật độc lập
- `NodeCreateResponse.SyncETAMs` trả về giá trị dựa trên lag thực của domain, không phải `0`
- Admin MCP có thể query trạng thái sync của một entity cụ thể
- Tất cả test hiện tại tiếp tục pass
