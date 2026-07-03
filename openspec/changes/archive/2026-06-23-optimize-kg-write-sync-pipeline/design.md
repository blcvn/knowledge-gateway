# Design: Optimize KG Write And Sync Pipeline

## Overview

Change này tập trung vào write-path và sync-path của `kg-service` khi ingest knowledge graph lớn.
Mục tiêu không phải đổi source-of-truth, mà là làm cho source write thật nhanh, còn projection sang
graph/vector/FTS được xử lý theo batch, song song, và có khả năng chịu tải tốt hơn.

## Current State

### Current data flow

```text
Producer / examples/codegraph
  -> POST /v1/kg/write/nodes hoặc /relationships
  -> write.Service
  -> Postgres transaction
       - kg_nodes
       - kg_relationships
       - kg_outbox_events
  -> HTTP response trả về ngay

Worker PollOnce
  -> scan kg_outbox_events
  -> với từng event:
       - đọc lại source record từ Postgres
       - upsert graph backend
       - build embedding
       - upsert vector backend
       - index FTS
       - upsert kg_projection_versions
       - update outbox status
```

### Database and backend roles

- `kg_nodes`: source-of-truth cho node metadata/properties
- `kg_relationships`: source-of-truth cho edge metadata
- `kg_outbox_events`: durable event stream cho projection
- `kg_projection_versions`: ledger theo dõi source version và projection lag
- graph backend (`neo4j`/`memgraph`/`nebula`): traversal projection
- vector backend (`pgvector`/`qdrant`/`milvus`): semantic search projection
- FTS backend (`postgres` hoặc memory): lexical search projection

### Observed bottlenecks from the current code

#### 1. Producer vẫn gửi tuần tự từng entity

`examples/codegraph/internal/bridge/sync.go` hiện reconcile node/relationship bằng cách gọi từng
`CreateNode`, `UpdateNode`, `CreateRelationship` trong vòng lặp tại `reconcileNodes` (sync.go:95)
và `reconcileRelationships` (sync.go:139).

Hệ quả:

- nhiều HTTP round-trip
- nhiều transaction nhỏ
- nhiều outbox events nhỏ

**Gap bổ sung — relationship delete bị bỏ sót:**
`reconcileRelationships` gán `state.Relationships = next` (sync.go:176) nhưng không gọi delete cho
các relationship không còn trong source. Relationship bị xóa khỏi code sẽ tồn tại vĩnh viễn trong
KG service vì bridge không tombstone chúng. Đây là correctness bug độc lập với performance, cần
sửa song song với việc chuyển sang bulk mode.

#### 2. Bulk API chưa phải bulk persistence thật

`CreateNodesBulkWithContext` và `CreateRelationshipsBulkWithContext` hiện vẫn loop từng item trong code.
Chúng gom vào một transaction, nhưng chưa:

- dùng `COPY` hay multi-row insert
- tạo outbox theo batch envelope
- dedupe hoặc reorder theo batch

#### 3. Worker xử lý projection tuần tự cho từng event

`internal/workers/runtime.go` hiện xử lý `NODE_UPSERTED` theo chuỗi:

1. graph upsert
2. embedding generation
3. vector upsert
4. FTS index
5. projection version update

Với dữ liệu lớn, đây là đường chậm nhất vì mỗi event kéo theo nhiều network calls hoặc compute steps.

**Gap bổ sung — outbox scan không có pagination:**
`PollOnce()` gọi `ListOutboxEvents()` (runtime.go:165) — load toàn bộ outbox vào memory. Ở quy mô
lớn với backlog cao, đây là memory explosion. Tương tự, `Reconcile()` gọi `ListNodes()`,
`ListRelationships()`, `ListProjectionVersions()` (runtime.go:244–514) mà không phân trang.
Mọi paginated access cần được định nghĩa ở Repository interface trước khi implement T4 và T7.

#### 4. Milvus flush theo từng upsert

`internal/platform/vectorstore/milvus.go` đang `Flush` sau từng `Upsert`, làm tăng đáng kể latency
và giảm throughput batch ingest.

#### 5. FK trên bảng nóng làm tăng cost ghi

Các migration hiện có giữ FK trực tiếp:

- `kg_relationships.from_node_id -> kg_nodes.id`
- `kg_relationships.to_node_id -> kg_nodes.id`
- `kg_vector_documents.node_id -> kg_nodes.id`
- cùng một số FK tới `domains`, `tenants`, `apps`

Với workload ingest lớn, FK validation và cascade behavior làm write path nặng hơn mức cần thiết cho
một hệ projection/outbox.

## Target State

### Target flow

```text
Producer / sync bridge
  -> bulk upsert API
  -> Postgres bulk transaction
       - bulk upsert kg_nodes
       - bulk upsert kg_relationships
       - bulk append kg_outbox_events
  -> ack nhanh

Projection workers
  -> claim batch từ outbox
  -> group theo entity kind / domain / backend
  -> xử lý song song theo pool:
       - graph projector
       - embedding/vector projector
       - FTS projector
  -> bulk update projection_versions
  -> bulk update outbox status

Background integrity jobs
  -> orphan detection
  -> relationship endpoint validation
  -> projection reconciliation
  -> cleanup/tombstone compaction
```

## Proposed changes

### 1. Introduce true bulk upsert contracts

Service nên có bulk contract thực sự cho:

- nodes upsert
- relationships upsert
- outbox append

Implementation target:

- dùng multi-row insert/upsert hoặc `COPY` staging table
- resolve `external_ref` theo batch thay vì `SELECT` từng record
- emit outbox theo batch hoặc theo chunk lớn, không từng entity lẻ nếu không cần

`examples/codegraph` phải đổi sang bulk-first mode, chỉ fallback về single-item khi batch thất bại cần isolate lỗi.

**Quyết định: batch failure semantics — partial success với per-item error list.**

Hai lựa chọn:

- *All-or-nothing per batch*: đơn giản nhất, nhưng một node sai schema block cả batch 500 node.
- *Partial success với per-item error list*: response trả về `{succeeded: [...], failed: [{index, external_ref, error}]}`.

Chọn partial success. Lý do: `examples/codegraph` sync hàng trăm node mỗi lần; một node lỗi validation
không nên block phần còn lại. Producer tự quyết định có retry item lỗi hay bỏ qua. Response contract:

```json
{
  "succeeded": ["node_id_1", "node_id_2"],
  "failed": [
    { "index": 3, "external_ref": "pkg:foo/bar", "error": "unknown node_type: FunctionX" }
  ]
}
```

Implementation cần xử lý validate trước khi mở transaction (preflight) và persist chỉ những item pass.

**Gap bổ sung — `DeleteNodesByExternalRefPrefixWithContext` là hotspot chưa được đề cập:**
`service.go:424` load toàn bộ node vào memory và loop xóa tuần tự. `examples/codegraph` gọi path này qua
`clearProjectByPrefix` khi reset state. Cần thêm bulk soft-delete by prefix support vào T2 hoặc T7.

### 2. Separate source write SLA from projection SLA

Write request chỉ nên chịu trách nhiệm:

- validate input
- persist source record
- append outbox

Không được để projection cost ảnh hưởng perceived ingest throughput.

Projection worker cần:

- claim outbox theo page/chunk
- process concurrently với worker pool có giới hạn
- retry độc lập theo backend
- cho phép graph/vector/FTS sync tiến độ khác nhau nhưng vẫn cập nhật ledger rõ ràng

**Quyết định: outbox claim pattern — `SELECT ... FOR UPDATE SKIP LOCKED`.**

Đây là cơ chế chuẩn cho competing-consumer outbox trên Postgres:

```sql
SELECT id, aggregate_type, aggregate_id, event_type, payload, retry_count
FROM kg_outbox_events
WHERE status = 'PENDING'
ORDER BY created_at
LIMIT $page_size
FOR UPDATE SKIP LOCKED;
```

`SKIP LOCKED` cho phép nhiều worker chạy song song mà không tranh nhau cùng event. Page size mặc định
100, cấu hình qua env. Worker claim một page, xử lý, cập nhật status, rồi lấy page tiếp. Sau khi
implement, `ListOutboxEvents()` trong Repository interface chỉ còn phục vụ test và admin — production
path dùng `ClaimOutboxBatch(ctx, pageSize int)` trả về `([]OutboxEvent, error)`.

### 3. Replace per-event projection with batch projection

Projection nên batch theo loại backend:

- graph batch cho node/relationship upsert
- vector batch cho embeddings + vector write
- FTS batch cho text index

Backend-specific notes:

- Milvus: bỏ `Flush` theo từng document, chuyển sang flush theo chunk/time window
- PgVector: hỗ trợ multi-row upsert, tránh update HNSW quá phân mảnh
- Neo4j/Memgraph: dùng `UNWIND` batch Cypher thay vì `MERGE` từng entity

### 4. Remove hot-path foreign keys and move integrity to code

Đề xuất mặc định của change này là bỏ FK ở các bảng projection/write-heavy:

- bỏ FK `kg_relationships.from_node_id`
- bỏ FK `kg_relationships.to_node_id`
- bỏ FK `kg_vector_documents.node_id`

**Quyết định: FK tới `domains`, `tenants`, `apps` — GIỮ LẠI.**

Lý do:
- Các bảng `domains`, `tenants`, `apps` có write frequency rất thấp so với `kg_nodes`/`kg_relationships`.
  FK validation trên chúng không có ý nghĩa bottleneck thực tế.
- Chúng cung cấp integrity quan trọng không thể dễ dàng thay bằng code: ngăn node/relationship bị gắn
  vào tenant/domain không tồn tại, bảo vệ multi-tenancy boundary.
- Scope change này chỉ bỏ FK trên hot write path giữa `kg_relationships`, `kg_vector_documents`, và
  `kg_nodes`. Không mở rộng ra identity/domain tables.

Danh sách FK thay đổi:

| FK | Action | Reason |
|----|--------|--------|
| `kg_relationships.from_node_id -> kg_nodes.id` | Drop | Hot path, validated by code |
| `kg_relationships.to_node_id -> kg_nodes.id` | Drop | Hot path, validated by code |
| `kg_vector_documents.node_id -> kg_nodes.id` | Drop | Projection table, orphan handled by cleanup job |
| `kg_*.domain_id -> domains.id` | Keep | Low write freq, multi-tenancy boundary |
| `kg_*.owner_tenant_id -> tenants.id` | Keep | Low write freq, auth boundary |
| `kg_*.owner_app_id -> apps.id` | Keep | Low write freq, auth boundary |

Ưu tiên giữ unique index và lookup index hơn là FK join enforcement

Sau khi bỏ FK, integrity sẽ do code và background jobs quản lý:

- validate endpoint existence ở write service trước khi insert relationship — pattern này **đã có** tại
  `service.go:898–905` (`createRelationshipInScope` check `from_node`/`to_node` tồn tại trước khi
  insert). Cần giữ và mở rộng pattern này cho bulk path.
- khi delete node, ghi tombstone/outbox và để cleanup job xử lý orphan relationships/vector docs
- reconciliation job phát hiện:
  - relationship trỏ tới node không tồn tại
  - vector doc không còn source node
  - projection ledger bị orphan
- cung cấp admin repair commands để rebuild hoặc purge orphan data

### 5. Introduce tombstone-friendly deletes

Thay vì dựa vào `ON DELETE CASCADE`, node delete nên đi theo hướng:

- soft delete node source
- emit delete/tombstone event
- worker cleanup graph/vector/FTS projections
- orphan cleanup job dọn nốt các relationship/vector records còn sót

Mô hình này phù hợp hơn với projection system lớn vì không khóa chặt lifecycle vào constraint của một DB duy nhất.

### 6. Standardize SQL scripts and migration structure

Để rollout các thay đổi nặng ở write path an toàn, repo cần một contract thống nhất cho SQL và migration.

Nguyên tắc chuẩn hóa:

- migration schema change phải tách biệt khỏi migration backfill lớn
- script verify và script repair không trộn vào file migration chính
- mọi DDL có chủ đích rerun phải idempotent khi có thể
- rollback boundary phải rõ: cái gì rollback bằng `down.sql`, cái gì chỉ repair-forward

## SQL and migration standardization

### Migration taxonomy

Đề xuất chuẩn hóa thành 4 nhóm artifact:

1. `migrations/*` cho schema migration chính thức
2. `scripts/sql/verify/*` cho câu SQL kiểm tra sau migrate
3. `scripts/sql/backfill/*` cho backfill hoặc reconcile chạy có chủ đích
4. `scripts/sql/repair/*` cho cleanup/orphan repair/rebuild hỗ trợ vận hành

### Migration rules

- mỗi migration chỉ nên có một mục tiêu chính:
  - schema create/alter/drop
  - index management
  - constraint change
  - data shape preparation nhỏ
- không nhồi backfill dài, scan toàn bảng lớn, hay repair logic phức tạp vào cùng migration DDL
- ưu tiên `IF EXISTS` / `IF NOT EXISTS` khi engine hỗ trợ và semantics an toàn
- comment ngắn trong SQL cho các bước không hiển nhiên, đặc biệt khi drop FK hoặc đổi write model

### Naming and ordering

- tiếp tục dùng prefix số thứ tự tăng dần như hiện tại
- tên migration phải phản ánh intent rollout, ví dụ:
  - `drop_hot_path_foreign_keys`
  - `add_bulk_write_support_indexes`
  - `prepare_projection_batch_claim_columns`
- script ngoài `migrations/` phải có prefix động từ rõ ràng:
  - `verify_*`
  - `backfill_*`
  - `repair_*`
  - `rebuild_*`

### Transaction and lock policy

- migration DDL cần ưu tiên thời gian lock ngắn
- index lớn hoặc backfill lớn phải được tách chiến lược rollout riêng thay vì gộp một phát
- với thay đổi write-hot tables, rollout ưu tiên nhiều pha:
  1. add-compatible
  2. dual-read/dual-write nếu cần
  3. backfill/reconcile
  4. drop old constraint/index/path

### Constraint and FK rollout policy

Khi bỏ FK hoặc relationship constraint trên bảng nóng:

- migration chỉ nên xử lý việc drop constraint và tạo index thay thế cần thiết
- kiểm tra orphan trước/sau rollout nằm ở `verify` scripts
- cleanup orphan hoặc data repair nằm ở `repair/backfill` scripts

Điều này giúp migration chính giữ deterministic behavior và tránh bị kéo dài bởi data-quality work.

### Verification contract

Mỗi thay đổi schema quan trọng nên có SQL verify tương ứng để xác nhận:

- constraint/index expected đã đúng
- không còn FK ngoài danh sách được giữ lại
- orphan counts nằm trong ngưỡng chấp nhận hoặc bằng 0
- query plan cơ bản của write/read hot path không regress nặng

### Forward-only vs reversible changes

- không phải migration nào cũng nên cố ép có `down.sql` đối xứng hoàn toàn
- các thay đổi như drop FK, rebuild projection, hay cleanup orphan nên được ghi rõ là:
  - reversible bằng `down.sql`, hoặc
  - forward-only và rollback bằng repair/recreate script

Spec này ưu tiên honesty hơn là tạo `down.sql` hình thức nhưng không thực sự an toàn.

## Data model direction

### Source-of-truth tables

- `kg_nodes`: giữ vai trò canonical node store
- `kg_relationships`: giữ edge source records nhưng không bắt FK nóng tới `kg_nodes`
- `kg_outbox_events`: append-only event log, cần tối ưu index theo `status`, `created_at`, có thể thêm `id`/`partition key`
- `kg_projection_versions`: ledger cho sync state, phục vụ monitor và reconcile

### Projection tables

- `kg_vector_documents` là projection table, không nên bị buộc lifecycle chặt bằng FK với `kg_nodes`
- graph backend cũng là projection ngoài Postgres, nên consistency model phải đồng nhất với vector projection: eventual consistency + reconciliation

## Integrity model after dropping FKs

### Write-time checks

- node phải tồn tại khi tạo relationship
- domain/node_type/rel_type vẫn được validate ở service layer
- duplicate detection vẫn dùng unique/index/app logic

### Async correctness checks

- periodic orphan scan cho `kg_relationships`
- periodic orphan scan cho `kg_vector_documents`
- projection-vs-source reconciliation dựa vào `kg_projection_versions`
- dead-letter handling và replay cho outbox events

### Operational safety

- metric cho outbox backlog
- metric cho graph lag/vector lag
- metric cho orphan counts
- script/job để rebuild projection từ source tables

## Migration strategy

### Phase 1

- thêm bulk APIs và bulk repository primitives
- cập nhật `examples/codegraph` sang batch mode

### Phase 2

- thêm worker batch claim + worker pool + backend batch adapters
- bỏ `Flush` per document ở Milvus

### Phase 3

- migration drop hot-path FKs
- thêm orphan/reconcile/repair jobs
- cập nhật delete semantics từ cascade-based sang tombstone-driven

### Phase 4

- chuẩn hóa taxonomy cho migration/verify/backfill/repair SQL
- thêm rollout guide cho constraint drops, index changes, và projection-related DB maintenance

## Key decisions

### 1. Postgres vẫn là source-of-truth duy nhất

Change này không chuyển source-of-truth sang graph DB hay vector DB.

### 2. Graph/vector/FTS là projections, không phải synchronous dependencies

Projection chậm không được chặn write throughput ngoài việc tăng outbox lag có thể quan sát được.

### 3. FK không phù hợp cho projection-heavy hot path

Với workload lớn, application-managed integrity + reconcile job phù hợp hơn database-enforced referential integrity trên mọi write.

### 4. Ưu tiên batch semantics hơn single-record correctness loops

Mục tiêu là tối ưu throughput mặc định cho producer lớn như `examples/codegraph`, thay vì chỉ tối ưu cho CRUD nhỏ lẻ.

### 5. SQL migration phải tách schema rollout khỏi data repair

Điều này giữ migration ngắn, dễ audit, và an toàn hơn khi rollout trên dữ liệu lớn.
