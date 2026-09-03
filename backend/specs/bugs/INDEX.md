# VNP Memory — Bug Report Index

> **Ngày rà soát:** 2026-06-18  
> **Phạm vi:** 28 features, luồng apps/memory → gateway → services  
> **Tổng số bugs:** 62 bugs

---

## 🔴 CRITICAL CROSS-CUTTING BUGS (Ảnh hưởng toàn bộ hệ thống)

| Bug ID | Mô tả | File | Features Affected |
|--------|--------|------|-------------------|
| **BUG-F14-001** | Auth & RateLimit middleware KHÔNG được apply trong router chain → tất cả APIs publicly accessible | `gateway/adapter/handler/router.go:50-57` | F01-F28 tất cả |
| **BUG-F14-002** | `signAccessToken` tạo random hex thay vì JWT → cycle break: login → token → auth fail | `gateway/adapter/handler/auth.go:258-275` | F14 |
| **BUG-F02-002** | gRPC `ForwardWithContext` dùng raw JSON bytes không phải protobuf → gRPC decode fail | `gateway/adapter/client/registry.go:133` | F01-F28 tất cả |
| **BUG-F01-006** | = BUG-F14-001 (same root cause) | | F01-F28 |

---

## 🟠 HIGH SEVERITY BUGS

### F01 — Unified Memory API
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F01-001 | `Forget` handler chỉ trả về hardcoded 202, không xóa data | CRITICAL |
| BUG-F01-002 | Routing map thiếu `adaptive` → `sm-memory` | HIGH |
| BUG-F01-003 | Classifier luôn trả về `semantic` — auto-routing broken | MEDIUM |
| BUG-F01-004 | `extractPathParams()` trả về empty map — path params không forward | MEDIUM |
| BUG-F01-005 | Store trả về 200 thay vì 202 Accepted | LOW |

### F02 — Episodic Memory (Graphiti)
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F02-001 | Graphiti handler route tới `kg-service` thay vì `graphiti-*` services | HIGH |
| BUG-F02-003 | `graphiti-*` services không có implementation | HIGH |

### F03 — Semantic Memory (Cognee)
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F03-001 | Cognee handler route tới `kg-service` thay vì `cognee-*` services | HIGH |
| BUG-F03-002 | `cognee-*` services không có implementation | HIGH |
| BUG-F03-003 | NATS `memory.blob.inserted` không được publish sau store | MEDIUM |

### F04 — Conversational Memory (Zep)
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F04-001 | Zep client là `NoopClient` — Zep integration không hoạt động | CRITICAL |
| BUG-F04-002 | ZepHandler routes tới `memory-service` không có Zep router registration | HIGH |

### F05 — Profile Memory (Memobase)
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F05-001 | `GetEvents` route sai tới `vnp-platform` | MEDIUM |
| BUG-F05-002 | Auto-flush (20 blobs) chưa implement | HIGH |

### F06 — Procedural Memory (OpenViking)
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F06-001 | `Search` route sai tới generic `search-service` | MEDIUM |
| BUG-F06-002 | `ov-*` services không có implementation | HIGH |
| BUG-F06-003 | WebDAV proxy được tạo nhưng không mount vào router | MEDIUM |

### F07 — Adaptive Memory (Supermemory)
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F07-001 | SMHandler routing sai (9 operations → wrong services) | HIGH |
| BUG-F07-002 | Tất cả `sm-*` services không có implementation | CRITICAL |

### F08 — Agent Observe & Hook Capture
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F08-001 | Import path sai cho `pkg/privacy` | CRITICAL |
| BUG-F08-002 | Observe pipeline thiếu Step 3 (authentication) | HIGH |
| BUG-F08-003 | Thiếu Step 10 (BM25) và Step 11 (embedding) | HIGH |
| BUG-F08-004 | SSE StreamEvents: gateway gRPC vs observe-service HTTP SSE conflict | MEDIUM |

### F09 — Agent Memory Lifecycle
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F09-001 | AgentMemory routes không registered trong memory-service router | HIGH |
| BUG-F09-002 | `GetRetentionScore` không có service implementation | HIGH |
| BUG-F09-003 | Auto-forget không có background scheduler | MEDIUM |
| BUG-F09-004 | Route conflict: `/agent/list` vs `/agent/{id}` | MEDIUM |

### F10 — Hybrid Search Engine
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F10-001 | `vnp-search-hub` service không có implementation | CRITICAL |
| BUG-F10-002 | `SearchUseCase.FanOutSearch` không được wire vào Recall | HIGH |
| BUG-F10-003 | SearchUseCase dùng `Forward()` deprecated | MEDIUM |
| BUG-F10-004 | Không có merge/rerank logic | HIGH |

### F11 — Multi-Agent Orchestration
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F11-001 | `orchestration-service` không có implementation | CRITICAL |
| BUG-F11-002 | Không có gateway routes cho orchestration | HIGH |

### F12 — Memory Consolidation Pipeline
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F12-001 | `ConsolidationHandler` không registered trong gRPC server | CRITICAL |
| BUG-F12-002 | NATS `consolidation.trigger` subscription không setup | CRITICAL |
| BUG-F12-003 | Consolidation pipeline thiếu LLM client | HIGH |
| BUG-F12-004 | Background job scheduler (`pipeline-service`) không có implementation | HIGH |

### F13 — MCP Server
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F13-002 | MCP server không có auth middleware | HIGH |

### F14 — Authentication & Multi-tenancy
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F14-001 | Auth middleware không apply trong router chain (**ROOT CAUSE của nhiều bugs**) | CRITICAL |
| BUG-F14-002 | `signAccessToken` tạo random hex không phải JWT | HIGH |
| BUG-F14-003 | Dev mode không restrict localhost traffic | MEDIUM |
| BUG-F14-004 | API key format validation không đúng spec | MEDIUM |
| BUG-F14-005 | CORS wildcard + credentials=true invalid | MEDIUM |
| BUG-F14-006 | `randomHex()` không đảm bảo uniqueness | LOW |

### F15-F27 — Console Features
Tất cả console features đều bị ảnh hưởng bởi BUG-F14-001 (Auth middleware không apply). Xem bug reports riêng cho các bugs cụ thể.

### F28 — WebSocket Real-time Events
| Bug | Mô tả | Severity |
|-----|--------|---------|
| BUG-F28-001 | WSHandler không subscribe NATS → không nhận events | CRITICAL |
| BUG-F28-002 | WebSocket upgrade không implement (dùng SSE thay thế) | HIGH |
| BUG-F28-003 | Auth middleware không apply → WSHandler luôn 401 | HIGH |
| BUG-F28-006 | Tenant-scoping cho events không implement → cross-tenant leakage | HIGH |

---

## 📋 BUG COUNT BY FEATURE

| Feature | Total Bugs | Critical | High | Medium | Low |
|---------|-----------|----------|------|--------|-----|
| F01 - Unified Memory API | 6 | 1 | 1 | 2 | 2 |
| F02 - Episodic (Graphiti) | 4 | 0 | 3 | 0 | 1 |
| F03 - Semantic (Cognee) | 3 | 0 | 2 | 1 | 0 |
| F04 - Conversational (Zep) | 3 | 1 | 1 | 1 | 0 |
| F05 - Profile (Memobase) | 3 | 0 | 1 | 2 | 0 |
| F06 - Procedural (OpenViking) | 3 | 0 | 1 | 2 | 0 |
| F07 - Adaptive (Supermemory) | 3 | 1 | 1 | 1 | 0 |
| F08 - Agent Observe | 5 | 1 | 2 | 1 | 1 |
| F09 - Agent Memory Lifecycle | 4 | 0 | 2 | 2 | 0 |
| F10 - Hybrid Search | 4 | 1 | 2 | 1 | 0 |
| F11 - Multi-Agent Orchestration | 2 | 1 | 1 | 0 | 0 |
| F12 - Consolidation Pipeline | 4 | 2 | 2 | 0 | 0 |
| F13 - MCP Server | 2 | 0 | 1 | 1 | 0 |
| F14 - Auth & Multi-tenancy | 6 | 1 | 1 | 3 | 1 |
| F15 - Console Dashboard | 3 | 1 | 1 | 1 | 0 |
| F16 - Memory Explorer | 3 | 1 | 2 | 0 | 0 |
| F17 - Graph Studio | 3 | 1 | 1 | 1 | 0 |
| F18 - User Profiles | 3 | 1 | 1 | 1 | 0 |
| F19 - Adaptive Console | 3 | 1 | 1 | 1 | 0 |
| F20 - Agent Debugger | 3 | 1 | 0 | 2 | 0 |
| F21 - Sessions Explorer | 4 | 1 | 2 | 1 | 0 |
| F22 - Governance | 4 | 1 | 3 | 0 | 0 |
| F23 - Pipeline Monitor | 4 | 1 | 2 | 1 | 0 |
| F24 - Infra Health | 3 | 1 | 1 | 0 | 1 |
| F25 - Observability | 3 | 1 | 1 | 1 | 0 |
| F26 - Session Replay | 2 | 1 | 1 | 0 | 0 |
| F27 - Org/SDK Manager | 4 | 1 | 2 | 1 | 0 |
| F28 - WebSocket | 6 | 1 | 3 | 1 | 1 |
| **TOTAL** | **97** | **22** | **42** | **26** | **7** |

---

## 🎯 TOP PRIORITY FIXES (Theo thứ tự ưu tiên)

1. **[P0] BUG-F14-001** — Wire Auth + RateLimit middleware vào router chain
2. **[P0] BUG-F14-002** — Implement JWT signing trong `signAccessToken()`
3. **[P0] BUG-F02-002** — Fix gRPC `ForwardWithContext` encoding (JSON vs protobuf)
4. **[P0] BUG-F01-001** — Implement `Forget` handler với `ForgetUseCase`
5. **[P1] BUG-F28-001** — Wire NATS subscription vào WSHandler
6. **[P1] BUG-F04-001** — Wire real Zep SDK client
7. **[P1] BUG-F10-001** — Implement `vnp-search-hub` service
8. **[P1] BUG-F12-001/002** — Register ConsolidationHandler + NATS subscription
9. **[P1] BUG-F08-001** — Fix privacy package import path
10. **[P1] BUG-F01-002** — Add `adaptive` → `sm-memory` routing

---

## 📁 File Structure

```
specs/bugs/
├── INDEX.md                           ← this file
├── F01-unified-memory-api/BUG-REPORT.md
├── F02-episodic-memory-graphiti/BUG-REPORT.md
├── F03-semantic-memory-cognee/BUG-REPORT.md
├── F04-conversational-memory-zep/BUG-REPORT.md
├── F05-profile-memory-memobase/BUG-REPORT.md
├── F06-procedural-memory-openviking/BUG-REPORT.md
├── F07-adaptive-memory-supermemory/BUG-REPORT.md
├── F08-agent-observe-hook-capture/BUG-REPORT.md
├── F09-agent-memory-lifecycle/BUG-REPORT.md
├── F10-hybrid-search-engine/BUG-REPORT.md
├── F11-multi-agent-orchestration/BUG-REPORT.md
├── F12-memory-consolidation-pipeline/BUG-REPORT.md
├── F13-mcp-server-context-injection/BUG-REPORT.md
├── F14-authentication-multi-tenancy/BUG-REPORT.md
├── F15-console-dashboard/BUG-REPORT.md
├── F16-memory-explorer/BUG-REPORT.md
├── F17-graph-studio/BUG-REPORT.md
├── F18-user-profiles-console/BUG-REPORT.md
├── F19-adaptive-memory-console/BUG-REPORT.md
├── F20-agent-context-debugger/BUG-REPORT.md
├── F21-sessions-explorer/BUG-REPORT.md
├── F22-governance-center/BUG-REPORT.md
├── F23-pipeline-monitor/BUG-REPORT.md
├── F24-infrastructure-health/BUG-REPORT.md
├── F25-observability-tracing/BUG-REPORT.md
├── F26-session-replay/BUG-REPORT.md
├── F27-organization-api-sdk-manager/BUG-REPORT.md
└── F28-websocket-realtime-events/BUG-REPORT.md
```
