# Change Requests — Supermemory Feature Parity

**Project:** VNP Memory  
**Domain:** Supermemory — Memory & Context Engine for AI  
**Path:** `specs/crs/v1/supermemory/`  
**Date:** 2026-06-16  
**Status:** Proposed

> Các Change Requests này được tạo từ phân tích đối chiếu giữa VNP Memory hiện tại và tài liệu Supermemory (`PRD v4.0.0`, `SRS`, `URD`, `specs/services`).
>
> **Benchmark mục tiêu của Supermemory**: #1 LongMemEval (81.6%), #1 LoCoMo, #1 ConvoMem — nhờ hệ thống Knowledge Graph với version chain và automatic forgetting.

---

## Danh sách Change Requests

| CR ID | Tên | Loại | Priority | Status |
|---|---|---|---|---|
| [CR-SM-001](./CR-SM-001-Document-Ingestion-Pipeline.md) | **Document Ingestion Pipeline** (11 content types, async worker, AST chunking) | 🆕 New Service | Critical | Proposed |
| [CR-SM-002](./CR-SM-002-Memory-Engine-Knowledge-Graph.md) | **Memory Engine & Knowledge Graph** (Fact extraction, updates/extends/derives, auto-forget) | 🆕 New Service | Critical | Proposed |
| [CR-SM-003](./CR-SM-003-Hybrid-Search-Engine.md) | **Hybrid Search Engine** (RAG + Memory, pgvector, query rewrite, rerank, metadata filter) | ⬆️ Upgrade Service | Critical | Proposed |
| [CR-SM-004](./CR-SM-004-User-Profile-Service.md) | **User Profile Service** (Static + Dynamic profile, < 100ms, Redis cache) | 🆕 New Service | High | Proposed |
| [CR-SM-005](./CR-SM-005-Connector-Service.md) | **External Connector Service** (Google Drive, Notion, OneDrive, GitHub, cron 4h) | 🆕 New Service | High | Proposed |
| [CR-SM-006](./CR-SM-006-MCP-Server.md) | **MCP Server** (4 tools, resources, prompts, session persistence, OAuth) | ⬆️ Upgrade Service | High | Proposed |
| [CR-SM-007](./CR-SM-007-Auth-Organization-RBAC.md) | **Auth Service & RBAC** (Organization, 4 roles, sm_ API keys, OAuth2 server) | ⬆️ Upgrade Service | High | Proposed |
| [CR-SM-008](./CR-SM-008-Project-Space-Management.md) | **Project & Space Management** (Container tags, space membership, doc-space M:M) | 🆕 New Service | Medium | Proposed |
| [CR-SM-009](./CR-SM-009-Analytics-Token-Economics.md) | **Analytics & Token Economics** (Usage tracking, token savings, cost USD) | 🆕 New Service | Medium | Proposed |
| [CR-SM-010](./CR-SM-010-Framework-Integrations-SDK.md) | **Framework Integrations & SDK** (Go/TS/Python SDKs, Vercel AI, LangChain, Mastra) | 🆕 New Packages | Medium | Proposed |

---

## Kiến trúc mục tiêu (Go Microservices)

Supermemory Enterprise Architecture: TypeScript Serverless → Go Microservices với Clean Architecture.

```
External Clients: AI SDKs (Go/TS/Py) · MCP Clients (Claude/Cursor) · Web Console · CLI
                                        │
                        ┌───────────────▼───────────────┐
                        │         API Gateway            │
                        │  REST · JWT/API Key Auth · RBAC│
                        └──┬────┬────┬────┬────┬────┬───┘
                           │    │    │    │    │    │   gRPC
          ┌────────────────┘    │    │    │    │    └────────────────┐
          │              ┌──────┘    │    │    └──────────────┐      │
          ▼              ▼           ▼    ▼                   ▼      ▼
    ┌──────────┐  ┌──────────┐ ┌──────┐ ┌────────┐  ┌──────────┐ ┌─────┐
    │ Document │  │  Memory  │ │Search│ │Profile │  │Connector │ │ MCP │
    │ Service  │  │ Service  │ │ Svc  │ │  Svc   │  │  Svc     │ │ Svc │
    │  :9001   │  │  :9002   │ │:9003 │ │ :9004  │  │  :9005   │ │:9006│
    └──────────┘  └──────────┘ └──────┘ └────────┘  └──────────┘ └─────┘
    ┌──────────────────────────────────────────────────────────────────────┐
    │  Auth :9007 │ Analytics :9008 │ Project :9009                        │
    └──────────────────────────────────────────────────────────────────────┘
                                    │
    ┌───────────────────────────────▼──────────────────────────────────────┐
    │  Infrastructure: PostgreSQL + pgvector │ Redis │ NATS JetStream │ S3  │
    └──────────────────────────────────────────────────────────────────────┘
```

---

## Tính năng độc đáo (Differentiators) của Supermemory

| Feature | Mô tả | CR liên quan |
|---------|-------|-------------|
| **Knowledge Graph** | Memories có quan hệ `updates/extends/derives`, version chain | CR-SM-002 |
| **Automatic Forgetting** | `forgetAfter` timestamp, semantic forget, contradiction resolution | CR-SM-002 |
| **Hybrid Search** | RAG (chunks) + Memory (facts) trong 1 query, pgvector HNSW | CR-SM-003 |
| **User Profile < 100ms** | Static + Dynamic profile, Redis cache, dedup priority | CR-SM-004 |
| **Connector Ecosystem** | Google Drive, Notion, OneDrive auto-sync mỗi 4 giờ | CR-SM-005 |
| **Token Economics** | Track token savings, tính cost saved USD | CR-SM-009 |
| **Multi-tenant spaces** | Container tags, space membership RBAC | CR-SM-008 |

---

## Lộ trình triển khai

| Wave | CR | Mô tả |
|------|-----|-------|
| **Wave 1 — Foundation** | CR-007, CR-008 | Auth/RBAC và Project/Space management |
| **Wave 2 — Core Memory** | CR-001, CR-002 | Document ingestion và Memory Engine (Knowledge Graph) |
| **Wave 3 — Intelligence** | CR-003, CR-004 | Hybrid Search và User Profile |
| **Wave 4 — Integrations** | CR-005, CR-006 | Connectors và MCP Server |
| **Wave 5 — Ecosystem** | CR-009, CR-010 | Analytics và SDK/Framework integrations |
