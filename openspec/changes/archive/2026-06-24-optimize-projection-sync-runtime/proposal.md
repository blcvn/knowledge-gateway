# Proposal: Optimize Projection Sync Runtime

## Problem

Quá trình sync từ `relationshipdb` sang `graphdb` và `vector db` hiện vẫn còn chậm dù outbox đã được
claim theo batch.

Qua rà soát code hiện tại:

- worker claim event theo lô bằng `ClaimOutboxBatch`, nhưng vẫn xử lý projection theo từng event riêng lẻ;
- mỗi `NODE_UPSERTED` đang chạy tuần tự theo chuỗi `graph upsert -> embedding -> vector upsert -> FTS index`;
- `processOutboxEvent` giữ global runtime lock trong suốt vòng đời xử lý event, làm giảm đáng kể mức
  song song thực tế;
- graph và vector projection chưa có cơ chế bulk/coalescing cho nhiều event của cùng một batch;
- projection version vẫn cần được bảo toàn chính xác khi một backend thành công còn backend kia thất bại.

Hệ quả là:

- throughput projection thấp hơn khả năng ingest của source write path;
- latency đồng bộ kéo dài khi backlog tăng;
- chi phí embedding và adapter round-trip tăng theo từng entity;
- khó tối ưu hiệu năng mà vẫn giữ được idempotency, ordering, và integrity giữa source với replicas.

## Proposed Solution

Tối ưu projection runtime theo hướng batch-first nhưng vẫn giữ `relationshipdb` là source-of-truth và
outbox là durable handoff.

1. **Batch và coalesce event trước khi project**
   - claim outbox theo page cố định;
   - group event theo aggregate kind;
   - coalesce nhiều event của cùng entity trong một page về mutation cuối cùng có source version mới nhất.

2. **Bỏ global critical section trên đường xử lý projection**
   - không giữ runtime mutex trong suốt network/embedding call;
   - chỉ lock ngắn khi cập nhật in-memory mirrors, projection ledger cache, và status cascade.

3. **Thêm backend-specific bulk projection**
   - graph adapters nhận batch node/relationship upsert-delete;
   - vector path nhận batch embedding/upsert và flush theo batch hoặc time window thay vì per-document;
   - worker có bounded concurrency và backpressure rõ ràng.

4. **Giữ integrity bằng version guard và per-backend ledger**
   - projection chỉ được phép advance replica version theo hướng tăng dần;
   - graph success không được ghi đè vector failure, và ngược lại;
   - stale event không được rollback replica về version cũ hơn.

5. **Bổ sung observability cho runtime**
   - backlog, batch size, coalescing ratio, queue age, graph/vector latency, partial failure rate;
   - đủ để đo hiệu quả tối ưu và phát hiện drift hoặc lag bất thường.

## Scope

### In scope

- projection worker runtime từ outbox sang `graphdb`, `vector db`, và FTS
- batch/coalesced processing cho node và relationship events
- concurrency control, backpressure, và retry boundary cho projection
- bulk adapter contracts cho graph/vector backends
- projection version/integrity rules để tránh stale overwrite
- observability trực tiếp phục vụ projection runtime

### Out of scope

- redesign source write contract của API
- thay đổi business schema của node/relationship
- thay đổi auth/RLS semantics
- redesign search ranking hoặc query APIs
- thay đổi ontology/domain model ngoài phần cần cho projection correctness

## Success Criteria

- projection runtime xử lý theo batch thực sự thay vì per-entity network round-trip trên critical path
- graph/vector sync có bounded concurrency nhưng không mất idempotency hoặc ordering theo entity
- stale event không thể overwrite replica đã ở source version mới hơn
- graph và vector lag/throughput có metrics đủ rõ để vận hành và tuning
- thiết kế đủ chi tiết để triển khai mà không làm mơ hồ boundary giữa performance và data integrity
