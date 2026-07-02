# Proposal: Sync Session — Single Event Per Upsert Run

## Problem

Khi `codegraph-sync/sync` chạy, nó gửi nhiều HTTP bulk request (mỗi batch 200 node, tương
tự với relationship). Mỗi request hiện tại tạo ra:

- một `GraphVersion` riêng lẻ per-node bên trong mỗi batch
- một outbox event `NODE_UPSERTED` per-node
- thêm các outbox event `RELATIONSHIP_UPSERTED` riêng biệt cho bridge rels (không có
  `graph_version_id`) — gây ra duplicate sync: bridge rels được project hai lần, một lần
  trong `handleGraphVersionEvent` của node event, một lần trong legacy event path

Với N node và M relationship, một lần sync = N + M + N×M_bridge outbox events, tương ứng
N + M + N×M_bridge projection cycle riêng lẻ. Projection worker vì vậy mang tải nặng
không cần thiết và không thể batch embed hiệu quả.

Yêu cầu đúng: **1 lần tương tác dữ liệu ở mức session commit = 1 graph version change = 1 outbox event → 1 projection cycle**.

Ngoài ra, không có cơ chế khoá nào ngăn hai sync session ghi đồng thời lên cùng một graph
scope, dẫn đến race condition trên `kg_graph_identifiers` và trên projection state.

## Proposed Solution

Giới thiệu **SyncSession** — một đơn vị ghi có vòng đời tường minh:

1. Client mở một session (`POST /v1/kg/write/sync-sessions`), truyền vào
   `domain_id` + `graph_scope`, nhận về `graph_version_id` và một **durable scope lease**
   trên đúng graph scope đó.
2. Tất cả bulk batch (node và relationship) gắn `graph_version_id` vào request. Service
   ghi dữ liệu vào DB và thêm entities vào GraphVersion đã mở — **không emit event**.
   Mỗi batch phải validate version còn ở trạng thái `PENDING_ENTITIES`, cùng owner, và cùng
   `graph_scope` với session đã mở.
   Mọi mutation của cùng interaction trên graph scope đó, bao gồm create/update và stale
   relationship deletes phát hiện trong reconcile, SHALL được gộp vào cùng session.
3. Sau khi toàn bộ batch thành công, client commit session
   (`POST /v1/kg/write/sync-sessions/{id}/commit`). Service finalize GraphVersion, emit
   **đúng 1 outbox event** kiểu `GRAPH_VERSION_SEALED`, và release scope lease
   **trong cùng transaction**.
4. Nếu bất kỳ batch nào thất bại sau khi retry cạn kiệt, client abandon session
   (`DELETE /v1/kg/write/sync-sessions/{id}`). GraphVersion được đánh dấu `ABANDONED`, scope
   lease được giải phóng, không có event nào được emit.

Worker xử lý event duy nhất bằng cách đọc toàn bộ entities của GraphVersion từ
`kg_graph_version_entities` và project chúng theo batch, dùng batch embedding (embed
nhiều văn bản cùng lúc thay vì từng node) để giảm thời gian projection.

Scope lease được persist trong DB bằng một lease record keyed theo
`(owner_tenant_id, owner_app_id, graph_scope)`, không phụ thuộc PostgreSQL connection state.
Với MemoryStore (test), lease được implement bằng in-memory mutex/map per scope.

## Scope

### In scope

- `SyncSession` lifecycle: open, write, commit, abandon
- Durable scope lease per graph scope — ngăn concurrent sync trên cùng scope
- `NodeBulkCreateRequest` và `RelationshipBulkCreateRequest` nhận optional
  `graph_version_id` (session mode)
- Stale relationship deletes trong `codegraph-sync` đi qua cùng session thay vì phát event
  legacy riêng
- Xóa bridge rel outbox events thừa trong bulk path (bug fix)
- Batch embedding trong `handleGraphVersionEvent` khi xử lý GRAPH_VERSION_SEALED
- Cơ chế expiry: GraphVersion ở `PENDING_ENTITIES` quá N giờ được cleanup job đánh dấu
  `ABANDONED` và release scope lease
- `codegraph-sync/bridge` cập nhật `reconcileNodes` và `reconcileRelationships` dùng session

### Out of scope

- Thay đổi schema outbox event cho các write path khác (single node, single rel, delete)
- Thay đổi reconciliation logic
- Đổi batch size defaults trong worker pool

## Success Criteria

- `codegraph-sync/sync` trên một graph scope, bao gồm create/update/delete thuộc cùng
  interaction, emit đúng 1 outbox event
- Projection worker xử lý 1000 node trong 1 event nhanh hơn hoặc bằng 1000 event nhờ
  batch embed
- Hai concurrent sync trên cùng graph scope: session thứ hai nhận lỗi conflict thay vì
  ghi đè dữ liệu
- `CommitSyncSession` an toàn khi retry: không thể tạo trạng thái `SEALED` mà thiếu
  `GRAPH_VERSION_SEALED` event
- Abandon session sau retry exhausted: không có event nào tồn tại ở trạng thái `PENDING`
  cho graph_version_id đó
- Bridge rels không còn bị project hai lần trong bulk path
