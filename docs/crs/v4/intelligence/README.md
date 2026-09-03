# v4 — Intelligence Layer

> **Mục tiêu:** User profiling, context efficiency, knowledge evolution, agent observability.
> **Phase:** Differentiation — làm VNP Memory thông minh hơn các competitors.
> **Actors hưởng lợi:** P1 (AI Agent Developer), P3 (ML/AI Engineer), P7 (AI Power User), P8 (Product Manager)

## Pain Points được giải quyết

| Pain Point | Actor | CR giải quyết |
|---|---|---|
| AI không nhớ user preferences | P7 | CR-INTEL-001 |
| Context token cost $0.50/call | P1, P3 | CR-INTEL-002 |
| Knowledge không tự update | P1, P7 | CR-INTEL-003 |
| Memory không tự quên stale info | P1 | CR-INTEL-004 |
| Không debug được agent decisions | P1, P4 | CR-INTEL-005, CR-INTEL-006 |
| Không trace context injection | P3 | CR-INTEL-006 |

## Change Requests

| CR | Title | Priority | Solution |
|---|---|---|---|
| [CR-INTEL-001](./CR-INTEL-001-User-Profile-Assembly.md) | User Profile Assembly — context < 100ms | 🔴 Critical | S5 |
| [CR-INTEL-002](./CR-INTEL-002-Tiered-Context-Injection.md) | Tiered Context Injection — L0/L1/L2 token budget | 🔴 Critical | S6 |
| [CR-INTEL-003](./CR-INTEL-003-Knowledge-Evolution.md) | Knowledge Evolution — contradiction resolution | 🟡 High | S4 |
| [CR-INTEL-004](./CR-INTEL-004-Memory-Decay-Eviction.md) | forgetAfter & Memory Decay — salience eviction | 🟡 High | S4 |
| [CR-INTEL-005](./CR-INTEL-005-Session-Replay.md) | Session Replay — JSONL import, timeline scrubbing | 🟡 High | S7 |
| [CR-INTEL-006](./CR-INTEL-006-Agent-Context-Debugger.md) | Agent Context Debugger — full context trace | 🟠 Medium | S7 |

## Tham chiếu

- [S4 — Knowledge Evolution](../../../bussiness/solutions/S4-knowledge-evolution.md)
- [S5 — User Profiling](../../../bussiness/solutions/S5-user-profiling.md)
- [S6 — Context Efficiency](../../../bussiness/solutions/S6-context-efficiency.md)
- [S7 — Agent Observability](../../../bussiness/solutions/S7-agent-observability.md)
