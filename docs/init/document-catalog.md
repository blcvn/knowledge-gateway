---
version: 1.2.0
last_updated: 2026-04-21
updated_by: doc-management-expert
status: Approved
scope: REPO-LEVEL
---

# Danh Mục Tài Liệu Quản Trị — Cấp Repo
## Requirement-to-UI Automation Platform

> **Phạm vi:** Tài liệu này quản lý **tài liệu cấp Repo** — các tài liệu áp dụng cho toàn bộ monorepo, toàn bộ team, và toàn bộ sản phẩm. Đây là nguồn chân lý tập trung (Single Source of Truth) cho kiến trúc, chuẩn kỹ thuật, quyết định thiết kế, và quy trình vận hành.
>
> **Phân biệt với Specs:** Tài liệu cấp Repo mô tả *hệ thống là gì và hoạt động như thế nào*. Specs (quản lý ở cấp package/service) mô tả *cần làm gì và làm như thế nào* để thay đổi hệ thống.

---

## Phạm Vi & Phân Cấp Tài Liệu

```
[REPO-LEVEL — Tài liệu quản lý tại đây]
knowledge-gateway/
│
├── docs/                              ← TẤT CẢ tài liệu cấp repo nằm ở đây
│   ├── product/                       # Yêu cầu sản phẩm, kiến trúc tổng thể
│   │   ├── prd.md                     # DOC-P01: Product Requirement Document
│   │   ├── architecture.md            # DOC-P02: System architecture overview
│   │   ├── workflow.md                # DOC-PL04: Workflow overview
│   │   ├── generation_algorithm.md    # DOC-PL02: Generation algorithm
│   │   ├── pipeline_flow[N]_*.md      # DOC-PL01: Pipeline flow per stage
│   │   ├── dev-guide.md               # DOC-PL05: Developer setup guide
│   │   ├── dev-demo-guide.md          # DOC-PL05: Demo developer guide
│   │   ├── domain/                    # DOC-PL03: Domain schema & ontology
│   │   │   ├── business.yaml
│   │   │   ├── schema.yaml
│   │   │   └── schema_diff.md
│   │   ├── design/                    # DOC-PL06: Design algorithms
│   │   ├── ui/                        # DOC-PL07: UI generation approaches
│   │   ├── ux/, runtime-ui/, specs-ui/, spec-preview/, figma/
│   │   └── design-system/
│   ├── adr/                           # DOC-P03: Architecture Decision Records
│   │   └── ADR-XXXX-[title].md
│   ├── standards/                     # Chuẩn kỹ thuật toàn repo
│   │   ├── api-conventions.md         # DOC-P05
│   │   ├── coding-standards.md        # DOC-P07
│   │   ├── data-glossary.md           # DOC-P04
│   │   └── security-policy.md         # DOC-P06
│   ├── releases/                      # DOC-P08: Release notes
│   │   └── vX.Y.Z.md
│   ├── audits/                        # DOC-G05: Audit reports
│   │   └── audit-YYYY-MM-DD.md
│   ├── approaches/                    # DOC-R01: Research & approach docs
│   │   └── *.md
│   ├── init/                          # Catalog & governance docs
│   │   ├── document-catalog.md        ← File này
│   │   ├── specs-catalog.md
│   │   └── skills-catalog.md
│   └── execute/                       # Tài liệu điều hành & kế hoạch
│       ├── agent-catalog.md           # DOC-G01
│       ├── sprint-[N]-plan.md         # DOC-G03
│       └── tech-debt.md               # DOC-G04
│
├── .agent/                            # Agent skills, rules, workflows
│
[PACKAGE/SERVICE-LEVEL — Specs quản lý tại đây — xem specs-catalog.md]
│
├── kgs-platform/                             # kgs-platform packages
│   └── specs/                                # Specs (xem specs-catalog.md)
└── services/*                                # Backend services
    └── specs/                                # Specs (xem specs-catalog.md)

> **Quy tắc vàng:**
> - **Tài liệu** (docs/) = mô tả hệ thống, cập nhật khi hệ thống thay đổi
> - **Specs** (kgs-platform/*/specs, services/*/specs) = chỉ thị cho AI thực thi thay đổi

---

## PHẦN A — Tài Liệu Cấp Sản Phẩm (Product-Level)

> Áp dụng toàn monorepo. Duy trì tập trung tại `docs/`. Mọi package và service phải tuân thủ. **Không đặt tài liệu loại này trong thư mục apps/ hoặc services/.**

---

### DOC-P01 · Product Requirement Document (PRD)
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/product/prd.md` |
| **Bắt buộc** | ✅ Có trước khi bắt đầu phát triển |
| **Chủ sở hữu** | Product Owner |
| **Cập nhật khi** | Thay đổi yêu cầu nghiệp vụ, thêm tính năng mới |
| **Audience** | Toàn bộ team (dev, design, QA, stakeholders) |

**Nội dung bắt buộc:**
- Vision & Goals của sản phẩm
- Target Personas (người dùng mục tiêu)
- Core Features & User Stories
- Non-functional Requirements (performance, security, scalability)
- Success Metrics (KPIs đo lường)
- Out of Scope (những gì sẽ không làm)

---

### DOC-P02 · System Architecture Overview
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/architecture/architecture.md` |
| **Bắt buộc** | ✅ Có trước khi bắt đầu phát triển |
| **Chủ sở hữu** | Software Architect |
| **Cập nhật khi** | Thêm/xóa service, thay đổi kiến trúc tổng thể |

**Nội dung bắt buộc:**
- System topology diagram (component & service map)
- Data flow diagram (luồng dữ liệu end-to-end)
- Inter-service communication patterns (REST/gRPC/Kafka)
- Technology stack & justification
- Scalability & availability strategy
- Deployment architecture (dev / staging / production)

---

### DOC-P03 · Architecture Decision Records (ADR)
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/adr/ADR-XXXX-[title].md` |
| **Bắt buộc** | ✅ Mỗi quyết định kiến trúc quan trọng |
| **Chủ sở hữu** | Software Architect + Team Lead |
| **Số lượng** | Không giới hạn — mỗi quyết định = 1 ADR |

**Khi nào tạo ADR:**
- Chọn công nghệ / framework / database
- Thay đổi pattern kiến trúc (e.g., chuyển từ REST sang gRPC)
- Quyết định có ảnh hưởng cross-service
- Migration hoặc breaking change lớn

**Template ADR:**
```markdown
# ADR-XXXX: [Tiêu đề]
- Date: YYYY-MM-DD
- Status: Proposed | Accepted | Deprecated | Superseded by ADR-XXXX
- Deciders: [Tên/vai trò]

## Context
## Considered Options
## Decision
## Consequences
```

---

### DOC-P04 · Data Model Glossary
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/standards/data-glossary.md` |
| **Bắt buộc** | ✅ Có trước khi thiết kế database |
| **Chủ sở hữu** | Software Architect + Tech Lead |
| **Cập nhật khi** | Thêm entity mới, thay đổi định nghĩa domain |

**Nội dung bắt buộc:**
- Canonical definitions cho mọi domain entity (Requirement, Entity, Actor, Field, Screen...)
- Naming conventions (camelCase vs snake_case, tiếng Anh là ngôn ngữ chuẩn)
- Relationship definitions giữa các entities
- Glossary thuật ngữ nghiệp vụ → thuật ngữ kỹ thuật

---

### DOC-P05 · API Standards & Conventions
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/standards/api-conventions.md` |
| **Bắt buộc** | ✅ Có trước khi xây dựng API đầu tiên |
| **Chủ sở hữu** | API Design Expert + Tech Lead |

**Nội dung bắt buộc:**
- URL naming convention (REST resource naming)
- HTTP status code usage guide
- Standard error response format (JSON schema)
- Authentication header conventions (Bearer token)
- API versioning strategy (`/v1/`, `/v2/`)
- Pagination standard (cursor-based vs offset)
- Request/Response naming conventions (camelCase vs snake_case)

---

### DOC-P06 · Security Policy
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/standards/security-policy.md` |
| **Bắt buộc** | ✅ Có trước khi deploy production |
| **Chủ sở hữu** | Security Engineer |

**Nội dung bắt buộc:**
- Authentication & Authorization standards (JWT, OAuth2, RBAC)
- Secrets management policy (không hardcode, dùng Vault/env)
- Input validation requirements
- Dependency vulnerability scanning requirements
- Security scanning tools bắt buộc trong CI/CD
- Incident response procedure
- Data classification & PII handling

---

### DOC-P07 · Coding Standards & Guidelines
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/standards/coding-standards.md` |
| **Bắt buộc** | ✅ Có trước sprint đầu tiên |
| **Chủ sở hữu** | Tech Lead + Senior Engineers |

**Nội dung bắt buộc:**
- Code style (linter config, formatter settings)
- Error handling conventions (Go: error wrapping; TS: Result types)
- Logging standards (structured logs, log levels, không log PII)
- Testing requirements (minimum coverage %, test types bắt buộc)
- Git workflow (branch naming, commit message format, PR checklist)
- Code review checklist

---

### DOC-P08 · Release Notes
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/releases/vX.Y.Z.md` |
| **Bắt buộc** | ✅ Trước mỗi release tag |
| **Chủ sở hữu** | Tech Lead / Release Manager |

**Nội dung bắt buộc:**
- Version number & release date
- Breaking changes (highlighted prominently)
- New features (Added)
- Bug fixes (Fixed)
- Performance improvements (Changed)
- Deprecated features
- Migration guide (nếu có breaking change)

---

## PHẦN B — Tài Liệu Cấp Service (Service-Level)

> Mỗi service/module độc lập phải duy trì bộ tài liệu riêng tại `services/[name]/docs/`.

---

### DOC-S01 · Service README
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/README.md` |
| **Bắt buộc** | ✅ Ngay khi tạo service (RULE-002) |
| **Chủ sở hữu** | Service Owner |

**Nội dung bắt buộc:**
- Service name & purpose (1 paragraph)
- Business capability owned
- Tech stack (language, framework, database)
- Links đến: `api.md`, `runbook.md`, `architecture.md`
- Owner / team contact
- Quick start (clone → run trong < 5 commands)

---

### DOC-S02 · API Reference
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/api.md` |
| **Bắt buộc** | ✅ Trước khi expose API (RULE-001) |
| **Cập nhật khi** | Thêm/sửa/xóa endpoint |

**Nội dung bắt buộc (mỗi endpoint):**
- Method + Path
- Authentication requirement
- Request body schema (với ví dụ)
- Response schema — mọi status code (200, 400, 401, 404, 500)
- Rate limiting (nếu có)
- Ví dụ curl request/response

---

### DOC-S03 · Service Architecture
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/architecture.md` |
| **Bắt buộc** | ✅ Ngay khi tạo service |
| **Cập nhật khi** | Thay đổi cấu trúc nội bộ |

**Nội dung bắt buộc:**
- Internal layer structure (e.g., handler → usecase → repository)
- Key design decisions & rationale
- External dependencies (third-party services, databases)
- Component diagram (text-based hoặc mermaid)
- Known limitations / technical debt

---

### DOC-S04 · Data Model
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/data-model.md` |
| **Bắt buộc** | ✅ Trước khi tạo database schema |
| **Cập nhật khi** | Database migration (RULE-008 analog) |

**Nội dung bắt buộc:**
- Mọi table/collection với fields, types, constraints
- Entity-relationship diagram (mermaid ERD)
- Index strategy
- Migration history summary

---

### DOC-S05 · Configuration Reference
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/configuration.md` |
| **Bắt buộc** | ✅ Trước khi deploy (RULE-008) |
| **Cập nhật khi** | Thêm environment variable mới |

**Nội dung bắt buộc (mỗi biến):**
- Variable name
- Description & purpose
- Type và valid values
- Default value
- Required / Optional
- Example

---

### DOC-S06 · Runbook (Operations Guide)
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/runbook.md` |
| **Bắt buộc** | ✅ Trước khi deploy production |
| **Audience** | SRE, DevOps, On-call engineers |

**Nội dung bắt buộc:**
- Startup / shutdown procedure
- Health check endpoints & expected responses
- Common error messages → diagnosis → resolution steps
- Deployment procedure & rollback procedure
- Monitoring dashboards links
- Alert runbook (mỗi alert → action cần thực hiện)
- Escalation contacts

---

### DOC-S07 · Service Changelog
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/changelog.md` |
| **Bắt buộc** | ✅ Khởi tạo khi tạo service; cập nhật mỗi release |
| **Format** | Keep a Changelog (keepachangelog.com) |

**Format chuẩn:**
```markdown
## [Unreleased]
### Added
### Changed
### Fixed

## [1.2.0] - 2026-04-20
### Added
- ...
```

---

## PHẦN C — Tài Liệu Quy Trình & Quản Trị

---

### DOC-G01 · Agent Catalog
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/execute/agent-catalog.md` |
| **Bắt buộc** | ✅ Đã có |
| **Cập nhật khi** | Thêm / thay đổi agent |

Xác định các agent AI cần thiết, skill sets, orchestration model, success criteria.

---

### DOC-G02 · Skill Set Catalog
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/init/skills-catalog.md` |
| **Bắt buộc** | ✅ Đã có |

Danh mục 16 bộ kỹ năng chuyên gia cần có để phát triển sản phẩm.

---

### DOC-G03 · Sprint Planning & Backlog
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/execute/sprint-[N]-plan.md` |
| **Bắt buộc** | ✅ Đầu mỗi sprint |
| **Chủ sở hữu** | Tech Lead / Scrum Master |

**Nội dung bắt buộc:**
- Sprint goal (1-2 câu)
- User stories theo thứ tự ưu tiên
- Acceptance criteria cho mỗi story
- Task breakdown với estimate
- Dependencies & blockers đã biết
- Definition of Done

---

### DOC-G04 · Technical Debt Register
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/execute/tech-debt.md` |
| **Bắt buộc** | ✅ Duy trì liên tục |
| **Cập nhật khi** | Phát hiện tech debt mới hoặc trả nợ xong |

**Mỗi mục tech debt:**
- ID, title, description
- Affected service/component
- Impact (High/Medium/Low)
- Estimated effort to fix
- Date identified
- Owner
- Status (Open / In Progress / Resolved)

---

### DOC-G05 · Documentation Audit Report
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/audits/audit-YYYY-MM-DD.md` |
| **Bắt buộc** | ✅ Trước mỗi release; đầu mỗi sprint |
| **Chủ sở hữu** | Documentation Management Agent |

Kết quả kiểm toán định kỳ: compliance rate, violations, stale references, action items.

---

## PHẦN D — Tài Liệu Chất Lượng & Testing

---

### DOC-Q01 · Test Plan
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/test-plan.md` |
| **Bắt buộc** | ✅ Trước khi bắt đầu QA sprint |
| **Chủ sở hữu** | QA Engineer |

**Nội dung bắt buộc:**
- Scope of testing (in-scope / out-of-scope)
- Test types: Unit, Integration, E2E, Performance, Security
- Test environments
- Entry / Exit criteria
- Risk areas & mitigation

---

### DOC-Q02 · Test Case Repository
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `services/[name]/docs/test-cases.md` |
| **Bắt buộc** | ✅ Trước khi test execution |
| **Format** | Given / When / Then (BDD) |

Bao gồm: Happy path, negative/sad path, boundary values, security test cases.

---

### DOC-Q03 · Bug Report
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/bugs/BUG-XXXX-[title].md` hoặc Issue tracker |
| **Bắt buộc** | ✅ Mỗi bug được phát hiện |

**Nội dung bắt buộc:**
- Severity, reproducibility, environment
- Steps to reproduce (numbered, exact)
- Expected vs actual result
- Screenshots / logs / network traces
- Root cause hypothesis

---

## Tóm Tắt — Checklist Theo Giai Đoạn Phát Triển

### Giai đoạn 1: Khởi động dự án
- [ ] DOC-P01 PRD ✅
- [ ] DOC-P02 System Architecture
- [ ] DOC-P04 Data Model Glossary
- [ ] DOC-P05 API Standards
- [ ] DOC-P06 Security Policy
- [ ] DOC-P07 Coding Standards

### Giai đoạn 2: Tạo mỗi Service mới
- [ ] DOC-S01 README ✅
- [ ] DOC-S02 API Reference
- [ ] DOC-S03 Service Architecture
- [ ] DOC-S04 Data Model
- [ ] DOC-S05 Configuration Reference
- [ ] DOC-S06 Runbook (stub → hoàn chỉnh trước production)
- [ ] DOC-S07 Changelog (khởi tạo)

### Giai đoạn 3: Mỗi Sprint
- [ ] DOC-G03 Sprint Plan
- [ ] DOC-G05 Documentation Audit
- [ ] DOC-G04 Tech Debt cập nhật

### Giai đoạn 4: Mỗi Release
- [ ] DOC-P03 ADR (cho mọi quyết định kiến trúc)
- [ ] DOC-P08 Release Notes
- [ ] DOC-Q01 Test Plan
- [ ] DOC-Q02 Test Cases
- [ ] DOC-G05 Audit Report

---

## PHẦN E — Tài Liệu Pipeline & Thuật Toán (Platform-Specific)

> Các tài liệu này đặc thù của nền tảng Requirement-to-UI — mô tả cách pipeline và thuật toán hoạt động.
> **Không đặt tài liệu loại này trong specs/ vì chúng là tài liệu sống (living docs), không phải chỉ thị thực thi.**

---

### DOC-PL01 · Pipeline Flow Documents
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/product/pipeline_flow[N]_[name].md` |
| **Bắt buộc** | ✅ Mỗi stage của pipeline phải có |
| **Chủ sở hữu** | Software Architect + Data Pipeline Expert |
| **Cập nhật khi** | Thay đổi luồng xử lý của bất kỳ stage nào |
| **Hiện tại** | ✅ 5 files (flow1, flow1a, flow1b, flow2, flow3) |

**Nội dung bắt buộc:**
- Input / Output contract của stage
- Sequence diagram (mermaid)
- Error handling & fallback strategy
- Integration points với stage trước/sau

---

### DOC-PL02 · Generation Algorithm
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/product/generation_algorithm.md` |
| **Bắt buộc** | ✅ Đã có (47KB) |
| **Chủ sở hữu** | Software Architect + LLM Engineer |
| **Cập nhật khi** | Thay đổi thuật toán sinh UI Schema từ KG |

---

### DOC-PL03 · Domain Schema & Ontology
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/product/domain/` |
| **Bắt buộc** | ✅ Đã có (`business.yaml`, `schema.yaml`, `schema_diff.md`) |
| **Chủ sở hữu** | Graph DB Expert + Software Architect |
| **Cập nhật khi** | Thay đổi ontology của Knowledge Graph (node types, relationships) |

> **Quan trọng:** Mọi thay đổi ontology phải kèm `schema_diff.md` cập nhật và ADR tương ứng.

---

### DOC-PL04 · Workflow Overview
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/product/workflow.md` |
| **Bắt buộc** | ✅ Đã có (16KB) |
| **Chủ sở hữu** | Product Owner + Software Architect |
| **Cập nhật khi** | Thay đổi luồng công việc tổng thể từ PRD đến UI |

---

### DOC-PL05 · Developer Guides
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/dev/dev-guide.md`, `docs/dev/dev-demo-guide.md` |
| **Bắt buộc** | ✅ Đã có |
| **Audience** | Frontend & backend developers mới join |
| **Cập nhật khi** | Thay đổi setup workflow, tool, hoặc pipeline execution |

---

### DOC-PL06 · Design Algorithms
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/product/design/` |
| **Bắt buộc** | ✅ Đã có (figma_to_design_algorithm.md, preview_to_design_algorithm.md) |
| **Chủ sở hữu** | UI/UX Design Expert |
| **Cập nhật khi** | Thay đổi thuật toán map từ preview sang Figma/design output |

---

### DOC-PL07 · UI Generation Approaches
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/product/ui/` |
| **Nội dung** | Các approach (approach_00, approach_03, approach_06...) — từng step algorithm |
| **Chủ sở hữu** | Software Architect + UI/UX Design Expert |
| **Cập nhật khi** | Thêm approach mới hoặc cập nhật step algorithm hiện có |

---

## PHẦN F — Tài Liệu Cấp App (Frontend Package-Level)

> Mỗi frontend app/package phải duy trì tài liệu tại `apps/[name]/docs/`.
> Nhẹ hơn service-level docs — không cần api.md, data-model.md, runbook.md.

---

### DOC-A01 · App README
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `apps/[name]/docs/README.md` |
| **Bắt buộc** | ✅ Ngay khi tạo app (RULE-010) |
| **Chủ sở hữu** | Frontend Lead |

**Nội dung bắt buộc:**
- App name & purpose
- Tech stack (framework, UI library, state management)
- Quick start (< 5 commands)
- Link đến architecture.md
- Danh sách shims/modules chứa trong app (đặc biệt cho apps/demo)

---

### DOC-A02 · App Architecture
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `apps/[name]/docs/architecture.md` |
| **Bắt buộc** | ✅ Ngay khi tạo app |
| **Cập nhật khi** | Thêm/xóa module, shim, thay đổi cấu trúc |

**Nội dung bắt buộc:**
- Module structure diagram
- Key design decisions
- Integration với backend services
- Build & deployment notes

---

### DOC-A03 · App Changelog
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `apps/[name]/docs/changelog.md` |
| **Bắt buộc** | ✅ Khởi tạo ngay; cập nhật mỗi release |
| **Format** | Keep a Changelog (keepachangelog.com) |

---

## PHẦN G — Tài Liệu Research & Tiếp Cận (Reference Only)

> Tài liệu nghiên cứu và khảo sát — **không cập nhật thường xuyên**, chỉ là tài liệu tham chiếu lịch sử.
> Không áp dụng RULE-005 (changelog) hay RULE-004 (metadata version tăng) cho nhóm này.

---

### DOC-R01 · Approach Research Documents
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/approaches/` |
| **Bắt buộc** | ⚪ Tùy chọn — tạo khi nghiên cứu approach mới |
| **Hiện tại** | ✅ 5 files (approaches.md, approach_to_ui.md, approache_rank.md, solutions.md, wireframe.md) |
| **Audience** | Tech leads, architects — không phải AI agent thực thi |

---

### DOC-R02 · Open Source Reference
| Thuộc tính | Giá trị |
|---|---|
| **Đường dẫn** | `docs/other/open_source.md` |
| **Bắt buộc** | ⚪ Tùy chọn |
| **Nội dung** | Danh sách open source libraries được xem xét |

---

## Tóm Tắt — Checklist Theo Giai Đoạn Phát Triển

### Giai đoạn 1: Khởi động dự án
- [ ] DOC-P01 PRD ✅
- [ ] DOC-P02 System Architecture
- [ ] DOC-P04 Data Model Glossary
- [ ] DOC-P05 API Standards
- [ ] DOC-P06 Security Policy
- [ ] DOC-P07 Coding Standards
- [ ] DOC-PL03 Domain Schema & Ontology ✅
- [ ] DOC-PL04 Workflow Overview ✅

### Giai đoạn 2: Mỗi Pipeline Stage mới
- [ ] DOC-PL01 Pipeline Flow Document
- [ ] DOC-PL02 Generation Algorithm (nếu có thuật toán)
- [ ] DOC-P03 ADR (nếu có quyết định kiến trúc)

### Giai đoạn 3: Tạo mỗi Service mới
- [ ] DOC-S01 README (RULE-002)
- [ ] DOC-S02 API Reference
- [ ] DOC-S03 Service Architecture
- [ ] DOC-S04 Data Model
- [ ] DOC-S05 Configuration Reference
- [ ] DOC-S06 Runbook (stub → hoàn chỉnh trước production)
- [ ] DOC-S07 Changelog

### Giai đoạn 4: Tạo mỗi App/Package mới
- [ ] DOC-A01 App README (RULE-010)
- [ ] DOC-A02 App Architecture
- [ ] DOC-A03 App Changelog

### Giai đoạn 5: Mỗi Sprint
- [ ] DOC-G03 Sprint Plan
- [ ] DOC-G05 Documentation Audit
- [ ] DOC-G04 Tech Debt cập nhật

### Giai đoạn 6: Mỗi Release
- [ ] DOC-P03 ADR (cho mọi quyết định kiến trúc)
- [ ] DOC-P08 Release Notes
- [ ] DOC-Q01 Test Plan
- [ ] DOC-Q02 Test Cases
- [ ] DOC-G05 Audit Report

---

## Trạng Thái Tài Liệu Hiện Tại

### Tài liệu Cấp Sản Phẩm
| Document | Đường dẫn | Trạng thái |
|---|---|---|
| PRD | `docs/product/prd.md` | ⚠️ Có nhưng thiếu metadata + sections |
| Agent Catalog | `docs/execute/agent-catalog.md` | ✅ Có |
| Skill Catalog | `docs/init/skills-catalog.md` | ✅ Có |
| Document Catalog | `docs/init/document-catalog.md` | ✅ File này |
| Specs Catalog | `docs/init/specs-catalog.md` | ✅ Có |
| System Architecture | `docs/product/architecture.md` | 🔴 Chưa có |
| API Standards | `docs/standards/api-conventions.md` | 🔴 Chưa có |
| Data Glossary | `docs/standards/data-glossary.md` | 🔴 Chưa có |
| Security Policy | `docs/standards/security-policy.md` | 🔴 Chưa có |
| Coding Standards | `docs/standards/coding-standards.md` | 🔴 Chưa có |
| ADR directory | `docs/adr/` | 🔴 Chưa có |
| Tech Debt Register | `docs/execute/tech-debt.md` | 🔴 Chưa có |

### Tài liệu Pipeline & Algorithm (Platform-Specific)
| Document | Đường dẫn | Trạng thái |
|---|---|---|
| Workflow Overview | `docs/product/workflow.md` | ✅ Có |
| Generation Algorithm | `docs/product/generation_algorithm.md` | ✅ Có |
| Pipeline Flow 1 | `docs/product/pipeline_flow1_data_to_preview.md` | ✅ Có |
| Pipeline Flow 1a | `docs/product/pipeline_flow1a_doc_to_kg.md` | ✅ Có |
| Pipeline Flow 1b | `docs/product/pipeline_flow1b_kg_to_preview.md` | ✅ Có |
| Pipeline Flow 2 | `docs/product/pipeline_flow2_preview_to_figma.md` | ✅ Có |
| Pipeline Flow 3 | `docs/product/pipeline_flow3_preview_to_code.md` | ✅ Có |
| Domain Schema | `docs/product/domain/` | ✅ Có |
| Design Algorithms | `docs/product/design/` | ✅ Có |
| UI Approaches | `docs/product/ui/` | ✅ Có |
| Dev Guide | `docs/product/dev-guide.md` | ✅ Có |
| Dev Demo Guide | `docs/product/dev-demo-guide.md` | ✅ Có |

### Tài liệu Cấp Service
| Service | README | API | Architecture | Config | Runbook | Changelog |
|---|---|---|---|---|---|---|
| chat-preview-service | ✅ (root+docs/) | 🔴 | 🔴 | 🔴 | 🔴 | 🔴 |
| doc_to_kg | ✅ (root+docs/) | 🔴 | 🔴 | 🔴 | 🔴 | 🔴 |
| knowledge-gateway | 🔴 | 🔴 | 🔴 | 🔴 | 🔴 | 🔴 |
| ui-knowledge-service | ✅ docs/ | ✅ docs/api.md | ✅ docs/ (Draft) | 🔴 | 🔴 | 🔴 |

### Tài liệu Cấp App
| App | README | Architecture | Changelog |
|---|---|---|---|
| apps/preview | ✅ docs/ | 🔴 | 🔴 |
| apps/demo | ✅ docs/ | 🔴 | 🔴 |
| apps/openui | 🔴 | 🔴 | 🔴 |
| apps/loveable | 🔴 | 🔴 | 🔴 |
