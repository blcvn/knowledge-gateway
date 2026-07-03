# Design: Sync Session — Single Event Per Upsert Run

## Overview

Ba mảng thay đổi độc lập, được ghép thành một change:

1. **SyncSession lifecycle** — open/write/commit/abandon với GraphVersion dùng chung
2. **Durable scope lease per graph scope** — ngăn concurrent sync race condition
3. **Batch embedding trong handleGraphVersionEvent** — projection worker xử lý large
   session hiệu quả

### Nguyên tắc DB trong change này

- **Không dùng**: unique index phụ trên bảng dữ liệu, FK constraint, advisory lock,
  `SELECT FOR UPDATE`.
- **Chỉ dùng**: PRIMARY KEY trên bảng lease (tối thiểu để lease hoạt động đúng dưới
  concurrent INSERT), DB transaction để đảm bảo atomicity commit/rollback.
- **Code control**: dedup entities, validate session ownership, check lease state — tất cả
  thực hiện ở tầng application trong cùng DB transaction.

---

## 1. SyncSession Lifecycle

### Vòng đời GraphVersion

```
OPEN (client mở session)
  SealGraphVersion(status=PENDING_ENTITIES)
  AcquireScopeLease(graph_scope)
       │
       │  bulk writes gắn graph_version_id
       │  ValidateGraphVersionSession(versionID, owner, graph_scope, status=PENDING_ENTITIES)
       │  AddGraphVersionEntities(versionID, entities...)
       │  (no outbox event)
       │
       ▼
COMMIT (client gọi commit)                ABANDON (client gọi abandon / expiry)
  FinalizeGraphVersion → SEALED             MarkGraphVersion → ABANDONED
  CreateOutboxEvents (1 GRAPH_VERSION_SEALED event)  ReleaseScopeLease
  ReleaseScopeLease                         (no event)
  (same transaction)
```

### SyncSession là alias của GraphVersion

Không cần bảng riêng. `session_id == graph_version_id`. Client giữ `graph_version_id` trả
về từ `OpenSyncSession` và dùng làm session ID cho tất cả API call tiếp theo.

### API endpoints mới

| Method | Path | Mô tả |
|--------|------|--------|
| `POST` | `/v1/kg/write/sync-sessions` | Mở session với `{domain_id, graph_scope}`, trả về `{session_id, graph_version_id, graph_identifier_id, version_number}` |
| `POST` | `/v1/kg/write/sync-sessions/{id}/commit` | Finalize + emit 1 event |
| `DELETE` | `/v1/kg/write/sync-sessions/{id}` | Abandon session, giải phóng scope lease |

### Thay đổi trên bulk endpoints hiện có

`NodeBulkCreateRequest` và `RelationshipBulkCreateRequest` thêm optional field:

```go
type NodeBulkCreateRequest struct {
    Nodes          []NodeCreateRequest `json:"nodes"`
    GraphVersionID string              `json:"graph_version_id,omitempty"`
}

type RelationshipBulkCreateRequest struct {
    Relationships  []RelationshipCreateRequest `json:"relationships"`
    GraphVersionID string                      `json:"graph_version_id,omitempty"`
}
```

**Session mode** (khi `graph_version_id != ""`):
- Ghi nodes/rels vào DB như bình thường
- Validate `graph_version_id` tồn tại, `version_status == PENDING_ENTITIES`, owner match, và
  graph scope của entity đang ghi khớp với session
- Thay vì `sealGraphVersionWithStatus` per-entity → gọi `AddGraphVersionEntities` vào
  version có sẵn
- Relationship deletes thuộc cùng sync interaction cũng phải append vào cùng version với
  `change_kind = DELETE`
- Xóa hoàn toàn việc tạo bridge rel outbox event thừa (bug fix)
- Không tạo outbox event

**Non-session mode** (backward compat, `graph_version_id == ""`):
- Giữ nguyên behavior hiện tại

### Repository interface mới

```go
// Thêm vào Writer interface trong internal/write/repository.go
AddGraphVersionEntities(ctx context.Context, versionID string, entities []GraphVersionEntityRecord) error
AbandonGraphVersion(ctx context.Context, versionID string) error
GetGraphVersionByID(ctx context.Context, versionID string) (GraphVersionRecord, bool)
GetGraphIdentityByID(ctx context.Context, identifierID string) (GraphIdentityRecord, bool)
AcquireScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string, expiresAt time.Time) error
ReleaseScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error
GetScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope string) (ScopeLeaseRecord, bool)
```

### Idempotency khi retry

Khi một batch bị retry, `AddGraphVersionEntities` cần tránh ghi duplicate entity vào
`kg_graph_version_entities`. Dedup được xử lý ở tầng code — không cần unique index trên
bảng:

```go
// AddGraphVersionEntities — code-level dedup (trong cùng DB transaction)
existing := fetchVersionEntitySet(ctx, tx, versionID)
// existing = map[entityKey]struct{}{} where entityKey = (entity_id, entity_kind)

toInsert := make([]GraphVersionEntityRecord, 0, len(entities))
for _, e := range entities {
    if _, dup := existing[entityKey{e.EntityID, e.EntityKind}]; !dup {
        toInsert = append(toInsert, e)
    }
}
if len(toInsert) > 0 {
    batchInsertVersionEntities(ctx, tx, toInsert)  // plain INSERT, không có ON CONFLICT
}
```

`SELECT` fetch existing và `INSERT` gói trong cùng transaction của request. Race giữa hai
request ghi cùng versionID không thể xảy ra vì scope lease đảm bảo tại mỗi thời điểm chỉ
có một session active trên mỗi `graph_scope`.

---

## 2. Durable Scope Lease Per Graph Scope

### Vấn đề

Hai sync session mở song song trên cùng `graph_scope` (cùng project/tenant/app):
- Cả hai `SealGraphVersion` thành công vì PostgreSQL `INSERT ... ON CONFLICT DO UPDATE`
  trên `kg_graph_identifiers` serializable
- Nhưng các entity write của session 2 đan xen với session 1 → event cuối của cả hai đều
  chứa không đầy đủ hoặc sai entities

### Giải pháp: Scope Lease

Không dùng PostgreSQL advisory session lock vì lifecycle của session kéo dài qua nhiều HTTP
request, trong khi service hiện tại không giữ cố định cùng DB connection giữa các request.
Thay vào đó, persist một lease record riêng trong DB:

```sql
CREATE TABLE kg_graph_scope_leases (
  owner_tenant_id text NOT NULL,
  owner_app_id text,
  graph_scope text NOT NULL,
  version_id text NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  PRIMARY KEY (owner_tenant_id, owner_app_id, graph_scope)
)
```

Acquire (code-level check, không dùng ON CONFLICT hay advisory lock):

```go
// AcquireScopeLease — trong một DB transaction
tx.Begin()
lease, found := SELECT FROM kg_graph_scope_leases
    WHERE owner_tenant_id = ? AND owner_app_id IS NOT DISTINCT FROM ? AND graph_scope = ?
if found {
    if lease.ExpiresAt.After(now) && lease.VersionID != versionID {
        tx.Rollback()
        return ErrScopeLocked   // active lease thuộc session khác
    }
    // lease hết hạn: xóa trước khi insert mới
    DELETE FROM kg_graph_scope_leases WHERE owner_tenant_id=? AND ...
}
err := INSERT INTO kg_graph_scope_leases (owner_tenant_id, owner_app_id, graph_scope, version_id, expires_at, ...)
if isPKViolation(err) {
    // Hai instance cùng pass SELECT check và race nhau — instance kia thắng
    tx.Rollback()
    return ErrScopeLocked
}
tx.Commit()
```

### Multi-instance behavior

Trong môi trường HA (nhiều service instance):

- **SELECT (fast path)**: Kiểm tra sớm để trả `ErrScopeLocked` ngay mà không cần thử INSERT.
  Không phải serialization point — chỉ là short-circuit tối ưu.
- **INSERT + PK violation (true serialization point)**: PostgreSQL đảm bảo tại mức row lock
  rằng chỉ một INSERT thành công cho mỗi `(tenant, app, scope)`. Instance thua cuộc nhận
  PK violation (`pq error code 23505`) → bắt lỗi này và trả `ErrScopeLocked`.
- **Không cần SELECT FOR UPDATE**: PostgreSQL implicit row lock trong INSERT đủ để serialize.
  SELECT FOR UPDATE sẽ lock bảng không cần thiết và giảm throughput.

Release:
- `DELETE ... WHERE owner_tenant_id=? AND owner_app_id IS NOT DISTINCT FROM ? AND graph_scope=? AND version_id=?`
- commit/abandon/expiry đều release bằng DB row delete trong cùng transaction cập nhật version status
- `version_id` trong WHERE tránh một instance release nhầm lease của instance khác

**MemoryStore (test double)**: `sync.Mutex` + `map[string]string` với key = `graph_scope`,
value = `versionID` active. Mutex đảm bảo check-then-set atomic trong process của test.
Production dùng PostgreSQL repository — lease được persist trong `kg_graph_scope_leases`,
multi-instance safe qua PK + INSERT race handling như mô tả ở trên.

### Scope lease chỉ áp dụng cho SyncSession

Single-node upsert và single-rel create không cần lease vì chúng không share GraphVersion —
chúng ghi nguyên tử và emit event ngay lập tức. Race condition giữa single upsert và sync
session được giải quyết tự nhiên: mỗi single-node write tạo GraphVersion riêng của mình và
không phụ thuộc vào session GraphVersion đang mở.

### Commit phải nguyên tử và idempotent

`CommitSyncSession` phải chạy trong **một transaction duy nhất**:

1. `GetGraphVersionByID(versionID)` — verify version tồn tại, owner match
2. `GetScopeLease(tenant, app, scope)` — verify lease còn thuộc về đúng `version_id`
3. `FinalizeGraphVersion(versionID)` — `UPDATE ... SET status='SEALED' WHERE version_id=? AND status='PENDING_ENTITIES'`; **check `rowsAffected`**
4. `CreateOutboxEvents([{EventType: GRAPH_VERSION_SEALED, ...}])`
5. `ReleaseScopeLease(...)`

**Xử lý `rowsAffected = 0` tại bước 3** (quan trọng cho HA):

```go
rowsAffected, err := FinalizeGraphVersion(ctx, tx, versionID)
if rowsAffected == 0 {
    // Version không còn ở trạng thái PENDING_ENTITIES — có thể do:
    // (a) instance khác đã commit → SEALED: idempotent success
    // (b) cleanup job đã abandon → ABANDONED: error
    current, _ := GetGraphVersionByID(ctx, tx, versionID)
    if current.Status == "SEALED" {
        tx.Rollback()
        return nil  // commit đã xảy ra, idempotent
    }
    tx.Rollback()
    return ErrSessionAbandoned
}
// rowsAffected == 1: commit bình thường, tiếp tục bước 4-5
```

Nếu transaction rollback thì version vẫn `PENDING_ENTITIES` và client có thể retry commit.

### Toàn bộ mutation của một interaction phải nằm trong cùng session

Để giữ invariant `1 interaction = 1 version change = 1 outbox event`, mọi mutation mà
`examples/codegraph` quyết định áp dụng cho cùng graph scope trong một lần reconcile phải đi qua
cùng `graph_version_id`. Điều này bao gồm:

- node create/update;
- relationship create/update;
- stale relationship deletes phát hiện trong `reconcileRelationships`.

`reconcileRelationships` vì vậy không được gọi legacy `DeleteRelationshipsBulk` phát event
riêng ngoài session. Thay vào đó cần một session-aware delete path:

- mở rộng `RelationshipBulkDeleteRequest` nhận `graph_version_id`, hoặc
- thêm endpoint write riêng để delete relationships vào session hiện tại.

Delete path trong session mode SHALL:
- ghi delete/soft-delete vào source DB;
- append `GraphVersionEntityRecord{EntityKind: "relationship", ChangeKind: "DELETE"}`;
- không tạo outbox event riêng;
- chỉ chờ event duy nhất `GRAPH_VERSION_SEALED` ở bước commit session.

---

## 3. Batch Embedding Trong handleGraphVersionEvent

### Vấn đề hiện tại

`handleGraphVersionEvent` (runtime.go:2160) xử lý entities tuần tự:

```go
for _, entity := range entities {
    case "node":
        graphSynced, vectorSynced, err := r.projectNode(node)  // embed 1 node/lần
```

`projectNode` gọi `embeddingProvider.Embed(ctx, text)` một lần cho một node. Với 1000 node
trong 1 session event, đây là 1000 sequential embed calls.

### Giải pháp: Batch Embed Path cho GraphVersion Events

Refactor `handleGraphVersionEvent` để dùng cùng batch infrastructure của
`projectCoalescedUnits`:

```
handleGraphVersionEvent(event)
  │
  ├─ GetGraphVersionEntities(versionID)   → []entity
  │
  ├─ group entities by kind
  │    nodeEntities = [node1, node2, ..., nodeN]
  │    relEntities  = [rel1, rel2, ..., relM]
  │
  ├─ batch load: store.GetNodesByIDs(nodeIDs) → map
  │
  ├─ Build []nodeProjectionWork (same as projectCoalescedUnits)
  │    để tính embedding text cho từng node
  │
  ├─ Batch embed: embeddingProvider.EmbedBatch(ctx, texts[]) → []embedding
  │    (tối đa embedBatchSize() texts/call, chunked)
  │
  ├─ applyNodeProjectionWork(nodeUpserts, false, results)   ← existing batch path
  ├─ applyRelationshipProjectionWork(relUpserts, false, results)
  │
  └─ commitAllProjectionResults(results)
       advanceGraphProjectionHead(identifierID, "graph", ...)
       advanceGraphProjectionHead(identifierID, "vector", ...)
```

**EmbedBatch interface** — thêm vào `vector.EmbeddingProvider`:

```go
type EmbeddingProvider interface {
    Embed(ctx context.Context, text string) ([]float64, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)  // mới
    ModelID() string
}
```

Default implementation: `EmbedBatch` gọi `Embed` tuần tự (backward compat). Các provider
HTTP có thể override để gọi batch endpoint.

**Chunk size**: dùng `r.embedBatchSize()` (env `EMBED_BATCH_SIZE`, default 16) — đã có sẵn
trong runtime.

---

## 4. examples/codegraph Changes

### SyncSession trong reconcile

`SyncProject` (bridge/sync.go) wrap toàn bộ reconcile trong một session:

```go
func SyncProject(ctx, cfg, dryRun, fullReindex) (SyncReport, error) {
    // ... setup ...

    session, err := client.OpenSyncSession(ctx, OpenSyncSessionRequest{
        DomainID:   cfg.KGDomainID,
        GraphScope: "project:" + cfg.ProjectID,
    })
    if err != nil { return err }

    committed := false
    defer func() {
        if !committed {
            _ = client.AbandonSyncSession(ctx, session.SessionID)
        }
    }()

    if err := reconcileNodes(ctx, client, cfg, nodes, &state, &report, session.GraphVersionID); err != nil {
        return err  // defer sẽ abandon
    }
    if err := reconcileRelationships(ctx, client, cfg, edges, &state, &report, session.GraphVersionID); err != nil {
        return err
    }

    if err := client.CommitSyncSession(ctx, session.SessionID); err != nil {
        return err
    }
    committed = true
}
```

`reconcileNodes` và `reconcileRelationships` nhận thêm `graphVersionID string` và truyền
vào từng `NodeBulkCreateRequest` / `RelationshipBulkCreateRequest`, đồng thời stale
relationship deletes cũng phải dùng cùng `graphVersionID`.

### KGServiceClient interface mới

```go
type KGServiceClient interface {
    // ... existing ...
    OpenSyncSession(ctx context.Context, req OpenSyncSessionRequest) (SyncSessionResponse, error)
    CommitSyncSession(ctx context.Context, sessionID string) error
    AbandonSyncSession(ctx context.Context, sessionID string) error
}
```

---

## 5. Expiry / Cleanup

Một background goroutine (hoặc periodic job) quét `kg_graph_versions WHERE version_status =
'PENDING_ENTITIES' AND created_at < NOW() - INTERVAL '2 hours'`, đánh dấu chúng
`ABANDONED` và xóa lease row tương ứng trong `kg_graph_scope_leases`.

Interval và threshold cấu hình qua env:
- `SYNC_SESSION_EXPIRY_HOURS` (default 2)
- `SYNC_SESSION_CLEANUP_INTERVAL_MINUTES` (default 30)

---

## 6. Không Thay Đổi

- Outbox event format cho single-node/single-rel write path
- Worker PollOnce flow (graphEvents / legacyEvents partition)
- Reconciliation logic
- `PROJECTION_BATCHING_ENABLED` flag
- Existing `coalesceOutboxEvents` logic

---

## Risks And Mitigations

| Risk | Mitigation |
|------|-----------|
| Scope lease bị orphaned nếu process crash | Cleanup job + `PENDING_ENTITIES` timeout |
| Batch embed API không available trên mọi provider | Default fallback sequential embed; interface backward compat |
| `handleGraphVersionEvent` với 5000 entities quá chậm | Batch embed + applyNodeProjectionWork đã có chunk loop |
| Concurrent single-node write và sync session trên cùng scope | Vẫn safe: single write dùng per-node GraphVersion, không phụ thuộc session GraphVersion |
| Retry batch ghi duplicate entity vào `kg_graph_version_entities` | Code-level dedup trong `AddGraphVersionEntities`: SELECT existing set trước khi INSERT — không cần unique index; scope lease đảm bảo không có race giữa hai goroutine trên cùng versionID |
| Hai HA instance race `AcquireScopeLease` — cả hai pass SELECT check | INSERT PK violation được bắt và map sang `ErrScopeLocked`; instance thua cuộc nhận 409, không lọt vào trạng thái dual-session |
| Hai HA instance commit cùng session_id — chỉ một nên phát event | `FinalizeGraphVersion` dùng `UPDATE WHERE status=PENDING_ENTITIES` + check `rowsAffected`: instance thứ hai nhận 0 rows → SELECT lại status → SEALED = idempotent return; ABANDONED = error |
