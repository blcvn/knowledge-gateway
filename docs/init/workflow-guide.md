---
version: 1.1.0
last_updated: 2026-04-22
updated_by: doc-management-expert
status: Approved
scope: REPO-LEVEL
source: docs/init/document-catalog.md, docs/init/specs-catalog.md
---

# Workflow Guide — Yêu Cầu → Phân Tích → Giải Pháp → Tác Vụ → Thực Thi

## VNP QA Platform

> **Mục tiêu:** Hướng dẫn quy trình chuẩn từ khi một yêu cầu thay đổi xuất hiện đến khi nó được hiện thực hoá thành code chạy được — với bước **thiết kế giải pháp kiến trúc** và **phân rã tác vụ có thể giám sát** là bắt buộc trước mọi việc viết code.
>
> **Đọc cùng với:** `document-catalog.md` (định nghĩa tài liệu) · `specs-catalog.md` (định nghĩa specs)

> **Nguyên tắc cốt lõi (v1.1):** Mọi thay đổi — dù nhỏ hay lớn — đều phải đi qua bước phân tích kiến trúc và thiết kế giải pháp trước khi tạo spec thực thi. AI không được implement trực tiếp từ yêu cầu mà không có Solution Spec (`SOL`) được duyệt.

---

## Tổng Quan Pipeline 4 Giai Đoạn

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│   💡 YÊU CẦU THAY ĐỔI                                                       │
│   (stakeholder, user feedback, bug, audit, PRD update, change request)       │
│                          │                                                   │
│                          ▼                                                   │
│   ┌──────────────────────────────────┐                                       │
│   │  GIAI ĐOẠN 1: PHÂN TÍCH TÁC ĐỘNG │  ← Đọc docs kiến trúc               │
│   │  "Thay đổi này ảnh hưởng gì?"    │    Xác định services/layers bị ảnh  │
│   │                                  │    hưởng, rủi ro, ràng buộc           │
│   └──────────────────┬───────────────┘                                       │
│                      │                                                       │
│                      ▼                                                       │
│   ┌──────────────────────────────────┐                                       │
│   │  GIAI ĐOẠN 2: THIẾT KẾ GIẢI PHÁP│  ← Tạo SOL spec                     │
│   │  "Giải quyết như thế nào?"       │    Phù hợp kiến trúc, rõ trade-offs  │
│   │  [Bắt buộc — kể cả thay đổi nhỏ]│    Phân rã thành tác vụ có thể track │
│   └──────────────────┬───────────────┘                                       │
│                      │                                                       │
│                      ▼                                                       │
│   ┌──────────────────────────────────┐                                       │
│   │  GIAI ĐOẠN 3: PHÂN RÃ TÁC VỤ    │  ← Tạo TASK/FEAT/BUG/FIX specs     │
│   │  "Làm gì, theo thứ tự nào?"      │    Mỗi task: 1 service, ≤1 ngày     │
│   │                                  │    Có AC rõ ràng để track            │
│   └──────────────────┬───────────────┘                                       │
│                      │                                                       │
│                      ▼                                                       │
│   ┌──────────────────────────────────┐                                       │
│   │  GIAI ĐOẠN 4: THỰC THI & GIÁM SÁT│  ← AI thực thi theo từng task      │
│   │  "Thực hiện và kiểm soát"         │    Human verify AC sau mỗi task     │
│   │                                  │    Cập nhật docs sau khi hoàn thành  │
│   └──────────────────────────────────┘                                       │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## GIAI ĐOẠN 1: Idea / Requirement → Docs

### 1.1 Mục Đích

> Docs mô tả **hệ thống là gì** — không phải "cần làm gì". Docs là nguồn chân lý tập trung (Single Source of Truth) để mọi người và AI agent hiểu context trước khi hành động.

### 1.2 Khi Nào Cần Cập Nhật Docs?

| Trigger | Tài liệu cần cập nhật | Ai cập nhật |
|---|---|---|
| Thêm tính năng mới vào PRD | `docs/product/prd_platform.md` (DOC-P01) | Product Owner |
| Thêm / xóa / thay đổi service | `docs/product/architecture.md` (DOC-P02) | Software Architect |
| Quyết định kiến trúc quan trọng | `docs/adr/ADR-XXXX.md` (DOC-P03) | Architect + Tech Lead |
| Thay đổi API contract | `services/[name]/docs/api.md` (DOC-S02) | Service Owner |
| Thay đổi database schema | `services/[name]/docs/data-model.md` (DOC-S04) | Service Owner |
| Thay đổi cấu hình / env var | `services/[name]/docs/configuration.md` (DOC-S05) | DevOps / Service Owner |
| Tạo service mới | Tất cả `services/[name]/docs/*.md` | Service Owner |

### 1.3 Quy Trình: Idea → Docs

```
[Ý tưởng / Yêu cầu phát sinh]
        │
        ▼
[Xác định loại thay đổi]
        │
        ├─ Thay đổi yêu cầu sản phẩm?
        │   └─ → Cập nhật DOC-P01 (PRD)
        │
        ├─ Thay đổi kiến trúc hệ thống?
        │   ├─ → Cập nhật DOC-P02 (Architecture)
        │   └─ → Tạo DOC-P03 (ADR) cho quyết định quan trọng
        │
        ├─ Thay đổi API của một service?
        │   └─ → Cập nhật DOC-S02 (api.md) của service đó
        │
        ├─ Tạo service hoàn toàn mới?
        │   └─ → Tạo đầy đủ docs/ cho service (DOC-S01 → DOC-S07)
        │
        └─ Bug / Lỗi vận hành?
            └─ → Cập nhật DOC-S06 (runbook.md) nếu cần
                 Tạo DOC-Q03 (bug report)
        │
        ▼
[Docs được cập nhật / tạo mới]
        │
        ▼
[→ Chuyển sang Giai Đoạn 2: Specs]
```

### 1.4 Cấu Trúc Docs Cần Tạo Cho Service Mới

```
services/[service-name]/docs/
├── README.md          ← DOC-S01: Mục đích, quick start, links
├── api.md             ← DOC-S02: Mọi endpoint, schema, ví dụ
├── architecture.md    ← DOC-S03: Layers, design decisions, diagrams
├── data-model.md      ← DOC-S04: Tables, ERD, migration history
├── configuration.md   ← DOC-S05: Mọi env var, type, default
├── runbook.md         ← DOC-S06: Startup/shutdown, alerts, escalation
└── changelog.md       ← DOC-S07: Keep a Changelog format
```

### 1.5 Ví Dụ Thực Tế

**Tình huống:** Product Owner yêu cầu thêm tính năng "AI tự động sinh test case từ requirement".

```
1. Product Owner cập nhật docs/product/prd_platform.md
   → Thêm mục: "AI Scenario Generation from Requirements"
   → Mô tả: actors, user story, acceptance criteria ở mức product

2. Software Architect cập nhật docs/product/architecture.md
   → Thêm service mới: ai-scenario-generator
   → Mô tả data flow từ requirement-service → ai-scenario-generator → test-case-service

3. Architect tạo docs/adr/ADR-0012-ai-scenario-generator-design.md
   → Ghi lại quyết định: dùng LLM streaming, BDD format, confidence threshold 0.85

4. Service Owner tạo services/ai-scenario-generator/docs/
   → README.md, api.md, architecture.md, data-model.md, ...
```

---

## GIAI ĐOẠN 2: Phân Tích & Thiết Kế Giải Pháp

### 2.1 Mục Đích

> **Bắt buộc với mọi thay đổi.** Trước khi viết bất kỳ spec thực thi nào, phải phân tích tác động lên kiến trúc hệ thống và thiết kế giải pháp phù hợp. Kết quả là một **Solution Spec (`SOL`)** tóm tắt giải pháp và danh sách tác vụ cần thực hiện.

### 2.2 Quy Trình Phân Tích Tác Động

```
[Yêu cầu thay đổi nhận được]
        │
        ▼
[Đọc docs kiến trúc liên quan]
   ├── docs/product/architecture.md   ← Tổng quan hệ thống
   ├── services/[name]/docs/architecture.md ← Service bị ảnh hưởng
   └── docs/adr/                      ← Quyết định kiến trúc đã có
        │
        ▼
[Phân tích tác động]
   □ Service nào bị ảnh hưởng?
   □ API contract thay đổi không? (breaking change?)
   □ Database schema thay đổi không?
   □ Có phụ thuộc ngược chiều (downstream consumers)?
   □ Rủi ro gì nếu triển khai sai thứ tự?
   □ Cần migration data không?
        │
        ▼
[Thiết kế giải pháp phù hợp kiến trúc]
   □ Approach nào phù hợp với pattern hiện tại?
   □ Trade-offs: đơn giản vs đúng kiến trúc?
   □ Phạm vi thay đổi: 1 service hay nhiều?
        │
        ▼
[Tạo SOL spec] → services/[name]/specs/solutions/SOL-NNN-*.md
        │
        ▼
[Phân rã thành danh sách TASK]
   → Chuyển sang Giai Đoạn 3
```

### 2.3 Template SOL Spec

```markdown
---
id: SOL-[NNN]
title: [Mô tả giải pháp]
service: [service-name] | cross-service
version: 1.0.0
status: Draft | Approved | Rejected
priority: P0 | P1 | P2 | P3
created: YYYY-MM-DD
updated: YYYY-MM-DD
linked_cr: [CR/FEAT/BUG mà SOL này giải quyết]
approved_by: [Architect / Tech Lead]
---

## Yêu Cầu Gốc
[Mô tả ngắn gọn yêu cầu thay đổi ban đầu]

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng
| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| [service-a] | API change | Cao |
| [service-b] | Config update | Thấp |

### Breaking Changes
- [ ] API response format thay đổi?
- [ ] Database schema migration cần thiết?
- [ ] Consumer downstream cần cập nhật?

### Ràng Buộc Kiến Trúc
[Những gì KHÔNG được phép thay đổi — pattern hiện tại, interface đã ký kết, ...]

## Giải Pháp Đề Xuất

### Approach
[Mô tả giải pháp: pattern nào dùng, layers nào thay đổi, tại sao đây là lựa chọn tốt nhất]

### Alternatives Đã Xem Xét
| Alternative | Lý do loại bỏ |
|---|---|
| [Phương án A] | [Lý do] |
| [Phương án B] | [Lý do] |

### Trade-offs
- **Ưu điểm:** [...]
- **Nhược điểm / Rủi ro:** [...]

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)
```
Task 1: [Tên] ← Phải làm trước (không phụ thuộc)
Task 2: [Tên] ← Sau Task 1
Task 3: [Tên] ← Sau Task 1, song song Task 2
Task 4: [Tên] ← Sau Task 2 + Task 3
```

### Danh Sách Tác Vụ
| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | [Tên] | TASK/FEAT/BUG/FIX | [service] | - | 4h |
| T02 | [Tên] | TASK | [service] | T01 | 2h |
| T03 | [Tên] | TASK | [service] | T01 | 3h |

### Rollback Plan
[Nếu triển khai thất bại: cách rollback từng task]

## Acceptance Criteria (Solution Level)
- [ ] SOL-AC-1: Tất cả tasks trong danh sách hoàn thành
- [ ] SOL-AC-2: Không có regression trên features liên quan
- [ ] SOL-AC-3: Docs được cập nhật phản ánh thay đổi
```

### 2.4 Ví Dụ: SOL Spec Thực Tế

**Tình huống:** Endpoint `/system/task-center/organization/options` trả về 404.

```markdown
---
id: SOL-001
title: Fix missing task-center organization routes in system-setting
service: system-setting
linked_cr: BUG-SS-16, BUG-SS-17
status: Approved
priority: P1
---

## Phân Tích Tác Động
- Services bị ảnh hưởng: ms-system-setting (handler, usecase, repository)
- Không có breaking change (thêm route mới, không sửa route cũ)
- ms-gateway không cần thay đổi (routing đã đúng)

## Giải Pháp
Thêm 2 GET routes + handler methods + usecase/repo calls theo pattern
handler→usecase→repository hiện tại của service.

## Danh Sách Tác Vụ
| ID | Task | Loại | Phụ thuộc |
|---|---|---|---|
| T01 | Add SystemOrganizationOptions handler | TASK | - |
| T02 | Add OrgOrganizationOptions handler | TASK | - |
| T03 | Register routes in router.go | TASK | T01, T02 |
| T04 | Build & hot-deploy binary | TASK | T03 |
```

---

## \ud83d\udc1e Luồng Đặc Biệt: Bug Handling Flow

> **Tại sao bug cần luồng riêng?** Bug yêu cầu điều tra và xác nhận root cause TRƯỚC khi thiết kế giải pháp. Thiếu bước này dẫn đến fix sai chỗ hoặc bỏ sót các lỗi tương tự trong hệ thống.

### Bug Pipeline (6 Bước)

```
┌────────────────────────────────────────────────────────────────────────────┐
│                                                                            │
│   🐛 BUG ĐƯỢC GHI NHẬN                                                    │
│   (production alert, user report, QA, code review)                        │
│                         │                                                  │
│                         ▼                                                  │
│   ┌──────────────────────────────┐                                         │
│   │  BƯỚC 1: GHI NHẬN BUG       │  → Tạo BUG spec (status: Draft)        │
│   │  Record bug report          │    Mô tả triệu chứng, môi trường        │
│   │                             │    Steps to reproduce                   │
│   └──────────────┬──────────────┘                                         │
│                  │                                                         │
│                  ▼                                                         │
│   ┌──────────────────────────────┐                                         │
│   │  BƯỚC 2: ĐIỀU TRA & RCA     │  → Phân tích log, trace, code          │
│   │  Investigate + Root Cause   │    Xác nhận root cause chính xác       │
│   │                             │    Ghi vào BUG spec: "Root Cause        │
│   │                             │    (Confirmed)" + Affected Code         │
│   └──────────────┬──────────────┘                                         │
│                  │                                                         │
│                  ▼                                                         │
│   ┌──────────────────────────────┐                                         │
│   │  BƯỚC 3: SYSTEM-WIDE SCAN   │  → Tìm pattern tương tự toàn hệ thống  │
│   │  Tìm lỗi tương tự           │    Kiểm tra tất cả services có cùng    │
│   │                             │    code pattern, query, config          │
│   │                             │    Ghi kết quả vào BUG spec            │
│   └──────────────┬──────────────┘                                         │
│                  │                                                         │
│                  ▼                                                         │
│   ┌──────────────────────────────┐                                         │
│   │  BƯỚC 4: SOL - GIẢI PHÁP    │  → Tạo SOL spec (cross-service nếu    │
│   │  Comprehensive Solution     │    tìm thấy nhiều nơi bị ảnh hưởng)    │
│   │                             │    Fix đồng bộ, không patch lẻ tẻ      │
│   └──────────────┬──────────────┘                                         │
│                  │                                                         │
│                  ▼                                                         │
│   ┌──────────────────────────────┐                                         │
│   │  BƯỚC 5: PHÂN RÃ TÁC VỤ    │  → BUG/FIX/TASK specs từ SOL          │
│   │  Task decomposition         │    Mỗi task: 1 service, rõ AC          │
│   └──────────────┬──────────────┘                                         │
│                  │                                                         │
│                  ▼                                                         │
│   ┌──────────────────────────────┐                                         │
│   │  BƯỚC 6: THỰC THI & STATUS  │  → AI implement từng task              │
│   │  Execute + Update Status    │    Human verify AC                      │
│   │                             │    Cập nhật SOL task status sau mỗi    │
│   │                             │    task hoàn thành                      │
│   └──────────────────────────────┘                                         │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### Checklist Điều Tra Bug (Bước 2)

```
□ Đọc logs của service bị ảnh hưởng (error, stack trace)
□ Kiểm tra request/response trên gateway logs
□ Xem xét code path từ handler → usecase → repository
□ Xác nhận: bug xuất hiện ở layer nào?
□ Reproduce được trên môi trường local/staging?
□ Root cause: file, function, line number cụ thể
□ Cập nhật BUG spec: "Root Cause (Confirmed)" + "Affected Code"
```

### System-wide Similar Bug Scan (Bước 3)

```
□ Tìm cùng pattern code trong tất cả services
   grep -r "[pattern]" services/
□ Kiểm tra services dùng chung library/util bị ảnh hưởng
□ Xem xét config tương tự (docker-compose, env vars)
□ Kiểm tra API routes có cùng issue (missing route, wrong method)
□ Ghi kết quả vào BUG spec: "Similar Bugs Found"
   - Nếu tìm thấy: mở rộng scope SOL thành cross-service
   - Nếu không tìm thấy: SOL chỉ scope 1 service
```

### Ví Dụ: Bug Flow Thực Tế

**Tình huống:** `/system/task-center/organization/options` → 404

```
Bước 1: Ghi nhận
  → BUG spec: BUG-SS-16
    Steps: GET /system/task-center/organization/options → 404
    Environment: production b1.openledger.vn

Bước 2: Điều tra & RCA
  → Kiểm tra GIN-debug logs: route không được register
  → Root cause: handler method SystemOrganizationOptions không có trong router.go
  → Affected code: services/system-setting/internal/handler/router.go:L364-370

Bước 3: System-wide scan
  → Grep tất cả services: "task-center/organization/options"
  → Kết quả: chỉ ms-system-setting bị ảnh hưởng
  → BUG-SS-17: /organization/task-center/organization/options cũng missing

Bước 4: SOL
  → SOL-SS-001: Fix missing organization options routes
  → Scope: ms-system-setting (handler + router + usecase + repo)
  → Không cross-service

Bước 5: Tasks
  T01 - Add SystemOrganizationOptions handler
  T02 - Add OrgOrganizationOptions handler
  T03 - Register routes in router.go
  T04 - Rebuild & hot-deploy binary

Bước 6: Execute & Status
  T01: ✅ Done (verified: 401 on direct call = route exists)
  T02: ✅ Done
  T03: ✅ Done
  T04: ✅ Done → SOL: Done
```

---

## GIAI ĐOẠN 3: Phân Rã Tác Vụ → Specs

### 3.1 Mục Đích

> Sau khi SOL spec được duyệt, mỗi task trong danh sách `Danh Sách Tác Vụ` của SOL được hiện thực hóa thành một spec thực thi riêng. Mỗi spec giải quyết đúng **một vấn đề** trong **một service/package**.

### 3.2 Loại Spec Cần Tạo

| Tình huống | Loại Spec | Thư mục |
|---|---|---|
| Tính năng mới từ PRD | `FEAT` | `specs/features/` |
| Business rule thay đổi | `CR` | `specs/changes/` |
| Refactor kiến trúc nội bộ | `ARCH` | `specs/architecture/` |
| Upgrade dependency / migrate DB | `TECH` | `specs/technical/` |
| Bug production / staging | `BUG` | `specs/bugs/` |
| Tăng coverage, tối ưu perf | `QA` | `specs/quality/` |
| Lỗ hổng bảo mật / CVE | `SEC` | `specs/security/` |
| Phân rã thành bước nhỏ hơn | `TASK` | `specs/tasks/` |
| Patch nhanh, hotfix | `FIX` | `specs/fixes/` |

### 3.3 Quy Trình: SOL → Specs

```
[SOL spec được Approved]
        │
        ▼
[Lấy danh sách tác vụ từ SOL."Danh Sách Tác Vụ"]
   ← Thực hiện theo đúng thứ tự phụ thuộc đã xác định
        │
        ▼
[Với mỗi task → tạo spec file tại đúng thư mục]
   Tên file: [TYPE]-[NNN]-[short-kebab-title].md
         hoặc [TYPE]-[SCOPE]-[NNN]-[short-kebab-title].md
        │
        ▼
[Điền đầy đủ YAML frontmatter]
   id, title, service/package, version, status: Draft
   priority, created, updated, linked_sol: SOL-NNN
        │
        ▼
[Viết nội dung spec theo template loại tương ứng]
   Bắt buộc: Acceptance Criteria (Given/When/Then)
        │
        ▼
[Review & hoàn chỉnh]
   status: Draft → Ready
        │
        ▼
[→ Chuyển sang Giai Đoạn 4: Thực Thi]
```

### 3.4 Docs Cần Đọc Trước Khi Tạo Spec

| Loại Spec | Đọc trước |
|---|---|
| `FEAT` | `docs/product/prd_platform.md` + `services/[name]/docs/architecture.md` |
| `CR` | `docs/product/prd_platform.md` + spec gốc `FEAT-NNN` |
| `ARCH` | `docs/product/architecture.md` + `docs/adr/` |
| `TECH` | `docs/standards/coding-standards.md` + `services/[name]/docs/configuration.md` |
| `BUG` | `services/[name]/docs/api.md` + `services/[name]/docs/architecture.md` |
| `QA` | `services/[name]/docs/test-plan.md` |
| `SEC` | `docs/standards/security-policy.md` |

### 3.5 Ví Dụ: Tạo FEAT Spec

**Tình huống tiếp theo** từ ví dụ trên — ai-scenario-generator cần API endpoint.

```markdown
# File: services/ai-scenario-generator/specs/features/FEAT-001-generate-scenarios-from-requirement.md

---
id: FEAT-001
title: Generate Test Scenarios From Requirement
service: ai-scenario-generator
version: 1.0.0
status: Ready
priority: P1
created: 2026-04-22
updated: 2026-04-22
linked_prd: Section 4.2 — AI Scenario Generation
linked_adr: ADR-0012
---

## Mục Tiêu
API cho phép gửi requirement text → nhận về danh sách test scenarios dạng BDD.

## Scope
### In Scope
- POST /api/v1/scenarios/generate
- Trả về tối đa 10 scenarios, mỗi scenario có Given/When/Then
- Confidence score cho mỗi scenario (0.0–1.0)

### Out of Scope
- Lưu scenarios vào database (spec riêng)
- UI rendering (spec riêng)

## Thiết Kế Kỹ Thuật
### API Contract
POST /api/v1/scenarios/generate
Request: { "requirement_id": "uuid", "requirement_text": "string" }
Response: { "scenarios": [...], "processing_time_ms": number }

### Business Logic
1. Validate requirement_text length (50–5000 chars)
2. Call LLM với prompt template từ prompt-registry
3. Parse response thành BDD format
4. Filter scenarios có confidence < 0.6
5. Return sorted by confidence desc

## Acceptance Criteria
- [ ] AC-1: Given valid requirement text (≥50 chars), When POST /generate, Then return ≥1 scenario
- [ ] AC-2: Given invalid text (<50 chars), When POST /generate, Then return 400 với error message
- [ ] AC-3: Given LLM timeout >10s, When POST /generate, Then return 503 với retry hint

## Test Requirements
- Unit tests: LLM response parser, BDD formatter
- Integration tests: end-to-end với mock LLM
- Minimum coverage: 80%
```

### 3.6 Ví Dụ: Phân Rã SOL → TASK

Khi FEAT quá lớn, tách thành TASK nhỏ:

```
FEAT-001-generate-scenarios-from-requirement.md
    └── tasks/
        ├── TASK-001-setup-handler-router.md        ← Handler layer
        ├── TASK-002-implement-llm-client.md         ← LLM integration
        ├── TASK-003-implement-bdd-parser.md         ← Parsing logic
        ├── TASK-004-write-unit-tests.md             ← Tests
        └── TASK-005-write-integration-tests.md      ← E2E tests
```

---

## GIAI ĐOẠN 4: Thực Thi & Giám Sát

### 4.1 Mục Đích

> AI agent đọc spec → implement → tự kiểm tra Acceptance Criteria → báo cáo kết quả. Human verify từng task. Tiến độ được track qua task status trong SOL spec.

### 4.2 Hướng Dẫn Giao Việc Cho AI Agent

Khi giao việc cho AI agent, cung cấp context theo thứ tự:

```
1. SOLUTION CONTEXT
   "Giải pháp: services/[name]/specs/solutions/SOL-NNN-*.md"

2. SKILL CONTEXT
   "Kỹ năng: .agents/.skills/[relevant-expert]/"

3. SPEC FILE
   "Thực hiện theo spec: services/[name]/specs/[type]/TASK-NNN-*.md"

4. SERVICE CONTEXT
   "Đọc architecture tại: services/[name]/docs/architecture.md"

5. STANDARDS
   "Tuân thủ: docs/standards/coding-standards.md"
```

### 4.3 Quy Trình: Spec → Code → Verify

```
[Task Spec status: Ready]
        │
        ▼
[AI agent đọc skill context + SOL context]
        │
        ▼
[AI agent đọc service docs + task spec]
   Scope • Acceptance Criteria • Business Logic
        │
        ▼
[AI implement]
   spec status → In Progress
   ├── Viết code theo business logic
   ├── Viết unit tests
   └── Viết integration tests
        │
        ▼
[AI tự kiểm tra AC]
   □ Mọi AC đã được implement?
   □ Unit tests pass, coverage ≥ 80%?
   □ Integration tests pass?
   □ Linter không có lỗi?
   □ Không có hardcoded secrets?
   □ Error handling đủ không?
        │
        ▼
[AI cập nhật docs]
   ├── services/[name]/docs/api.md (nếu có API change)
   └── services/[name]/docs/changelog.md
        │
        ▼
[spec status → Done • Cập nhật SOL task list]
        │
        ▼
[Human verify AC • Test lại endpoint/feature]
        │
        ▼
[Tiếp theo task kế tiếp trong SOL (theo thứ tự phụ thuộc)]
        │
        ▼
[Khi tất cả tasks Done → SOL status → Done → Merge → Production]
```

### 4.4 Giám Sát Tiến Độ (Progress Monitoring)

Sau khi deploy SOL, cập nhật bảng task trong SOL spec theo mẫ dưới:

```markdown
### Trạng Thái Thực Thi
| ID | Task | Status | Assigned | Verify | Ghi chú |
|---|---|---|---|---|---|
| T01 | Add SystemOrganizationOptions | ✅ Done | AI | Human ✓ | 2026-04-22 |
| T02 | Add OrgOrganizationOptions | ✅ Done | AI | Human ✓ | 2026-04-22 |
| T03 | Register routes in router.go | ✅ Done | AI | Human ✓ | 2026-04-22 |
| T04 | Build & deploy binary | ✅ Done | AI | Human ✓ | 2026-04-22 |
| T05 | End-to-end test on production | ⏳ In Progress | Human | - | |
```

**Ký hiệu trạng thái task:**
- ⏳ `Draft` — Chưa bắt đầu  
- 🔄 `In Progress` — Đang thực hiện
- ✅ `Done` — Xong, đã verify
- ❌ `Blocked` — Bị chặn bởi phụ thuộc
- ⚠️ `Failed` — AC không pass, cần xử lý

### 4.5 Vòng Đời Đầy Đủ Của Một SOL

```
SOL: Draft → Approved
                │
                ▼ (tạo task specs)
TASK-T01: Draft → Ready → In Progress → Done ✔
TASK-T02: Draft → Ready → In Progress → Done ✔
TASK-T03: Draft → Ready → In Progress → Done ✔
                │
                ▼ (khi tất cả tasks Done)
SOL: Done → Merge → Cập nhật api.md + changelog.md
```

---

## Tóm Tắt Nhanh: Bảng Quyết Định

| Câu hỏi | Câu trả lời | Hành động |
|---|---|---|
| Nhận được bất kỳ yêu cầu nào? | **Luôn bắt đầu bằng SOL** | Giai đoạn 2: Tạo SOL spec |
| SOL cần tạo FEAT spec? | Tính năng mới hoàn toàn | `FEAT` trong `specs/features/` |
| SOL cần thay đổi business rule? | Behavior hiện tại cần sửa | `CR` trong `specs/changes/` |
| SOL cần refactor nội bộ? | Không thay đổi behavior | `ARCH` trong `specs/architecture/` |
| SOL cần upgrade / migrate? | Đổi dep hoặc DB | `TECH` trong `specs/technical/` |
| SOL giải quyết bug? | Đọc api.md + architecture.md | `BUG` trong `specs/bugs/` |
| Task trong SOL quá lớn? | Tách nhỏ ≤1 ngày/task | Phân rã thêm `TASK` |
| Hotfix khẩn, riêng biệt? | Ít hơn 4h, không cần SOL | `FIX` spec, note lại lý do |
| Có lỗ hổng bảo mật? | Đọc security-policy.md trước | SOL → `SEC` spec |

---

## Anti-Patterns Cần Tránh

| ❌ Sai | ✅ Đúng |
|---|---|
| Implement trực tiếp từ yêu cầu | Luôn qua SOL spec trước |
| Tạo spec mà không có SOL được duyệt | SOL → task specs → code |
| SOL không có thứ tự phụ thuộc | Mọi SOL phải có Dependency Order rõ ràng |
| Viết code mà không có spec | Luôn có spec trước khi code |
| Spec quá lớn (>1 service, >1 vấn đề) | 1 spec = 1 service = 1 vấn đề |
| Spec không có Acceptance Criteria | Mọi spec phải có AC (Given/When/Then) |
| AI implement từ `tdd.md` trực tiếp | Tách `tdd.md` → spec files riêng trước |
| Không cập nhật task status trong SOL | Cập nhật trạng thái sau mỗi task Done |
| Docs đặt trong `specs/` | Docs → `docs/`, Specs → `specs/` |
| Spec không có YAML header | Mọi spec phải có YAML frontmatter |

---

## Quick Reference: Tên File & Thư Mục

```
services/[name]/specs/
├── solutions/    SOL-001-ten-giai-phap.md     ← Bắt buộc truớc tiên
├── features/     FEAT-001-ten-tinh-nang.md
├── changes/      CR-001-ten-thay-doi.md
├── architecture/ ARCH-001-ten-refactor.md
├── technical/    TECH-001-ten-upgrade.md
├── bugs/         BUG-001-ten-bug.md
│                 BUG-GW-01-ten-bug-co-scope.md
├── quality/      QA-001-ten-cai-thien.md
├── security/     SEC-001-ten-lo-hong.md
├── tasks/        TASK-001-ten-task.md
│                 TASK-FE-001-ten-task-co-scope.md
├── fixes/        FIX-001-ten-patch.md
├── testcases/    test-scenario-*.md
└── tdd.md        ← Technical Design Document (alias: technical_design.md)
```

---

## Xem Thêm

- **Document Catalog:** [`docs/init/document-catalog.md`](document-catalog.md) — Định nghĩa tất cả loại tài liệu
- **Specs Catalog:** [`docs/init/specs-catalog.md`](specs-catalog.md) — Định nghĩa 9 loại spec, templates, naming convention
- **Skills Catalog:** [`docs/init/skills-catalog.md`](skills-catalog.md) — Bộ kỹ năng AI agent cần đọc
- **Migration Tool:** [`script/migrate-specs.py`](../../script/migrate-specs.py) — Tự động chuẩn hoá cấu trúc specs
