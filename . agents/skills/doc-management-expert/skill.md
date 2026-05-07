---
skill_id: SKILL-015
version: 1.0.0
status: active
priority: existing
group: Quản lý & Governance
created_at: 2026-04-24
---

# SKILL-015 · Documentation Management & Governance

## Mô tả

Định nghĩa và phân loại tài liệu (Product-level vs Service-level), quản lý version, đảm bảo code ↔ doc consistency, enforce governance rules.

## Agents sử dụng

- `doc-consistency-agent`

---

## Năng lực cốt lõi

### 1. Document Taxonomy

```
Knowledge Gateway Documentation Structure:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

docs/
├── product/           → PRODUCT-LEVEL (cross-service)
│   ├── ARCHITECTURE.md         → System-wide architecture
│   ├── PRD.md                  → Product Requirements
│   ├── ROADMAP.md              → Feature roadmap
│   └── DECISION_LOG.md         → High-level decisions
│
├── adr/               → Architecture Decision Records
│   ├── ADR-001-graph-db-selection.md
│   ├── ADR-002-async-pipeline.md
│   └── ...
│
├── standards/         → Engineering Standards
│   ├── coding-standards.md
│   ├── security-policy.md
│   ├── api-design-guide.md
│   └── testing-guide.md
│
├── init/              → Bootstrap documents
│   ├── skills-catalog.md       → This file
│   ├── agents-catalog.md
│   └── workflow-guide.md
│
└── product/           → Product design documents

services/{service-name}/docs/   → SERVICE-LEVEL
├── README.md           → Service overview
├── api.md              → API documentation
├── runbook.md          → Operations runbook
├── changelog.md        → Service changelog
└── architecture.md     → Service-specific architecture
```

### 2. Document Metadata Standard

```yaml
# Required frontmatter for all documents
---
doc_id: "DOC-G02"              # Unique ID (G=global, S=service-specific)
version: "1.1.0"               # Semantic version
status: "Approved"             # Draft | Review | Approved | Deprecated
last_updated: "2026-04-24"
updated_by: "ai-agent-architect"
scope: "REPO-LEVEL"           # REPO-LEVEL | SERVICE-LEVEL
---
```

### 3. ADR Management

```markdown
# ADR-{NNN}: {Decision Title}

## Status
[Proposed | Accepted | Deprecated | Superseded by ADR-{N}]
Date: YYYY-MM-DD

## Context
Mô tả vấn đề và bối cảnh. Tại sao cần phải ra quyết định này?

## Decision
Quyết định cụ thể được đưa ra.

## Rationale
Lý do chọn option này. Trade-offs được xem xét.

## Alternatives Considered
1. **Option A**: [mô tả] — Bị loại vì [lý do]
2. **Option B**: [mô tả] — Bị loại vì [lý do]

## Consequences
**Positive:** [benefits]
**Negative (trade-offs):** [costs/risks]

## References
- [link to related docs]
```

### 4. Version Management

```
Semantic Versioning for Documents:
MAJOR.MINOR.PATCH

- MAJOR: Breaking structural change (major reorganization, new taxonomy)
- MINOR: New sections added, significant content update
- PATCH: Typo fixes, clarification, minor updates

Changelog format (per document):
## Changelog
### v1.1.0 — 2026-04-24
- Added SKILL-008 through SKILL-016 skill definitions
- Updated priority matrix

### v1.0.0 — 2026-04-21
- Initial release
```

### 5. Code ↔ Doc Consistency (Drift Detection)

```bash
# Check for API drift: compare OpenAPI spec vs implementation
swagger-diff old-spec.yaml new-spec.yaml

# Check for config drift: env vars in code vs documented
grep -r 'os.Getenv\|env.Get' services/ | \
  awk -F'"' '{print $2}' | sort -u > code-env-vars.txt
grep -r '^\s*[A-Z_]*:' docs/standards/env-vars.md | \
  awk '{print $1}' | sort -u > documented-env-vars.txt
diff code-env-vars.txt documented-env-vars.txt

# Automated doc update trigger in CI
# .github/workflows/doc-check.yml
on:
  pull_request:
    paths:
      - 'services/*/internal/handler/**'
      - 'services/*/docs/api.md'
```

### 6. Governance Rules

```
Documentation Governance:

RULE-DOC-01: API changes require api.md update in same PR
RULE-DOC-02: New ADR required for every architecture decision
RULE-DOC-03: Service runbook must be updated before production deploy
RULE-DOC-04: doc_id and version metadata required in all docs
RULE-DOC-05: Deprecated docs must have "Superseded by" reference
RULE-DOC-06: All docs must be reviewed by doc-consistency-agent before merge

Audit Workflow:
1. Developer submits PR
2. CI triggers doc-consistency-agent check
3. Agent compares code changes vs doc changes
4. If drift detected → PR blocked with specific recommendations
5. Developer updates docs
6. Agent confirms consistency → PR approved to merge
```

---

## Checklist (per PR)

- [ ] API changes reflected in `api.md`
- [ ] New environment variables documented
- [ ] Breaking changes noted in `changelog.md`
- [ ] ADR created for architecture decisions
- [ ] Document version bumped (minor for additions, patch for fixes)
- [ ] `doc_id` and `status` metadata present
- [ ] Links to related documents verified (not broken)
- [ ] Runbook updated if operational procedure changed
