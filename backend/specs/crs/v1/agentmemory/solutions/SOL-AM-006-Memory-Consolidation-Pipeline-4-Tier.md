# SOL-AM-006 — Solution: Memory Consolidation Pipeline (4-Tier)

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-006 |
| **CR** | CR-AM-006 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/pipeline-service` |

---

## 1. Giải pháp

See SOL-ENT-002 (Consolidation Pipeline) for full implementation.

AgentMemory context: pipeline-service is in AgentMemory layer, triggered by observe-service via NATS.

Trigger chain:
1. observe-service: `POST /v1/observe/sessions/{id}/end`
2. Publish: `agent.session.complete` → NATS
3. pipeline-service: subscribes → RunForSession(sessionID)
4. Results: blobs → memory-service, summaries → memobase, procedures → openviking

## 2. Acceptance Criteria

Same as SOL-ENT-002. Plus:
- [ ] Triggered within 1s of session end
- [ ] Results distributed to correct engines (not stored in pipeline-service)

