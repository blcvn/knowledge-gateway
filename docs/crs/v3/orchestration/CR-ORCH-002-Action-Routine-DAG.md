# Change Request: CR-ORCH-002 — Action Task Graph & Routines (Workflow Templates)

**CR ID:** CR-ORCH-002
**Component:** `backend/services/orchestration-service`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Orchestration
**Feature:** [F11](../../../features/11-multi-agent-orchestration/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-03 | Agent Developer | Multi-step agent workflows không có state tracking |

---

## 2. Action Task Graph (State Machine)

```
Action states: pending → running → completed | failed | cancelled

DAG execution:
  - Nodes: individual actions (function calls, API calls, memory ops)
  - Edges: dependencies (B runs after A completes)
  - Parallel: independent nodes run concurrently
  - Error handling: retry policy per action

Action model:
  {
    id, tenant_id, agent_id, session_id,
    name, type: "llm_call|memory_store|api_call|signal",
    payload, status,
    depends_on: [action_id],
    retry_count, max_retries,
    started_at, completed_at, error
  }
```

---

## 3. Routines (Workflow Templates)

```
Routine = reusable workflow template with parameterized steps.

Example routine: "Research and Store"
  Step 1: Recall relevant memories (memory.recall)
  Step 2: LLM analysis with context (llm.complete)
  Step 3: Store findings (memory.store, type=semantic)
  Step 4: Notify agent (signal)

Parameterized: {topic, depth, store_as}

Instantiate: POST /v1/orchestration/routines/{id}/run
  → Creates concrete Action DAG from template
```

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/orchestration/actions` | Create action |
| `GET` | `/v1/orchestration/actions` | List agent's actions |
| `GET` | `/v1/orchestration/actions/{id}` | Action detail + status |
| `POST` | `/v1/orchestration/actions/{id}/cancel` | Cancel action |
| `POST` | `/v1/orchestration/routines` | Create routine template |
| `GET` | `/v1/orchestration/routines` | List templates |
| `POST` | `/v1/orchestration/routines/{id}/run` | Instantiate routine |

---

## 5. Acceptance Criteria

- [ ] DAG topological sort: independent nodes run in parallel
- [ ] Retry policy: configurable per action (max 3, exponential backoff)
- [ ] State transitions persisted in DB
- [ ] Routine instantiation creates full Action DAG
- [ ] Action status queryable in real-time
