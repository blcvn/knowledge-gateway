# Tasks

- [x] **T1** — Refactor projection runtime thành pipeline `claim -> coalesce -> project -> commit`.
  Yêu cầu:
  - giữ `ClaimOutboxBatch(... FOR UPDATE SKIP LOCKED)` làm cơ chế claim chuẩn;
  - bỏ global runtime lock khỏi network/embedding path;
  - chỉ giữ lock ngắn cho shared in-memory state và status cascade.

- [x] **T2** — Implement coalescing trong một outbox page cho node/relationship events.
  Yêu cầu:
  - gộp nhiều event cùng entity về mutation cuối;
  - giữ delete semantics idempotent;
  - event cũ hơn trong cùng page không được ép projector chạy lại vô ích.

- [x] **T3** — Thêm batch source loading và chunk dispatch.
  Yêu cầu:
  - batch-load node/relationship source rows theo danh sách ID thay vì fetch từng event;
  - chunk riêng cho graph, vector, embedding, và FTS path;
  - hỗ trợ tuning batch size qua env/config.

- [x] **T4** — Thêm bulk graph adapter contracts và production implementations.
  Yêu cầu:
  - `UpsertNodesBatch`, `DeleteNodesBatch`, `UpsertRelationshipsBatch`, `DeleteRelationshipsBatch`;
  - Neo4j/Memgraph dùng `UNWIND` batch query;
  - response phải map được entity success/failure để cập nhật ledger chính xác.

- [x] **T5** — Thêm batch vector adapter contracts và flush policy.
  Yêu cầu:
  - `UpsertBatch`, `DeleteBatch`;
  - Milvus không flush per-document;
  - pgvector dùng multi-row upsert;
  - lưu `_kg_sync_version` cho mọi document để chặn stale overwrite.

- [x] **T6** — Tách graph success và vector success trong projection ledger/update path.
  Yêu cầu:
  - graph và vector cập nhật version/timestamp độc lập;
  - stale event được đánh dấu no-op thành công thay vì retry;
  - event chỉ dead-letter khi còn backend bắt buộc chưa sync được sau retry policy.

- [x] **T7** — Thêm version guard cho projection commit.
  Yêu cầu:
  - replica version chỉ được advance theo chiều tăng;
  - event cũ đến muộn không được ghi đè state mới hơn;
  - integration tests phải cover out-of-order events và concurrent processing.

- [x] **T8** — Bổ sung observability cho projection runtime.
  Yêu cầu:
  - metrics cho claim size, coalesced entity count, queue age, graph/vector/embedding latency;
  - metrics cho stale skip và partial backend failure;
  - alerting guideline cho backlog tăng và queue age kéo dài.
