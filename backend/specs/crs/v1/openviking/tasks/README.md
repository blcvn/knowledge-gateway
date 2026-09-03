# OpenViking — Implementation Tasks

**Dự án:** VNP Memory — OpenViking Context Database  
**Nguồn:** [Solutions](../solutions/)  
**Ngày tạo:** 2026-06-17  
**Trạng thái:** Ready for Execution

> Các tác vụ này được thiết kế để **AI agent thực thi tuần tự**. Mỗi task là một đơn vị công việc độc lập, có đầu vào rõ ràng, tiêu chí hoàn thành có thể kiểm chứng bằng code/test.

---

## Quy tắc thực thi

1. **Thứ tự Wave là bắt buộc** — Wave N phải hoàn thành trước Wave N+1
2. **Trong cùng Wave** — các task có thể chạy song song
3. **Mỗi task kết thúc** bằng `go build ./...` không có lỗi
4. **Unit test phải pass** trước khi sang task tiếp theo
5. **Không được sửa** code trong `services/kgs-platform/` (read-only external service)

---

## Tổng quan theo Wave

### Wave 1 — Foundation (`pkg/`)
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-OV-001 | [TASK-OV-001-pkg-viking.md](./TASK-OV-001-pkg-viking.md) | `pkg/viking/` — URI system, Identity, RBAC, Errors | 2h |
| TASK-OV-002 | [TASK-OV-002-pkg-vikingfs.md](./TASK-OV-002-pkg-vikingfs.md) | `pkg/vikingfs/` — LocalFileSystem + PathLock | 3h |
| TASK-OV-003 | [TASK-OV-003-pkg-adapters.md](./TASK-OV-003-pkg-adapters.md) | `pkg/adapters/` — 5 infrastructure interfaces + implementations | 4h |
| TASK-OV-004 | [TASK-OV-004-pkg-infra.md](./TASK-OV-004-pkg-infra.md) | `pkg/` infra — nats, resilience, middleware, observability, auth, parse | 4h |

### Wave 2 — Security Services
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-OV-005 | [TASK-OV-005-crypto-service.md](./TASK-OV-005-crypto-service.md) | `services/openviking-crypto` — OVE1 format, AES-256-GCM, KMS adapters | 4h |
| TASK-OV-006 | [TASK-OV-006-admin-service.md](./TASK-OV-006-admin-service.md) | `services/openviking-admin` — Accounts, Users, API Keys, Health aggregation | 5h |

### Wave 3 — Storage Service
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-OV-007 | [TASK-OV-007-fs-domain-usecase.md](./TASK-OV-007-fs-domain-usecase.md) | `services/openviking-fs` — Domain model + Use cases (read/write/rm/mv) | 4h |
| TASK-OV-008 | [TASK-OV-008-fs-grep-relations.md](./TASK-OV-008-fs-grep-relations.md) | `services/openviking-fs` — Grep pool, Glob, Relations, Privacy config | 3h |
| TASK-OV-009 | [TASK-OV-009-fs-grpc-nats.md](./TASK-OV-009-fs-grpc-nats.md) | `services/openviking-fs` — gRPC server, NATS events, integration | 3h |

### Wave 4 — Search Service
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-OV-010 | [TASK-OV-010-search-retriever.md](./TASK-OV-010-search-retriever.md) | `services/openviking-search` — HierarchicalRetriever 6-step + Priority Queue | 5h |
| TASK-OV-011 | [TASK-OV-011-search-index-nats.md](./TASK-OV-011-search-index-nats.md) | `services/openviking-search` — Index/Hotness/NATS sync + gRPC server | 3h |

### Wave 5 — Context Services
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-OV-012 | [TASK-OV-012-session-phase1.md](./TASK-OV-012-session-phase1.md) | `services/openviking-session` — Phase 1 archive, AddMessages, token tracking | 4h |
| TASK-OV-013 | [TASK-OV-013-session-phase2.md](./TASK-OV-013-session-phase2.md) | `services/openviking-session` — Phase 2 WM v2, memory extraction, redo log | 4h |
| TASK-OV-014 | [TASK-OV-014-resource-pipeline.md](./TASK-OV-014-resource-pipeline.md) | `services/openviking-resource` — 7-step ingestion pipeline, source adapters | 5h |
| TASK-OV-015 | [TASK-OV-015-resource-watch-skills.md](./TASK-OV-015-resource-watch-skills.md) | `services/openviking-resource` — WatchManager, Skill loading, Task tracking | 3h |

### Wave 6 — Gateway
| Task | File | Mô tả | Ước tính |
|---|---|---|---|
| TASK-OV-016 | [TASK-OV-016-gateway-rest.md](./TASK-OV-016-gateway-rest.md) | `services/openviking-gateway` — REST router, 17 route groups, auth middleware | 5h |
| TASK-OV-017 | [TASK-OV-017-gateway-mcp-webdav.md](./TASK-OV-017-gateway-mcp-webdav.md) | `services/openviking-gateway` — MCP 9 tools, WebDAV proxy, circuit breakers | 4h |

---

## Dependency Graph

```
TASK-001 (pkg/viking)
    ↓
TASK-002 (pkg/vikingfs) ──┐
TASK-003 (pkg/adapters) ──┤
TASK-004 (pkg/infra)    ──┘
    ↓
TASK-005 (crypto) ──┐
TASK-006 (admin)  ──┘
    ↓
TASK-007 (fs domain+usecase) → TASK-008 (fs grep) → TASK-009 (fs grpc)
    ↓
TASK-010 (search retriever) → TASK-011 (search index)
    ↓
TASK-012 (session phase1) → TASK-013 (session phase2)
TASK-014 (resource pipeline) → TASK-015 (resource watch)
    ↓
TASK-016 (gateway REST) → TASK-017 (gateway MCP+WebDAV)
```

---

## Monorepo Paths (Target)

```
vnp-memory/
├── pkg/
│   ├── viking/
│   ├── vikingfs/
│   ├── adapters/{vectordb,embedder,vlm,reranker,kms}/
│   ├── nats/
│   ├── resilience/
│   ├── middleware/{auth,logging,ratelimit,recovery}/
│   ├── observability/
│   ├── auth/
│   └── parse/
└── services/
    ├── openviking-crypto/
    ├── openviking-admin/
    ├── openviking-fs/
    ├── openviking-search/
    ├── openviking-session/
    ├── openviking-resource/
    └── openviking-gateway/
```

**Tổng ước tính:** ~58 giờ AI execution time
