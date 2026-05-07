---
version: 1.0.0
created_at: 2026-04-24
scope: REPO-LEVEL
doc_id: DOC-AGENTS-INDEX
---

# Skills Index — Knowledge Gateway

> Thư mục tổng hợp toàn bộ skill sets của project. Mỗi skill tương ứng với một thư mục con chứa `skill.md`.

## Cấu trúc thư mục

```
.agents/
├── INDEX.md                         ← File này
└── skills/
    ├── software-architect/          → SKILL-001 (P1)
    │   └── skill.md
    ├── ui-ux-design-expert/         → SKILL-002 (existing)
    │   └── skill.md
    ├── api-design-expert/           → SKILL-003 (P2)
    │   └── skill.md
    ├── golang-expert/               → SKILL-004 (existing)
    │   └── skill.md
    ├── graph-db-expert/             → SKILL-005 (P0)
    │   └── skill.md
    ├── react-ts-expert/             → SKILL-006 + SKILL-007 (existing)
    │   ├── skill.md
    │   └── json-schema-rendering.md
    ├── llm-engineer/                → SKILL-008 (P0)
    │   └── skill.md
    ├── nlp-expert/                  → SKILL-009 (P0)
    │   └── skill.md
    ├── data-pipeline-expert/        → SKILL-010 (P1)
    │   └── skill.md
    ├── ui-testing-expert/           → SKILL-011 (existing)
    │   └── skill.md
    ├── golang-testing-expert/       → SKILL-012 (existing)
    │   └── skill.md
    ├── security-engineer/           → SKILL-013 (P2)
    │   └── skill.md
    ├── devops-expert/               → SKILL-014 (P3)
    │   └── skill.md
    ├── doc-management-expert/       → SKILL-015 (existing)
    │   └── skill.md
    └── ai-agent-architect/          → SKILL-016 (existing)
        └── skill.md
```

---

## Quick Reference

| ID | Skill | Path | Priority | Status |
|----|-------|------|----------|--------|
| SKILL-001 | Software Architecture & System Design | `skills/software-architect/` | 🟠 P1 | ✅ Created |
| SKILL-002 | UI/UX Design | `skills/ui-ux-design-expert/` | existing | ✅ Created |
| SKILL-003 | API Design & Integration Patterns | `skills/api-design-expert/` | 🟡 P2 | ✅ Created |
| SKILL-004 | Backend Development — Golang | `skills/golang-expert/` | existing | ✅ Created |
| SKILL-005 | Graph Database Engineering | `skills/graph-db-expert/` | 🔴 P0 | ✅ Created |
| SKILL-006 | Frontend Development — React + TypeScript | `skills/react-ts-expert/` | existing | ✅ Created |
| SKILL-007 | Dynamic UI Schema Rendering | `skills/react-ts-expert/json-schema-rendering.md` | existing | ✅ Created |
| SKILL-008 | LLM Engineering & Prompt Design | `skills/llm-engineer/` | 🔴 P0 | ✅ Created |
| SKILL-009 | Natural Language Processing (NLP) | `skills/nlp-expert/` | 🔴 P0 | ✅ Created |
| SKILL-010 | Data Pipeline Engineering | `skills/data-pipeline-expert/` | 🟠 P1 | ✅ Created |
| SKILL-011 | UI Testing & Automation | `skills/ui-testing-expert/` | existing | ✅ Created |
| SKILL-012 | Backend Testing — Golang | `skills/golang-testing-expert/` | existing | ✅ Created |
| SKILL-013 | Security Engineering & Hardening | `skills/security-engineer/` | 🟡 P2 | ✅ Created |
| SKILL-014 | DevOps, CI/CD & Infrastructure | `skills/devops-expert/` | 🟢 P3 | ✅ Created |
| SKILL-015 | Documentation Management & Governance | `skills/doc-management-expert/` | existing | ✅ Created |
| SKILL-016 | AI Agent Architecture & Orchestration | `skills/ai-agent-architect/` | existing | ✅ Created |

---

## Skill × Agent Matrix

| Skill | Parser | Extractor | KG | Schema Gen | Renderer | Traceability | Doc | QA |
|-------|:------:|:---------:|:--:|:----------:|:--------:|:------------:|:---:|:--:|
| SKILL-001 Arch Design | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| SKILL-002 UI/UX | | | | ✓ | ✓ | | | |
| SKILL-003 API Design | | | ✓ | ✓ | | | | ✓ |
| SKILL-004 Golang | ✓ | ✓ | ✓ | ✓ | | | | |
| SKILL-005 Graph DB | | | ✓ | | | ✓ | | |
| SKILL-006 React+TS | | | | ✓ | ✓ | | | |
| SKILL-007 JSON Schema | | | | ✓ | ✓ | | | |
| SKILL-008 LLM Eng | ✓ | ✓ | | | | | | |
| SKILL-009 NLP | ✓ | ✓ | | | | | | |
| SKILL-010 Data Pipeline | ✓ | ✓ | ✓ | ✓ | | | | |
| SKILL-011 UI Testing | | | | | | | | ✓ |
| SKILL-012 Go Testing | | | | | | ✓ | | ✓ |
| SKILL-013 Security | | | | | | | ✓ | ✓ |
| SKILL-014 DevOps | | | | | | | ✓ | ✓ |
| SKILL-015 Doc Mgmt | | | | | | | ✓ | |
| SKILL-016 AI Arch | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

---

## Priority Implementation Order

| Priority | Skill | Lý do |
|----------|-------|-------|
| 🔴 P0 | SKILL-008 LLM Engineering | Cốt lõi pipeline — không có LLM thì không có sản phẩm |
| 🔴 P0 | SKILL-005 Graph DB | Trái tim của Knowledge Graph layer |
| 🔴 P0 | SKILL-009 NLP | Tiền xử lý input cho LLM |
| 🟠 P1 | SKILL-001 Software Architecture | Thiết kế tổng thể — định hướng mọi service |
| 🟠 P1 | SKILL-010 Data Pipeline | Gắn kết các stages với nhau |
| 🟡 P2 | SKILL-003 API Design | Cần khi build inter-service communication |
| 🟡 P2 | SKILL-013 Security Engineering | Cần trước production deployment |
| 🟢 P3 | SKILL-014 DevOps & CI/CD | Cần khi chuẩn bị release |

---

*Generated from: `docs/init/skills-catalog.md` (DOC-G02 v1.1.0)*
