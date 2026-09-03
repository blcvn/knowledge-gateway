# S8 — Distributed Agent Coordination

> **Giải quyết Pain Points:** PP-P1-08
> **Actor chính:** P1 (AI Agent Developer)
> **Features:** F11 (Multi-Agent Orchestration)

---

## Vấn đề cần giải quyết

Khi nhiều AI agents chạy song song và truy cập shared memory, xảy ra race conditions: Agent A và B cùng update memory → last-write-wins → data corruption. Không có mechanism để agents phối hợp, handoff tasks, hoặc share state an toàn.

---

## Giải pháp: Distributed Agent Coordination Layer

### 1 — Distributed Leases (Mutual Exclusion)

Trước khi modify shared resource, agent phải acquire lease:

```
Agent A muốn update "project budget memory"
        │
        ▼
POST /v1/orchestration/leases
{
  "resource_id": "memory:project-budget",
  "agent_id": "agent-A",
  "ttl_seconds": 30
}
        │
        ├── Lease available → Grant (lease_id: "lease-001")
        │         Agent A update memory
        │         Agent A release lease
        │
        └── Lease taken (Agent B đang giữ) → 409 Conflict
                  Agent A retry sau X giây
                  (Exponential backoff built-in)
```

**Guarantee:** Không thể có 2 agents cùng modify 1 resource → Zero race conditions.

---

### 2 — Inter-Agent Signals (Async Communication)

Agents giao tiếp qua structured signals thay vì shared state:

```http
POST /v1/orchestration/signals
{
  "from_agent": "research-agent",
  "to_agent": "writer-agent",
  "signal_type": "handoff",
  "payload": {
    "task": "write_report",
    "context_memory_id": "mem-research-001",
    "priority": "high"
  }
}
```

**4 Signal Types:**
| Type | Dùng khi |
|---|---|
| `handoff` | Agent A chuyển giao task cho Agent B |
| `alert` | Thông báo lỗi nghiêm trọng cần human review |
| `update` | Thông báo state change |
| `query` | Agent A hỏi thông tin từ Agent B |

**Delivery:** NATS JetStream → guaranteed delivery (at-least-once).

---

### 3 — Action DAG (Task Dependencies)

Agents định nghĩa task graph với dependencies, tự động serialize execution:

```
workflow: "Deploy new service"

Actions:
  lint  ──→  test  ──→  build  ──→  push  ──→  deploy
                                              ↑
  security_scan ─────────────────────────────┘

Rules:
- lint phải complete trước test
- build chỉ start khi test passed
- deploy chỉ start khi push VÀ security_scan đều passed
```

```http
POST /v1/orchestration/actions
{
  "name": "deploy",
  "dependencies": ["push", "security_scan"],
  "retry_policy": {"max_attempts": 3, "backoff": "exponential"}
}
```

**State machine:** pending → in_progress → completed | failed | retrying

---

### 4 — Sketches & Crystals (Draft → Commit Pattern)

```
Agent làm việc trong phiên:
  → Tạo "Sketch" (ephemeral draft)
  → Modify sketch thoải mái
  → Khi confident: "Crystallize" → Permanent memory

Code:
POST /v1/orchestration/sketches    ← Create draft
PUT  /v1/orchestration/sketches/{id}  ← Update
POST /v1/orchestration/sketches/{id}/crystallize  ← Commit
```

**Pattern:** Giống Git's working tree (sketch) → commit (crystal). Tránh lưu permanent memory sai do agent đang "suy nghĩ".

---

### 5 — Sentinels (Event-driven Triggers)

Agents định nghĩa background watchers, tự động react khi condition met:

```http
POST /v1/orchestration/sentinels
{
  "name": "error_rate_alert",
  "condition": "error_count > 10 in last 5 minutes",
  "action": {
    "type": "signal",
    "to_agent": "supervisor-agent",
    "signal_type": "alert"
  }
}
```

Không cần polling loop — Sentinel xử lý event-driven.

---

## Luồng Multi-Agent Collaboration

```
Research Agent:
  1. Acquire lease: "memory:research-topic"
  2. Store research findings (Sketches)
  3. Crystallize → Permanent knowledge
  4. Release lease
  5. Send signal(handoff) → Writer Agent
        │
        ▼
Writer Agent:
  1. Receive handoff signal
  2. Acquire lease: "memory:draft-report"
  3. Read research crystals (memory_read)
  4. Write draft report (sketch)
  5. Sentinel: "Draft complete → notify Review Agent"
        │
        ▼
Review Agent:
  1. Receive sentinel trigger
  2. Read draft
  3. Send signal(update) → Writer Agent với feedback
```

---

## Kết quả

| Scenario | Trước | Sau |
|---|---|---|
| 2 agents update cùng memory | Race condition / data corruption | Leases → serialized access |
| Agent handoff với context | Copy-paste context manually | Structured handoff signal |
| Task dependencies | Tự implement DAG | Built-in Action DAG |
| React to events | Polling loop | Sentinel event-driven |
| Draft vs permanent | Không có | Sketch → Crystal pattern |
