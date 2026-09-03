# VNP Memory — Feature Catalog

> Tài liệu này liệt kê tất cả các feature của VNP Memory. Mỗi feature có một thư mục riêng chứa mô tả business logic và dataflow chi tiết.

| # | Feature | Thư mục | Loại |
|---|---------|---------|------|
| 01 | Unified Memory API | [01-unified-memory-api](./01-unified-memory-api/) | Core Platform |
| 02 | Episodic Memory (Graphiti) | [02-episodic-memory-graphiti](./02-episodic-memory-graphiti/) | Memory Engine |
| 03 | Semantic Memory (Cognee) | [03-semantic-memory-cognee](./03-semantic-memory-cognee/) | Memory Engine |
| 04 | Conversational Memory (Zep) | [04-conversational-memory-zep](./04-conversational-memory-zep/) | Memory Engine |
| 05 | Profile Memory (Memobase) | [05-profile-memory-memobase](./05-profile-memory-memobase/) | Memory Engine |
| 06 | Procedural Memory (OpenViking) | [06-procedural-memory-openviking](./06-procedural-memory-openviking/) | Memory Engine |
| 07 | Adaptive Memory (Supermemory) | [07-adaptive-memory-supermemory](./07-adaptive-memory-supermemory/) | Memory Engine |
| 08 | Agent Observe & Hook Capture | [08-agent-observe-hook-capture](./08-agent-observe-hook-capture/) | AgentMemory |
| 09 | Agent Memory Lifecycle | [09-agent-memory-lifecycle](./09-agent-memory-lifecycle/) | AgentMemory |
| 10 | Hybrid Search Engine | [10-hybrid-search-engine](./10-hybrid-search-engine/) | Search |
| 11 | Multi-Agent Orchestration | [11-multi-agent-orchestration](./11-multi-agent-orchestration/) | AgentMemory |
| 12 | Memory Consolidation Pipeline | [12-memory-consolidation-pipeline](./12-memory-consolidation-pipeline/) | AgentMemory |
| 13 | MCP Server & Context Injection | [13-mcp-server-context-injection](./13-mcp-server-context-injection/) | Integration |
| 14 | Authentication & Multi-tenancy | [14-authentication-multi-tenancy](./14-authentication-multi-tenancy/) | Platform |
| 15 | Console Dashboard | [15-console-dashboard](./15-console-dashboard/) | Console UI |
| 16 | Memory Explorer | [16-memory-explorer](./16-memory-explorer/) | Console UI |
| 17 | Graph Studio | [17-graph-studio](./17-graph-studio/) | Console UI |
| 18 | User Profiles Console | [18-user-profiles-console](./18-user-profiles-console/) | Console UI |
| 19 | Adaptive Memory Console | [19-adaptive-memory-console](./19-adaptive-memory-console/) | Console UI |
| 20 | Agent Context Debugger | [20-agent-context-debugger](./20-agent-context-debugger/) | Console UI |
| 21 | Sessions Explorer | [21-sessions-explorer](./21-sessions-explorer/) | Console UI |
| 22 | Governance Center | [22-governance-center](./22-governance-center/) | Compliance |
| 23 | Pipeline Monitor | [23-pipeline-monitor](./23-pipeline-monitor/) | Operations |
| 24 | Infrastructure Health | [24-infrastructure-health](./24-infrastructure-health/) | Operations |
| 25 | Observability & Tracing | [25-observability-tracing](./25-observability-tracing/) | Operations |
| 26 | Session Replay | [26-session-replay](./26-session-replay/) | AgentMemory |
| 27 | Organization & API SDK Manager | [27-organization-api-sdk-manager](./27-organization-api-sdk-manager/) | Platform |
| 28 | WebSocket Real-time Events | [28-websocket-realtime-events](./28-websocket-realtime-events/) | Platform |

---

## Phân loại theo domain

### Memory Engines (6 engines)
- Episodic → Graphiti (temporal graph)
- Semantic → Cognee (knowledge extraction)
- Conversational → Zep (session context)
- Profile → Memobase (YOLO engine)
- Procedural → OpenViking (VikingFS, tiered L0/L1/L2)
- Adaptive → Supermemory (living KG, auto-forget)

### AgentMemory Layer
- Observe Service (hook capture, 14-step pipeline)
- Memory Lifecycle (versioning, eviction, decay)
- Hybrid Search (BM25 + Vector + RRF)
- Orchestration (leases, signals, actions, sentinels)
- Consolidation Pipeline (4-tier compression)
- Session Replay (SSE stream, scrub timeline)

### Core Platform
- Unified Memory API (store/recall/forget/timeline)
- MCP Server (37+ tools, JSON-RPC 2.0)
- Authentication & Multi-tenancy
- WebSocket Streaming

### Console UI (12 sections)
- Dashboard, Memory Explorer, Graph Studio
- User Profiles, Adaptive Memory, Agent Debugger
- Sessions, Governance, Pipelines, Infrastructure
- Observability, Organization/SDK

### Compliance & Operations
- Governance Center (GDPR, Audit Trail, OPA Policies)
- Pipeline Monitor, Infrastructure Health, Observability
