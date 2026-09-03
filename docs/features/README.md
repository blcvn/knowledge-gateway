# VNP Memory — Feature Catalog

> Tài liệu này liệt kê tất cả 28 features của VNP Memory. Mỗi feature có một thư mục riêng chứa:
> - Mô tả business logic và dataflow chi tiết
> - **Business Value**: Pain points giải quyết, actors hưởng lợi, ROI
>
> **Tài liệu liên quan:**
> - [Research Insights](../bussiness/research/README.md) — Neuroscience + Market foundations
> - [Competitive Analysis](../bussiness/competitive/README.md) — 5 competitors vs VNP Memory
> - [Pain Points](../bussiness/painpoints/README.md) — Các vấn đề của 8 actors
> - [Solutions](../bussiness/solutions/README.md) — Giải pháp kỹ thuật chi tiết
> - [PRD v2](../product/v2/PRD.md) — Product Requirements Document

---

## Danh sách Features

| # | Feature | Loại | Solutions | Actors |
|---|---------|------|-----------|--------|
| 01 | [Unified Memory API](./01-unified-memory-api/) | Core Platform | S2, S10 | P1, P6, P2 |
| 02 | [Episodic Memory (Graphiti)](./02-episodic-memory-graphiti/) | Memory Engine | S3 | P1, P3 |
| 03 | [Semantic Memory (Cognee)](./03-semantic-memory-cognee/) | Memory Engine | S2 | P1, P3 |
| 04 | [Conversational Memory (Zep)](./04-conversational-memory-zep/) | Memory Engine | S1, S3 | P1, P3 |
| 05 | [Profile Memory (Memobase)](./05-profile-memory-memobase/) | Memory Engine | S1, S5 | P1, P7, P8 |
| 06 | [Procedural Memory (OpenViking)](./06-procedural-memory-openviking/) | Memory Engine | S1, S6 | P1, P5 |
| 07 | [Adaptive Memory (Supermemory)](./07-adaptive-memory-supermemory/) | Memory Engine | S1, S4 | P1, P7 |
| 08 | [Agent Observe & Hook Capture](./08-agent-observe-hook-capture/) | AgentMemory | S7 | P1, P3, P4 |
| 09 | [Agent Memory Lifecycle](./09-agent-memory-lifecycle/) | AgentMemory | S4 | P1 |
| 10 | [Hybrid Search Engine](./10-hybrid-search-engine/) | Search | S2 | P1, P3 |
| 11 | [Multi-Agent Orchestration](./11-multi-agent-orchestration/) | AgentMemory | S8 | P1 |
| 12 | [Memory Consolidation Pipeline](./12-memory-consolidation-pipeline/) | AgentMemory | S6 | P1, P2 |
| 13 | [MCP Server & Context Injection](./13-mcp-server-context-injection/) | Integration | S2, S6 | P1, P5, P6 |
| 14 | [Authentication & Multi-tenancy](./14-authentication-multi-tenancy/) | Platform | S9 | P2, P4 |
| 15 | [Console Dashboard](./15-console-dashboard/) | Console UI | S10 | P2, P3 |
| 16 | [Memory Explorer](./16-memory-explorer/) | Console UI | S9 | P4, P7 |
| 17 | [Graph Studio](./17-graph-studio/) | Console UI | S3 | P3 |
| 18 | [User Profiles Console](./18-user-profiles-console/) | Console UI | S5 | P7, P8 |
| 19 | [Adaptive Memory Console](./19-adaptive-memory-console/) | Console UI | S4 | P1, P3 |
| 20 | [Agent Context Debugger](./20-agent-context-debugger/) | Console UI | S7 | P1, P3 |
| 21 | [Sessions Explorer](./21-sessions-explorer/) | Console UI | S7 | P1, P3 |
| 22 | [Governance Center](./22-governance-center/) | Compliance | S9 | P4, P2 |
| 23 | [Pipeline Monitor](./23-pipeline-monitor/) | Operations | S10 | P2, P3 |
| 24 | [Infrastructure Health](./24-infrastructure-health/) | Operations | S10 | P2 |
| 25 | [Observability & Tracing](./25-observability-tracing/) | Operations | S10 | P2, P3, P8 |
| 26 | [Session Replay](./26-session-replay/) | AgentMemory | S7 | P1, P3, P4 |
| 27 | [Organization & API SDK Manager](./27-organization-api-sdk-manager/) | Platform | S9, S10 | P2, P6 |
| 28 | [WebSocket Real-time Events](./28-websocket-realtime-events/) | Platform | S10 | P2, P1 |

---


---

## Research Backing — Neuroscience Foundation

Mỗi memory type trong VNP Memory được thiết kế dựa trên neuroscience:

| Memory Type | Neuroscience Analog | Research Source |
|---|---|---|
| Episodic (Graphiti) | Event memory với timestamps (hippocampus) | [sleep.md](../research/sleep.md) — hippocampus replay |
| Semantic (Cognee) | Schema networks trong neocortex | [personal-memory.md](../research/personal-memory.md) |
| Conversational (Zep) | Working memory + session encoding | [sensor.md](../research/sensor.md) — 5-step pipeline |
| Profile (Memobase) | Cortical representation of self | [predictive-processing.md](../research/predictive-processing.md) |
| Adaptive (Supermemory) | Living synaptic weights + pruning | [synapse.md](../research/synapse.md) + [sleep.md](../research/sleep.md) |
| Procedural (OpenViking) | Neocortex L0/L1/L2 hierarchy | [neocortex.md](../research/neocortex.md) |
| AgentMemory (Consolidation) | Sleep consolidation: NREM→REM→insight | [sleep.md](../research/sleep.md) — 9 sleep functions |
| Agent Observe | Prediction error capture | [predictive-processing.md](../research/predictive-processing.md) |

> Chi tiết: [Research Insights](../bussiness/research/README.md)

## Phân loại theo domain

### Memory Engines (6 engines) — Giải quyết S1, S3, S4, S5
- **Episodic** → Graphiti (temporal graph, validity windows)
- **Semantic** → Cognee (knowledge extraction, 15+ strategies)
- **Conversational** → Zep (session context, custom ontology)
- **Profile** → Memobase (YOLO engine, < 100ms context)
- **Procedural** → OpenViking (VikingFS, tiered L0/L1/L2)
- **Adaptive** → Supermemory (living KG, auto-forget, version chain)

### AgentMemory Layer — Giải quyết S6, S7, S8
- Observe Service (12 hooks, 14-step pipeline, SSE)
- Memory Lifecycle (Jaccard versioning, eviction, decay)
- Hybrid Search (BM25 + Vector + RRF fusion)
- Orchestration (leases, signals, Action DAG, sentinels)
- Consolidation Pipeline (4-tier: raw → summary → procedure → insight)
- Session Replay (JSONL import, timeline scrubbing)

### Core Platform — Giải quyết S2, S9, S10
- Unified Memory API (store/recall/forget/timeline)
- MCP Server (37+ tools, JSON-RPC 2.0, context injection)
- Authentication & Multi-tenancy (TenantID isolation, API key lifecycle)
- WebSocket Streaming (real-time events)

### Console UI (12 sections) — Giải quyết S5, S7, S9
- Dashboard, Memory Explorer, Graph Studio
- User Profiles, Adaptive Memory, Agent Debugger
- Sessions Explorer, Session Replay
- Governance Center, Pipelines, Infrastructure, Observability
- Organization/SDK Manager

---

## Ma trận Pain Point → Features

| Pain Point | Features giải quyết |
|---|---|
| AI mất context sau session | F01, F04, F05, F07 |
| Memory fragmented | F01, F10 |
| RAG không hiểu thời gian | F02, F04, F09 |
| Knowledge không tự update | F07, F09 |
| Không có user profile | F05 |
| Context tốn token/chậm | F05, F06, F12, F13 |
| Không debug được agent | F08, F20, F21, F26 |
| Multi-agent race conditions | F11 |
| GDPR / Governance gap | F14, F16, F22 |
| Infrastructure phức tạp | F01, F15, F23, F24, F25 |
| AI coding assistant quên project | F06, F13 |
| No standard API | F01, F13 |
