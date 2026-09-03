# Vai trò của Ontology trong ai-kg-service

> Tài liệu tổng hợp — Tham chiếu: DOC 3 (KG Service Architecture), API Specs v1.1

---

## Tổng quan

Ontology là **"bộ từ điển & luật lệ" của Knowledge Graph** — nó định nghĩa *những gì được phép tồn tại* trong đồ thị trước khi bất kỳ dữ liệu nào được ghi vào.

Không có Ontology, KG chỉ là một đống node/edge vô tổ chức. Ontology là thứ biến nó thành đồ thị tri thức có ngữ nghĩa và có thể truy vấn có ý nghĩa.

---

## 1. Schema Registry — "Từ điển" của KG

Ontology quản lý 2 loại schema:

### Entity Types — định nghĩa các loại node

| Entity Type | Properties tiêu biểu |
|-------------|----------------------|
| `Requirement` | `priority: HIGH\|MEDIUM\|LOW`, `status: draft\|approved\|implemented`, `source: string` |
| `UseCase` | `actor: string`, `goal: string` |
| `Actor` | ... |
| `APIEndpoint` | ... |
| `DataModel` | ... |

API: `POST /v1/ontology/entities`

### Relation Types — định nghĩa các loại cạnh + ràng buộc đầu cuối

| Relation Type | source_types | target_types | Properties |
|---------------|-------------|-------------|------------|
| `DEPENDS_ON` | `Requirement`, `UseCase` | `Requirement`, `UseCase`, `NFR` | `strength: 0..1`, `notes: string` |
| `IMPLEMENTS` | `APIEndpoint` | `DataModel` | — |
| `CONFLICTS_WITH` | `Requirement` | `Requirement` | — |
| `TRACED_TO` | `UseCase` | `Requirement` | — |

API: `POST /v1/ontology/relations`

---

## 2. Validation Gate — "Người gác cổng"

Mọi write vào KG đều phải đi qua Ontology validation trước khi được ghi xuống Neo4j:

```
ai-planner gửi entity lên KG
         │
         ▼
  Graph Service nhận CreateNode
         │
         ▼
  Ontology lookup:
  "Requirement" có tồn tại không? ──── Không → 400 ERR_SCHEMA_INVALID
         │ Có
         ▼
  properties_json có match JSON Schema
  của EntityType "Requirement" không? ─ Không → 400 ERR_SCHEMA_INVALID
         │ Có
         ▼
  OPA Access Control check ──────────── Từ chối → 403 ERR_FORBIDDEN
         │ Pass
         ▼
  Write vào Neo4j ✅
```

Tương tự với Edge — `source_types` và `target_types` phải hợp lệ trước khi ghi cạnh.

---

## 3. Projection Rules Provider — "Bộ lọc theo vai trò"

Ontology lưu trữ **ProjectionRule theo role** — được Projection Engine sử dụng khi trả kết quả search, đảm bảo mỗi role chỉ thấy đúng phần thông tin thuộc phạm vi của mình:

| Role | Entity Types được thấy | Edge Types được thấy | PII bị mask | Min Confidence |
|------|------------------------|----------------------|-------------|----------------|
| `BA` | `Requirement`, `UseCase`, `Actor`, `BusinessRule`, `Risk` | `DEPENDS_ON`, `CONFLICTS_WITH`, `TRACED_TO` | `internal_code`, `implementation_detail` | 0.70 |
| `DEV` | `APIEndpoint`, `DataModel`, `Integration`, `NFR`, `Sequence` | `CALLS`, `IMPLEMENTS`, `EXTENDS` | `stakeholder_name`, `business_justification` | 0.65 |
| `PO` | `Epic`, `UserStory`, `Feature`, `Stakeholder` | `PART_OF`, `BLOCKS`, `DELIVERS_VALUE_TO` | — | 0.60 |
| `DESIGNER` | `UserFlow`, `Screen`, `Persona`, `Interaction` | `NAVIGATES_TO`, `TRIGGERED_BY` | — | 0.65 |

Projection Engine gọi `getRules(role, domains)` để lấy rules từ Ontology trước khi filter kết quả trả về client.

---

## 4. Các service phụ thuộc Ontology

```
                    ┌─────────────────────────────┐
                    │         ONTOLOGY            │
                    │                             │
                    │  EntityType registry        │
                    │  RelationType registry      │
                    │  ProjectionRules per role   │
                    └──────────┬──────────────────┘
                               │ phục vụ
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                    ▼
   Graph Service        Projection Engine     Rule Engine
   (validate schema     (filter kết quả      (áp dụng
    khi write node/      search theo role)    business rules)
    edge)
```

---

## 5. Tóm tắt vai trò

| Vai trò | Mô tả |
|---------|-------|
| **Schema Registry** | Định nghĩa EntityType + RelationType + JSON Schema properties |
| **Validation Gate** | Chặn mọi write không hợp lệ (sai type, sai schema) |
| **Constraint Enforcer** | Ràng buộc source/target của edge phải đúng type |
| **Projection Rules Store** | Lưu rules filter theo role (BA/DEV/PO/DESIGNER) + PII masking |
| **Vocabulary Provider** | Đảm bảo toàn bộ KG dùng chung ngôn ngữ nhất quán |

---

## 6. API Reference

| Method | Endpoint | Mô tả |
|--------|----------|-------|
| `POST` | `/v1/ontology/entities` | Tạo Entity Type mới |
| `GET` | `/v1/ontology/entities` | Liệt kê tất cả Entity Types |
| `POST` | `/v1/ontology/relations` | Tạo Relation Type mới |
| `GET` | `/v1/ontology/relations` | Liệt kê tất cả Relation Types |

> **Proto:** `api/ontology/v1/ontology.proto` — gRPC Service: `api.ontology.v1.Ontology`