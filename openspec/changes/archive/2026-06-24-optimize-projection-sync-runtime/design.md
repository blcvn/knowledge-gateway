# Design: Optimize Projection Sync Runtime

## Overview

Change này tối ưu đúng đoạn bạn đang quan tâm: pipeline sync từ `relationshipdb` qua outbox sang
`graphdb` và `vector db`.

Mục tiêu là cải thiện:

- **throughput**: xử lý được backlog lớn hơn với ít round-trip hơn;
- **latency**: giảm thời gian từ source commit đến lúc projection sẵn sàng;
- **integrity**: không để event cũ ghi đè replica mới hơn, không để graph/vector che lỗi của nhau.

Change này không thay đổi source-of-truth. `relationshipdb` vẫn là canonical state; projection vẫn là
asynchronous replicas được dẫn bởi outbox.

## Current State

### Runtime flow hiện tại

```text
write commit -> kg_outbox_events
worker PollOnce
  -> ClaimOutboxBatch(page)
  -> fan-out goroutines
  -> processOutboxEvent(event)
       -> runtime mutex held
       -> fetch source row
       -> graph upsert/delete
       -> embed
       -> vector upsert/delete
       -> fts index/delete
       -> update projection version
       -> update outbox status
```

### Bottlenecks chính

1. **Batch claim nhưng per-event projection**
   - worker đã claim event theo page, nhưng graph/vector/FTS vẫn gọi theo từng entity.

2. **Global lock làm nghẽn concurrency**
   - runtime giữ mutex xuyên suốt lúc xử lý event, bao gồm cả I/O và embedding call.
   - điều này làm worker pool có nhiều goroutine nhưng mức song song thực tế thấp.

3. **Không coalesce event**
   - nếu cùng một node bị update nhiều lần trong backlog, worker vẫn có thể project từng event thay vì
     chỉ trạng thái cuối cùng.

4. **Chưa có bulk adapter contract**
   - graph adapter và vector adapter hiện thiên về single-item upsert/delete.
   - Milvus/pgvector/Neo4j/Memgraph đều có cơ hội tối ưu tốt hơn khi nhận batch.

5. **Integrity boundary còn mỏng khi tối ưu hiệu năng**
   - nếu tăng song song mà không có version guard, event cũ có thể đến sau và overwrite replica mới hơn.
   - nếu graph thành công còn vector thất bại, ledger phải phản ánh đúng partial progress.

## Goals

### Performance goals

- giảm số network round-trip tới graph/vector backends trên mỗi outbox page;
- tăng hiệu quả worker pool bằng cách bỏ lock quá rộng;
- cho phép tuning batch size độc lập cho outbox claim, embedding, graph write, vector write.

### Integrity goals

- mỗi entity chỉ advance replica version theo chiều tăng;
- stale event không được áp lên replica nếu source version mới hơn đã được sync;
- graph/vector có retry và ledger độc lập, không che khuất partial failure.

## Target Architecture

### 1. Batch claim và coalescing

Worker tiếp tục dùng `SELECT ... FOR UPDATE SKIP LOCKED`, nhưng thay vì dispatch từng event ngay,
runtime sẽ xử lý theo các bước:

1. claim một page outbox;
2. chuẩn hóa event thành `entity mutation intent`;
3. group theo `aggregate_type`;
4. coalesce theo `(entity_kind, entity_id)` để chỉ giữ mutation cuối cùng trong page;
5. batch-load source rows tương ứng;
6. gửi sang graph/vector projectors theo chunk.

### Coalescing rules

- `NODE_UPSERTED` sau `NODE_UPSERTED` cùng entity trong cùng page -> giữ event mới nhất;
- `NODE_DELETED` xuất hiện sau mọi node upsert trong page -> mutation cuối là delete;
- `RELATIONSHIP_DELETED` xuất hiện sau `RELATIONSHIP_UPSERTED` -> mutation cuối là delete;
- event không còn source row nhưng mutation cuối là delete -> vẫn cho phép projector chạy delete idempotent.

Coalescing chỉ áp dụng trong phạm vi page đang claim. Nó không thay đổi durability semantics của outbox.

### 2. Bounded concurrency thay cho global lock

`PollOnce` sẽ chia thành ba tầng:

1. **claim/coalesce stage**
   - đơn luồng, nhẹ, chỉ xử lý metadata;
2. **projection stage**
   - graph/vector/FTS chạy bằng worker pool giới hạn;
3. **commit stage**
   - cập nhật outbox status, projection version, in-memory mirrors theo kết quả batch.

Runtime mutex chỉ được giữ ở commit stage khi cần cập nhật shared memory state. Không giữ lock qua:

- đọc source rows;
- graph upsert/delete;
- embedding generation;
- vector upsert/delete;
- FTS index/delete.

Nếu cần tránh race trên cùng entity giữa hai page khác nhau, dùng một trong hai chiến lược:

- per-entity lock theo `entity_id`; hoặc
- version guard ở lúc ghi replica và cập nhật ledger.

Change này chọn **version guard là lớp bảo vệ bắt buộc**, còn per-entity lock là tối ưu tùy chọn cho hot keys.

### 3. Independent backend projectors

Runtime tách projection thành các projector độc lập:

- **graph projector**
  - nhận batch node/relationship upsert-delete;
  - gọi adapter bulk API;
  - trả kết quả per-entity.

- **embedding/vector projector**
  - nhận batch node upsert;
  - gom text để embed theo batch hoặc bounded mini-batches;
  - gọi vector adapter bulk upsert;
  - delete vector docs theo batch cho tombstone path.

- **FTS projector**
  - index/delete theo chunk riêng;
  - không chặn graph success hoặc vector success.

### 4. Version guard và partial progress ledger

Mỗi projector phải làm việc với `source_version` của entity.

Rules:

- graph replica chỉ được update nếu `incoming_source_version >= stored_graph_version`;
- vector replica chỉ được update nếu `incoming_source_version >= stored_vector_version`;
- nếu event cũ hơn replica hiện tại, projector bỏ qua như no-op idempotent;
- `ProjectionVersionRecord` cập nhật độc lập cho graph và vector timestamps/versions;
- trạng thái outbox `DONE` chỉ khi các backend bắt buộc của event đã hoàn tất hoặc được xác nhận no-op do stale.

### Quyết định: backend failure semantics

Chọn **partial backend progress với ledger độc lập**, không all-or-nothing giữa graph và vector.

Lý do:

- graph và vector có latency profile khác nhau;
- embedding có thể fail tạm thời trong khi graph vẫn thành công;
- ép all-or-nothing sẽ làm tăng retry cost và kéo dài lag không cần thiết.

Hệ quả:

- event cần lưu rõ backend nào đã sync;
- retry chỉ chạy phần còn thiếu;
- reconcile dựa vào ledger để báo lag riêng từng replica.

### 5. Bulk adapter contracts

#### Graph adapter

Thêm bulk APIs:

- `UpsertNodesBatch`
- `DeleteNodesBatch`
- `UpsertRelationshipsBatch`
- `DeleteRelationshipsBatch`

Backend guidance:

- Neo4j/Memgraph: dùng `UNWIND` để gửi batch Cypher;
- adapter phải trả kết quả chi tiết đủ để mapping entity nào fail.

#### Vector adapter

Thêm bulk APIs:

- `UpsertBatch`
- `DeleteBatch`

Embedding stage nên hỗ trợ mini-batch size riêng với vector upsert size riêng.

Backend guidance:

- Milvus: không flush sau từng document; flush theo batch hoàn chỉnh hoặc time window ngắn;
- pgvector: multi-row upsert để giảm churn index;
- các adapter phải lưu `_kg_sync_version` để no-op stale writes và phục vụ reconcile.

## Operational Tuning

Các knobs cấu hình đề xuất:

- `OUTBOX_PAGE_SIZE`
- `GRAPH_BATCH_SIZE`
- `VECTOR_BATCH_SIZE`
- `EMBED_BATCH_SIZE`
- `FTS_BATCH_SIZE`
- `PROJECTION_WORKER_POOL_SIZE`
- `VECTOR_FLUSH_INTERVAL_MS`

Mặc định nên thiên về an toàn:

- outbox page 100
- graph/vector chunk 25-50
- embed batch 16-32
- worker pool 4-8

## Observability

Runtime cần xuất ít nhất các metric sau:

- `kg_projection_outbox_claim_size`
- `kg_projection_coalesced_entities`
- `kg_projection_queue_age_seconds`
- `kg_graph_batch_latency_seconds`
- `kg_vector_batch_latency_seconds`
- `kg_embedding_batch_latency_seconds`
- `kg_projection_partial_failure_total`
- `kg_projection_stale_event_skips_total`

Các metric này là bắt buộc để tuning batch size mà không làm mù vận hành.

## Rollout Plan

1. thêm bulk adapter interfaces nhưng vẫn giữ single-item fallback;
2. thêm runtime batching/coalescing phía sau feature flag;
3. chạy integration tests với mixed success/failure paths;
4. bật canary cho batch projection;
5. theo dõi backlog, queue age, partial failure, stale skip;
6. bỏ đường single-item cũ khi metrics ổn định.

## Risks

### Risk: Coalescing làm mất semantics retry

Giảm thiểu:

- chỉ coalesce trong phạm vi page hiện tại;
- không xóa event history trước khi backend commit xong;
- ledger ghi source event/version cuối cùng đã áp dụng.

### Risk: Batch lớn làm tăng tail latency

Giảm thiểu:

- chunk theo backend riêng;
- tách embed batch khỏi vector batch;
- có backpressure và giới hạn worker pool.

### Risk: Stale overwrite khi xử lý song song

Giảm thiểu:

- version guard bắt buộc trên graph/vector/ledger;
- stale event được xem là successful no-op, không retry vô hạn.

## Spec Impact

Change này sửa ba vùng spec:

- `sync-consistency`
- `graph-db-adapter`
- `vector-store-adapter`

Không tạo capability business mới; thay vào đó siết chặt contract runtime projection để tối ưu hiệu
năng nhưng vẫn giữ tính toàn vẹn dữ liệu.
