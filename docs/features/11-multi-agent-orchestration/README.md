# Feature 11 — Multi-Agent Orchestration

> **Loại:** AgentMemory | **Priority:** High | **Status:** Implemented (CR-AM-004)

## Mô tả

Multi-Agent Orchestration cung cấp infrastructure để nhiều AI Agents cùng làm việc trên shared state mà không conflict. Bao gồm: **distributed leases** (prevent write conflicts), **inter-agent signals** (handoff, alert), **action task graph** (state machine), **routines** (workflow templates), **checkpoints** (human approval gates), và **sentinels** (event watchers).

---

## Business Logic

### Distributed Leases

Khi nhiều agents cùng truy cập shared resource (e.g., một memory slot), leases prevent conflicts:

- Agent A muốn modify resource R → request lease.
- Nếu resource chưa có lease → grant lease (TTL: configurable, e.g., 30s).
- Nếu resource đã có lease (từ Agent B) → Agent A phải chờ hoặc bị reject.
- Lease hết TTL → auto-release (prevent deadlock nếu agent crash).

### Inter-Agent Signals

Agents communicate qua typed signals:
- `handoff`: Agent A chuyển giao task cho Agent B kèm context.
- `alert`: Thông báo khẩn cấp (e.g., lỗi nghiêm trọng cần human review).
- `update`: Thông báo state change.
- `query`: Agent A request information từ Agent B.

Signals được deliver qua NATS JetStream → guaranteed delivery.

### Action Task Graph (State Machine)

Actions là các task có structure, chạy như state machine:

- `pending` → `in_progress` → `completed` | `failed`
- Dependencies giữa actions (DAG — Directed Acyclic Graph).
- Agent chỉ start action khi dependencies đã `completed`.
- Retry policy per action.

### Routines (Workflow Templates)

Routines là reusable workflow templates:
- Define một sequence of actions + conditions.
- Instantiate thành actual actions khi needed.
- Ví dụ: "Deploy workflow" = [lint, test, build, push, deploy].

### Checkpoints (Human Approval Gates)

Checkpoint là điểm dừng trong workflow, đợi human approval:
- Agent pause execution tại checkpoint.
- Notify human (via webhook hoặc UI).
- Human approve/reject → Agent tiếp tục hoặc rollback.

### Sentinels (Event Watchers)

Sentinels là background watchers trigger action khi điều kiện xảy ra:
- Monitor memory changes, NATS events, time-based triggers.
- Khi condition match → trigger pre-defined action.
- Ví dụ: "Nếu error count > 10 trong 5 phút → alert human".

### Sketches & Crystals

- **Sketches**: Ephemeral drafts — temporary working state chưa commit.
- **Crystals**: Permanent knowledge — extracted từ sketches sau khi verified.

Pattern: Agent tạo sketch trong khi làm, sau khi xác nhận đúng → crystallize thành permanent memory.

---

## Dataflow

### Lease Request Flow

```
Agent A wants to modify resource R
        │
        ▼
POST /v1/orchestration/leases
        │
        ├── Input: {resource_id: "memory-slot-xyz", ttl_seconds: 30}
        │
        ▼
orchestration-service
        │
        ├── Check existing lease on resource_id
        │         ├── No lease → GRANT: create lease record, return lease_token
        │         └── Active lease exists (not expired) → REJECT: 409 Conflict
        │
Agent A does work (holds lease)
        │
        ▼
DELETE /v1/orchestration/leases/{lease_id}  → Release lease
        │
        └── OR: lease expires automatically after TTL
```

### Signal Flow

```
Agent A → Agent B handoff
        │
POST /v1/orchestration/signals
        │
        ├── Input: {target_agent_id: "B", type: "handoff", payload: {...context...}}
        │
        ▼
orchestration-service
        │
        ├── Store signal in PostgreSQL (agent_signals table)
        ├── Publish NATS: orchestration.signal.{target_agent_id}
        │
        ▼
Agent B (subscribed to NATS)
        │
        └── Receive signal → process handoff context → continue task
```

### Action State Machine

```
POST /v1/orchestration/actions
        │
        ├── Create action: {name, dependencies: [...], routine_id}
        │         State: pending
        │
        ▼
Background runner (orchestration-service)
        │
        ├── Poll: any actions where all dependencies = completed?
        │         → Set state: in_progress
        │         → Execute action logic
        │         → Set state: completed | failed
        │
        ├── On checkpoint:
        │         → Set state: awaiting_approval
        │         → Notify human
        │         → Wait for PUT /v1/orchestration/checkpoints/{id}/approve|reject
        │
        └── DAG visualization: GET /v1/orchestration/actions (tree view)
```

### Sentinel Watch Flow

```
POST /v1/orchestration/sentinels
        │
        ├── Input: {condition: "error_count > 10", window: "5m", action: "alert"}
        │
        ▼
orchestration-service (background watcher)
        │
        ├── Subscribe to relevant NATS topics / poll metrics
        ├── Evaluate condition on each event
        └── Condition met → trigger action (create signal, webhook, etc.)
```

---

## API Endpoints (inferred from CR-AM-004)

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/orchestration/leases` | Request lease |
| `DELETE` | `/v1/orchestration/leases/{id}` | Release lease |
| `POST` | `/v1/orchestration/signals` | Send inter-agent signal |
| `GET` | `/v1/orchestration/signals` | List signals for agent |
| `POST` | `/v1/orchestration/actions` | Create action |
| `GET` | `/v1/orchestration/actions` | List actions (DAG view) |
| `POST` | `/v1/orchestration/routines` | Create routine template |
| `POST` | `/v1/orchestration/checkpoints/{id}/approve` | Human approve checkpoint |
| `POST` | `/v1/orchestration/checkpoints/{id}/reject` | Human reject checkpoint |
| `POST` | `/v1/orchestration/sentinels` | Create sentinel watcher |
| `POST` | `/v1/orchestration/sketches` | Create ephemeral sketch |
| `POST` | `/v1/orchestration/crystals` | Crystallize sketch |

---

## Database Tables

| Table | Nội dung |
|-------|---------|
| `agent_actions` | Action records, state machine, dependencies |
| `agent_leases` | Active leases (resource_id, holder, expiry) |
| `agent_signals` | Inter-agent signals, delivery status |
| `agent_routines` | Workflow templates |
| `agent_checkpoints` | Human approval gates |
| `agent_sentinels` | Event watchers, conditions |
| `agent_sketches` | Ephemeral drafts |
| `agent_crystals` | Permanent crystallized knowledge |

---

## Service

| Service | Vai trò |
|---------|---------|
| `orchestration-service` | Leases, signals, actions, checkpoints, sentinels, sketches/crystals |

---

## Business Value

### Pain Points được giải quyết

- **PP-P1-08 (Multi-agent race conditions)**

### Actors hưởng lợi

P1 AI Agent Developer

### Giải pháp tham chiếu

- [S8 — Distributed Agent Coordination](../../bussiness/solutions/S8-multi-agent.md)

### ROI / Kết quả đo được

> Zero race conditions via leases | Guaranteed signal delivery (NATS) | Action DAG built-in

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
