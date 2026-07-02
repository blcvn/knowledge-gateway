# Tasks

## Milestone: Schema & Repository

- [x] **S1** — Migration SQL tạo bảng `kg_graph_scope_leases`.
  Yêu cầu:
  - PRIMARY KEY `(owner_tenant_id, owner_app_id, graph_scope)` — đây là constraint DB
    **duy nhất** của toàn bộ change này;
  - không thêm unique index hay foreign key trên bất kỳ bảng dữ liệu nào
    (`kg_graph_version_entities`, `kg_graph_versions`, v.v.);
  - không thêm index phụ trên `kg_graph_scope_leases` — bảng có số row = số session active
    (rất nhỏ), full scan trong cleanup là chấp nhận được;
  - các field cần có: `version_id text`, `expires_at timestamptz`, `created_at timestamptz`,
    `updated_at timestamptz`.

- [x] **S3** — Thêm các method mới vào `Writer` interface trong
  `internal/write/repository.go`:
  `AddGraphVersionEntities`, `AbandonGraphVersion`, `GetGraphVersionByID`,
  `GetGraphIdentityByID`, `AcquireScopeLease`, `ReleaseScopeLease`, `GetScopeLease`.
  Yêu cầu:
  - giữ interface tối giản — không expose DB lease internals cụ thể ra ngoài;
  - các method phải có signature nhận `context.Context` để cancel được.

- [x] **S4** — Implement các method mới trong `internal/platform/postgres/repository.go`.
  Yêu cầu:
  - `AddGraphVersionEntities`: dedup ở tầng code — SELECT existing `(entity_id, entity_kind)`
    cho `versionID` trong cùng transaction, lọc ra entities chưa có, sau đó plain INSERT
    (không dùng `ON CONFLICT`); không cần unique index trên bảng;
  - `AcquireScopeLease`: trong một transaction, SELECT lease hiện tại; nếu tồn tại và
    chưa hết hạn và `version_id` khác → trả `ErrScopeLocked`; nếu tồn tại nhưng đã hết
    hạn → DELETE trước; INSERT lease mới; **bắt PK violation (`pq` error code `23505`)
    từ INSERT và trả `ErrScopeLocked`** — đây là điểm serialization thực sự trong HA;
  - `ReleaseScopeLease`: DELETE lease row theo `(owner, app, scope, version_id)`;
  - `GetGraphVersionByID`: SELECT trên `kg_graph_versions`;
  - `GetGraphIdentityByID`: SELECT trên `kg_graph_identifiers`.

- [x] **S5** — Implement các method mới trong `internal/write/store.go` (MemoryStore).
  Yêu cầu:
  - `AcquireScopeLease`: dùng `sync.Mutex` + `map[string]string` với key = `graph_scope`,
    value = `versionID`; kiểm tra trong critical section, trả `ErrScopeLocked` nếu đã tồn
    tại versionID khác (không dùng `sync.Map` vì cần read-check-write atomic);
  - `ReleaseScopeLease`: trong critical section, xóa entry chỉ khi value == versionID của
    caller — tránh release nhầm lease của session khác;
  - `AddGraphVersionEntities`: trong critical section, dedup `(entity_id, entity_kind)` theo
    set trong memory trước khi append vào `graphVersionEntities[versionID]`.

## Milestone: Service Layer

- [x] **W1** — Thêm method `OpenSyncSession` vào `internal/write/service.go`.
  Yêu cầu:
  - validate actor có write permission trên domain;
  - request phải nhận explicit `GraphScope string` thay vì tự derive mơ hồ từ domain;
  - gọi `AcquireScopeLease` và `SealGraphVersion` trong cùng transaction;
  - nếu lease thất bại → trả `ErrScopeLocked` (HTTP 409);
  - `SealGraphVersion` với `VersionStatus = "PENDING_ENTITIES"`, không có entities ban đầu;
  - trả `SyncSessionResponse` gồm `session_id = graph_version_id`.

- [x] **W2** — Thêm method `CommitSyncSession` vào `internal/write/service.go`.
  Yêu cầu:
  - verify `GraphVersion` tồn tại và `version_status == "PENDING_ENTITIES"`;
  - verify actor là owner của version (tenant + app match) và `graph_scope` lấy từ
    `GraphIdentity` của version;
  - verify active lease của scope vẫn thuộc về chính `version_id`;
  - `FinalizeGraphVersion`, `CreateOutboxEvents`, và `ReleaseScopeLease` phải chạy trong cùng
    transaction;
  - `FinalizeGraphVersion` dùng `UPDATE WHERE status='PENDING_ENTITIES'` và **phải check
    `rowsAffected`**: nếu 0 row → SELECT lại status của version; nếu `SEALED` → return nil
    (idempotent, instance khác đã commit); nếu `ABANDONED` → return `ErrSessionAbandoned`;
  - emit đúng 1 `OutboxEvent` kiểu `GRAPH_VERSION_SEALED` với payload đủ để
    `hasGraphVersionMetadata` trả `true` (`graph_version_id`, `graph_identifier_id`,
    `graph_version_number`);
  - commit phải idempotent: hai HA instance commit cùng session_id → chỉ một phát event.

- [x] **W3** — Thêm method `AbandonSyncSession` vào `internal/write/service.go`.
  Yêu cầu:
  - gọi `AbandonGraphVersion` → ABANDONED và `ReleaseScopeLease` trong cùng transaction;
  - không emit event.

- [x] **W4** — Sửa `CreateNodesBulkWithContext` để hỗ trợ session mode.
  Yêu cầu:
  - nếu `req.GraphVersionID != ""`: ghi nodes + bridge rels vào DB, gọi
    `AddGraphVersionEntities` với entity kind `"node"` và `"embeddable_relationship"`, không
    tạo outbox event nào;
  - trước khi append entities phải verify session version tồn tại, `PENDING_ENTITIES`, owner
    match, và `graph_scope` của node khớp với session;
  - xóa hoàn toàn việc tạo bridge rel `RELATIONSHIP_UPSERTED` events thừa trong session mode
    (fix bug duplicate sync);
  - nếu `req.GraphVersionID == ""`: giữ nguyên behavior hiện tại.

- [x] **W5** — Sửa `CreateRelationshipsBulkWithContext` để hỗ trợ session mode.
  Yêu cầu:
  - nếu `req.GraphVersionID != ""`: ghi rels vào DB, gọi `AddGraphVersionEntities` với entity
    kind `"relationship"`, không tạo outbox event;
  - trước khi append entities phải verify session version tồn tại, `PENDING_ENTITIES`, owner
    match, và `graph_scope` của relationship khớp với session;
  - nếu `req.GraphVersionID == ""`: giữ nguyên behavior.

- [x] **W5b** — Sessionize stale relationship delete path.
  Yêu cầu:
  - delete path dùng bởi `codegraph-sync` phải nhận `graph_version_id` khi chạy trong
    session mode;
  - delete/soft-delete source rows và append graph-version entity với `change_kind = DELETE`
    phải xảy ra mà không phát outbox event riêng;
  - invariant phải giữ: một sync interaction của cùng graph scope chỉ tạo đúng một
    `GRAPH_VERSION_SEALED` event tại `CommitSyncSession`.

- [x] **W6** — Thêm các type mới vào `internal/write/types.go`.
  Yêu cầu:
  - `OpenSyncSessionRequest { DomainID string; GraphScope string }`;
  - `SyncSessionResponse { SessionID, GraphVersionID, GraphIdentifierID string; GraphVersionNumber int64 }`;
  - thêm field `GraphVersionID string` vào `NodeBulkCreateRequest`;
  - thêm field `GraphVersionID string` vào `RelationshipBulkCreateRequest`.

## Milestone: HTTP Handler

- [x] **H1** — Thêm 3 handler mới vào `internal/write/handler.go` và wiring vào router.
  Yêu cầu:
  - `POST /v1/kg/write/sync-sessions` → `OpenSyncSession`; trả HTTP 202 với body
    `SyncSessionResponse`;
  - `POST /v1/kg/write/sync-sessions/{id}/commit` → `CommitSyncSession`; trả HTTP 200;
  - `DELETE /v1/kg/write/sync-sessions/{id}` → `AbandonSyncSession`; trả HTTP 204;
  - `ErrScopeLocked` → HTTP 409 với error code `SYNC_SCOPE_LOCKED`.

## Milestone: Worker — Batch Embedding

- [x] **E1** — Thêm method `EmbedBatch` vào `vector.EmbeddingProvider` interface.
  Yêu cầu:
  - signature: `EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)`;
  - default implementation trong `DeterministicProvider` và `DirectRouter`: gọi `Embed`
    tuần tự (backward compat);
  - HTTP embedding provider có thể override để dùng batch endpoint nếu available.

- [x] **E2** — Refactor `handleGraphVersionEvent` trong `internal/workers/runtime.go` để dùng
  batch projection path.
  Yêu cầu:
  - load tất cả node/rel entities từ `GetGraphVersionEntities(versionID)`;
  - batch load nodes bằng `store.GetNodesByIDs(nodeIDs)`;
  - build `[]nodeProjectionWork` cho tất cả nodes (cùng struct đã dùng trong
    `projectCoalescedUnits`);
  - batch embed: nhóm embedding texts theo `embedBatchSize()`, gọi `EmbedBatch` cho mỗi
    chunk;
  - gọi `applyNodeProjectionWork` và `applyRelationshipProjectionWork` với full batches;
  - gọi `commitProjectionResult` cho từng result;
  - gọi `advanceGraphProjectionHead` cho "graph" và "vector" sau khi tất cả thành công.

- [x] **E3** — Thêm expiry cleanup trong `internal/workers/runtime.go` (hoặc background
  goroutine riêng).
  Yêu cầu:
  - scan `kg_graph_versions WHERE version_status = 'PENDING_ENTITIES' AND created_at <
    NOW() - INTERVAL '2 hours'`;
  - đánh dấu ABANDONED và xóa scope lease tương ứng cho từng version;
  - chạy định kỳ theo `SYNC_SESSION_CLEANUP_INTERVAL_MINUTES` (default 30);
  - log số lượng session được cleanup.

## Milestone: codegraph-sync

- [x] **C1** — Thêm 3 method mới vào `KGServiceClient` interface và `Client` struct trong
  `codegraph-sync/internal/bridge/kgservice.go`.
  Yêu cầu:
  - `OpenSyncSession(ctx, OpenSyncSessionRequest) (SyncSessionResponse, error)` →
    `POST /v1/kg/write/sync-sessions`;
  - `CommitSyncSession(ctx, sessionID string) error` →
    `POST /v1/kg/write/sync-sessions/{id}/commit`;
  - `AbandonSyncSession(ctx, sessionID string) error` →
    `DELETE /v1/kg/write/sync-sessions/{id}`.

- [x] **C2** — Sửa `SyncProject` trong `codegraph-sync/internal/bridge/sync.go` để dùng
  session.
  Yêu cầu:
  - mở session trước khi reconcile;
  - `defer` abandon session nếu chưa commit;
  - truyền explicit `GraphScope = "project:" + cfg.ProjectID` khi open session;
  - truyền `graphVersionID` vào `reconcileNodes` và `reconcileRelationships`;
  - stale relationship deletes trong `reconcileRelationships` phải đi qua session-aware delete
    path thay vì legacy `DeleteRelationshipsBulk` phát event riêng;
  - gọi `CommitSyncSession` sau khi cả hai reconcile thành công;
  - nếu `OpenSyncSession` trả 409 `SYNC_SCOPE_LOCKED` → trả lỗi rõ ràng (không retry tự
    động, để operator xử lý).

- [x] **C3** — Sửa `reconcileNodes` và `reconcileRelationships` nhận thêm tham số
  `graphVersionID string`.
  Yêu cầu:
  - truyền `graphVersionID` vào mỗi `NodeBulkCreateRequest` / `RelationshipBulkCreateRequest`;
  - xử lý partial failure đúng: nếu một batch fail sau max retry → return error (session sẽ
    được abandon bởi defer trong `SyncProject`).

## Milestone: Tests

- [x] **T1** — Unit test cho `OpenSyncSession`, `CommitSyncSession`, `AbandonSyncSession`
  trên MemoryStore.
  Yêu cầu:
  - happy path: open → bulk write (session mode) → commit → verify 1 outbox event với
    graph_version_id đúng;
  - abandon path: open → bulk write → abandon → verify 0 outbox events;
  - concurrent open trên cùng scope → session thứ hai nhận `ErrScopeLocked`;
  - commit non-existent session → `ErrNotFound`;
  - double commit: commit lần 2 trên cùng session (mô phỏng HA retry) → idempotent return
    nil, không tạo event thứ hai;
  - commit sau abandon → `ErrSessionAbandoned`.

- [x] **T2** — Unit test bulk write trong session mode.
  Yêu cầu:
  - `CreateNodesBulkWithContext` với `GraphVersionID` set → entities được add vào version,
    không có outbox event;
  - bridge rels trong session mode không tạo RELATIONSHIP_UPSERTED event thừa;
  - retry của cùng batch (same node IDs) không tạo duplicate entities trong version.

- [x] **T3** — Unit test `handleGraphVersionEvent` với `GRAPH_VERSION_SEALED` event.
  Yêu cầu:
  - event với N entities → `applyNodeProjectionWork` được gọi với đúng N nodes;
  - `EmbedBatch` được gọi thay vì N lần `Embed` riêng lẻ;
  - `advanceGraphProjectionHead` được gọi sau khi projection xong.

- [x] **T4** — Integration test end-to-end sync session.
  Yêu cầu:
  - open session → gửi create/update/delete mutations của cùng graph interaction → commit →
    worker poll → verify projection sync hoàn thành với đúng 1 outbox event;
  - so sánh projection state (graph + vector) với expected state.

## Parity Follow-up Notes

- [x] **P1** — Sessionize full node mutation parity trong `internal/write/service.go`.
  Trạng thái hiện tại:
  - `CreateNodesBulkWithContext(..., GraphVersionID)` đã chạy ở session mode mà không phát outbox trước commit;
  - `UpdateNodeWithContext` và `DeleteNodeWithContext` đã chuyển sang session-aware path;
  - mọi node mutation trong cùng sync interaction giờ chỉ append `GraphVersionEntities` cho đến lúc `CommitSyncSession`.

- [x] **P2** — Hoàn thiện `codegraph-sync` cho full create/update/delete parity trong cùng session.
  Trạng thái hiện tại:
  - `SyncProject` truyền `graphVersionID` cho node create/update/delete và stale relationship delete;
  - stale node delete cũng đi qua session-aware delete path;
  - invariant “1 interaction = 1 `GRAPH_VERSION_SEALED` event” đã được giữ end-to-end.

- [x] **P3** — Hardening expiry cleanup để an toàn hơn ở production/HA.
  Trạng thái hiện tại:
  - cleanup đã đi qua helper atomic `CleanupExpiredSyncSession`;
  - version được `ABANDONED` và scope lease được gỡ trong cùng flow;
  - thêm test cho session cleanup để tránh regression.

- [x] **P4** — Chỉ đóng `T4` sau khi verify full parity thật.
  Trạng thái hiện tại:
  - `T4` đã được bổ sung và pass cùng các test trọng điểm khác;
  - parity end-to-end đã được verify bằng compile + targeted tests trong workers, write, postgres, và bridge.
