# Báo cáo đối chiếu: Ontology đã được triển khai tuân thủ vai trò hay chưa?

> **Ngày đối chiếu:** 10/03/2026
> **Tài liệu gốc:** [ai_kg_service_ontology.md](ai_kg_service_ontology.md)
> **Đối chiếu với:**
> - [LLD](lld_ai_kg_service.md) (Low Level Design)
> - [Kế hoạch triển khai](ai_kg_service_impl_plan.md) (Implementation Plan)
> - [Kế hoạch Fix 3 Issues](ai_kg_service_impl_plan_01.md) (Implementation Plan 01)
> - [Báo cáo triển khai](report_impl_ai_kg_service.md) (Implementation Report)
> - [Đánh giá Coverage](lld_coverage_ai_kg_service.md) (LLD Coverage Assessment)

---

## Tổng kết nhanh

| Vai trò Ontology | Mức tuân thủ | Ghi chú |
|-------------------|-------------|---------|
| **1. Schema Registry** | ✅ **Đạt** | EntityType + RelationType CRUD đầy đủ, GORM models hoàn chỉnh |
| **2. Validation Gate** | ⚠️ **Chưa đạt** | Ontology tồn tại nhưng KHÔNG được gọi khi write node/edge vào Neo4j |
| **3. Constraint Enforcer** | ⚠️ **Chưa đạt** | RelationType có `FromType`/`ToType` nhưng không enforce khi CreateEdge |
| **4. Projection Rules Store** | ⚠️ **Đạt một phần** | ProjectionRule hardcode trong code, không lấy từ Ontology store |
| **5. Vocabulary Provider** | ⚠️ **Đạt ngầm** | EntityType registry tồn tại nhưng không có cơ chế enforce vocabulary |

### Điểm tổng: **2/5 vai trò được triển khai đúng ý nghĩa thiết kế**

---

## Chi tiết từng vai trò

### 1. Schema Registry — "Từ điển" của KG ✅ ĐẠT

**Ontology yêu cầu:**
- Định nghĩa EntityType (Requirement, UseCase, Actor, APIEndpoint, DataModel...) với properties schema
- Định nghĩa RelationType (DEPENDS_ON, IMPLEMENTS, CONFLICTS_WITH, TRACED_TO...) với source/target constraints
- API: `POST /v1/ontology/entities`, `POST /v1/ontology/relations`

**Thực tế triển khai:**

| Item | Trạng thái | Bằng chứng |
|------|-----------|------------|
| EntityType GORM model | ✅ | `internal/biz/ontology.go` — fields: ID, AppID, TenantID, Name, Description, Properties (JSONField), Domain |
| RelationType GORM model | ✅ | `internal/biz/ontology.go` — fields: ID, AppID, TenantID, Name, FromType, ToType, Cardinality |
| Ontology CRUD service | ✅ | `internal/service/ontology.go` + `internal/data/ontology.go` — full CRUD hoạt động |
| Proto definition | ✅ | `api/ontology/v1/ontology.proto` — gRPC Service `api.ontology.v1.Ontology` |
| API endpoints | ✅ | `POST/GET /v1/ontology/entities`, `POST/GET /v1/ontology/relations` hoạt động |

**Đánh giá:** Schema Registry triển khai **đầy đủ và đúng thiết kế**. Đây là phần Ontology được triển khai tốt nhất, đúng với đánh giá LLD Coverage (Section 9: 70%).

**Nguồn xác nhận:**
- LLD Coverage Assessment §9: "Ontology CRUD service ✅"
- Implementation Report §2.1: "Registry/Ontology/Rules/Policy — các module riêng — `service/*.go` ✅"

---

### 2. Validation Gate — "Người gác cổng" ⚠️ CHƯA ĐẠT

**Ontology yêu cầu:**
- Mọi write vào KG **phải** đi qua Ontology validation trước khi ghi Neo4j
- Kiểm tra EntityType có tồn tại không → nếu không → `400 ERR_SCHEMA_INVALID`
- Kiểm tra properties_json match JSON Schema của EntityType → nếu không → `400 ERR_SCHEMA_INVALID`
- Kiểm tra OPA Access Control → nếu từ chối → `403 ERR_FORBIDDEN`

**Thực tế triển khai:**

| Validation step | Trạng thái | Chi tiết |
|----------------|-----------|---------|
| Ontology lookup khi CreateNode | ❌ | Graph service KHÔNG gọi Ontology service trước khi write |
| Properties JSON Schema validation | ❌ | Không có schema validation logic nào connect Ontology → Graph write |
| OPA Access Control trước write | ✅ | `opa_client.go` tồn tại và hoạt động, nhưng flow chưa được wire vào Graph CRUD |

**Phân tích theo từng giai đoạn:**

1. **LLD Coverage (đánh giá ban đầu ~25%):** Graph API handlers là stubs — `CreateNode()`, `GetNode()`, `CreateEdge()` return `&Reply{}` rỗng, không gọi biz layer. → **Validation Gate không thể hoạt động khi handler là stub.**

2. **Implementation Report (sau triển khai 7.5/10):** API endpoints đã implement đầy đủ, `CreateNode` hoạt động thực. Tuy nhiên, flow `CreateNode → Ontology lookup → Schema validate → OPA check → Neo4j write` **không được mô tả** trong report.

3. **Implementation Plan:** Phase 0 (P0.1) chỉ focus vào "Wire biz logic vào CreateNode handler" — **không đề cập** Ontology validation gate.

**Kết luận:** Ontology data (EntityType, RelationType) tồn tại trong DB nhưng **không được sử dụng như validation gate** — đây là gap nghiêm trọng nhất so với tài liệu Ontology. Write operations có thể ghi entity với type bất kỳ mà không bị reject.

---

### 3. Constraint Enforcer — Ràng buộc source/target ⚠️ CHƯA ĐẠT

**Ontology yêu cầu:**
- RelationType định nghĩa `source_types` và `target_types`
- Khi CreateEdge: source entity type phải thuộc `source_types`, target entity type phải thuộc `target_types`
- Ví dụ: `DEPENDS_ON` chỉ cho phép source là `Requirement|UseCase`, target là `Requirement|UseCase|NFR`

**Thực tế triển khai:**

| Constraint | Trạng thái | Chi tiết |
|-----------|-----------|---------|
| RelationType có FromType/ToType | ✅ | GORM model có `FromType string` và `ToType string` |
| Validate source/target khi CreateEdge | ❌ | `CreateEdge` handler KHÔNG lookup RelationType để validate |
| Reject edge với source/target sai type | ❌ | Không có validation logic |

**Phân tích:** RelationType model lưu thông tin `FromType` và `ToType`, nhưng logic CreateEdge (cả ở LLD Coverage lẫn Implementation Report) không có step nào thực hiện cross-reference với Ontology RelationType. Edge có thể được tạo giữa bất kỳ 2 node nào mà không bị ràng buộc.

---

### 4. Projection Rules Provider — "Bộ lọc theo vai trò" ⚠️ ĐẠT MỘT PHẦN

**Ontology yêu cầu:**
- Ontology lưu trữ ProjectionRule **theo role** (BA, DEV, PO, DESIGNER)
- Projection Engine gọi `getRules(role, domains)` để lấy rules **từ Ontology** trước khi filter
- Mỗi role có: Entity Types được thấy, Edge Types được thấy, PII bị mask, Min Confidence

**Thực tế triển khai — 2 giai đoạn khác nhau:**

#### Giai đoạn 1: LLD Coverage (~25%)

| Item | Trạng thái |
|------|-----------|
| ProjectionRule per role | ❌ Stub |
| `view_resolver.go` | ⚠️ Stub — hardcode return `queryResult` không filter |
| `projection/` package | ❌ Không tồn tại |

#### Giai đoạn 2: Implementation Report (7.5/10)

| Item | Trạng thái |
|------|-----------|
| `projection/projection.go` (293 lines) | ✅ Tồn tại |
| ViewDefinition CRUD (GORM) | ✅ Hoạt động |
| Role-based field filtering | ✅ Hoạt động |
| PII Masking (`projection/mask.go`) | ✅ Hoạt động |

**Tuy nhiên, vấn đề cốt lõi:**

Theo LLD §7.7, ProjectionRule được **hardcode** trong code:

```go
var defaultProjectionRules = map[string]ProjectionRule{
    "BA": { IncludeEntityTypes: []string{"Requirement","UseCase",...} },
    "DEV": { IncludeEntityTypes: []string{"APIEndpoint","DataModel",...} },
    ...
}
```

Tài liệu Ontology yêu cầu rules được **lưu trữ trong Ontology store** và lấy ra qua `getRules(role, domains)`. Thực tế:
- **LLD**: Hardcode `defaultProjectionRules` — không lấy từ Ontology
- **Impl Report**: `ViewDefinition` CRUD có (lưu trong Postgres) — nhưng đây là entity riêng, **không phải** là phần Ontology store

**Kết luận:** Projection Engine hoạt động và có PII masking, nhưng **rules không được lưu trong/lấy từ Ontology** như tài liệu mô tả. Rules hoặc hardcode hoặc lưu trong `ViewDefinition` table riêng biệt.

---

### 5. Vocabulary Provider — Đảm bảo KG dùng chung ngôn ngữ ⚠️ ĐẠT NGẦM

**Ontology yêu cầu:**
- Đảm bảo toàn bộ KG dùng chung ngôn ngữ nhất quán
- Không có Ontology, KG chỉ là một đống node/edge vô tổ chức

**Thực tế triển khai:**

| Item | Trạng thái | Chi tiết |
|------|-----------|---------|
| EntityType registry | ✅ | Tồn tại và có thể query danh sách entity types |
| Enforce vocabulary khi write | ❌ | Write operation không kiểm tra entityType có tồn tại trong registry |
| Auto-suggest/validate type names | ❌ | Không có cơ chế này |

**Kết luận:** EntityType registry tồn tại như một "từ điển" có thể tham khảo, nhưng **không có cơ chế enforce** — nghĩa là KG vẫn có thể chứa entity types không có trong Ontology registry.

---

## 6. API Reference — Đối chiếu

| API (Ontology doc) | Proto | Service | Hoạt động? |
|---------------------|-------|---------|-----------|
| `POST /v1/ontology/entities` | ✅ `ontology.proto` | ✅ `service/ontology.go` | ✅ Có |
| `GET /v1/ontology/entities` | ✅ | ✅ | ✅ Có |
| `POST /v1/ontology/relations` | ✅ | ✅ | ✅ Có |
| `GET /v1/ontology/relations` | ✅ | ✅ | ✅ Có |

**Kết luận:** Tất cả 4 API endpoints của Ontology đã được triển khai và hoạt động. Proto file đúng: `api/ontology/v1/ontology.proto` — gRPC Service: `api.ontology.v1.Ontology`.

---

## 7. Đối chiếu sơ đồ phụ thuộc

**Ontology document mô tả 3 service phụ thuộc:**

```
              ONTOLOGY
                │ phục vụ
    ┌───────────┼───────────┐
    ▼           ▼           ▼
Graph Service  Projection  Rule Engine
(validate)     Engine       (business rules)
```

| Service phụ thuộc | Ontology mô tả | Thực tế |
|-------------------|----------------|---------|
| **Graph Service** | Validate schema khi write node/edge | ❌ Graph Service KHÔNG gọi Ontology khi write |
| **Projection Engine** | Filter kết quả search theo role | ⚠️ Projection hoạt động nhưng rules không lấy từ Ontology store |
| **Rule Engine** | Áp dụng business rules | ✅ Rule Engine hoạt động (event-driven + scheduled), nhưng không liên quan trực tiếp Ontology |

---

## 8. Tổng hợp Gap Analysis

### Gaps theo mức nghiêm trọng

| Priority | Gap | Vai trò bị ảnh hưởng | Mô tả |
|----------|-----|----------------------|-------|
| 🔴 **P0** | **Ontology Validation Gate không được wire** | Validation Gate, Constraint Enforcer, Vocabulary Provider | Ontology data tồn tại nhưng không được sử dụng để validate writes — entity/edge bất kỳ đều được ghi vào Neo4j |
| 🟡 **P1** | **Projection rules không lấy từ Ontology** | Projection Rules Store | Rules hardcode hoặc lưu trong table riêng, không phải Ontology store |
| 🟢 **P2** | **Thiếu JSON Schema validation cho properties** | Schema Registry (nâng cao) | EntityType có field `Properties JSONField` nhưng không dùng để validate entity properties trước khi write |

### Gaps KHÔNG được đề cập trong kế hoạch triển khai

Điểm đáng chú ý: **Kế hoạch triển khai (impl_plan.md)** có 6 Phase nhưng **KHÔNG CÓ phase nào** đề cập việc wire Ontology Validation Gate vào Graph write flow:

- Phase 0: Fix stubs, auth, namespace — ❌ không đề cập Ontology validation
- Phase 1: Lock, batch, graph algorithms — ❌ không đề cập
- Phase 2: Qdrant, hybrid search — ❌ không đề cập
- Phase 3: Overlay, versioning, NATS — ❌ không đề cập
- Phase 4: Analytics, projection — chỉ đề cập projection, ❌ không wire Ontology validation
- Phase 5: Observability, deploy, E2E — ❌ không đề cập

**Kế hoạch Fix 3 Issues (impl_plan_01.md)** cũng chỉ focus vào NATS, X-Org-ID, Embedding — không đề cập Ontology validation.

→ **Ontology Validation Gate là một gap chưa được lên kế hoạch khắc phục.**

---

## 9. Kết luận

### Ontology đã tuân thủ vai trò chưa?

**Trả lời: CHƯA HOÀN TOÀN.**

Ontology hiện tại hoạt động như một **passive registry** (kho lưu trữ thụ động) — tức là nó lưu trữ thông tin EntityType/RelationType và cho phép CRUD, nhưng **không thực hiện vai trò active enforcement** như tài liệu thiết kế mô tả.

Cụ thể:

| Tính chất | Thiết kế (Ontology doc) | Thực tế |
|-----------|------------------------|---------|
| **Active gatekeeper** | Chặn mọi write không hợp lệ | ❌ Không chặn gì |
| **Schema enforcer** | Validate JSON Schema properties | ❌ Không validate |
| **Constraint checker** | Kiểm tra source/target types | ❌ Không kiểm tra |
| **Rules provider** | Cung cấp projection rules cho Projection Engine | ⚠️ Rules không từ Ontology |
| **Vocabulary authority** | Đảm bảo mọi write dùng đúng vocabulary | ❌ Không enforce |
| **Registry** | Lưu trữ EntityType + RelationType | ✅ Hoạt động tốt |

### Đề xuất

1. **Bổ sung Ontology Validation Middleware** — Tạo middleware/interceptor trong Graph write flow:
   - `CreateNode` → lookup `EntityType` → validate `properties` against JSON Schema → proceed hoặc reject `400`
   - `CreateEdge` → lookup `RelationType` → validate source/target entity types → proceed hoặc reject `400`

2. **Bổ sung vào kế hoạch triển khai** — Thêm Phase mới (hoặc bổ sung vào Phase 0/1) cho Ontology enforcement:
   - Task: Wire `OntologyUsecase.GetEntityType()` vào `GraphUsecase.CreateNode()`
   - Task: Wire `OntologyUsecase.GetRelationType()` vào `GraphUsecase.CreateEdge()`
   - Task: Implement JSON Schema validation cho entity properties

3. **Liên kết Projection Rules với Ontology** — Chuyển `defaultProjectionRules` từ hardcode sang lưu trong Ontology store hoặc liên kết `ViewDefinition` với Ontology EntityType/RelationType.

---

## Bảng tổng hợp cuối cùng

| # | Vai trò Ontology | Thiết kế | LLD | Impl Plan | Impl Report | Thực tế |
|---|------------------|----------|-----|-----------|-------------|---------|
| 1 | Schema Registry | ✅ Định nghĩa rõ | ✅ §9.1 | — (có sẵn) | ✅ Hoạt động | ✅ **ĐẠT** |
| 2 | Validation Gate | ✅ Flow chi tiết | ❌ Không wire | ❌ Không có task | Không đề cập | ❌ **CHƯA ĐẠT** |
| 3 | Constraint Enforcer | ✅ source/target rules | ⚠️ Model có, logic không | ❌ Không có task | Không đề cập | ❌ **CHƯA ĐẠT** |
| 4 | Projection Rules Store | ✅ Per-role rules từ Ontology | ⚠️ Hardcode | P4 — Projection | ✅ Projection hoạt động | ⚠️ **MỘT PHẦN** |
| 5 | Vocabulary Provider | ✅ Enforce vocabulary | ⚠️ Implicit | ❌ Không có task | Không đề cập | ⚠️ **NGẦM** |
