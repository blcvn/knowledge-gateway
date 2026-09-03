# Architecture Decision Records — VNP Memory

> ADRs ghi lại các quyết định kiến trúc quan trọng, lý do chọn, và các phương án đã cân nhắc.
> **Format:** [Michael Nygard ADR template](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)

---

## Index

| ADR | Tiêu đề | Status | Date |
|---|---|---|---|
| [ADR-001](./ADR-001-monolith-first.md) | Monolith-first với InProcessRegistry | ✅ Accepted | 2026-01 |
| [ADR-002](./ADR-002-grpc-bufconn.md) | gRPC + bufconn cho inter-service communication | ✅ Accepted | 2026-01 |
| [ADR-003](./ADR-003-nats-jetstream.md) | NATS JetStream làm message broker | ✅ Accepted | 2026-02 |
| [ADR-004](./ADR-004-postgres-primary.md) | PostgreSQL + pgvector làm primary data store | ✅ Accepted | 2026-01 |
| [ADR-005](./ADR-005-six-memory-engines.md) | 6 memory engines thay vì 1 unified engine | ✅ Accepted | 2026-02 |
| [ADR-006](./ADR-006-tenantid-isolation.md) | TenantID column-level isolation (không dùng schema separation) | ✅ Accepted | 2026-03 |
| [ADR-007](./ADR-007-memobase-yolo.md) | Memobase YOLO Engine: 3 fixed LLM calls | ✅ Accepted | 2026-03 |
| [ADR-008](./ADR-008-consolidation-4tier.md) | Memory Consolidation 4-tier pipeline (Sleep model) | ✅ Accepted | 2026-04 |
| [ADR-009](./ADR-009-mcp-dual-transport.md) | MCP Server: SSE + HTTP Streamable (dual transport) | ✅ Accepted | 2026-04 |
| [ADR-010](./ADR-010-rrf-hybrid-search.md) | Hybrid Search: BM25 + Vector + RRF Fusion | ✅ Accepted | 2026-03 |
| [ADR-011](./ADR-011-distributed-leases.md) | Distributed Leases cho Multi-Agent Coordination | ✅ Accepted | 2026-05 |
| [ADR-012](./ADR-012-bifrost-llm-routing.md) | Bifrost làm LLM multi-provider router | ✅ Accepted | 2026-02 |

---

## Hướng dẫn đọc

- **Status:** `Proposed` → `Accepted` → `Deprecated` → `Superseded`
- Mỗi ADR có: Context → Decision → Consequences → Alternatives Considered
- ADR là immutable sau khi Accepted; nếu thay đổi → tạo ADR mới Superseding ADR cũ
