# v3 — Core Memory & Integration

> **Mục tiêu:** Hoàn thiện Unified Memory API, kết nối đầy đủ 6 engines, temporal reasoning, MCP integration.
> **Phase:** MVP Completion
> **Actors hưởng lợi:** P1 (AI Agent Developer), P5 (IDE Plugin User), P6 (Framework Integrator)

## Pain Points được giải quyết

| Pain Point | Actor | CR giải quyết |
|---|---|---|
| Memory fragmented — 6 APIs rời | P1, P6 | CR-CORE-001, CR-CORE-002 |
| AI mất context sau session | P1, P5 | CR-CORE-004 |
| RAG không hiểu thời gian | P1 | CR-CORE-005 |
| No standard API cho frameworks | P6 | CR-CORE-001, CR-CORE-006 |
| GDPR forget không cascading | P4 | CR-CORE-003 |
| IDE phải brief AI mỗi sáng | P5 | CR-CORE-004, CR-CORE-006 |

## Change Requests

| CR | Title | Priority | Solution |
|---|---|---|---|
| [CR-CORE-001](./CR-CORE-001-Unified-Memory-Router.md) | Unified Memory Router — Auto-type & routing | 🔴 Critical | S2 |
| [CR-CORE-002](./CR-CORE-002-Cross-Engine-Recall.md) | Cross-Engine Recall — RRF Hybrid Search | 🔴 Critical | S2 |
| [CR-CORE-003](./CR-CORE-003-Cascading-Forget.md) | Cascading Forget — GDPR delete 6 engines | 🔴 Critical | S9 |
| [CR-CORE-004](./CR-CORE-004-Persistent-Session-Context.md) | Persistent Session Context — cross-session | 🟡 High | S1 |
| [CR-CORE-005](./CR-CORE-005-Temporal-Reasoning.md) | Temporal Reasoning — isLatest versioning | 🟡 High | S3 |
| [CR-CORE-006](./CR-CORE-006-MCP-Server.md) | MCP Server — 37+ tools, dual transport | 🟡 High | S2, S6 |

## Tham chiếu

- [S2 — Unified Memory API](../../../bussiness/solutions/S2-unified-api.md)
- [S1 — Persistent Memory](../../../bussiness/solutions/S1-persistent-memory.md)
- [S3 — Temporal Reasoning](../../../bussiness/solutions/S3-temporal-reasoning.md)
- [Pain Points P1](../../../bussiness/painpoints/P1-ai-agent-developer.md)
