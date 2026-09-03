# VNP Memory — High Level Design

> **Standard:** C4 Model (Simon Brown) — Context → Container → Component → Code
> **Version:** 1.0.0 | **Updated:** 2026-09-03
>
> **Tham chiếu:**
> - [Feature Catalog](../features/README.md) — 28 features
> - [PRD v2.3.0](../product/v2/PRD.md)
> - [ADR Index](../adr/README.md)

---

## C4 Model Overview

```
Level 1 — CONTEXT    : VNP Memory trong hệ sinh thái AI
Level 2 — CONTAINER  : Các runtime process và data stores
Level 3 — COMPONENT  : Các module bên trong từng container
Level 4 — CODE       : Domain models, interfaces (tham khảo source)
```

## Documents

| Level | File | Mô tả |
|---|---|---|
| L1 | [C1-context.md](./C1-context.md) | System Context — Actors & External Systems |
| L2 | [C2-container.md](./C2-container.md) | Container Diagram — Processes & Data Stores |
| L3 | [C3-component.md](./C3-component.md) | Component Diagram — Internal Modules |
| L4 | [C4-code.md](./C4-code.md) | Key Domain Models & Interfaces |
| — | [deployment.md](./deployment.md) | Deployment Diagram — Dev vs Production |
| — | [data-flow.md](./data-flow.md) | Key Data Flows (Store, Recall, Observe) |

---

## Quick Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                      VNP Memory System                          │
│                                                                 │
│  REST API :8080          MCP Server :8082      Health :8083     │
│      │                        │                    │           │
│      └──────────┬─────────────┘                    │           │
│                 │                                   │           │
│           API Gateway                          Prometheus       │
│                 │ (InProcessRegistry / gRPC)                    │
│    ┌────────────┼─────────────────────────────┐                │
│    │ Memory Engines (6)    AgentMemory (4)     │                │
│    │  Cognee  Graphiti     Observe  Lifecycle  │                │
│    │  Memobase Zep         Orchestration       │                │
│    │  Supermemory          Consolidation       │                │
│    │  OpenViking                               │                │
│    └─────────────────────────────────────────-┘                │
│                 │                                               │
│    ┌────────────┼──────────────────┐                           │
│    PostgreSQL  Neo4j   Redis  NATS  MinIO  pgvector         │
└─────────────────────────────────────────────────────────────────┘
```
