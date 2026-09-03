# Memobase — Implementation Tasks

**Dự án:** VNP Memory — Memobase Profile-Based Memory System  
**Nguồn:** [Solutions](../solutions/)  
**Ngày tạo:** 2026-06-17  
**Trạng thái:** Ready for Execution

> Các tác vụ này được thiết kế để **AI agent thực thi tuần tự theo Wave**. Mỗi task là một đơn vị công việc độc lập, có đầu vào rõ ràng, tiêu chí hoàn thành có thể kiểm chứng.

---

## Quy tắc thực thi

1. **Thứ tự Wave là bắt buộc** — Wave N phải hoàn thành trước Wave N+1
2. **Trong cùng Wave** — các task song song được phép chạy đồng thời
3. **Mỗi task kết thúc** bằng `go build ./...` không có lỗi
4. **Unit test phải pass** trước khi sang task tiếp theo
5. **Monolith integration** — đăng ký vào `apps/memory/internal/bootstrap/` sau khi mỗi service hoàn thành
6. **Composite PK** — mọi table đều dùng `(id, project_id)` primary key

---

## Tổng quan theo Wave

### Wave 1 — Data Layer (song song)
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-MB-001 | [TASK-MB-001-pkg-foundation.md](./TASK-MB-001-pkg-foundation.md) | `pkg/tokenizer/`, `pkg/config/` — nền tảng dùng ngay từ Wave 1 | 2h |
| TASK-MB-002 | [TASK-MB-002-admin-service.md](./TASK-MB-002-admin-service.md) | `services/memobase-admin` — User, Project, Billing, API Keys | 5h |
| TASK-MB-003 | [TASK-MB-003-ingestion-domain.md](./TASK-MB-003-ingestion-domain.md) | `services/memobase-ingestion` — Domain + DB migrations + Blob CRUD | 3h |
| TASK-MB-004 | [TASK-MB-004-ingestion-usecases.md](./TASK-MB-004-ingestion-usecases.md) | `services/memobase-ingestion` — InsertBlob, FlushBuffer, AutoFlush, gRPC | 4h |

### Wave 2 — LLM Processing
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-MB-005 | [TASK-MB-005-pkg-llm-adapters.md](./TASK-MB-005-pkg-llm-adapters.md) | `pkg/adapters/llm/`, `pkg/adapters/embedder/`, `pkg/resilience/` | 4h |
| TASK-MB-006 | [TASK-MB-006-pkg-prompt.md](./TASK-MB-006-pkg-prompt.md) | `pkg/prompt/` — EN/ZH prompt registry, 5 prompt templates | 3h |
| TASK-MB-007 | [TASK-MB-007-engine-domain-pipeline.md](./TASK-MB-007-engine-domain-pipeline.md) | `services/memobase-engine` — Domain + 3-call LLM pipeline | 5h |
| TASK-MB-008 | [TASK-MB-008-engine-yolo-merge.md](./TASK-MB-008-engine-yolo-merge.md) | `services/memobase-engine` — YOLO merge, organize, validate, gRPC | 4h |

### Wave 3 — Read Path (song song)
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-MB-009 | [TASK-MB-009-context-service.md](./TASK-MB-009-context-service.md) | `services/memobase-context` — Profile read, Redis cache, context assembly | 5h |
| TASK-MB-010 | [TASK-MB-010-event-service.md](./TASK-MB-010-event-service.md) | `services/memobase-event` — Event timeline, pgvector search, NATS | 5h |

### Wave 4 — Access Layer
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-MB-011 | [TASK-MB-011-pkg-observability.md](./TASK-MB-011-pkg-observability.md) | `pkg/observability/`, `pkg/middleware/` — OTel, Prometheus, auth middleware | 3h |
| TASK-MB-012 | [TASK-MB-012-gateway-rest.md](./TASK-MB-012-gateway-rest.md) | Gateway REST routes — User, Blob, Memory, Context endpoints | 5h |
| TASK-MB-013 | [TASK-MB-013-gateway-mcp.md](./TASK-MB-013-gateway-mcp.md) | Gateway MCP server — 8 tools cho AI agents | 4h |

---

## Dependency Graph

```
TASK-001 (pkg: tokenizer, config)
    ↓
TASK-002 (admin) ──────────────────────────────────────────┐
TASK-003 (ingestion domain) → TASK-004 (ingestion usecases)│
    ↓                                                       │
TASK-005 (pkg: llm, embedder, resilience)                  │
TASK-006 (pkg: prompt)                                      │
    ↓                                                       │
TASK-007 (engine domain+pipeline) → TASK-008 (engine YOLO) │
    ↓                                                       │
TASK-009 (context service) ← ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─┘
TASK-010 (event service)   ← (song song với TASK-009)
    ↓
TASK-011 (pkg: observability, middleware)
    ↓
TASK-012 (gateway REST) → TASK-013 (gateway MCP)
```

---

## Monorepo Paths (Target)

```
vnp-memory/
├── pkg/
│   ├── tokenizer/
│   ├── config/
│   ├── adapters/{llm,embedder}/
│   ├── prompt/{en,zh}/
│   ├── resilience/
│   ├── observability/
│   └── middleware/
└── services/
    ├── memobase-ingestion/   (port 9041)
    ├── memobase-engine/      (port 9042)
    ├── memobase-context/     (port 9043)
    ├── memobase-event/       (port 9044)
    └── memobase-admin/       (port 9045)
```

**Tổng ước tính:** ~48 giờ AI execution time
