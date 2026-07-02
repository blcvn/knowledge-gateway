# Proposal: Optimize KG Write And Sync Pipeline

## Problem

Luồng `insert/update/upsert` của `kg-service` hiện đang chậm khi ingest/sync dữ liệu lớn vào knowledge graph.
Qua rà soát code hiện tại, độ trễ đến từ cả phía producer lẫn projection runtime:

- `codegraph-sync` đang gọi `CreateNode` và `CreateRelationship` theo từng item, chưa dùng bulk API.
- bulk API hiện có trong `internal/write/service.go` vẫn lặp từng entity trong cùng transaction, chưa có bulk SQL hay bulk outbox thật sự.
- mỗi `NODE_UPSERTED` event được worker xử lý tuần tự theo chuỗi `graph upsert -> embedding -> vector upsert -> FTS index`.
- adapter Milvus đang `Flush` sau từng document upsert.
- các bảng nóng của write path vẫn giữ foreign key trực tiếp giữa `kg_relationships`, `kg_vector_documents`, `kg_nodes`, và các bảng identity/domain, làm tăng cost validate constraint khi ghi lớn.

Hệ quả là thời gian upsert kéo dài, backlog outbox tăng, và sync giữa source-of-truth với graph/vector projections dễ bị lag khi tải tăng.

## Proposed Solution

Thiết kế lại write/sync pipeline theo hướng tối ưu ingest khối lượng lớn, nhưng vẫn giữ semantics hiện tại: Postgres là source-of-truth, graph/vector là projection bất đồng bộ.

1. **Vẽ lại current-state flow và target-state flow**
   - làm rõ đường đi của dữ liệu qua `kg_nodes`, `kg_relationships`, `kg_outbox_events`, `kg_projection_versions`, graph backend, vector backend, và FTS.
   - chỉ ra rõ đâu là synchronous write path, đâu là asynchronous projection path.

2. **Chuẩn hóa bulk-first write path**
   - thêm contract bulk upsert thực sự cho nodes/relationships/outbox thay vì loop từng item.
   - cho phép producer như `codegraph-sync` gửi theo batch thay vì từng request đơn lẻ.

3. **Tách source write khỏi projection cost**
   - giữ API write chỉ commit source records + outbox càng nhanh càng tốt.
   - đẩy embedding, graph upsert, vector upsert, FTS index sang worker batch hoặc parallel worker pool có backpressure.

4. **Nới lỏng ràng buộc quan hệ ở tầng database cho bảng nóng**
   - xem xét bỏ foreign key trực tiếp trên các bảng write-heavy/projection-heavy như `kg_relationships.from_node_id`, `kg_relationships.to_node_id`, `kg_vector_documents.node_id`.
   - chuyển phần lớn quản lý relationship/reference integrity sang application code, reconciliation jobs, và cleanup jobs.

5. **Chuẩn hoá SQL scripts và migration contract**
   - định nghĩa chuẩn chung cho cấu trúc file migration, naming, idempotency, và rollback boundary.
   - tách rõ DDL schema changes, data backfill, verification SQL, và repair SQL để rollout thay đổi lớn an toàn hơn.

## Scope

### In scope

- analysis và redesign cho write path `nodes/relationships/outbox/projection`
- flow dữ liệu qua Postgres, graph backend, vector backend, và FTS
- bulk ingest/upsert contract cho producer và service
- giảm FK/relational coupling ở các bảng nóng để tăng tốc ghi/đồng bộ
- cơ chế app-managed integrity, reconciliation, orphan cleanup, và retry
- chuẩn hoá SQL scripts, migration layout, và rollout contract cho DB changes

### Out of scope

- thay đổi semantics auth/RLS hiện tại
- redesign ontology/domain schema
- thay đổi business meaning của node/relationship
- tối ưu query/read API ngoài các ảnh hưởng trực tiếp từ write-model mới

## Success Criteria

- có một tài liệu flow rõ ràng cho current state và target state qua tất cả database/backend liên quan
- xác định được bottleneck chính và chuyển chúng thành backlog implementation cụ thể
- có thiết kế bulk write + batch sync đủ rõ để `codegraph-sync` và các producer khác dùng được
- có quyết định rõ ràng về FK nào sẽ bỏ, FK nào giữ lại, và integrity sẽ được thay bằng cơ chế code/runtime nào
- có chuẩn migration/SQL rõ ràng để rollout các thay đổi schema, backfill, và cleanup mà không làm pipeline khó vận hành
- target design giảm đáng kể round-trip ghi, giảm projection lag, và phù hợp với dữ liệu lớn
