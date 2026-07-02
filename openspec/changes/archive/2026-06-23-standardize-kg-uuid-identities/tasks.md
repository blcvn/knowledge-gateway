# Tasks

- [x] **T1** — Inventory toàn bộ các nơi đang sinh, parse, persist, hoặc giả định `node_*` / `rel_*` / text IDs trong write path, bridge sync, outbox, projection ledger, graph adapter, vector adapter, bootstrap scripts, và validation scripts.
- [x] **T2** — Thiết kế và implement canonical UUID generation/reuse flow cho node và relationship writes, đồng thời giữ `external_ref` làm idempotent lookup key cho repeated CodeGraph sync.
- [x] **T3** — Revert các thay đổi schema/migration/code gần đây đã nới service-owned identity columns từ UUID sang `TEXT`, và bổ sung migration/backfill strategy để chuyển legacy rows/ref sang UUID hợp lệ.
- [x] **T4** — Cập nhật CodeGraph bridge/sync state để resolve, cache, và reuse canonical UUIDs khi upsert node/relationship thay vì tự sinh text IDs.
- [x] **T5** — Cập nhật graph/vector projection và adapters để dùng canonical UUID identity nhất quán, bao gồm fix Qdrant point-id compatibility mà không làm lệch source-of-truth contract.
- [x] **T6** — Mở rộng test/migration/runtime validation để chứng minh sync rerun vẫn idempotent theo `external_ref`, còn persisted IDs và backend projection IDs đều là UUID hợp lệ.
