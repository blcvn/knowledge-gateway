# ADR-005 — 6 Memory Engines thay vì 1 Unified Engine

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-02 |
| **Deciders** | Product + Platform Team |
| **Feature** | F02-F07 (6 Memory Engines), F01 (Unified API) |

---

## Context

VNP Memory cần support nhiều loại memory (episodic, semantic, conversational, profile, procedural, adaptive). Câu hỏi: **1 engine tổng quát hay nhiều engines chuyên biệt?**

Market research cho thấy:
- Graphiti: Tốt nhất cho temporal reasoning (valid_at/invalid_at)
- Memobase: Tốt nhất cho user profiling (YOLO engine, < 100ms)
- Zep: Tốt nhất cho conversational context (sub-200ms, custom ontology)
- Cognee: Tốt nhất cho knowledge extraction (7-step cognify, 15+ search types)
- Supermemory: Tốt nhất cho adaptive memory (living KG, forgetAfter)
- OpenViking: Designed specifically cho procedural/filesystem memory

---

## Decision

**6 engines chuyên biệt, được orchestrate bởi Unified API.**

```
POST /v1/memory/store
    │
    ▼
Gateway (type detection)
    │
    ├── type=episodic   → Graphiti   (temporal KG, valid_at)
    ├── type=semantic   → Cognee     (knowledge extraction)
    ├── type=conversational → Zep    (session context)
    ├── type=profile    → Memobase   (YOLO engine)
    ├── type=procedural → OpenViking (L0/L1/L2 tiered)
    ├── type=adaptive   → Supermemory (living KG, forgetAfter)
    └── type=auto       → LLM classify → route
```

Developer không cần biết engines — chỉ gọi 1 API.

---

## Consequences

**Positive:**
- Mỗi engine được tối ưu cho use case của nó (best-of-breed)
- Dễ swap/upgrade 1 engine mà không ảnh hưởng engines khác
- Competitive với tất cả specialist tools (Zep, Memobase, Graphiti...) đồng thời
- Cross-engine recall: search tất cả 6 engines → RRF fusion

**Negative:**
- Operational complexity: 6 systems × nhiều services mỗi system
- Inconsistent behavior giữa engines (mỗi engine có quirks riêng)
- Cross-engine transactions không possible (eventual consistency)
- Storage duplication: cùng data có thể trong nhiều engines

**Mitigations:**
- Monolith-first (ADR-001) giảm operational complexity
- Unified API ẩn complexity khỏi developer
- TenantID consistency đảm bảo tất cả engines isolate đúng

---

## Alternatives Considered

### A1 — Build 1 custom engine (từ đầu)
- **Rejected:** 12-24 tháng để match feature parity với 5 mature open-source engines; không realistic

### A2 — Chỉ 1 engine (Zep hoặc Graphiti)
- **Rejected:** Không ai có đủ tính năng (user profiling, adaptive memory, filesystem); customers sẽ cần integrate thêm tools

### A3 — Plugin architecture (dynamic engine loading)
- **Deferred:** Quá complex cho current stage; có thể xem xét sau khi core stable
