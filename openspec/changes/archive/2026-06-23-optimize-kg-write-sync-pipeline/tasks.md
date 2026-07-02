# Tasks

- [x] **T1** — Tạo flow diagram (Mermaid hoặc PlantUML) cho current-state và target-state của toàn
  bộ pipeline `write -> outbox -> graph/vector/FTS projection`. Diagram phải thể hiện rõ vai trò của
  `kg_nodes`, `kg_relationships`, `kg_outbox_events`, `kg_projection_versions`, và các backend ngoài
  Postgres, phân biệt synchronous write path và asynchronous projection path. *(Design.md đã có mô tả
  dạng text; T1 sản xuất diagram artifact riêng — không lặp lại nội dung design.md.)*

- [x] **T2** — Implement true bulk upsert cho nodes/relationships/outbox ở service và repository layer.
  Yêu cầu cụ thể:
  - Multi-row insert/upsert (hoặc COPY staging table) thay vì loop từng item trong transaction.
  - Batch resolve `external_ref` thay vì `SELECT` từng record.
  - Batch outbox append cho cả batch node/relationship trong một lần.
  - **Batch failure semantics: partial success với per-item error list** (đã chốt trong design.md). Response
    trả về `succeeded[]` và `failed[{index, external_ref, error}]`. Preflight validate trước khi mở
    transaction, persist chỉ item pass.
  - **Bulk soft-delete by external_ref prefix**: thay thế `DeleteNodesByExternalRefPrefixWithContext`
    (`service.go:424`) đang load toàn bộ node vào memory và loop xóa tuần tự bằng query bulk soft-delete
    trực tiếp trên Postgres.
  - Idempotency rules: upsert trên `external_ref` không tạo duplicate, retry an toàn.

- [x] **T3** — Cập nhật `codegraph-sync` sang bulk-first producer. Yêu cầu cụ thể:
  - Thay `reconcileNodes` / `reconcileRelationships` (sync.go:95, sync.go:139) gọi từng API bằng bulk
    upsert API mới từ T2.
  - **Fix relationship delete semantics — correctness bug**: `reconcileRelationships` (sync.go:176) gán
    `state.Relationships = next` nhưng không tombstone relationship không còn trong source. Cần bổ sung:
    1. Sau khi build `next`, tính `stale = state.Relationships - next`.
    2. Gọi bulk delete/tombstone cho `stale` relationships qua KG service API.
    3. Chỉ ghi `state.Relationships = next` sau khi delete stale thành công.
  - State manifest strategy: nếu bulk upsert trả về partial success, chỉ ghi vào state những item trong
    `succeeded` list.
  - Xác định batch size (đề xuất khởi điểm: 200 nodes/batch, 100 relationships/batch, cấu hình qua env).

- [x] **T4** — Thiết kế lại projection runtime theo batch/worker-pool cho graph, embedding/vector, và FTS.
  Yêu cầu cụ thể:
  - **Outbox claim pattern: `SELECT ... FOR UPDATE SKIP LOCKED`** (đã chốt trong design.md). Thêm
    `ClaimOutboxBatch(ctx, pageSize int) ([]OutboxEvent, error)` vào Repository interface. `ListOutboxEvents()`
    chỉ còn dùng cho test/admin, không dùng trong production worker loop.
  - Worker pool với giới hạn goroutine (đề xuất: `WORKER_POOL_SIZE` env, mặc định 10). Group events
    theo backend sau khi claim: graph projector, embedding+vector projector, FTS projector chạy song song.
  - Backpressure: khi pool đầy, dừng claim batch mới thay vì tích lũy goroutine không giới hạn.
  - Retry độc lập theo backend: graph fail không ảnh hưởng vector; projection version ledger cập nhật
    riêng từng backend.
  - **Pagination cho Reconcile()**: thay `ListNodes()`, `ListRelationships()`, `ListProjectionVersions()`
    (runtime.go:244) bằng paginated scan. Định nghĩa `ListNodesBatch`, `ListRelationshipsBatch`,
    `ListProjectionVersionsBatch` trong Repository interface trước khi implement.

- [x] **T5** — Backend-specific optimizations cho projection adapters:
  - **Milvus**: bỏ `client.Flush(ctx, ...)` per-document (milvus.go:81). Thêm `UpsertBatch([]VectorDocument)`
    nhận cả batch, flush sau khi upsert xong batch hoặc theo time window (đề xuất: flush mỗi 5 giây
    hoặc sau batch hoàn chỉnh).
  - **PgVector**: multi-row upsert, tránh update HNSW index quá phân mảnh.
  - **Neo4j/Memgraph**: dùng `UNWIND` batch Cypher thay vì `MERGE` từng entity.

- [x] **T6** — Finalize FK drop và viết migration. Yêu cầu cụ thể:
  - Áp dụng bảng quyết định trong design.md: drop 3 FK trên `kg_relationships`/`kg_vector_documents`,
    giữ FK tới `domains`, `tenants`, `apps`.
  - Migration script chỉ xử lý DROP CONSTRAINT và tạo index thay thế cần thiết — không nhồi backfill
    hay orphan repair vào cùng migration.
  - `verify` script kiểm tra sau migrate: đúng FK còn lại, orphan count trong ngưỡng chấp nhận.
  - Ghi rõ rollback policy: forward-only hay có `down.sql` reversible.

- [x] **T7** — Implement app-managed integrity sau khi bỏ FK:
  - **Write-time validation**: giữ và mở rộng pattern check node existence ở `createRelationshipInScope`
    (service.go:898) cho bulk path — batch-resolve node existence thay vì check từng item.
  - Orphan scan `kg_relationships`: periodic job tìm relationship có `from_node_id`/`to_node_id` không
    còn trong `kg_nodes` (non-deleted).
  - Orphan scan `kg_vector_documents`: periodic job tìm vector doc có `node_id` không còn source node
    chưa bị soft-delete.
  - Tombstone-driven delete: soft-delete node → emit `NODE_DELETED` → worker cleanup graph/vector/FTS
    → orphan job dọn relationship/vector còn sót. *(Pattern đã có ở `deleteNodeInScope` — T7 mở rộng
    cho orphan cleanup và lifecycle sau khi bỏ CASCADE.)*
  - Admin repair commands: rebuild projection từ source tables, purge orphan data.

- [x] **T8** — Chuẩn hoá spec cho SQL scripts và migrations: taxonomy `migration/verify/backfill/repair`,
  naming rules, idempotency rules, rollback boundary, và lock/transaction policy cho bảng nóng.

- [x] **T9** — Định nghĩa rollout pattern chuẩn cho DB changes lớn: add-compatible, backfill/reconcile,
  verify, cutover, drop-old; áp dụng rõ cho các thay đổi drop FK, thêm bulk-write indexes, và
  projection maintenance.

- [x] **T10** — Instrumentation và observability cho write/sync pipeline:
  - Metric: `kg_outbox_backlog` — số event PENDING/FAILED còn tồn đọng.
  - Metric: `kg_graph_lag_seconds` và `kg_vector_lag_seconds` — median và p99 từ `kg_projection_versions`.
  - Metric: `kg_orphan_relationships_count` và `kg_orphan_vector_docs_count` — output từ orphan scan.
  - Metric: `kg_bulk_write_batch_size` histogram và `kg_bulk_write_partial_failure_rate`.
  - Alert: outbox backlog vượt ngưỡng, lag class STUCK tăng.
  - T10 nên hoàn thành trước hoặc cùng lúc T4/T7: không có metrics thì không biết optimization có
    hiệu quả hay không.
