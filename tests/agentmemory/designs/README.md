# agentmemory — Bộ Test Design

Thư mục này chứa **Test Design** (Thiết kế Kiểm thử) chi tiết cho hệ thống **agentmemory**.

## Mối quan hệ với Test Requirements

```
Test Requirements (../requirements/)
        ↓  [WHAT to test]
Test Design     (./designs/)
        ↓  [HOW to test - strategy, approach, test cases]
Test Implementation (../specs/)
        ↓  [actual test code]
```

## Cấu trúc thư mục

```
designs/
├── README.md                         # File này
├── TD-000-test-infrastructure.md     # Chiến lược, môi trường, fixtures
├── TD-001-session-management.md
├── TD-002-observe-pipeline.md
├── TD-003-compress-synthetic.md
├── TD-004-search-bm25.md
├── TD-005-search-vector.md
├── TD-006-search-hybrid.md
├── TD-007-memory-management.md
├── TD-008-knowledge-graph.md
├── TD-009-consolidation-pipeline.md
├── TD-010-context-injection.md
├── TD-011-multi-agent.md
├── TD-012-orchestration.md
├── TD-013-governance-audit.md
├── TD-014-mcp-server.md
├── TD-015-rest-api.md
├── TD-016-session-replay.md
├── TD-017-memory-slots.md
├── TD-018-provider-system.md
├── TD-019-health-diagnostics.md
├── TD-020-security.md
├── TD-021-performance.md
├── TD-022-deployment.md
└── TD-023-export-import.md
```

## Quy ước Test Design

### Cấu trúc Test Case

Mỗi test case gồm:
- **ID:** `TC-NNN`
- **Requirement:** tham chiếu TR-XXX-YYY-NNN
- **Type:** `unit` / `integration` / `e2e` / `performance` / `security`
- **Priority:** 🔴 P0 / 🟠 P1 / 🟡 P2
- **Given / When / Then:** mô tả ngôn ngữ tự nhiên
- **Test Data:** dữ liệu đầu vào cụ thể
- **Kỹ thuật:** boundary value, equivalence partition, state transition, v.v.

### Không có code trong Test Design

Test Design là tài liệu **thiết kế** — mô tả *cái gì* sẽ được kiểm tra và *tại sao*, không phải *code thế nào*. Implementation được đặt riêng ở `tests/agentmemory/specs/`.

## Ma trận Traceability

| Test Design | Requirement | Source Module |
|---|---|---|
| TD-001 | TR-001 | `functions/observe.ts` |
| TD-002 | TR-002 | `functions/observe.ts`, `dedup.ts`, `privacy.ts` |
| TD-003 | TR-003 | `functions/compress-synthetic.ts` |
| TD-004 | TR-004 | `state/search-index.ts` |
| TD-005 | TR-005 | `state/vector-index.ts` |
| TD-006 | TR-006 | `state/hybrid-search.ts` |
| TD-007 | TR-007 | `functions/remember.ts` |
| TD-008 | TR-008 | `functions/graph.ts` |
| TD-009 | TR-009 | `functions/consolidation-pipeline.ts` |
| TD-010 | TR-010 | `functions/context.ts` |
| TD-011 | TR-011 | `functions/leases.ts`, `signals.ts` |
| TD-012 | TR-012 | `functions/actions.ts`, `routines.ts`, `sketches.ts` |
| TD-013 | TR-013 | `functions/governance.ts`, `audit.ts`, `privacy.ts` |
| TD-014 | TR-014 | `mcp/server.ts`, `mcp/standalone.ts` |
| TD-015 | TR-015 | `triggers/api.ts` |
| TD-016 | TR-016 | `functions/replay.ts`, `viewer/server.ts` |
| TD-017 | TR-017 | `functions/slots.ts` |
| TD-018 | TR-018 | `providers/` |
| TD-019 | TR-019 | `health/`, `functions/diagnostics.ts` |
| TD-020 | TR-020 | `auth.ts`, `functions/privacy.ts` |
| TD-021 | TR-021 | System-wide |
| TD-022 | TR-022 | CLI, deployment |
| TD-023 | TR-023 | `functions/export-import.ts`, `functions/migrate.ts` |
