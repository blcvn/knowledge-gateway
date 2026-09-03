# TASK-ORCH-004 — Checkpoint/Sentinel/Sketch HTTP Handlers

| Field | Value |
|---|---|
| **Task ID** | TASK-ORCH-004 |
| **Wave** | 3 |
| **Solution** | [SOL-ORCH-001](../solutions/SOL-ORCH-001-Checkpoints-Sentinels.md) §6 |
| **Component** | `gateway/adapter/handler/orchestration_handler.go` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-ORCH-002, TASK-ORCH-003 |
| **Estimated** | 2h |

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** HTTP/gRPC handler for checkpoint endpoints not in gateway router
---

## Mục tiêu

Thêm checkpoint/sentinel/sketch endpoints vào orchestration handler.

---

## Công việc cụ thể

### 1. Sửa `gateway/adapter/handler/orchestration_handler.go` [MODIFY]

Thêm các methods:
- `POST /v1/orchestration/checkpoints` → `checkpointUC.Create()`
- `GET /v1/orchestration/checkpoints/{id}` → `checkpointUC.GetStatus()` (agent polls)
- `POST /v1/orchestration/checkpoints/{id}/approve` → `checkpointUC.Resolve(approved=true)`
- `POST /v1/orchestration/checkpoints/{id}/reject` → `checkpointUC.Resolve(approved=false)`
- `POST /v1/orchestration/sentinels` → `sentinelUC.Create()`
- `GET /v1/orchestration/sentinels` → `sentinelUC.List()`
- `DELETE /v1/orchestration/sentinels/{id}` → `sentinelUC.Remove()`
- `POST /v1/orchestration/sketches` → `sketchUC.Create()`
- `POST /v1/orchestration/crystals` → `sketchUC.Crystallize()`

### 2. Routes trong `gateway/adapter/handler/router.go` [MODIFY]

```go
// Checkpoints
r.Post("/v1/orchestration/checkpoints", orchHandler.CreateCheckpoint)
r.Get("/v1/orchestration/checkpoints/{id}", orchHandler.GetCheckpoint)
r.Post("/v1/orchestration/checkpoints/{id}/approve", orchHandler.ApproveCheckpoint)
r.Post("/v1/orchestration/checkpoints/{id}/reject", orchHandler.RejectCheckpoint)

// Sentinels
r.Post("/v1/orchestration/sentinels", orchHandler.CreateSentinel)
r.Get("/v1/orchestration/sentinels", orchHandler.ListSentinels)
r.Delete("/v1/orchestration/sentinels/{id}", orchHandler.DeleteSentinel)

// Sketch/Crystal
r.Post("/v1/orchestration/sketches", orchHandler.CreateSketch)
r.Post("/v1/orchestration/crystals", orchHandler.CrystallizeSketch)
```

---

## Acceptance Criteria

- [ ] `POST /v1/orchestration/checkpoints` → 202 + checkpoint_id
- [ ] Agent polls `GET /{id}` → returns pending/approved/rejected
- [ ] Human approve → agent poll returns approved within 1s
- [ ] `POST /v1/orchestration/sentinels` → sentinel created, NATS sub active
- [ ] Sketch create → 201, crystal → 200 + memory stored

## Files

```
gateway/adapter/handler/orchestration_handler.go  [MODIFY]
gateway/adapter/handler/router.go                 [MODIFY]
```
