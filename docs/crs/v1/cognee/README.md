# Change Requests — Cognee Feature Parity

**Project:** VNP Memory  
**Domain:** Cognee Engine  
**Path:** `specs/crs/v1/cognee/`  
**Date:** 2026-06-16  
**Status:** Proposed

> Các Change Requests này được tạo dựa trên phân tích đối chiếu giữa codebase hiện tại (`services/cognee-*`, `gateway/`) và tài liệu tham chiếu (`references/cognee/docs/PRD.md`, `SRS.md`, `URD.md`, `specs/services/*.md`).

---

## Tổng quan

| CR ID | Tên | Component | Priority | Status |
|---|---|---|---|---|
| [CR-COGNEE-001](./CR-COGNEE-001-Memify-Enrichment.md) | Graph Enrichment (Memify) | `cognee-cognify`, `vnp-gateway` | High | Implemented |
| [CR-COGNEE-002](./CR-COGNEE-002-NodeSets-Scoping.md) | NodeSets Memory Scoping | `cognee-ingestion`, `cognee-search`, `vnp-gateway` | High | Implemented |
| [CR-COGNEE-003](./CR-COGNEE-003-DataPoint-Schema.md) | DataPoint Custom Schema | `cognee-ingestion`, `cognee-cognify` | Medium | Implemented |
| [CR-COGNEE-004](./CR-COGNEE-004-Advanced-Loaders.md) | Advanced Loaders & DLT | `cognee-ingestion` | Medium | Implemented |
| [CR-COGNEE-005](./CR-COGNEE-005-Feedback-Loop.md) | Feedback Loop & Self-Improvement | `cognee-search`, `cognee-memory`, `vnp-gateway` | Medium | Implemented |
| [CR-COGNEE-006](./CR-COGNEE-006-Custom-Pipelines.md) | Custom Pipelines Orchestration | `cognee-cognify`, `vnp-gateway` | Low | Implemented |
| [CR-COGNEE-007](./CR-COGNEE-007-MCP-Parity.md) | MCP Server Parity | `vnp-gateway` (MCP adapter) | High | Implemented |

---

## Feature Gap Matrix

| Feature | Cognee Spec | VNP Memory hiện tại | CR |
|---|---|---|---|
| `memify()` — graph enrichment | ✅ PRD §4.1.5 | ❌ Chưa có | CR-001 |
| NodeSets tagging | ✅ PRD §4.2 | ❌ Chưa có | CR-002 |
| DataPoint custom schema | ✅ PRD §4.3 | ❌ Chưa có | CR-003 |
| Layout PDF (bảng biểu) | ✅ SRS FR-ING-01 | ⚠️ Flat text only | CR-004 |
| Web readability scraping | ✅ SRS FR-ING-01 | ⚠️ Basic only | CR-004 |
| Tabular FK edges | ✅ SRS FR-ING-01 | ❌ Chưa có | CR-004 |
| `save_interaction=true` | ✅ PRD §4.5 | ❌ Chưa có | CR-005 |
| SearchType `FEEDBACK` | ✅ SRS §2.3 | ❌ Chưa có | CR-005 |
| Edge weight reinforcement | ✅ PRD §4.5 | ❌ Chưa có | CR-005 |
| Custom pipeline steps | ✅ PRD §4.4 | ❌ Hardcoded 7 steps | CR-006 |
| Pipeline templates | ✅ SRS FR-PIPE-01 | ❌ Chưa có | CR-006 |
| MCP `cognify` tool | ✅ PRD §6.3 | ⚠️ Tên khác (`cognee_add`) | CR-007 |
| MCP `save_interaction` tool | ✅ PRD §6.3 | ❌ Chưa có | CR-007 |
| MCP `list_data` tool | ✅ PRD §6.3 | ❌ Chưa có | CR-007 |
| MCP `delete_dataset` tool | ✅ PRD §6.3 | ⚠️ Gián tiếp qua `cognee_add` | CR-007 |
| MCP `cognify_status` tool | ✅ PRD §6.3 | ❌ Chưa có | CR-007 |

---

## Dependency Graph

```
CR-004 (Advanced Loaders)
  └─ uses → CR-003 (DataPoint) for TabularFK path

CR-005 (Feedback Loop)
  └─ uses → CR-007 (MCP) for save_interaction MCP tool

CR-006 (Custom Pipelines)
  └─ enhances → CR-001 (Memify) can be a pipeline template

CR-002 (NodeSets)
  └─ integrates with → CR-003 (DataPoint) — node_sets on datapoints
```

---

## Recommended Implementation Order

| Wave | CRs | Rationale |
|---|---|---|
| **Wave 1** | CR-002, CR-007 | High impact, high demand từ Agent/IDE users |
| **Wave 2** | CR-001, CR-003 | Core functionality gap, medium complexity |
| **Wave 3** | CR-005, CR-004 | Enhancement, thêm loader + feedback |
| **Wave 4** | CR-006 | Refactoring, cần Wave 1+2+3 stable trước |
