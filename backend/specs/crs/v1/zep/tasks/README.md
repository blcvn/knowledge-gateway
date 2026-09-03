# Zep Feature Parity — Implementation Tasks

**Dự án:** VNP Memory — Zep End-to-End Context Engineering Platform  
**Nguồn:** [Solutions](../solutions/)  
**Ngày tạo:** 2026-06-17  
**Trạng thái:** Ready for Execution

> Các tác vụ này được thiết kế để **AI agent thực thi tuần tự theo Wave**. Mỗi task là một đơn vị công việc độc lập, có input rõ ràng, acceptance criteria kiểm chứng được bằng code/test.

---

## Quy tắc Thực thi

1. **Thứ tự Wave là bắt buộc** — Wave N phải hoàn thành trước Wave N+1
2. **Trong cùng Wave** — các task KHÔNG có dependency với nhau có thể chạy song song
3. **Mỗi task kết thúc** bằng `go build ./...` HOẶC `make ci` không có lỗi
4. **Unit test phải pass** trước khi sang task tiếp theo
5. **KHÔNG được sửa** code trong `services/kgs-platform/` (read-only external)
6. **Infrastructure first:** TASK-ZEP-010 (Neo4j + Graphiti) phải chạy trước Wave 4

---

## Task Overview theo Wave

### Wave 1 — Foundation (`pkg/`) — **8 ngày**
> Shared packages dùng bởi tất cả Zep services. PHẢI hoàn thành trước.

| Task | File | Mô tả | Ước tính |
|------|------|-------|---------|
| TASK-ZEP-001 | [TASK-ZEP-001-pkg-resilience.md](./TASK-ZEP-001-pkg-resilience.md) | `pkg/resilience/` — Circuit Breaker (sony/gobreaker) + Exponential Backoff Retry | 3h |
| TASK-ZEP-002 | [TASK-ZEP-002-pkg-metadata-advisory-lock.md](./TASK-ZEP-002-pkg-metadata-advisory-lock.md) | `pkg/metadata/` — PostgreSQL Advisory Lock shared package | 2h |
| TASK-ZEP-003 | [TASK-ZEP-003-pkg-middleware-stack.md](./TASK-ZEP-003-pkg-middleware-stack.md) | `pkg/middleware/` — 10-layer chi middleware stack | 2h |
| TASK-ZEP-004 | [TASK-ZEP-004-pkg-telemetry.md](./TASK-ZEP-004-pkg-telemetry.md) | `pkg/telemetry/` — Anonymous opt-out telemetry tracker | 1h |

### Wave 2 — Core CRUD — **15 ngày**
> Thread lifecycle management + Admin/project management. Có thể parallel.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-ZEP-005 | [TASK-ZEP-005-thread-domain-schema.md](./TASK-ZEP-005-thread-domain-schema.md) | `services/zep-thread/` — Domain model + PostgreSQL schema + advisory lock | ZEP-002 | 2h |
| TASK-ZEP-006 | [TASK-ZEP-006-thread-usecases-grpc.md](./TASK-ZEP-006-thread-usecases-grpc.md) | `services/zep-thread/` — 9 use cases + gRPC server + proto | ZEP-005 | 3h |
| TASK-ZEP-007 | [TASK-ZEP-007-admin-service.md](./TASK-ZEP-007-admin-service.md) | `services/zep-admin/` — Project CRUD + `vnp_` API keys + health aggregation + NATS cascade | ZEP-001 | 4h |

### Wave 3 — Memory Core — **8 ngày**
> Message ingestion (sub-200ms) + Context assembly. Depends on Thread Service.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-ZEP-008 | [TASK-ZEP-008-memory-domain-put-memory.md](./TASK-ZEP-008-memory-domain-put-memory.md) | `services/memory-service/` — Message domain upgrade (RoleType 6 values) + PutMemory sub-200ms | ZEP-006 | 4h |
| TASK-ZEP-009 | [TASK-ZEP-009-memory-get-context.md](./TASK-ZEP-009-memory-get-context.md) | `services/memory-service/` — GetMemory (graceful degradation) + GetUserContext formatter | ZEP-008 | 3h |

### Wave 4 — Graph Intelligence — **24.5 ngày**
> Temporal KG extraction + Semantic search. Infrastructure phải sẵn sàng.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-ZEP-010 | [TASK-ZEP-010-infra-neo4j-graphiti.md](./TASK-ZEP-010-infra-neo4j-graphiti.md) | **Infrastructure:** Neo4j 5.22+ upgrade + Graphiti Python service deploy + schema | ZEP-008 | 2h |
| TASK-ZEP-011 | [TASK-ZEP-011-graph-domain-neo4j.md](./TASK-ZEP-011-graph-domain-neo4j.md) | `services/zep-graph/` — 9-node ontology + TemporalEdge (ValidAt/InvalidAt) + Neo4j repos | ZEP-010 | 4h |
| TASK-ZEP-012 | [TASK-ZEP-012-graph-extraction-pipeline.md](./TASK-ZEP-012-graph-extraction-pipeline.md) | `services/zep-graph/` — NATS consumer → Graphiti → Neo4j + AddGraphData + SetOntology | ZEP-011 | 4h |
| TASK-ZEP-013 | [TASK-ZEP-013-search-graph-rerankers.md](./TASK-ZEP-013-search-graph-rerankers.md) | `services/search-service/` — Multi-scope search + 5 rerankers + Redis cache + NATS invalidation | ZEP-011 | 5h |

### Wave 5 — Integration — **19.5 ngày**
> MCP server + Framework Python packages. Có thể parallel.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-ZEP-014 | [TASK-ZEP-014-mcp-server-13-tools.md](./TASK-ZEP-014-mcp-server-13-tools.md) | `services/zep-mcp/` — 13 read-only tools, stdio+HTTP dual transport, Docker distroless | ZEP-009, ZEP-013 | 4h |
| TASK-ZEP-015 | [TASK-ZEP-015-python-integrations.md](./TASK-ZEP-015-python-integrations.md) | 4 Python packages: AutoGen + CrewAI + ADK + LiveKit, mypy strict, coverage > 90% | ZEP-009 | 6h |

### Wave 6 — Quality — **10 ngày**
> Evaluation harness. Yêu cầu full system đang chạy.

| Task | File | Mô tả | Depends on | Ước tính |
|------|------|-------|-----------|---------|
| TASK-ZEP-016 | [TASK-ZEP-016-eval-harness.md](./TASK-ZEP-016-eval-harness.md) | `tools/eval-harness/` — 4-step eval pipeline + LoCoMo + LongMemEval benchmarks | ALL | 5h |

---

## Dependency Graph

```
Wave 1 (Foundation — parallel):
TASK-001 (pkg/resilience) ─────────────────────────────────┐
TASK-002 (pkg/metadata)   ─────────────────────────────────┤
TASK-003 (pkg/middleware)  ─────────────────────────────────┤
TASK-004 (pkg/telemetry)   ─────────────────────────────────┘
                                          │ All Wave 1 done
Wave 2 (parallel pairs):                  │
├── TASK-005 (thread domain) → TASK-006 (thread usecases) ──┤
└── TASK-007 (admin service) ─────────────────────────────── │
                                          │ Wave 2 done
Wave 3 (sequential):                      │
TASK-008 (put_memory) → TASK-009 (get_memory) ─────────────┤
                                          │ Wave 3 done
Wave 4 (sequential start, then parallel): │
TASK-010 (neo4j+graphiti) → TASK-011 (graph domain) ────── │
                               ├── TASK-012 (extraction pipeline)
                               └── TASK-013 (search + rerankers)
                                          │ Wave 4 done
Wave 5 (parallel):                        │
├── TASK-014 (MCP 13 tools) ─────────────┤
└── TASK-015 (Python integrations) ───── ┘
                                          │ Wave 5 done
Wave 6:                                   │
TASK-016 (eval harness) ─────────────────┘
```

---

## Monorepo Target Paths

```
vnp-memory/
├── pkg/
│   ├── resilience/          ← TASK-001
│   ├── metadata/            ← TASK-002
│   ├── middleware/          ← TASK-003
│   └── telemetry/           ← TASK-004
├── services/
│   ├── zep-thread/          ← TASK-005, TASK-006
│   ├── zep-admin/           ← TASK-007
│   ├── memory-service/      ← TASK-008, TASK-009 (Zep domain upgrade)
│   ├── zep-graph/           ← TASK-011, TASK-012
│   ├── search-service/      ← TASK-013 (Zep graph search upgrade)
│   └── zep-mcp/             ← TASK-014
├── packages/integrations/python/
│   ├── vnp-autogen/         ← TASK-015
│   ├── vnp-crewai/          ← TASK-015
│   ├── vnp-adk/             ← TASK-015
│   └── vnp-livekit/         ← TASK-015
├── tools/
│   └── eval-harness/        ← TASK-016
└── deploy/dev/
    ├── docker-compose.server.yaml  ← TASK-010 (Neo4j upgrade + Graphiti)
    └── neo4j/                      ← TASK-010 (schema)
```

---

## Infrastructure Prerequisites (TASK-ZEP-010)

> [!IMPORTANT]
> Các components sau PHẢI sẵn sàng trước Wave 4:
>
> | Component | Action | Ghi chú |
> |-----------|--------|---------|
> | **Neo4j 5.22+** | UPGRADE từ version hiện tại | Vector index requires 5.22+ |
> | **Graphiti service** | NEW deploy (Python container) | Cần OPENAI_API_KEY |
> | **OPENAI_API_KEY** | Set trong .env | Cho LLM entity extraction |

---

## Key Design Decisions

| Quyết định | Task | Lý do |
|-----------|------|-------|
| `pkg/metadata` là shared package | ZEP-002 | Thread + Memory + Admin đều dùng advisory lock |
| Advisory lock key = SHA-256 → int64 | ZEP-002 | Collision probability 1/(2^64) |
| PutMemory sub-200ms (NATS async) | ZEP-008 | Graph extraction 10-20s là OK (async) |
| Graceful degradation GetMemory | ZEP-009 | Search down → messages still returned |
| TemporalEdge.ValidAt từ Graphiti LLM | ZEP-011 | Core Zep differentiator |
| 13 MCP tools read-only ONLY | ZEP-014 | Safety by design |
| mypy --strict cho Python packages | ZEP-015 | Type safety cho framework interfaces |

---

## Tổng ước tính: ~38h AI execution time

| Wave | Tasks | Thời gian |
|------|-------|---------|
| Wave 1 (Foundation) | 4 tasks | 8h |
| Wave 2 (CRUD) | 3 tasks | 9h |
| Wave 3 (Memory) | 2 tasks | 7h |
| Wave 4 (Graph) | 4 tasks | 15h |
| Wave 5 (Integration) | 2 tasks | 10h |
| Wave 6 (Quality) | 1 task | 5h |
| **TOTAL** | **16 tasks** | **~54h** |
