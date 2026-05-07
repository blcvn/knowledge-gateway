---
version: 1.2.0
last_updated: 2026-04-21
updated_by: doc-management-expert
status: Approved
scope: PACKAGE-AND-SERVICE-LEVEL
---

# Specs Catalog — Cấp Package & Service
## Requirement-to-UI Automation Platform

> **Phạm vi:** Specs được quản lý tại cấp **package** (`apps/*/specs/`) và **service** (`services/*/specs/`). Mỗi spec là một **chỉ thị thực thi** có phạm vi giới hạn trong đúng một package hoặc service.
>
> **Phân biệt với Tài liệu cấp Repo:** Tài liệu (docs/) mô tả *hệ thống là gì*. Spec mô tả *cần làm gì cụ thể* trong một package/service để thực hiện sự thay đổi, và AI agent sử dụng spec như lệnh thực thi.
>
> **Mục tiêu:** Định nghĩa hệ thống tài liệu Spec chuẩn để AI agent có thể đọc và thực thi việc viết code, kiểm tra, và kiểm soát chất lượng một cách chính xác và nhất quán.

---

## Nguyên Tắc Cốt Lõi

1. **Spec là lệnh cho AI:** Mỗi spec file phải đủ rõ để AI agent thực thi mà không cần hỏi thêm.
2. **Một spec — một mục tiêu:** Mỗi file spec giải quyết đúng một vấn đề có thể kiểm tra được.
3. **Spec phải có Acceptance Criteria:** AI chỉ biết mình "xong việc" khi có tiêu chí rõ ràng.
4. **Mọi thay đổi bắt đầu từ Spec:** Không có code change nào mà không có spec tương ứng.
5. **Spec được version:** Spec thay đổi → version tăng → AI tái thực thi những gì bị ảnh hưởng.

---

## Phân Loại Specs — 7 Loại Chính

| Loại | Mã | Kích hoạt khi | AI thực hiện |
|---|---|---|---|
| Feature Spec | `FEAT` | Yêu cầu tính năng mới từ PRD | Viết code tính năng mới |
| Change Request Spec | `CR` | Thay đổi yêu cầu sản phẩm/nghiệp vụ | Sửa code hiện có theo yêu cầu mới |
| Architecture Spec | `ARCH` | Thay đổi kiến trúc hoặc pattern | Refactor cấu trúc, không thay đổi behavior |
| Technical Spec | `TECH` | Thay đổi kỹ thuật thuần túy (upgrade, migration) | Migrate/upgrade không thay đổi business logic |
| Bug Fix Spec | `BUG` | Lỗi được phát hiện và cần sửa | Sửa lỗi, viết regression test |
| Quality Spec | `QA` | Cải thiện chất lượng, coverage, performance | Refactor, tối ưu, bổ sung tests |
| Security Spec | `SEC` | Lỗ hổng bảo mật hoặc hardening | Patch vulnerability, tăng cường bảo mật |

---

## Cấu Trúc Tổ Chức Specs Trong Monorepo

```
vnp-design-platform/
│
[FRONTEND PACKAGES — apps/]
├── apps/
│   ├── preview/
│   │   ├── specs/                     ← SPECS cho package này
│   │   │   ├── features/
│   │   │   ├── changes/
│   │   │   ├── architecture/
│   │   │   ├── technical/
│   │   │   ├── bugs/
│   │   │   ├── quality/
│   │   │   └── security/
│   │   └── docs/                     ← DOCS cho package này
│   │       ├── README.md
│   │       ├── architecture.md
│   │       └── changelog.md
│   ├── demo/
│   │   ├── specs/ ...
│   │   └── docs/ ...
│   └── openui/
│       ├── specs/ ...
│       └── docs/ ...
│
[BACKEND SERVICES — services/]
└── services/
    ├── chat-preview-service/
    │   ├── specs/                     ← SPECS cho service này
    │   │   ├── features/
    │   │   ├── changes/
    │   │   ├── architecture/
    │   │   ├── technical/
    │   │   ├── bugs/
    │   │   ├── quality/
    │   │   └── security/
    │   └── docs/                     ← DOCS cho service này
    │       ├── README.md
    │       ├── api.md
    │       ├── architecture.md
    │       ├── data-model.md
    │       ├── configuration.md
    │       ├── runbook.md
    │       └── changelog.md
    ├── doc_to_kg/
    │   ├── specs/ ...
    │   └── docs/ ...
    ├── knowledge-gateway/
    │   ├── specs/ ...
    │   └── docs/ ...
    └── ui-knowledge-service/
        ├── specs/ ...
        └── docs/ ...
```

> **Quy tắc cốt lõi:**
> - **`specs/`** chứa chỉ thị thực thi có thời hạn (Done xong là khép lại)
> - **`docs/`** chứa tài liệu sống (cập nhật liên tục theo sự phát triển của package/service)
> - Specs tham chiếu đến docs nhưng không bao giờ ngược lại
> - **`technical_design.md`** là ngoại lệ được phép — xem quy tắc đặc biệt bên dưới

**Quy tắc đặt tên file spec:**
- `[TYPE]-[NNN]-[short-kebab-title].md`
- Ví dụ: `FEAT-001-requirement-parsing-api.md`, `BUG-023-goroutine-leak-on-timeout.md`

**Thư mục được phép trong `specs/`:**

| Thư mục | Loại spec chứa | Bắt buộc |
|---|---|---|
| `features/` | `FEAT-NNN-*.md` | ⚪ Khi có feature mới |
| `changes/` | `CR-NNN-*.md` | ⚪ Khi có change request |
| `architecture/` | `ARCH-NNN-*.md` | ⚪ Khi refactor kiến trúc |
| `technical/` | `TECH-NNN-*.md` | ⚪ Khi upgrade/migrate |
| `bugs/` | `BUG-NNN-*.md` | ⚪ Khi phát hiện bug |
| `quality/` | `QA-NNN-*.md` | ⚪ Khi cải thiện chất lượng |
| `security/` | `SEC-NNN-*.md` | ⚪ Khi có lỗ hổng bảo mật |

> **❌ KHÔNG hợp lệ trong `specs/`:**
> - `data/` — dữ liệu test đặt tại `services/[name]/testdata/`
> - `test/` — dùng `quality/` hoặc để tại `services/[name]/tests/`
> - `kg/`, `pipelines/`, `crs/` — tổ chức theo loại, không theo domain

**Header bắt buộc trong mọi spec file:**
```yaml
---
id: [TYPE]-[NNN]
title: ...
package: apps/preview    # hoặc
service: chat-preview-service
version: 1.0.0
status: Draft | Ready | In Progress | Done | Cancelled
---
```

---

## LOẠI 1: Feature Spec (`FEAT`)

**Khi nào tạo:** Khi PRD hoặc Product Owner yêu cầu xây dựng tính năng mới.

**AI đọc spec này để:** Hiểu đầy đủ tính năng → viết code đúng behavior → viết tests đủ coverage.

### Template FEAT Spec
```markdown
---
id: FEAT-[NNN]
title: [Tên tính năng]
service: [service-name]
version: 1.0.0
status: Draft | Ready | In Progress | Done | Cancelled
priority: P0 | P1 | P2 | P3
created: YYYY-MM-DD
updated: YYYY-MM-DD
linked_prd: [Section trong PRD liên quan]
linked_adr: [ADR-XXXX nếu có]
---

## Mục Tiêu
[1-2 câu mô tả tính năng này làm gì và tại sao cần nó]

## Bối Cảnh Nghiệp Vụ
[Giải thích business context, actor nào sử dụng, trigger là gì]

## Scope
### In Scope (AI phải implement)
- [Item 1]
- [Item 2]

### Out of Scope (AI không được implement trong spec này)
- [Item A]

## Thiết Kế Kỹ Thuật

### API Contract
[Nếu có API: method, path, request schema, response schema, error codes]

### Data Model Changes
[Tables/fields mới, migrations cần tạo]

### Business Logic
[Thuật toán, rules, luồng xử lý — đủ cụ thể để AI viết code]

### Internal Architecture
[Service layers cần thay đổi: handler / usecase / repository]

## Acceptance Criteria
> AI tự kiểm tra: Nếu tất cả các tiêu chí này pass → tính năng hoàn chỉnh.

- [ ] AC-1: [Given <context> When <action> Then <expected outcome>]
- [ ] AC-2: [...]
- [ ] AC-3: [Edge case handling]

## Test Requirements
- **Unit tests:** [Những function/method nào bắt buộc có unit test]
- **Integration tests:** [Scenario cần test end-to-end]
- **Minimum coverage:** 80%

## Definition of Done
- [ ] Code implement đủ Acceptance Criteria
- [ ] Unit tests pass, coverage ≥ 80%
- [ ] Integration tests pass
- [ ] API documentation cập nhật trong `api.md`
- [ ] Không có lint errors
- [ ] Peer review approved
```

---

## LOẠI 2: Change Request Spec (`CR`)

**Khi nào tạo:** Khi yêu cầu sản phẩm hoặc nghiệp vụ thay đổi — tính năng đã có cần hoạt động khác đi.

**AI đọc spec này để:** Hiểu behavior hiện tại vs behavior mới → sửa code đúng chỗ → không làm hỏng những gì đang hoạt động.

### Template CR Spec
```markdown
---
id: CR-[NNN]
title: [Mô tả thay đổi]
service: [service-name]
version: 1.0.0
status: Draft | Ready | In Progress | Done
linked_feat: [FEAT-NNN nếu là thay đổi từ tính năng cũ]
breaking_change: true | false
---

## Lý Do Thay Đổi
[Tại sao requirement thay đổi — business reason]

## So Sánh: Trước & Sau

### Behavior Hiện Tại (Before)
[Mô tả behavior hiện tại của hệ thống]

### Behavior Mới (After)
[Mô tả behavior mong muốn sau khi thay đổi]

### Delta (Những gì thay đổi chính xác)
- [+] Thêm: ...
- [-] Bỏ: ...
- [~] Sửa: ...

## Breaking Changes
[Nếu có: API signature thay đổi, data format thay đổi, behavior thay đổi gây ảnh hưởng consumer]

## Migration Plan
[Nếu cần: Hướng dẫn migrate data hoặc update consumer]

## Acceptance Criteria
- [ ] AC-1: Behavior cũ [X] không còn xảy ra
- [ ] AC-2: Behavior mới [Y] hoạt động đúng
- [ ] AC-3: Regression: các tính năng liên quan không bị ảnh hưởng

## Regression Test Checklist
[Liệt kê các test cases hiện có cần re-run để confirm không bị regression]
```

---

## LOẠI 3: Architecture Spec (`ARCH`)

**Khi nào tạo:** Khi cần thay đổi cấu trúc code, pattern, hoặc kiến trúc mà không thay đổi business behavior.

**AI đọc spec này để:** Refactor code theo kiến trúc mới — đảm bảo behavior giữ nguyên, chỉ cấu trúc thay đổi.

### Template ARCH Spec
```markdown
---
id: ARCH-[NNN]
title: [Mô tả thay đổi kiến trúc]
service: [service-name]
linked_adr: ADR-XXXX
behavior_change: false   # ARCH spec không thay đổi behavior
---

## Vấn Đề Kiến Trúc Hiện Tại
[Mô tả vấn đề: coupling cao, khó test, vi phạm SOLID, etc.]

## Kiến Trúc Mới Đề Xuất
[Mô tả pattern/structure mới]

## Phạm Vi Refactor
### Files cần tạo mới
- `[path]`: [lý do]

### Files cần sửa
- `[path]`: [thay đổi gì]

### Files cần xóa
- `[path]`: [lý do]

## Invariants (Không được thay đổi)
> AI phải đảm bảo những điều sau KHÔNG thay đổi sau refactor:
- [ ] API response format giữ nguyên
- [ ] Business logic giữ nguyên
- [ ] Existing tests pass 100%

## Verification
- [ ] Toàn bộ existing tests pass sau refactor
- [ ] Không có new linter errors
- [ ] Performance không tệ hơn (benchmark comparison)
```

---

## LOẠI 4: Technical Spec (`TECH`)

**Khi nào tạo:** Upgrade dependency, migration database, thay đổi infrastructure config, thay đổi tooling.

### Template TECH Spec
```markdown
---
id: TECH-[NNN]
title: [Mô tả thay đổi kỹ thuật]
service: [service-name]  # hoặc "all-services" nếu áp dụng toàn hệ thống
risk_level: Low | Medium | High
rollback_plan: [Mô tả cách rollback nếu thất bại]
---

## Mô Tả Thay Đổi
[Thay đổi kỹ thuật gì: upgrade từ X→Y, migrate từ A→B]

## Lý Do
[Breaking change bắt buộc / security patch / performance / EOL]

## Các Bước Thực Hiện
1. [Bước 1 — AI thực hiện]
2. [Bước 2]
3. [Bước 3]

## Risk & Mitigation
| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| [Risk 1] | High/Med/Low | High/Med/Low | [Cách giảm thiểu] |

## Rollback Plan
[Chi tiết cách rollback nếu thay đổi gây ra vấn đề]

## Verification Checklist
- [ ] Build thành công
- [ ] Toàn bộ tests pass
- [ ] Smoke test trên staging pass
- [ ] Performance không giảm
```

---

## LOẠI 5: Bug Fix Spec (`BUG`)

**Khi nào tạo:** Khi phát hiện bug trong production hoặc testing.

**AI đọc spec này để:** Hiểu root cause → fix đúng chỗ → viết regression test → không tạo ra bug mới.

### Template BUG Spec
```markdown
---
id: BUG-[NNN]
title: [Mô tả bug ngắn gọn]
service: [service-name]
severity: Critical | High | Medium | Low
reproducibility: Always | Intermittent | Rare
environment: Production | Staging | Dev
discovered_by: [QA / User / Monitoring]
created: YYYY-MM-DD
---

## Mô Tả Bug
[Mô tả rõ ràng: cái gì xảy ra sai]

## Steps To Reproduce
1. [Bước 1]
2. [Bước 2]
3. [Bước 3 — bug xuất hiện]

## Expected Behavior
[Hệ thống đáng lẽ phải làm gì]

## Actual Behavior
[Hệ thống thực sự làm gì sai]

## Evidence
- Console errors: [paste error message]
- Stack trace: [paste nếu có]
- Network request/response: [relevant data]

## Root Cause Analysis
### Hypothesis
[AI phân tích: tại sao bug xảy ra — layer nào, code nào]

### Root Cause (Confirmed)
[Sau khi điều tra: đây là nguyên nhân gốc]

### Affected Code
- File: `[path]`, Line: [N], Function: `[name]`

## Fix Plan
[Mô tả cách sửa — đủ cụ thể để AI implement]

## Regression Test
[Test case mới bắt buộc phải viết để đảm bảo bug không tái phát]

## Acceptance Criteria
- [ ] Bug không còn reproducible theo steps trên
- [ ] Regression test được viết và pass
- [ ] Các feature liên quan không bị ảnh hưởng
```

---

## LOẠI 6: Quality Spec (`QA`)

**Khi nào tạo:** Khi cần cải thiện chất lượng code, tăng test coverage, tối ưu performance, giảm technical debt.

### Template QA Spec
```markdown
---
id: QA-[NNN]
title: [Mô tả cải thiện chất lượng]
service: [service-name]
type: Coverage | Performance | Refactor | Debt-Reduction | Observability
---

## Vấn Đề Chất Lượng Hiện Tại
[Metric hiện tại: coverage 45%, latency 800ms, cyclomatic complexity 15...]

## Mục Tiêu Sau Cải Thiện
[Metric mục tiêu: coverage 80%, latency < 200ms, complexity < 10]

## Phạm Vi Công Việc
[AI cần làm gì cụ thể: thêm test cases cho module X, tối ưu query Y, extract method Z]

## Acceptance Criteria
- [ ] [Metric 1] đạt [target]
- [ ] [Metric 2] đạt [target]
- [ ] Không có behavior regression

## Không Được Làm
[Những gì AI không được thay đổi trong scope này]
```

---

## LOẠI 7: Security Spec (`SEC`)

**Khi nào tạo:** Phát hiện lỗ hổng bảo mật, kết quả security audit, CVE được publish cho dependency đang dùng.

### Template SEC Spec
```markdown
---
id: SEC-[NNN]
title: [Mô tả lỗ hổng / hardening]
service: [service-name]
severity: Critical | High | Medium | Low
cve: CVE-XXXX-XXXXX  # nếu có
disclosure: Private  # không publish chi tiết lỗ hổng trước khi fix
---

## Mô Tả Lỗ Hổng
[Mô tả lỗ hổng — đủ để AI hiểu nhưng không để lộ thông tin nhạy cảm]

## Attack Vector
[Attacker khai thác bằng cách nào]

## Impact
[Hậu quả nếu bị khai thác]

## Fix Plan
[Mô tả cách vá — cụ thể, actionable cho AI]

## Verification
- [ ] Lỗ hổng không còn có thể khai thác
- [ ] Security scan (gosec/npm audit) không còn báo lỗi này
- [ ] Regression tests pass
```

---

## Quy Tắc Đặc Biệt: Technical Design Documents

> Thực tế các service có file `specs/technical_design.md` rất lớn (32-78KB) chứa cả thiết kế tổng thể lẫn spec chi tiết. Đây là **ngoại lệ được chấp nhận** nhưng có ràng buộc rõ ràng.

### Lifecycle của `technical_design.md`

```
[Khởi tạo service]
        ↓
[Tạo technical_design.md] → Trạng thái: Living Design Doc
        ↓
[Bắt đầu sprint implement]
        ↓
[Bắt buộc: Tách thành spec files riêng]
   ARCH-001-*.md (kiến trúc)
   FEAT-001-*.md (feature 1)
   FEAT-002-*.md (feature 2)
        ↓
[technical_design.md → chuyển thành Reference Doc]
        ↓ (sau 1 sprint)
[Archived hoặc di chuyển vào docs/ nếu là tài liệu kiến trúc]
```

**Quy tắc:**
- ✅ `technical_design.md` được phép tồn tại trong `specs/` khi `status: Draft`
- ✅ Được dùng như nguồn để tạo spec files riêng
- ❌ Không được implement trực tiếp từ `technical_design.md` mà không có spec file riêng
- ❌ Sau khi specs riêng được tạo, `technical_design.md` không được cập nhật implementation details

**Tài liệu trong `specs/` bị nhầm lẫn là tài liệu (docs):**

| File thực tế | Hành động đúng |
|---|---|
| `specs/api_description.md` | → Di chuyển vào `docs/api.md` (DOC-S02) |
| `specs/service_spec.md` (>30KB) | → Tách: phần architecture → `docs/architecture.md`; phần specs → FEAT/ARCH files |
| `specs/external_services.md` | → Di chuyển vào `docs/` hoặc `docs/architecture.md` |
| `specs/services.md` | → Di chuyển vào `docs/README.md` (DOC-S01) |

---

## Quy Tắc Đặc Biệt: Frontend Package Specs

`apps/[name]/specs/` có một số khác biệt so với service specs:

**ĐƯỢC PHÉP:**
- `prd.md` / `urd.md`: Package-level requirements — chấp nhận, không cần đổi tên
  - Nhưng **PHẢI** có metadata header theo RULE-004
- `design/` subfolder: Chứa design specs và wireframe references
- Spec files theo naming convention chuẩn `[TYPE]-NNN-*.md`

**KHÔNG ĐƯỢC PHÉP:**
- Đặt tài liệu system-level (docs/) vào specs/
- Tạo spec file không có Acceptance Criteria

---

## Cross-Reference: Spec → Tài Liệu Repo-Level

> Trước khi tạo spec, AI agent **PHẢI đọc** các tài liệu repo-level tương ứng để hiểu context.

| Loại Spec | Đọc trước khi tạo |
|---|---|
| `FEAT` | `docs/product/prd.md` (DOC-P01) + pipeline flows liên quan (DOC-PL01) |
| `CR` | `docs/product/prd.md` + spec gốc (FEAT-NNN) |
| `ARCH` | `docs/product/architecture.md` (DOC-P02) + `docs/adr/` (DOC-P03) |
| `TECH` | `docs/standards/coding-standards.md` (DOC-P07) + `services/[name]/docs/configuration.md` |
| `BUG` | `services/[name]/docs/api.md` (DOC-S02) + `services/[name]/docs/architecture.md` |
| `QA` | `services/[name]/docs/test-plan.md` (DOC-Q01) |
| `SEC` | `docs/standards/security-policy.md` (DOC-P06) |

---

## Quy Trình Sử Dụng Specs Với AI

### 1. Vòng đời của một Spec

```
[Phát sinh nhu cầu]
        ↓
[Tạo Spec file] → status: Draft
        ↓
[Review & hoàn chỉnh Acceptance Criteria] → status: Ready
        ↓
[AI đọc Spec → Implement] → status: In Progress
        ↓
[AI tự kiểm tra Acceptance Criteria]
        ↓
[Human review / QA Agent verify] → status: Done
        ↓
[Cập nhật api.md, changelog.md nếu cần]
```

### 2. Hướng Dẫn AI Đọc Spec

Khi giao việc cho AI, cung cấp context theo thứ tự:
1. **Skill context:** `Đọc skill từ .agent/skills/[relevant-expert]/`
2. **Spec file:** `Thực hiện theo spec: services/[name]/specs/[TYPE]-NNN-title.md`
3. **Service context:** `Đọc architecture tại: services/[name]/docs/architecture.md`
4. **Standards:** `Tuân thủ: docs/standards/coding-standards.md`

### 3. AI Tự Kiểm Tra Chất Lượng

Sau khi implement, AI phải tự chạy checklist:
```
□ Mọi Acceptance Criteria đã được implement?
□ Unit tests đã viết cho logic mới?
□ Integration tests pass?
□ Linter/formatter không có lỗi?
□ API docs (api.md) đã cập nhật nếu có API change?
□ Changelog đã thêm entry mới?
□ Không có hardcoded secrets?
□ Error handling đủ không?
```

---

## Ví Dụ Cấu Trúc Thực Tế

```
services/requirement-parser/specs/
├── features/
│   ├── FEAT-001-document-ingestion-api.md        ← API nhận PRD upload
│   ├── FEAT-002-block-segmentation.md            ← Phân đoạn thành blocks
│   └── FEAT-003-block-classification-llm.md      ← Phân loại bằng LLM
├── changes/
│   └── CR-001-support-vietnamese-prd.md          ← Hỗ trợ PRD tiếng Việt
├── architecture/
│   └── ARCH-001-split-parser-extractor.md        ← Tách parser và extractor
├── technical/
│   └── TECH-001-upgrade-go-1.22.md               ← Upgrade Go version
├── bugs/
│   └── BUG-001-context-leak-on-timeout.md        ← Context không cancel khi timeout
├── quality/
│   └── QA-001-increase-test-coverage-to-80.md   ← Tăng coverage
└── security/
    └── SEC-001-add-request-size-limit.md         ← Giới hạn kích thước request
```

---

## Mapping: Loại Thay Đổi → Loại Spec

| Tình huống | Loại Spec | Ví dụ |
|---|---|---|
| PRD yêu cầu tính năng mới | `FEAT` | "Thêm API upload PRD" |
| Product Owner thay đổi business rule | `CR` | "Actor 'Viewer' giờ không được xóa" |
| Yêu cầu nghiệp vụ thay đổi flow | `CR` | "Approval flow thêm bước kiểm duyệt" |
| Refactor kiến trúc nội bộ | `ARCH` | "Tách handler → usecase → repo" |
| Thêm pattern mới (Circuit Breaker) | `ARCH` | "Wrap LLM calls với Circuit Breaker" |
| Upgrade dependency | `TECH` | "Upgrade neo4j-driver 5.x → 6.x" |
| Migrate database | `TECH` | "Migrate từ PostgreSQL sang Neo4j" |
| Bug production | `BUG` | "Goroutine leak khi LLM timeout" |
| Test coverage thấp | `QA` | "Coverage parser service < 40%" |
| Performance chậm | `QA` | "KG query > 2s" |
| CVE trong dependency | `SEC` | "CVE-2024-XXXX trong golang.org/x/net" |
| Kết quả security audit | `SEC` | "SQL injection trong search endpoint" |
