# overlay-service — Overlay Graph Service

> **Role:** Quản lý Overlay Graphs — cho phép app tạo "nhánh làm việc" tạm thời trên graph, thực hiện thay đổi thử nghiệm, rồi commit hoặc discard mà không ảnh hưởng đến main graph.

---

## 1. Trách Nhiệm (Single Responsibility)

`overlay-service` chịu trách nhiệm **duy nhất** cho:
- **Overlay Session Management**: Tạo, liệt kê, đóng các overlay sessions
- **Delta Tracking**: Ghi lại mọi thay đổi (create/update/delete entity/edge) trong overlay
- **Commit**: Merge overlay changes vào main graph (gọi `graph-service`)
- **Discard**: Hủy bỏ overlay và xóa tất cả delta
- **Conflict Detection**: Phát hiện conflicts khi commit (version mismatch)
- **Session Expiry**: Auto-expire overlay sessions sau TTL (default 1 giờ)

---

## 2. Concept: Overlay Graph

```
Main Graph (PostgreSQL/Neo4j)
─────────────────────────────
[REQ-001: APPROVED]  [UC-001]  [TC-001]
          └─── HAS_USECASE ───┘

Overlay Session "draft-session-xyz" (Redis, TTL=1h)
─────────────────────────────────────────────────────
Deltas:
  + CREATE: REQ-002 {title: "New Requirement", status: "DRAFT"}
  ~ UPDATE: REQ-001 {priority: "MUST" → "SHOULD"}  ← change tracking
  + CREATE: EDGE (REQ-002) -[HAS_USECASE]→ (UC-001)

View with Overlay Applied:
─────────────────────────────
[REQ-001: APPROVED, priority=SHOULD]  [REQ-002: DRAFT]  [UC-001]  [TC-001]
  └─── HAS_USECASE ──────────────────────────────────┘ └────────── HAS_USECASE ───┘

On Commit: Deltas applied to main graph via graph-service
On Discard: Overlay deleted from Redis, no trace left
```

---

## 3. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         overlay-service                                  │
│                                                                         │
│  gRPC Server (port 9008)                                                │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  OverlayServiceServer                             │   │
│  │                                                                  │   │
│  │  CreateSession()    GetSession()    ListSessions()               │   │
│  │  DiscardSession()                                                │   │
│  │  AddEntityDelta()   AddEdgeDelta()   RemoveDelta()              │   │
│  │  GetDelta()         ListDeltas()                                 │   │
│  │  CommitSession()                    [merge to main graph]        │   │
│  │  ApplyOverlay()                     [read: PG + overlay deltas]  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Overlay Business Logic                          │   │
│  │                                                                  │   │
│  │  OverlayUsecase                                                  │   │
│  │  ├── SessionManager (Redis-backed, TTL=1h)                       │   │
│  │  ├── DeltaTracker (entity/edge changes)                          │   │
│  │  ├── ConflictDetector (version-based)                            │   │
│  │  └── CommitExecutor → gRPC calls to graph-service                │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                               │                                          │
│  ┌────────────────────────────▼────────────────────────────────────┐   │
│  │                   Data Layer                                      │   │
│  │  Redis:   Overlay sessions and deltas (TTL-based)                │   │
│  │  NATS:    Publish session events (created, committed, discarded)  │   │
│  │  gRPC:    graph-service (for read base + commit write)           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Models (Redis)

### 4.1 OverlaySession (Redis Hash)

```
Key: overlay:session:{session_id}
TTL: 1 hour (refreshed on activity)

Fields:
  session_id:  "sess-uuid-xxx"
  app_id:      "ba_agent"
  tenant_id:   "tenant_001"
  name:        "sprint-3-draft"      // Optional label
  status:      "ACTIVE"              // ACTIVE | COMMITTING | COMMITTED | DISCARDED
  created_by:  "user_123"
  created_at:  "2026-06-11T00:00:00Z"
  expires_at:  "2026-06-11T01:00:00Z"
  delta_count: "5"
```

### 4.2 OverlayDelta (Redis Hash per delta)

```
Key: overlay:delta:{session_id}:{entity_id_or_edge_id}
TTL: Same as session

Fields:
  delta_type:   "ENTITY" | "EDGE"
  operation:    "CREATE" | "UPDATE" | "DELETE"
  entity_id:    "ba_agent__Requirement__REQ-002"   // For entities
  edge_id:      "edge-uuid-xxx"                    // For edges
  entity_type:  "Requirement"
  base_version: "1"              // Version in main graph (for conflict detection)
  payload:      '{"title": "New Requirement", ...}'
  created_at:   "..."
```

### 4.3 Delta Index (Redis Set)

```
Key: overlay:deltas:{session_id}
Members: [delta_keys...]
// Used to list all deltas of a session
```

---

## 5. gRPC API

```protobuf
service OverlayService {
  // Session Management
  rpc CreateSession(CreateSessionRequest) returns (OverlaySession);
  rpc GetSession(GetSessionRequest) returns (OverlaySession);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc DiscardSession(DiscardSessionRequest) returns (google.protobuf.Empty);

  // Delta Operations
  rpc AddEntityDelta(AddEntityDeltaRequest) returns (Delta);
  rpc AddEdgeDelta(AddEdgeDeltaRequest) returns (Delta);
  rpc RemoveDelta(RemoveDeltaRequest) returns (google.protobuf.Empty);
  rpc ListDeltas(ListDeltasRequest) returns (ListDeltasResponse);

  // Commit (merge to main graph)
  rpc CommitSession(CommitSessionRequest) returns (CommitResult);

  // Read with overlay applied
  rpc ApplyOverlay(ApplyOverlayRequest) returns (ApplyOverlayResponse);
  // Returns: base entity + overlay deltas merged
}

message CommitResult {
  string session_id = 1;
  int32 entities_committed = 2;
  int32 edges_committed = 3;
  repeated ConflictInfo conflicts = 4; // Empty if no conflicts
  bool success = 5;
}

message ConflictInfo {
  string entity_id = 1;
  string conflict_type = 2;  // "VERSION_MISMATCH" | "ENTITY_DELETED" | "SCHEMA_CHANGED"
  string resolution = 3;     // "SKIPPED" | "OVERWRITTEN" | "MANUAL_REQUIRED"
}
```

---

## 6. Commit Flow

```
CommitSession(session_id):

1. Lock session (Redis SETNX) — prevent concurrent commits
2. Validate: session.status == "ACTIVE"
3. Set session.status = "COMMITTING"
4. Load all deltas from Redis

5. For each delta:
   a. CHECK CONFLICT:
      - Load current entity from graph-service.GetNode(entity_id)
      - Compare base_version vs current entity.version
      - If mismatch: ConflictInfo{type: "VERSION_MISMATCH"}
        → Default resolution: SKIP (configurable to OVERWRITE)

   b. APPLY (if no conflict or resolution=OVERWRITE):
      - For CREATE deltas: graph-service.CreateNode(...)
      - For UPDATE deltas: graph-service.UpdateNode(...)
      - For DELETE deltas: graph-service.DeleteNode(...)
      - For EDGE deltas:   graph-service.CreateEdge/DeleteEdge(...)

6. Set session.status = "COMMITTED"
7. Delete all deltas from Redis (cleanup)
8. Publish NATS: "overlay.session.committed" {session_id, app_id, delta_count}

9. Return CommitResult{success: true, entities_committed, conflicts}
```

---

## 7. Conflict Resolution Strategies

| Strategy | Mô tả | Khi nào dùng |
|----------|-------|-------------|
| `SKIP` (default) | Bỏ qua delta bị conflict, không write | An toàn nhất, preserves main graph |
| `OVERWRITE` | Ghi đè main graph bất kể conflict | Khi overlay luôn đúng |
| `MANUAL` | Abort commit, return conflict list cho app xử lý | Cần human review |

---

## 8. Overlay Read (ApplyOverlay)

App có thể đọc entity với overlay applied:

```go
// ApplyOverlay trả về entity với deltas từ overlay session applied on top:

// Base entity từ main graph (PostgreSQL/Neo4j):
{
  "node_id": "ba_agent__Requirement__REQ-001",
  "properties": { "title": "Login Feature", "priority": "MUST" }
}

// Overlay delta: UPDATE priority "MUST" → "SHOULD"
{
  "operation": "UPDATE",
  "payload": { "priority": "SHOULD" }
}

// ApplyOverlay result:
{
  "node_id": "ba_agent__Requirement__REQ-001",
  "properties": { "title": "Login Feature", "priority": "SHOULD" },
  "is_overlay": true,       // Indicates overlay was applied
  "overlay_session": "sess-xxx"
}
```

---

## 9. Session Auto-Expiry

```
NATS Listener: overlay.session.close
  - Subscribe to session expiry events
  - When Redis key expires (TTL=1h): publish NATS event
  - OverlayService handles cleanup:
    1. Mark session.status = "DISCARDED" (if possible)
    2. Log expiry event
    3. Alert if session had uncommitted deltas
```

---

## 10. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | Scope | Mô tả |
|--------|------|-------|-------|
| POST | `/v1/overlay` | `graph:write` | Tạo overlay session |
| GET | `/v1/overlay` | `graph:read` | List sessions |
| GET | `/v1/overlay/:id` | `graph:read` | Chi tiết session |
| DELETE | `/v1/overlay/:id` | `graph:write` | Discard session |
| POST | `/v1/overlay/:id/deltas/entity` | `graph:write` | Thêm entity delta |
| POST | `/v1/overlay/:id/deltas/edge` | `graph:write` | Thêm edge delta |
| GET | `/v1/overlay/:id/deltas` | `graph:read` | List deltas |
| POST | `/v1/overlay/:id/commit` | `graph:write` | Commit session |

---

## 11. Use Cases

### Use Case 1: Thử nghiệm thay đổi hàng loạt

```
1. BA Agent tạo overlay session "sprint-4-requirements"
2. Thêm 20 requirement mới vào overlay (không ảnh hưởng main graph)
3. Review toàn bộ trong overlay → chạy rule engine trên overlay view
4. Manager approve → CommitSession() → 20 requirements vào main graph
5. Nếu reject → DiscardSession() → không có gì thay đổi
```

### Use Case 2: Multi-user collaboration

```
User A: session "alice-draft"
User B: session "bob-draft"

Cả hai làm việc độc lập trên main graph
Khi commit: ConflictDetector kiểm tra VERSION
Ai commit trước thành công, người còn lại nhận ConflictInfo
```

---

## 12. Configuration

```yaml
# configs/overlay.yaml
overlay_service:
  grpc_port: 9008

  redis:
    addr: redis:6379
    session_ttl: 1h
    lock_ttl: 30s

  nats:
    addr: nats:4222
    subjects:
      publish:
        session_created: overlay.session.created
        session_committed: overlay.session.committed
        session_discarded: overlay.session.discarded

  dependencies:
    graph_service: graph-service:9003

  commit:
    default_conflict_strategy: SKIP
    max_deltas_per_commit: 10000
    commit_timeout: 5m

  observability:
    metrics_port: 9098
```

---

## 13. Observability

| Metric | Mô tả |
|--------|-------|
| `overlay_sessions_active_total{app_id}` | Số sessions đang active |
| `overlay_sessions_committed_total{app_id}` | Số sessions đã commit |
| `overlay_sessions_discarded_total{app_id}` | Số sessions bị discard/expire |
| `overlay_deltas_total{app_id, operation}` | Số deltas theo loại |
| `overlay_commit_duration_seconds{app_id}` | Thời gian commit |
| `overlay_conflicts_total{app_id, conflict_type}` | Số conflicts phát hiện |
| `overlay_session_expired_with_deltas_total` | Sessions expire có uncommitted deltas |
