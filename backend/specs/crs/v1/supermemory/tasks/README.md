# Supermemory Feature Parity — Implementation Tasks

**Dự án:** VNP Memory — Supermemory End-to-End Platform  
**Nguồn:** [Solutions](../solutions/)  
**Ngày tạo:** 2026-06-17  
**Trạng thái:** Ready for Execution

> Các tác vụ này được thiết kế để **AI agent thực thi tuần tự theo Wave**. Mỗi task là một đơn vị công việc độc lập, có input rõ ràng, acceptance criteria kiểm chứng được bằng code/test.

---

## Quy tắc Thực thi

1. **Wave order bắt buộc** — Wave N phải hoàn thành trước Wave N+1
2. **Trong cùng Wave** — tasks không phụ thuộc nhau có thể chạy **song song**
3. **Mỗi task kết thúc** bằng `go build ./...` HOẶC `make ci` không có lỗi
4. **Unit test phải pass** trước khi sang task tiếp theo
5. **KHÔNG được sửa** code trong `services/kgs-platform/` (read-only external)
6. **Auth trước tất cả** — Wave 1 (TASK-SM-001/002/003) phải xong trước Wave 2+

---

## Task Overview theo Wave

### Wave 1 — Foundation: Auth & RBAC — **9h**
> Auth là foundation cho tất cả services. Phải hoàn thành TRƯỚC mọi Wave khác.

| Task | File | Mô tả | Ước tính |
|------|------|-------|---------|
| TASK-SM-001 | [TASK-SM-001-auth-apikey-rbac.md](./TASK-SM-001-auth-apikey-rbac.md) | `sm_` API Key (base62) + RBAC 4-role + Redis token cache | 3h |
| TASK-SM-002 | [TASK-SM-002-auth-organization-invitation.md](./TASK-SM-002-auth-organization-invitation.md) | Organization model + OrgMember + Invitation (7-day token) | 3h |
| TASK-SM-003 | [TASK-SM-003-auth-oauth2-server.md](./TASK-SM-003-auth-oauth2-server.md) | OAuth2 Authorization Server + PKCE + token endpoint | 3h |

### Wave 2 — Core Memory: Document & Memory Engine — **9h**
> Document ingestion pipeline + Knowledge Graph. Parallel: SM-004+SM-005 và SM-006 độc lập.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-SM-004 | [TASK-SM-004-document-domain-extractors.md](./TASK-SM-004-document-domain-extractors.md) | Document domain (11 types, 7-stage status) + Extractor Registry + PDF/HTML/Image | SM-001 | 4h |
| TASK-SM-005 | [TASK-SM-005-document-chunker-pipeline.md](./TASK-SM-005-document-chunker-pipeline.md) | AST/Semantic/Fixed chunkers + Ingestion Pipeline + NATS Worker Pool | SM-004 | 4h |
| TASK-SM-006 | [TASK-SM-006-memory-fact-extraction-kg.md](./TASK-SM-006-memory-fact-extraction-kg.md) | MemoryEntry upgrade + Fact Extraction LLM + Relation Graph + Auto-Forget | SM-005 | 5h |

### Wave 3 — Intelligence: Search & Profile — **8h**
> Hybrid Search + User Profile. Có thể parallel SM-007 và SM-008.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-SM-007 | [TASK-SM-007-search-hybrid-engine.md](./TASK-SM-007-search-hybrid-engine.md) | 3-goroutine parallel search (chunks+memories+docs) + JSONB filter + V4 API | SM-006 | 5h |
| TASK-SM-008 | [TASK-SM-008-profile-service.md](./TASK-SM-008-profile-service.md) | UserProfile (Static/Dynamic) + Redis cache + NATS invalidation + ProfileSearch combo | SM-006 | 3h |

### Wave 4 — Integrations: Connectors & MCP — **9h**
> External connectors + MCP server. Có thể parallel SM-009 và SM-010.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-SM-009 | [TASK-SM-009-connector-google-notion.md](./TASK-SM-009-connector-google-notion.md) | Connector Service: Google Drive OAuth2 + Notion API + AES-GCM token vault + Sync cron | SM-001, SM-005 | 5h |
| TASK-SM-010 | [TASK-SM-010-mcp-server-tools.md](./TASK-SM-010-mcp-server-tools.md) | MCP Server upgrade: 8 tools, OAuth2 auth, RBAC enforcement | SM-007, SM-008, SM-005 | 4h |

### Wave 5 — Ecosystem: SDK & Analytics — **9h**
> SDK + integrations + analytics. Có thể parallel SM-011 và SM-012.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-SM-011 | [TASK-SM-011-analytics-token-economics.md](./TASK-SM-011-analytics-token-economics.md) | Analytics Service: token usage + memory stats + daily aggregation | SM-005, SM-006 | 4h |
| TASK-SM-012 | [TASK-SM-012-sdk-framework-integrations.md](./TASK-SM-012-sdk-framework-integrations.md) | Go SDK + LangChain Python + Vercel AI TypeScript | SM-007, SM-008 | 5h |

---

## Dependency Graph

```
Wave 1 (parallel):
TASK-SM-001 (sm_ apikey + RBAC) ─────────────────────────────┐
TASK-SM-002 (Organization + Invitation) ─────────────────────┤
TASK-SM-003 (OAuth2 Server) ─────────────────────────────────┘
                                          │ Wave 1 done
Wave 2:                                   │
TASK-SM-004 (document domain+extractors) ─┤
  ↓                                       │
TASK-SM-005 (chunkers+pipeline) ──────────┤ (parallel with SM-006 start)
  ↓                                       │
TASK-SM-006 (memory fact extraction) ─────┘
                                          │ Wave 2 done
Wave 3 (parallel):                        │
├── TASK-SM-007 (hybrid search) ──────────┤
└── TASK-SM-008 (profile service) ────────┘
                                          │ Wave 3 done
Wave 4 (parallel):                        │
├── TASK-SM-009 (connectors) ─────────────┤
└── TASK-SM-010 (MCP server) ─────────────┘
                                          │ Wave 4 done
Wave 5 (parallel):                        │
├── TASK-SM-011 (analytics) ──────────────┤
└── TASK-SM-012 (SDK + integrations) ─────┘
```

---

## Monorepo Target Paths

```
vnp-memory/
├── services/
│   ├── vnp-platform/            ← TASK-SM-001, SM-002, SM-003
│   ├── document-service/        ← TASK-SM-004, SM-005 (NEW)
│   ├── memory-service/          ← TASK-SM-006 (upgrade SM domain)
│   ├── search-service/          ← TASK-SM-007 (upgrade)
│   ├── profile-service/         ← TASK-SM-008 (NEW)
│   ├── connector-service/       ← TASK-SM-009 (NEW)
│   └── analytics-service/       ← TASK-SM-011 (NEW)
├── gateway/adapter/
│   └── mcp/                     ← TASK-SM-010 (upgrade)
└── packages/
    ├── sdk/go/                  ← TASK-SM-012
    └── integrations/
        ├── python/vnp-langchain/ ← TASK-SM-012
        └── typescript/vnp-vercel-ai/ ← TASK-SM-012
```

---

## Key Design Decisions

| Quyết định | Task | Lý do |
|-----------|------|-------|
| `sm_` prefix cho API keys | SM-001 | Phân biệt rõ với internal tokens, dễ grep trong logs |
| PKCE required cho OAuth2 | SM-003 | Public clients (MCP) không có client secret |
| SHA-256 contentHash dedup | SM-004 | Deterministic, không cần lookup extra |
| 3-goroutine parallel search | SM-007 | Latency p95 < 500ms với HNSW indexes |
| AES-GCM cho OAuth tokens | SM-009 | Industry standard, IV random → different ciphertext |
| Redis cache TTL 5 min | SM-001, SM-008 | Hot path: token validation + profile < 100ms |
| Batch INSERT 100 events | SM-011 | Analytics write throughput without DB lock contention |

---

## Tổng ước tính: ~44h AI execution time

| Wave | Tasks | Thời gian |
|------|-------|---------|
| Wave 1 (Auth/RBAC) | 3 tasks | 9h |
| Wave 2 (Core Memory) | 3 tasks | 13h |
| Wave 3 (Intelligence) | 2 tasks | 8h |
| Wave 4 (Integrations) | 2 tasks | 9h |
| Wave 5 (Ecosystem) | 2 tasks | 9h |
| **TOTAL** | **12 tasks** | **~48h** |
