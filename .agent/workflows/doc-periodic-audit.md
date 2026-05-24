# Workflow: Periodic Documentation Audit

> **Trigger:** Run this workflow at the start of each sprint OR before every major release to proactively detect and remediate documentation drift across the entire project.

---

## Agents Involved
- **Documentation Management Agent** — Leads the audit, produces the report, and drives remediation.

---

## Steps

### Step 1: Collect Scope
- Identify all services in the project by scanning `services/` directory.
- Identify the current product version from `package.json`, `go.mod`, or equivalent.

### Step 2: Product-Level Document Audit
Check the following product-level documents exist and have current metadata:
- [ ] `docs/product/README.md`
- [ ] `docs/product/architecture.md` — Does it reflect all current services?
- [ ] `docs/standards/api-conventions.md`
- [ ] `docs/standards/data-glossary.md`
- [ ] `docs/standards/security-policy.md`
- [ ] `docs/releases/` — Does a release note file exist for the current version?

### Step 3: Per-Service Document Audit
For EACH service, verify:
- [ ] `README.md` exists and is current
- [ ] `api.md` exists — Cross-check against the actual route definitions in code. Flag any endpoint in code with no corresponding doc entry.
- [ ] `configuration.md` exists — Cross-check against `.env.example` or actual config loading code. Flag undocumented variables.
- [ ] `data-model.md` exists — Cross-check against migration files or ORM models. Flag undocumented tables.
- [ ] `architecture.md` exists and reflects the current internal structure.
- [ ] `runbook.md` exists and is not a stub (services in production must have complete runbooks).
- [ ] `changelog.md` exists and has an entry for the current version.

### Step 4: ADR Completeness Check
- Review recent architectural changes in Git history (commits/PRs mentioning "refactor", "migrate", "adopt", "replace").
- For each such change, verify a corresponding ADR exists in `docs/adr/`.
- Flag any undocumented architectural decision.

### Step 5: Stale Reference Scan
- Search all documents for references to specific code files (`handlers/`, `services/`, specific function names).
- Verify each referenced file/function still exists in the current codebase.
- Flag stale references as **documentation bugs**.

### Step 6: Metadata Compliance Check
- Scan all `.md` files in `docs/` and `services/*/docs/`.
- Flag any document missing the required metadata header (RULE-004).

### Step 7: Produce Audit Report
Generate a `docs/audits/audit-YYYY-MM-DD.md` report with:
```markdown
# Documentation Audit Report — YYYY-MM-DD

## Summary
- Total documents audited: X
- Compliant: X
- Issues found: X (Y critical, Z warnings)

## Critical Issues (Block Release)
- [RULE violation, service name, document, description]

## Warnings (Require Remediation This Sprint)
- [Item, location, description]

## Stale References
- [Document path, stale reference, correct reference or status]

## Recommended Actions
- [Prioritized list of actions]
```

---

## Success Criteria
- Audit report generated and committed to the repository.
- All **Critical Issues** (RULE violations) have remediation tasks created.
- Product-level and per-service docs are verified as consistent with the current codebase.

## Cadence
- **Mandatory:** Before every release tag.
- **Recommended:** Start of every sprint.
