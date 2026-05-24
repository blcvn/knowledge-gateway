# Documentation Governance Rules & Constraints

> These rules are MANDATORY and apply to all contributors, agents, and automated processes working within this project.
> Violations block merges, deployments, or task completions.

---

## RULE-001: No Undocumented Public API
**Constraint:** Any public or internal API endpoint that is merged into the main branch MUST have a corresponding entry in the service's `api.md` before the PR can be approved.

**Applies to:** All services, all contributors, all agents.

**Enforcement:** PR checklist item. Automated doc-lint check (if configured).

---

## RULE-002: New Service Must Have Complete Bootstrap Documentation
**Constraint:** A new service is not considered created until the following documents exist:
- `services/[name]/docs/README.md`
- `services/[name]/docs/api.md`
- `services/[name]/docs/configuration.md`
- `services/[name]/docs/architecture.md`

**Enforcement:** Any agent or contributor scaffolding a new service MUST create these files as part of the task. The task is incomplete without them.

---

## RULE-003: Architectural Decisions Must Be Recorded
**Constraint:** Any significant technical decision — technology selection, major refactoring, pattern adoption, third-party integration — MUST be recorded as an ADR in `docs/adr/` before or concurrent with implementation.

**What qualifies as "significant":** A decision that, if made differently, would require substantial rework.

---

## RULE-004: Document Metadata is Mandatory
**Constraint:** Every document MUST contain a valid metadata header:
```markdown
---
version: X.Y.Z
last_updated: YYYY-MM-DD
updated_by: [author]
status: [Draft | Review | Approved | Deprecated]
---
```
Documents without valid metadata are considered **unmanaged** and flagged for remediation.

---

## RULE-005: Changelog Entry Required for Every Release
**Constraint:** No service version may be tagged/released without a corresponding entry in its `changelog.md` and the product-level `docs/releases/vX.Y.Z.md`.

---

## RULE-006: Deprecated Documents Must Be Explicitly Marked
**Constraint:** Documents that are no longer valid MUST NOT be silently deleted. They must first be updated with:
- `status: Deprecated` in metadata
- A deprecation notice at the top: `> ⚠️ DEPRECATED as of vX.Y.Z. Superseded by [link to new doc].`

They may be archived or deleted only after one full release cycle.

---

## RULE-007: Doc-to-Code Reference Must Be Verifiable
**Constraint:** When a document references a specific code implementation (e.g., "This handler is implemented in `handlers/user.go`"), the referenced file and function MUST exist. Stale code references in documents are treated as documentation bugs.

---

## RULE-008: Configuration Variables Must Be Documented Before Deployment
**Constraint:** Any new environment variable or feature flag used in code MUST be listed in `services/[name]/docs/configuration.md` with:
- Variable name
- Purpose/description
- Type and valid values
- Default value
- Whether it is required or optional

---

## RULE-009: Technical Design Must Be Migrated to Spec Files Before Implementation
**Constraint:** `technical_design.md` in `specs/` is allowed as a starting document but MUST be decomposed into individual spec files (`FEAT-NNN`, `ARCH-NNN`, etc.) before any implementation sprint begins.

**What this means:**
- AI agents MUST NOT implement directly from `technical_design.md` without a corresponding typed spec file
- Each implementable unit in `technical_design.md` must become its own spec with Acceptance Criteria
- After decomposition, `technical_design.md` transitions to a read-only reference doc

**Applies to:** All services and packages with `specs/technical_design.md`.

**Enforcement:** Sprint planning checklist. Agent task is incomplete without spec files.

---

## RULE-010: New App/Package Must Have Bootstrap Documentation
**Constraint:** A new frontend app or package is not considered created until the following documents exist:
- `apps/[name]/docs/README.md` (DOC-A01)
- `apps/[name]/docs/architecture.md` (DOC-A02)

**Applies to:** All packages under `apps/` — preview, demo, openui, loveable, and any future packages.

**Enforcement:** Any agent or contributor scaffolding a new app MUST create these files as part of the task.

---

## CONSTRAINT SUMMARY TABLE

| Rule | Constraint | Blocking? |
|---|---|---|
| RULE-001 | Public API endpoints must be documented | ✅ Yes — blocks PR |
| RULE-002 | New service requires bootstrap docs | ✅ Yes — task incomplete |
| RULE-003 | Significant decisions require ADR | ✅ Yes — blocks implementation |
| RULE-004 | All docs require metadata header | ⚠️ Flagged for remediation |
| RULE-005 | Release requires changelog entry | ✅ Yes — blocks release |
| RULE-006 | Deprecated docs must be marked | ✅ Yes — silent deletion not allowed |
| RULE-007 | Code references must be verifiable | ⚠️ Flagged as doc bug |
| RULE-008 | Env vars documented before deploy | ✅ Yes — blocks deployment |
| RULE-009 | technical_design.md must be decomposed before sprint | ✅ Yes — blocks implementation |
| RULE-010 | New app requires bootstrap docs | ✅ Yes — task incomplete |
