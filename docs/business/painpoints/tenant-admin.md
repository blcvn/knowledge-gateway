# Pain Points — Tenant Admin

> **Actor**: Tenant Admin  
> **Phạm vi**: Người quản lý tenant — define ontology domain, publish query templates, cấu hình lifecycle, quản lý visibility cho apps trong tenant  
> **Loại**: Domain modeling & Ontology governance pain points  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Tổng quan

Tenant Admin là người **chịu trách nhiệm về chất lượng knowledge** của tenant mình — từ việc định nghĩa ontology đúng, publish query templates hữu ích, đến việc đảm bảo các apps trong tenant thấy đúng những gì cần thấy.

Đây là role **đứng ở giữa** Platform Admin và App Integrator — phải hiểu cả technical (ontology modeling, API) lẫn business (domain knowledge, access policy). Hiện tại, không có tooling nào hỗ trợ role này một cách đầy đủ.

---

## Pain Points chi tiết

### PP-TA-01 — Không có công cụ hỗ trợ thiết kế ontology domain — phải tự figure out từ đầu

**Mức độ**: 🔴 Critical  
**Tần suất**: Khi khởi tạo domain mới hoặc mở rộng domain hiện có  

**Mô tả**:  
Trước khi app integrator có thể write bất cứ gì, Tenant Admin phải định nghĩa domain ontology:
- Node types (vd: `Requirement`, `UserStory`, `TestCase`)
- Relationship types (vd: `BREAKS_DOWN_TO`, `IMPLEMENTED_BY`, `VALIDATED_BY`)
- Attribute schemas cho từng node type
- Lifecycle/status rules

Hiện tại, không có hướng dẫn nào về:
- Làm thế nào để biết ontology mình định nghĩa là "đúng"?
- Node types nên granular đến mức nào?
- Relationship types nên unidirectional hay bidirectional?
- Có nên inherit từ shared ontology hay tạo domain-specific hoàn toàn?

**Ví dụ thực tế**:
```
Tenant Admin của "Payment Team" muốn model domain "payment-errors":
→ Cần node type "ErrorCode"? Hay "PaymentError"? Hay "ErrorDefinition"?
→ Relationship với "PaymentFlow" là "TRIGGERS"? "BELONGS_TO"? "CAUSES"?
→ "severity" là attribute của ErrorCode hay là separate node type?
→ Không có ontology design guide → mỗi admin làm khác nhau
→ Sau 3 tháng → ontology inconsistent, hard to query
```

**Hệ quả kinh doanh**:
- Bad ontology design → query templates trả về wrong/incomplete data
- Phải refactor ontology sau khi đã ingest data → rất tốn công
- Không có best practices → mỗi tenant có ontology style khác nhau → không thể cross-tenant query

**Giải pháp cần có**:
- Ontology Design Guide per domain type (product, engineering, finance, compliance)
- AI-assisted ontology generator: "Tôi muốn model payment error handling" → AI suggest initial ontology
- `GET /v1/ontology/templates` — standard ontology templates để start from
- Ontology linter: validate ontology definition trước khi activate

---

### PP-TA-02 — Query template lifecycle phức tạp — không có IDE/preview để test template trước khi publish

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi khi cần thêm hoặc update query template  

**Mô tả**:  
Query templates là cách App Integrators truy vấn knowledge — nhưng toàn bộ lifecycle của template hiện tại hoàn toàn blind:

```
Bước 1: Viết template definition (graph query language syntax)
→ Không có IDE support, không có syntax highlighting
→ Không biết valid query syntax cho backend đang dùng (Neo4j? MemGraph? FalkorDB?)
→ Phải test trial-and-error

Bước 2: Register template
POST /v1/tenants/{t}/ontology/domains/{d}/templates
→ Nếu syntax sai → cryptic error message

Bước 3: Activate template
POST /v1/tenants/{t}/ontology/domains/{d}/templates/{id}/activate
→ Nếu quên activate → app integrator gọi template → 404

Bước 4: App integrator test → nếu kết quả sai → báo lại admin
→ Admin không biết sai ở đâu: syntax? data? ontology?
→ Debug mất nhiều giờ
```

**Hệ quả kinh doanh**:
- Template errors chỉ phát hiện khi app integrator dùng → delayed feedback loop
- Admin phải coordinate với integrator mỗi khi update template → overhead
- Không có template versioning → update template ảnh hưởng tất cả consumers ngay lập tức

**Giải pháp cần có**:
- Template preview mode: `POST /v1/ontology/templates/preview` — run template với sample data, trả về kết quả mà không activate
- Template versioning: v1, v2 của cùng template → controlled migration
- Template validation: syntax check trước khi register
- `GET /v1/ontology/templates/{id}/usage` — app nào đang dùng template này → assess impact trước khi update

---

### PP-TA-03 — Không có visibility vào effective access của apps trong tenant

**Mức độ**: 🔴 Critical  
**Tần suất**: Khi setup mới và khi debug access issues  

**Mô tả**:  
URD yêu cầu Tenant Admin phải "confirm effective visibility for the tenant's apps" và "understand which apps can see which domains". Nhưng để làm điều này, hiện tại phải:

```
1. Gọi GET /v1/access/resolve với app key A → xem app A thấy gì
2. Gọi GET /v1/access/resolve với app key B → xem app B thấy gì
3. So sánh thủ công → không có diff view
4. Lặp lại cho từng app trong tenant
→ Tenant có 5 apps → 5 separate API calls → mental merge
```

Không có:
- Dashboard: "App X thấy domains: [D1, D2], App Y thấy domains: [D1, D3, D4]"
- Reason: "App X không thấy D3 vì grant chưa được thêm"
- Alert: "App X đã bị mất access vào D2 trong 24 giờ qua"

**Hệ quả kinh doanh**:
- Access misconfiguration discovered muộn (khi user complaint)
- Debugging access issues mất nhiều giờ vì phải correlate nhiều API calls
- Tenant Admin không confident về access state → thêm grant dư thừa (over-grant) để "chắc ăn" → security risk

**Giải pháp cần có**:
- `GET /v1/tenants/{t}/access/summary` — human-readable access summary cho toàn bộ tenant
- Access change event: "24 giờ qua, access của app X đã thay đổi như sau..."
- Access simulation: "Nếu tôi revoke grant Y → app nào sẽ mất access gì?"

---

### PP-TA-04 — Không có cách manage lifecycle rules mà không biết domain-specific business logic

**Mức độ**: 🟠 High  
**Tần suất**: Khi setup domain có status workflow (requirement lifecycle, issue status...)  

**Mô tả**:  
SRS yêu cầu hỗ trợ "status/lifecycle configuration per domain". Ví dụ, Requirement có lifecycle: `Draft → In Review → Approved → Deprecated`. Nhưng:
- Không có UI hay CLI để define state machine
- Không có validation: state transition `Draft → Deprecated` có hợp lệ không?
- Không có way để enforce: app integrator có thể set status thành bất cứ gì nếu không có guard
- Không có lifecycle visualization: xem state machine của domain như diagram

**Hệ quả kinh doanh**:
- Lifecycle rules không được enforce → data inconsistency (node trong state "Deprecated" vẫn có active relationships)
- Không có transition guards → app bugs đặt node vào wrong state mà không ai biết
- Reporting sai: "Có bao nhiêu requirements đang In Review?" → không accurate vì status không controlled

**Giải pháp cần có**:
- Lifecycle editor: Visual state machine editor → export to domain config
- Transition validation API: `POST /v1/ontology/domains/{d}/lifecycle/validate` — check if transition is valid
- Status enforcement: Node write API reject invalid status transitions automatically

---

### PP-TA-05 — Onboarding app mới vào tenant phải hướng dẫn thủ công — không có self-service flow

**Mức độ**: 🟠 High  
**Tần suất**: Mỗi khi có team mới muốn dùng domain của tenant  

**Mô tả**:  
Khi một app integrator team mới muốn dùng domain của tenant, Tenant Admin phải:
1. Explain ontology: node types, relationship types, query templates nào có sẵn
2. Hướng dẫn cách gọi API
3. Cấu hình grant nếu cần cross-tenant access
4. Là single point of contact khi integrator gặp vấn đề

Không có:
- Domain documentation auto-generated từ ontology definition
- Self-service onboarding flow cho app integrator
- Sandbox environment để integrator test mà không ảnh hưởng production data

**Hệ quả kinh doanh**:
- Tenant Admin trở thành bottleneck — phải support nhiều teams
- Integrator phải chờ Tenant Admin respond → slow adoption
- Undocumented ontology → integrators dùng sai → bad data vào KG

**Giải pháp cần có**:
- `GET /v1/tenants/{t}/ontology/docs` — auto-generated documentation từ ontology definition
- Sandbox mode: app integrator có thể test write/read mà không affect production
- Onboarding checklist: Tenant Admin chỉ cần click "Onboard App X" → system guide cả hai bên

---

## Ma trận Pain Points — Tenant Admin

| ID | Pain Point | Mức độ | Impact | Giải pháp cần có |
|:---|:---|:---:|:---|:---|
| PP-TA-01 | Không có hỗ trợ thiết kế ontology | 🔴 | Bad ontology → bad data quality | Ontology templates + AI assist |
| PP-TA-02 | Query template lifecycle thiếu preview/versioning | 🔴 | Template bugs ảnh hưởng consumers | Template preview + versioning |
| PP-TA-03 | Không có visibility vào effective access | 🔴 | Access misconfiguration, over-granting | Access summary API |
| PP-TA-04 | Lifecycle rules không có tooling | 🟠 | Data inconsistency, wrong reporting | Lifecycle editor + enforcement |
| PP-TA-05 | Onboard app mới phải support thủ công | 🟠 | Bottleneck, slow adoption | Auto-generated docs + sandbox |

---

## Tại sao Tenant Admin phải dùng kg-service

1. **Single source of truth cho domain knowledge**: Không có nơi nào khác trong stack để store ontology-aware, queryable domain knowledge với ACL
2. **Template-based query abstraction**: App integrators không cần viết raw graph queries — Tenant Admin define templates một lần, integrators dùng mãi mãi
3. **Cross-tenant knowledge sharing**: Chỉ kg-service cung cấp explicit grant mechanism để share knowledge an toàn giữa teams
4. **Graph + Vector projection**: Domain knowledge được tự động project sang cả graph (traversal) lẫn vector (semantic search) mà không cần maintain hai stores riêng

> **Kết luận**: Tenant Admin là người hưởng lợi nhiều nhất khi kg-service trưởng thành — vì service giải phóng họ khỏi việc maintain domain knowledge trên nhiều tools khác nhau (Confluence, spreadsheet, Word). Pain points trên là bước cần thiết để cải thiện adoption.
