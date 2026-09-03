# agentmemory — Bộ Test Cases

Thư mục này chứa **Test Cases** (Kịch bản Kiểm thử) chi tiết cho hệ thống **agentmemory**.

## Mối quan hệ trong bộ test

```
Test Requirements (../requirements/)
        ↓
Test Design     (../designs/)
        ↓
Test Cases      (./test-cases/)    ← Bạn đang ở đây
        ↓
Test Scripts    (../test-scripts/) ← Implementation (code)
```

## Cấu trúc thư mục

```
test-cases/
├── README.md
├── TC-001-session-management.md
├── TC-002-observe-pipeline.md
├── TC-003-compress-synthetic.md
├── TC-004-search-bm25.md
├── TC-005-search-vector.md
├── TC-006-search-hybrid.md
├── TC-007-memory-management.md
├── TC-008-knowledge-graph.md
├── TC-009-consolidation-pipeline.md
├── TC-010-context-injection.md
├── TC-011-multi-agent.md
├── TC-012-orchestration.md
├── TC-013-governance-audit.md
├── TC-014-mcp-server.md
├── TC-015-rest-api.md
├── TC-016-session-replay.md
├── TC-017-memory-slots.md
├── TC-018-provider-system.md
├── TC-019-health-diagnostics.md
├── TC-020-security.md
├── TC-021-performance.md
├── TC-022-deployment.md
└── TC-023-export-import.md
```

## Quy ước Test Case

### Test Case ID Format
`TC-[Module]-[Sequence]`  
Ví dụ: `TC-001-001` = Module 001 (Session), Test case thứ 1

### Cấu trúc mỗi Test Case

| Trường | Mô tả |
|---|---|
| **ID** | Định danh duy nhất |
| **Tên** | Mô tả ngắn gọn mục tiêu |
| **Yêu cầu** | Tham chiếu TR/TD |
| **Loại** | unit / integration / e2e / performance / security |
| **Ưu tiên** | 🔴 P0 (critical) / 🟠 P1 (high) / 🟡 P2 (medium) |
| **Điều kiện tiên quyết** | Trạng thái hệ thống trước khi test |
| **Dữ liệu đầu vào** | Input cụ thể (values, không phải code) |
| **Các bước thực hiện** | Step-by-step actions |
| **Kết quả mong đợi** | Expected output / state sau khi thực hiện |
| **Tiêu chí Pass/Fail** | Điều kiện để test PASS |
| **Ghi chú** | Kỹ thuật, edge cases, liên kết |

### Ký hiệu trạng thái

- ☐ Chưa thực hiện
- ⚙️ Đang chạy
- ✅ Pass
- ❌ Fail
- ⏭️ Skip

## Ma trận Traceability

| Test Cases | Test Design | Requirements |
|---|---|---|
| TC-001-xxx | TD-001 | TR-001 |
| TC-002-xxx | TD-002 | TR-002 |
| TC-003-xxx | TD-003 | TR-003 |
| TC-004-xxx | TD-004 | TR-004 |
| TC-005-xxx | TD-005 | TR-005 |
| TC-006-xxx | TD-006 | TR-006 |
| TC-007-xxx | TD-007 | TR-007 |
| TC-008-xxx | TD-008 | TR-008 |
| TC-009-xxx | TD-009 | TR-009 |
| TC-010-xxx | TD-010 | TR-010 |
| TC-011-xxx | TD-011 | TR-011 |
| TC-012-xxx | TD-012 | TR-012 |
| TC-013-xxx | TD-013 | TR-013 |
| TC-014-xxx | TD-014 | TR-014 |
| TC-015-xxx | TD-015 | TR-015 |
| TC-016-xxx | TD-016 | TR-016 |
| TC-017-xxx | TD-017 | TR-017 |
| TC-018-xxx | TD-018 | TR-018 |
| TC-019-xxx | TD-019 | TR-019 |
| TC-020-xxx | TD-020 | TR-020 |
| TC-021-xxx | TD-021 | TR-021 |
| TC-022-xxx | TD-022 | TR-022 |
| TC-023-xxx | TD-023 | TR-023 |

## Thống kê Test Cases

| Module | P0 | P1 | P2 | Total |
|---|---|---|---|---|
| TC-001 Session Management | 6 | 5 | 2 | 13 |
| TC-002 Observe Pipeline | 10 | 8 | 3 | 21 |
| TC-003 Synthetic Compression | 8 | 6 | 4 | 18 |
| TC-004 BM25 Search | 8 | 9 | 5 | 22 |
| TC-005 Vector Index | 6 | 7 | 4 | 17 |
| TC-006 Hybrid Search | 5 | 6 | 3 | 14 |
| TC-007 Memory Management | 7 | 8 | 4 | 19 |
| TC-008 Knowledge Graph | 5 | 6 | 3 | 14 |
| TC-009 Consolidation | 3 | 4 | 3 | 10 |
| TC-010 Context Injection | 4 | 5 | 2 | 11 |
| TC-011 Multi-Agent | 5 | 5 | 2 | 12 |
| TC-012 Orchestration | 3 | 5 | 3 | 11 |
| TC-013 Governance/Audit | 4 | 4 | 2 | 10 |
| TC-014 MCP Server | 4 | 3 | 2 | 9 |
| TC-015 REST API | 6 | 6 | 2 | 14 |
| TC-016 Session Replay | 3 | 3 | 1 | 7 |
| TC-017 Memory Slots | 5 | 4 | 2 | 11 |
| TC-018 Provider System | 4 | 4 | 2 | 10 |
| TC-019 Health/Diagnostics | 4 | 2 | 1 | 7 |
| TC-020 Security | 7 | 4 | 1 | 12 |
| TC-021 Performance | 5 | 3 | 2 | 10 |
| TC-022 Deployment | 5 | 4 | 1 | 10 |
| TC-023 Export/Import | 5 | 5 | 3 | 13 |
| **TỔNG** | **122** | **116** | **57** | **295** |
