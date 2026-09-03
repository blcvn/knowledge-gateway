# agentmemory — Bộ Test Requirements

Thư mục này chứa bộ **test requirements** (yêu cầu kiểm thử) cho hệ thống **agentmemory** — Persistent Memory Engine cho AI Coding Agents.

## Nguồn tài liệu

Bộ test requirements này được tổng hợp từ:
- `docs/PRD.md` — Product Requirements Document
- `docs/SRS.md` — Software Requirements Specification  
- `docs/URD.md` — User Requirements Document
- `specs/architecture.md` — Architecture Document
- `specs/tdd.md` — Technical Design Document
- Source code tại `src/` và test suite tại `test/`

## Cấu trúc thư mục

```
requirements/
├── README.md                          # File này
├── TR-000-overview.md                 # Tổng quan & cross-cutting concerns
├── TR-001-session-management.md       # Session lifecycle tests
├── TR-002-observe-pipeline.md         # Observation capture & dedup tests
├── TR-003-compress-synthetic.md       # Synthetic compression tests
├── TR-004-search-bm25.md             # BM25 search engine tests
├── TR-005-search-vector.md           # Vector index tests
├── TR-006-search-hybrid.md           # Hybrid search (RRF fusion) tests
├── TR-007-memory-management.md       # Long-term memory & versioning tests
├── TR-008-knowledge-graph.md         # Knowledge graph tests
├── TR-009-consolidation-pipeline.md  # Memory consolidation pipeline tests
├── TR-010-context-injection.md       # Context building & injection tests
├── TR-011-multi-agent.md             # Multi-agent coordination tests
├── TR-012-orchestration.md           # Actions, routines, checkpoints tests
├── TR-013-governance-audit.md        # Governance, audit, privacy tests
├── TR-014-mcp-server.md              # MCP server & tools tests
├── TR-015-rest-api.md                # REST API tests
├── TR-016-session-replay.md          # Session replay tests
├── TR-017-memory-slots.md            # Memory slots & working memory tests
├── TR-018-provider-system.md         # LLM & embedding providers tests
├── TR-019-health-diagnostics.md      # Health monitoring & diagnostics tests
├── TR-020-security.md                # Security & authentication tests
├── TR-021-performance.md             # Performance & benchmark tests
├── TR-022-deployment.md              # Deployment & configuration tests
└── TR-023-export-import.md           # Export/import & migration tests
```

## Quy ước đặt tên Test Case

Mỗi test case sử dụng định danh: `TR-XXX-YYY-NNN`
- `TR` = Test Requirement
- `XXX` = Module ID (3 chữ số)
- `YYY` = Category code (3 chữ cái viết hoa)
- `NNN` = Sequential number trong category

**Ví dụ:** `TR-002-OBS-001` = Test Requirement, Module Observe, case số 1

## Priority Levels

| Priority | Ký hiệu | Ý nghĩa |
|---|---|---|
| Critical | 🔴 P0 | Phải pass để hệ thống hoạt động |
| High | 🟠 P1 | Core features, phải test trước release |
| Medium | 🟡 P2 | Important features, test trong regression |
| Low | 🟢 P3 | Edge cases, nice-to-have |

## Test Types

| Type | Ký hiệu | Mô tả |
|---|---|---|
| Unit | `[UNIT]` | Pure function, không cần iii-engine |
| Integration | `[INT]` | Mock iii-sdk, test business logic |
| E2E | `[E2E]` | Full stack, cần iii-engine running |
| Performance | `[PERF]` | Latency/throughput benchmarks |
| Security | `[SEC]` | Security & privacy validation |

## Traceability Matrix (Summary)

| Requirement ID | Test File | Test Count |
|---|---|---|
| FR-SESSION-001..004 | TR-001 | 15 |
| FR-OBS-001..004 | TR-002 | 20 |
| FR-COMPRESS-001..004 | TR-003 | 12 |
| FR-SEARCH-001..005 | TR-004, TR-005, TR-006 | 35 |
| FR-MEM-001..005 | TR-007 | 18 |
| FR-GRAPH-001..004 | TR-008 | 16 |
| FR-CONSOL-001..004 | TR-009 | 12 |
| FR-CTX-001..002 | TR-010 | 10 |
| FR-MULTI-001..005 | TR-011 | 15 |
| FR-ORCH-001..005 | TR-012 | 20 |
| FR-GOV-001..003 | TR-013 | 14 |
| FR-REPLAY-001..002 | TR-016 | 10 |
| FR-SLOTS-001..002 | TR-017 | 10 |
| FR-DIAG-001..003 | TR-019 | 12 |
| UR-001..040 | All files | mapped |
| NFR (Performance) | TR-021 | 8 |
| NFR (Security) | TR-020 | 12 |

**Tổng số test requirements:** ~240+

---

*Tài liệu này được tạo tự động từ phân tích PRD + SRS + URD + TDD của agentmemory.*
