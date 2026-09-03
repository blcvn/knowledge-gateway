# VNP Memory — Product Documentation v2

> **Version:** 2.1.0 | **Updated:** 2026-09-03

Tài liệu product v2 được cập nhật đầy đủ từ [Feature Catalog](../features/README.md) với **28 features**.

## Documents

| Document | Mô tả |
|---|---|
| [PRD.md](PRD.md) | Product Requirements Document v2.1.0 — đầy đủ 28 features, AgentMemory layer, 37+ MCP tools |

## Thay đổi so với v1

| Mục | v1 | v2 |
|---|---|---|
| MCP Tools | 16 tools | **37+ tools** |
| Feature coverage | F01-F07 + Console | **F01-F28** (28 features) |
| AgentMemory Layer | Không có | **F08-F13, F26** (Observe, Lifecycle, Search, Orchestration, Consolidation, Session Replay) |
| Component paths | `gateway/`, `services/` | `backend/gateway/`, `backend/services/` |
| Roadmap Phase 1 | Partially done | ✅ **Complete** (incl. AgentMemory) |
| Session Replay | Không có | **F26** — JSONL import, timeline scrubbing |
| Organization SDK | Không có | **F27** — API key lifecycle, SSO |
| WebSocket Events | Listed only | **F28** — event categories, filter |
| Observe pipeline | Không có | **14-step pipeline** với privacy redaction |
| Consolidation | Không có | **4-tier** (L1 session → L4 core knowledge) |

## Feature Catalog

Xem đầy đủ 28 features tại: [../features/README.md](../features/README.md)

## Previous Version

Tài liệu v1 vẫn được giữ nguyên tại [../v1/](../v1/) để tham khảo.
