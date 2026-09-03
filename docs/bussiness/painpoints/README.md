# VNP Memory — Pain Points Analysis

> **Mục đích:** Mô tả các vấn đề thực tế mà từng actor gặp phải trong công việc hàng ngày — những vấn đề này chính là lý do họ cần và sử dụng VNP Memory.
> 
> **Nguồn:** Tổng hợp từ [Feature Catalog](../../features/README.md) và [Product Docs](../product/v2/PRD.md).

---

## Actors (8 vai trò)

| # | Actor | Tần suất dùng | Tài liệu |
|---|---|---|---|
| P1 | [AI Agent Developer](./P1-ai-agent-developer.md) | Hàng ngày | Primary |
| P2 | [Platform / DevOps Engineer](./P2-platform-engineer.md) | Hàng tuần | Primary |
| P3 | [ML / AI Engineer](./P3-ml-ai-engineer.md) | Hàng tuần | Primary |
| P4 | [Enterprise Architect](./P4-enterprise-architect.md) | Hàng tháng | Primary |
| P5 | [IDE Plugin User](./P5-ide-plugin-user.md) | Hàng ngày | Secondary |
| P6 | [AI Framework Integrator](./P6-framework-integrator.md) | Theo dự án | Secondary |
| P7 | [AI Power User](./P7-ai-power-user.md) | Hàng ngày | Secondary |
| P8 | [Product Manager](./P8-product-manager.md) | Hàng tuần | Secondary |

---

## Bức tranh toàn cảnh — Vì sao AI Memory là bài toán chưa có lời giải?

### Thị trường đang chuyển dịch

```
RAG → Agentic RAG → Persistent Memory Systems
```

Các hệ thống hiện tại chỉ giải quyết được **một mảnh**:

| Giải pháp hiện tại | Mạnh về | Thiếu |
|---|---|---|
| Zep | Conversational memory | User profiling, adaptive memory |
| Mem0 | Lightweight memory | Temporal reasoning, enterprise |
| Graphiti | Temporal graph memory | Context assembly, user profiles |
| Cognee | Extraction pipeline | Session management |
| Memobase | User profiling | Graph memory, temporal reasoning |
| Supermemory | Adaptive KG + connectors | Session management |
| Vector DB (Qdrant, Pinecone) | Semantic search | Temporal, governance, structure |

> **Không ai unify toàn bộ stack** — và đó là lý do VNP Memory tồn tại.

---

## Ma trận Pain Points × Features

| Pain Point | Actor | Features giải quyết |
|---|---|---|
| Agent quên context sau phiên | P1, P5, P7 | F01, F04, F05 |
| Memory fragmented ở nhiều hệ thống | P1, P3 | F01, F10 |
| RAG không hiểu quan hệ thời gian | P1, P3 | F02, F04 |
| Knowledge cũ không tự cập nhật | P1, P7 | F07, F09 |
| Không có user profile có cấu trúc | P1, P7, P8 | F05 |
| Context assembly chậm, đắt | P1, P3 | F05, F06, F13 |
| Không track được agent đã làm gì | P1, P2 | F08, F21, F26 |
| Multi-agent race conditions | P1, P3 | F11 |
| Observation storage explosion | P1, P2 | F12 |
| Không debug được context assembly | P1, P3 | F20, F08 |
| Governance / GDPR gap | P4 | F14, F22 |
| Audit trail thiếu | P4, P2 | F22 |
| Không monitor được pipeline | P2, P3 | F23, F24, F25 |
| Quá nhiều infrastructure phức tạp | P1, P2 | F01 (monolith mode) |
| AI coding assistant quên project | P5 | F06, F13 |
| Framework integration khó | P6 | F01, F13 |
| AI không cá nhân hóa | P7 | F05, F07 |
| Không biết AI đang nhớ gì | P4, P7 | F16, F18 |
