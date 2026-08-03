# Pain Points — Quá trình Số hóa Tài liệu thành Knowledge Graph (KG Digitization)

> **Phạm vi**: Data Engineer, Architect, BA, Developer, AI Engineer  
> **Loại**: Process pain points — ảnh hưởng trực tiếp đến adoption của VNP-KGP và KG Service  
> **Giải pháp hướng đến**: [VNP Knowledge Governance Platform](../../../vnp-ontology/) + [Knowledge Graph Service](../../../knowledge-graph-service/) + **PIKB AI-Assisted Digitization**  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-02

---

## Tổng quan

Khi tổ chức quyết định số hóa tài liệu và tri thức vào **Knowledge Graph (KG)**, quá trình này hiện tại phải thực hiện **hoàn toàn bằng tay** qua nhiều công đoạn phức tạp, tốn nhiều thời gian, và đòi hỏi kiến thức chuyên sâu về ontology, schema design, và KG service API. Đây là **rào cản adoption** lớn nhất khiến tổ chức không thể scale việc số hóa tri thức.

Chuỗi công đoạn thủ công hiện tại:

```
Tài liệu gốc (Word/PDF/Docs...)
        │
        ▼ [THỦ CÔNG] Đọc + phân tích
Xác định ontology domain phù hợp
        │
        ▼ [THỦ CÔNG] Tra cứu 46 domains trong vnp-ontology
Kiểm tra xem entity đã tồn tại chưa
        │
        ├─ Chưa có → [THỦ CÔNG] Mở rộng ontology (RFC process)
        │               - Viết YAML entity definition
        │               - Submit RFC
        │               - Chờ review/approve
        │               - Merge vào vnp-ontology
        │
        ▼ [THỦ CÔNG] Generate schema từ ontology
Sinh JSON Schema / GraphQL / Protobuf
        │
        ▼ [THỦ CÔNG] Đăng ký schema vào KGP Schema Registry
POST /v1/schemas/{domain}/{entity}
        │
        ▼ [THỦ CÔNG] Chuyển đổi dữ liệu sang KG format
Map từng field tài liệu → ontology attribute
        │
        ▼ [THỦ CÔNG] Ingest nodes vào KG Service
POST /v1/kg/write/nodes (từng node một)
        │
        ▼ [THỦ CÔNG] Tìm và tạo relationships
Phân tích để biết entity này relate đến entity kia như thế nào
        │
        ▼ [THỦ CÔNG] Mapping dữ liệu cũ với dữ liệu mới
Identify overlaps, deduplicate, resolve conflicts
        │
        ▼ Kết quả cuối: KG có dữ liệu
           (sau 2–4 tuần làm việc cho 1 domain)
```

---

## Pain Points chi tiết

### PP-KGD-01 — Không có công cụ hỗ trợ tự động xác định ontology domain phù hợp cho tài liệu

**Mức độ**: 🔴 Critical  
**Actors bị ảnh hưởng**: Data Engineer, BA, Architect, Developer  
**Tần suất**: Mỗi lần có tài liệu mới cần số hóa

**Mô tả**:
Khi muốn đưa một tài liệu vào KG, bước đầu tiên phải là xác định **domain ontology phù hợp** (payment, lending, risk, compliance, software-dev, knowledge...). VNP Ontology hiện có **46 domains** với hàng trăm entity types.

Hiện tại người thực hiện phải:
- Đọc toàn bộ tài liệu
- Tra cứu thủ công trong `vnp-ontology/data/ontology/` để tìm domain phù hợp
- Đọc GUIDELINES.md để hiểu nguyên tắc phân loại
- Dùng AI (ChatGPT, Claude) để phỏng đoán — nhưng AI không biết nội dung vnp-ontology nên kết quả không chính xác
- Hỏi người có kinh nghiệm về ontology — bottleneck vì chỉ có vài người biết

**Ví dụ thực tế**:
```
BA có tài liệu "Payment Error Handling Specification"
→ Cần xác định: domain là "payment"? "engineering"? "software-dev"?
→ Entity type là "ErrorCode"? "FunctionSpec"? "BusinessRule"?
→ Tra cứu thủ công trong 46 domains mất 2–4 giờ
→ Vẫn không chắc chắn → hỏi Architect → blocked 1–2 ngày chờ response
```

**Hệ quả kinh doanh**:
- Bottleneck tại bước đầu tiên → toàn bộ quá trình số hóa bị block
- Sai domain → phải làm lại toàn bộ mapping → waste effort
- Chỉ một số ít người có thể thực hiện → không scale
- Team ngại số hóa vì không biết bắt đầu từ đâu

**Giải pháp cần có**:
- **AI-Assisted Ontology Matching**: Upload tài liệu → AI đọc nội dung → suggest top-3 domain + entity types với confidence score và lý do giải thích
- **Search Ontology API**: `POST /v1/ontology/match` — input: document text, output: ranked domain suggestions
- **Ontology Explorer UI**: Interactive tree để browse 46 domains, filter by keyword, xem examples per entity
- **PIKB AI Copilot biết vnp-ontology**: Vì được grounded với KG Service, AI biết chính xác 46 domains và suggest đúng

**Feature cần**: F-16 Document Intelligence (AI ontology matching) + KG Service `/v1/ontology/match`  
**Xem giải pháp**: [knowledge-governance-solutions.md](../solutions/knowledge-governance-solutions.md)

---

### PP-KGD-02 — Quy trình mở rộng ontology (khi thiếu entity) phức tạp, nhiều bước thủ công

**Mức độ**: 🔴 Critical  
**Actors bị ảnh hưởng**: Data Engineer, Architect, BA  
**Tần suất**: Mỗi khi gặp khái niệm mới chưa có trong ontology

**Mô tả**:
Khi tài liệu chứa khái niệm chưa có trong vnp-ontology, người thực hiện phải chạy qua **toàn bộ RFC process** thủ công:

**Quy trình hiện tại (ước tính 3–10 ngày)**:
```
Bước 1: Viết entity YAML definition
  vnp-ontology/data/ontology/{domain}/entities/{EntityName}.yaml
  → Phải biết cú pháp YAML của vnp-ontology
  → Phải định nghĩa đúng attributes, types, constraints
  → Phải biết relationship types hợp lệ
  ⏱️ 2–4 giờ nếu biết, 1–2 ngày nếu mới

Bước 2: Kiểm tra conflict với entities hiện có
  → Không có tool để check tự động
  → Phải đọc tay toàn bộ entities trong domain
  ⏱️ 1–4 giờ

Bước 3: Submit RFC
  POST /v1/governance/rfcs
  → Phải biết RFC format
  → Viết description, motivation, impact assessment
  ⏱️ 1–2 giờ

Bước 4: Chờ review
  → Reviewer (Architect/Domain Owner) có thể bận
  → Review cycle 1–5 ngày
  ⏱️ 1–5 ngày blocked

Bước 5: Sau approve — merge vào vnp-ontology
  → Git workflow, PR, CI/CD pipeline
  ⏱️ 0.5–1 ngày

Bước 6: Generate schema và đăng ký vào KGP Schema Registry
  → Chạy pikb-cli codegen generate
  → POST /v1/schemas/{domain}/{entity}
  ⏱️ 1–2 giờ
```

**Tổng thời gian**: 3–10 ngày làm việc **chỉ để thêm 1 entity type mới**

**Hệ quả kinh doanh**:
- Quá trình số hóa bị block hoàn toàn khi gặp khái niệm mới
- Team không muốn mở rộng ontology → cố ép data vào entity không phù hợp → data quality thấp
- Ontology trở thành bottleneck thay vì enabler
- Chỉ có 1–2 người biết cách mở rộng → single point of failure

**Giải pháp cần có**:
- **AI-Assisted Entity Generation**: Từ mô tả tự nhiên → AI draft entity YAML → người review và adjust → submit
- **Conflict Detection**: Tự động check entity mới có overlap với entity hiện có không
- **Accelerated RFC**: Với low-risk entity additions (chỉ add optional attribute), auto-approve với notification
- **Ontology Extension Wizard**: UI step-by-step guide để định nghĩa entity mới — không cần biết YAML syntax

**Feature cần**: VNP-KGP AI-Assisted RFC + F-18 Knowledge Governance với fast-track approval  
**Xem giải pháp**: [knowledge-governance-solutions.md](../solutions/knowledge-governance-solutions.md)

---

### PP-KGD-03 — Phải sinh schema thủ công và đăng ký vào KGP Service trước khi có thể ingest data

**Mức độ**: 🔴 Critical  
**Actors bị ảnh hưởng**: Data Engineer, Developer  
**Tần suất**: Mỗi lần ingest loại data mới

**Mô tả**:
Trước khi ingest bất kỳ dữ liệu nào vào KG Service, phải hoàn thành **3 bước tiên quyết** theo đúng thứ tự:

```
Bước A: Ontology phải đã được define trong vnp-ontology
         (nếu chưa → PP-KGD-02 process)

Bước B: Schema phải được generate từ ontology
  pikb-cli codegen generate --entity Payment --format json_schema
  → Tạo ra payment.schema.json

Bước C: Schema phải được đăng ký vào KGP Schema Registry
  POST /v1/schemas/payment/Payment
  Content-Type: application/json
  Body: { payment.schema.json }
  → Nhận schema_id: "schema-payment-payment-v1"

Bước D: Mapping config phải được đăng ký
  POST /v1/tenants/{tenant}/ontology/domains/{domain}/mappings
  → Định nghĩa cách map từ source format → canonical schema

CHỈ SAU KHI XONG A+B+C+D mới có thể:
  POST /v1/kg/write/nodes (với schema_id reference)
```

Vấn đề:
- Nếu làm sai thứ tự → API trả về cryptic errors
- Nếu ontology version không khớp với schema version → ingest fail
- Mỗi version update của entity → phải re-generate schema → re-register
- Không có tooling để automate pipeline này → làm tay từng bước

**Ví dụ thực tế**:
```
Engineer cố ingest PaymentError documents:
→ POST /v1/kg/write/nodes → 400 Error: "schema_id not registered"
→ Đăng ký schema → 409 Error: "ontology entity not found"
→ Check ontology → entity chưa có trong đúng domain
→ Phải mở rộng ontology (PP-KGD-02) trước
→ Mất 3 ngày để unblock
```

**Hệ quả kinh doanh**:
- Prerequisites không rõ ràng → engineer waste nhiều giờ debug errors
- Pipeline fragile — một bước fail → toàn bộ ingest process stop
- Version drift giữa ontology, schema, và KG data → silent data inconsistency
- Knowledge về prerequisites chỉ có trong đầu 1–2 người

**Giải pháp cần có**:
- **Auto-provisioning**: Khi submit entity definition → hệ thống tự động generate schema + register + create mapping template
- **Dependency validation**: Trước khi ingest → validate tự động rằng ontology + schema + mapping đều sẵn sàng
- **Pipeline orchestration**: Single command: `pikb-cli provision --entity Payment --domain payment --from-doc payment-prd.md`
- **Clear error messages**: Thay vì cryptic errors → "Schema not registered. Run: pikb-cli codegen generate && pikb-cli schema register"

**Feature cần**: PIKB CLI auto-provisioning + F-19 Schema Code Generator với auto-register  
**Xem giải pháp**: [knowledge-governance-solutions.md](../solutions/knowledge-governance-solutions.md)

---

### PP-KGD-04 — Chuyển đổi dữ liệu từ tài liệu sang KG nodes phải làm tay, tốn nhiều thời gian

**Mức độ**: 🔴 Critical  
**Actors bị ảnh hưởng**: Data Engineer, BA, Developer  
**Tần suất**: Mỗi lần số hóa một tài liệu hoặc dataset

**Mô tả**:
Sau khi ontology và schema đã sẵn sàng, công đoạn **chuyển đổi tài liệu thành KG nodes** vẫn phải làm thủ công:

**Tình huống 1 — Tài liệu có cấu trúc (JSON, YAML, CSV)**:
```python
# Engineer phải tự viết transformation script
with open('payment-errors.json') as f:
    data = json.load(f)

for item in data:
    # Phải biết KG Service API format
    node = {
        "domain_id": "payment",
        "node_type": "ErrorCode",
        "external_ref": item["error_code"],
        "attributes": {
            "title": item["title"],          # map field name
            "severity": item["severity"],     # may need value transformation
            "description": item["message"]   # field rename
        }
    }
    # Phải handle pagination, retry, rate limiting
    requests.post("/v1/kg/write/nodes", json=node)
```

**Tình huống 2 — Tài liệu phi cấu trúc (Word, PDF, Confluence page)**:
```
Không có automated tool
→ Engineer phải đọc từng trang
→ Copy-paste thủ công vào KG node format
→ 50 pages = 50 nodes = 50 API calls = 1–2 ngày làm việc
→ Error-prone, thiếu consistency
```

**Tình huống 3 — Tài liệu bán cấu trúc (Markdown với headers, tables)**:
```
Có thể parse nhưng:
→ Không có standard parser
→ Phải viết custom parser cho từng document format
→ Mỗi team có format riêng → N parsers
```

**Hệ quả kinh doanh**:
- 1 domain với 100 documents × 2 ngày = 200 ngày công → không feasible
- Error rate cao vì làm tay → data quality thấp trong KG
- Chỉ có người biết Python/API mới làm được → BA và PO không tự làm được
- Quá trình số hóa không thể scale

**Giải pháp cần có**:
- **AI Document Parser**: Upload PDF/Word/Confluence → AI extract entities → preview → approve → batch ingest
- **No-code Data Mapping UI**: BA tự map field nguồn → field ontology → hệ thống generate transformation và ingest
- **Batch Import API**: `POST /v1/kg/write/ingest/batch` với document content → AI extract nodes tự động
- **Format-specific connectors**: Built-in parsers cho Confluence, Google Docs, Markdown, JSON, CSV

**Feature cần**: F-16 Document Intelligence (AI extraction) + KG Service batch ingest endpoint  
**Xem giải pháp**: [knowledge-governance-solutions.md](../solutions/knowledge-governance-solutions.md)

---

### PP-KGD-05 — Phải tìm và tạo relationships thủ công — không biết entity nào liên kết với entity nào

**Mức độ**: 🔴 Critical  
**Actors bị ảnh hưởng**: Data Engineer, Architect, BA  
**Tần suất**: Mỗi lần số hóa (đây là phần tốn thời gian nhất)

**Mô tả**:
Nếu nodes là phần khó, thì **relationships (edges) là phần khó hơn nhiều**. Knowledge Graph chỉ có giá trị khi có relationships phong phú — nhưng tạo relationships yêu cầu:

1. **Biết entity nào relate với entity nào** (không phải lúc nào cũng hiển nhiên)
2. **Biết relationship type phù hợp** (IMPLEMENTS? DEPENDS_ON? VALIDATES? DESCRIBES? GOVERNS?)
3. **Biết ID của target node** (phải tìm hoặc đã biết trước)
4. **Tạo relationship theo đúng direction** (from → to hay to → from)

**Ví dụ thực tế**:
```
Engineer vừa ingest: 45 Requirement nodes, 30 UserStory nodes, 20 TestCase nodes
Cần tạo relationships:
  Requirement --[BREAKS_DOWN_TO]--> UserStory  (45 × n relationships)
  UserStory --[IMPLEMENTED_BY]--> CodeFunction (30 × m relationships)
  CodeFunction --[VALIDATED_BY]--> TestCase    (??? relationships)

Vấn đề:
→ Phải đọc từng requirement để biết nó relate đến UserStory nào
→ Phải tra cứu ID của UserStory target
→ Phải biết chính xác relationship type ("BREAKS_DOWN_TO" hay "HAS_STORY"?)
→ Phải tạo từng relationship bằng API call riêng

100 nodes × 3 avg relationships = 300 API calls
Với mỗi call cần biết exact node IDs → lookup trước
→ Thực tế: 300 lookups + 300 creates = 600 API calls thủ công
```

**Hệ quả kinh doanh**:
- Phần tốn thời gian nhất trong toàn bộ digitization process
- KG thường được ingest với nodes nhưng thiếu relationships → KG vô nghĩa (chỉ là flat store)
- Relationships thường bị bỏ qua vì quá tốn công → mất đi giá trị của KG
- Không có cách verify relationships đã đúng chưa trước khi commit

**Giải pháp cần có**:
- **AI Relationship Suggestion**: Sau khi ingest nodes → AI analyze nội dung và suggest relationships với confidence score
- **Relationship Builder UI**: Drag-and-drop interface để tạo relationships giữa nodes
- **Auto-link by reference**: Nếu Requirement document mention "US-045" → tự động create link đến UserStory node US-045
- **Relationship validation**: Check relationship type có hợp lệ không trước khi create (vd: `Requirement VALIDATED_BY TestCase` là hợp lệ, nhưng `TestCase BREAKS_DOWN_TO Requirement` là sai direction)

**Feature cần**: F-17 Requirements Traceability (auto-link) + KG Service AI relationship suggestion  
**Xem giải pháp**: [knowledge-governance-solutions.md](../solutions/knowledge-governance-solutions.md)

---

### PP-KGD-06 — Mapping dữ liệu hiện có với nhau cực kỳ phức tạp khi nhiều nguồn cùng lúc

**Mức độ**: 🔴 Critical  
**Actors bị ảnh hưởng**: Data Engineer, Architect  
**Tần suất**: Khi integrate nhiều data sources vào cùng KG

**Mô tả**:
Khi tổ chức có nhiều nguồn dữ liệu cùng lúc cần đưa vào KG (vd: Jira issues + Confluence pages + Git commits + Google Docs PRD), vấn đề phát sinh là **cross-source data mapping**:

**Vấn đề 1 — Entity deduplication**:
```
Source A (Jira): Issue "PAY-123 — QR timeout handling"
Source B (Confluence): Page "QR Payment Timeout Spec"
Source C (Google Docs): PRD Section "F-PAY-007 — Timeout"

→ Cả 3 nói về cùng 1 concept nhưng khác format, khác ID
→ Phải detect: đây có phải cùng entity không?
→ Nếu là cùng entity → merge hay keep separate với cross-link?
→ Không có automated deduplication tool
```

**Vấn đề 2 — ID mapping**:
```
Jira ID: PAY-123
Confluence page ID: 1234567
Git commit hash: a3b4c5d6
Google Doc heading ID: h.abc123xyz

→ Phải tự build lookup table để biết:
  PAY-123 ≈ confluence-1234567 ≈ prd-F-PAY-007
→ Không có tool → làm Excel spreadsheet → error prone
```

**Vấn đề 3 — Semantic conflicts**:
```
Jira: "priority: High"
Confluence: "priority: P1"
PRD: "priority: Must Have"

→ Phải normalize về canonical priority: "high"
→ Không có shared value vocabulary
→ Phải viết transformation logic cho mỗi source
```

**Vấn đề 4 — Temporal mapping**:
```
Jira issue created: 2024-01-15
Confluence page updated: 2024-03-20
PRD section added: 2024-02-10

→ Cái nào là canonical version?
→ Thay đổi nào xảy ra trước? Impact chain như thế nào?
→ Không có timeline graph
```

**Hệ quả kinh doanh**:
- Duplicate nodes trong KG → khi query ra kết quả trùng lặp → confusing
- Sai mapping → sai relationships → sai impact analysis → wrong decisions
- Mapping work chiếm 40–60% tổng thời gian digitization
- Mapping logic không được document → next person phải rediscover

**Giải pháp cần có**:
- **AI Entity Resolution**: Upload data từ 2 sources → AI detect potential duplicates với similarity score
- **Mapping Registry (VNP-KGP F05)**: Lưu permanent mapping giữa external IDs → internal KG IDs
- **Value Normalization Rules**: Define canonical vocabulary → auto-normalize incoming values
- **Merge/Link Decision UI**: Review suggested duplicates → click "Merge" or "Link" or "Separate"
- **Provenance tracking**: Mỗi node biết nó đến từ source nào, merge từ bao nhiêu sources

**Feature cần**: VNP-KGP F05 Mapping Registry + KG Service entity resolution API  
**Xem giải pháp**: [knowledge-governance-solutions.md](../solutions/knowledge-governance-solutions.md)

---

### PP-KGD-07 — Thiếu observability trong quá trình digitization — không biết tiến độ và chất lượng

**Mức độ**: 🟠 High  
**Actors bị ảnh hưởng**: Data Engineer, Architect, PO  
**Tần suất**: Trong toàn bộ quá trình số hóa

**Mô tả**:
Trong quá trình số hóa tài liệu vào KG, người thực hiện không có visibility về:

**Thiếu progress tracking**:
```
Bắt đầu ingest 500 documents từ Confluence...
→ Chạy script...
→ Chờ...
→ Sau 2 giờ: script crash ở document 237/500
→ Không biết: 237 documents đã vào KG đúng chưa?
→ Có duplicate không?
→ Relationships đã được tạo đúng chưa?
→ Rollback hay tiếp tục từ 238?
```

**Thiếu data quality metrics**:
```
Sau khi ingest xong:
→ Không biết: có bao nhiêu nodes thiếu mandatory attributes?
→ Không biết: có bao nhiêu nodes không có relationships?
→ Không biết: data completeness rate là bao nhiêu?
→ Không biết: có entity nào bị duplicate không?
```

**Thiếu validation**:
```
→ Không có dry-run mode để test trước khi ingest real data
→ Không có rollback mechanism nếu ingest sai
→ Không có diff view giữa trước và sau khi ingest
```

**Hệ quả kinh doanh**:
- Ingest errors discovered muộn → phải clean up KG data tốn công hơn
- Không confident về data quality → không dám dùng KG cho production use cases
- Phải re-ingest nhiều lần → waste time và compute

**Giải pháp cần có**:
- **Ingest Job Dashboard**: Real-time progress, ETA, error count, success count
- **Dry-run mode**: `POST /v1/kg/write/ingest/dry-run` → validate without committing
- **Data Quality Score**: Sau ingest → auto-score: completeness, relationship density, orphan nodes
- **Rollback**: Ingest job có transaction ID → có thể rollback nếu cần
- **Diff view**: "Before ingest: 0 nodes, After ingest: 45 nodes, 73 relationships — Preview here"

**Feature cần**: KG Service ingest observability + F-10 Observability & Quality (VNP-KGP)  
**Xem giải pháp**: [knowledge-governance-solutions.md](../solutions/knowledge-governance-solutions.md)

---

### PP-KGD-08 — Knowledge về quy trình số hóa chỉ nằm trong đầu 1–2 người, không được document

**Mức độ**: 🟠 High  
**Actors bị ảnh hưởng**: Toàn bộ tổ chức  
**Tần suất**: Khi người mới muốn thực hiện số hóa

**Mô tả**:
Quy trình số hóa tài liệu vào KG là một **tribal knowledge** — chỉ có 1–2 người biết làm đúng toàn bộ pipeline. Khi họ vắng mặt hoặc nghỉ việc:

```
Câu hỏi thường gặp từ người mới:
→ "Làm sao biết phải dùng domain nào?"
→ "Có cần tạo RFC trước không?"
→ "Sau khi RFC approve thì làm gì tiếp?"
→ "Schema register ở đâu?"
→ "Ingest node xong thì relationships tạo như thế nào?"
→ "Nếu bị lỗi thì làm gì?"
→ "Làm sao biết data trong KG đúng chưa?"

Hiện tại: không có documentation, không có runbook, không có playbook
→ Phải hỏi người biết → bottleneck
→ Learn by trial and error → tốn nhiều ngày
```

**Hệ quả kinh doanh**:
- Bus factor = 1 → single point of failure
- Onboarding người mới vào quá trình digitization mất 2–4 tuần
- Không có way để audit: "người này làm đúng quy trình chưa?"
- Mỗi người làm theo cách riêng → inconsistent KG data

**Giải pháp cần có**:
- **PIKB Digitization Playbook**: Step-by-step guide với examples cho từng loại tài liệu
- **Interactive Wizard UI**: Guided workflow — hỏi từng bước, validate input, suggest next action
- **CLI với contextual help**: `pikb-cli digitize --help` → full documentation + examples
- **Video walkthroughs**: Record digitization sessions → searchable knowledge base

**Feature cần**: PIKB Documentation Hub + CLI interactive mode

---

## Ma trận Pain Point vs. Giải pháp

| Pain Point | Mức độ | Actors chính | Giải pháp chính | Tool/API cần có |
|:---|:---:|:---|:---|:---|
| PP-KGD-01: Xác định ontology domain thủ công | 🔴 | Data Eng, BA, Arch | AI Ontology Matching | `/v1/ontology/match` |
| PP-KGD-02: Mở rộng ontology phức tạp, nhiều bước | 🔴 | Data Eng, Arch | AI-Assisted RFC + Fast-track | Ontology Extension Wizard |
| PP-KGD-03: Schema provision thủ công nhiều bước | 🔴 | Data Eng, Dev | Auto-provisioning pipeline | `pikb-cli provision` |
| PP-KGD-04: Chuyển đổi tài liệu → nodes thủ công | 🔴 | Data Eng, BA | AI Document Parser + Batch Import | `/v1/kg/write/ingest/batch` |
| PP-KGD-05: Tạo relationships thủ công, khó | 🔴 | Data Eng, Arch | AI Relationship Suggestion + UI | Relationship Builder |
| PP-KGD-06: Mapping nhiều nguồn data phức tạp | 🔴 | Data Eng, Arch | AI Entity Resolution + Mapping Registry | F05 + Entity Resolution API |
| PP-KGD-07: Không có observability trong digitization | 🟠 | Data Eng, PO | Ingest Job Dashboard + Dry-run | Job tracking API |
| PP-KGD-08: Tribal knowledge, không có documentation | 🟠 | Tất cả | Digitization Playbook + Wizard | CLI interactive mode |

---

## Thời gian ước tính: Thủ công vs. Tự động hóa

| Công đoạn | Thủ công (hiện tại) | Với Tool hỗ trợ | Tiết kiệm |
|:---|:---:|:---:|:---:|
| Xác định ontology domain | 2–4 giờ | < 5 phút (AI suggest) | 96% |
| Mở rộng ontology (nếu cần) | 3–10 ngày | 2–4 giờ (AI draft + fast review) | 90% |
| Provision schema + register | 2–4 giờ | < 1 phút (auto-provision) | 99% |
| Chuyển đổi 100 documents → nodes | 5–10 ngày | 1–2 giờ (AI extraction + batch) | 95% |
| Tạo relationships | 3–7 ngày | 4–8 giờ (AI suggest + UI) | 85% |
| Mapping nhiều nguồn data | 5–15 ngày | 1–3 ngày (AI resolution) | 80% |
| **Tổng cho 1 domain vừa** | **3–6 tuần** | **2–4 ngày** | **~90%** |

---

## Root Cause: Tại sao quá trình này khó?

### 1. Thiếu abstraction layer phù hợp cho non-expert

KG Service và VNP-KGP được thiết kế cho expert users (Data Engineers, Architects). Không có layer nào cho BA hoặc Developer thông thường sử dụng mà không cần kiến thức sâu về:
- Ontology modeling
- Graph database concepts
- KG Service API internals
- YAML schema syntax

### 2. Thiếu AI assistance trong toàn bộ pipeline

Hiện tại AI assistance chỉ có ở output (query KG, RAG). Chưa có AI assistance ở input (số hóa vào KG). Đây là **asymmetry** lớn:
```
✅ AI có thể QUERY knowledge từ KG → tốt
❌ AI không giúp được khi DIGITIZE knowledge vào KG → bottleneck
```

### 3. Quá nhiều manual steps không có automation

Mỗi bước trong pipeline đều là separate manual action, không có orchestration. Thiếu:
- Pipeline automation tool
- Idempotent operations (chạy lại không bị duplicate)
- Error recovery mechanism

### 4. Feedback loop chậm

Không biết data quality tốt không cho đến khi query và thấy kết quả sai. Không có:
- Pre-ingest validation
- Post-ingest quality score
- Continuous monitoring

---

## Quotes từ thực tế

> *"Tôi mất 3 tuần để số hóa một domain, phần lớn thời gian là figure out phải dùng ontology nào và làm sao create relationships đúng"*  
> — Data Engineer feedback

> *"Chúng tôi muốn đưa toàn bộ SRS vào KG nhưng không dám bắt đầu vì không biết quy trình và sợ làm sai mất công làm lại"*  
> — Business Analyst feedback

> *"Phần khó nhất không phải là học KG Service API, mà là học VNP Ontology đủ để biết nên map data của mình vào đâu"*  
> — Developer feedback
