# Workflow: Documentation Review on Code Change

> **Trigger:** Whenever a code change (PR / merge) touches a service's source code, this workflow MUST be executed by the Documentation Management Agent to ensure doc ↔ code consistency.

---

## Agents Involved
- **Documentation Management Agent** — Reads code diff, identifies documentation gaps, and creates/updates required documents.
- **[Optional] Code Expert Agent** — Consulted to clarify implementation intent when documentation is ambiguous.

---

## Steps

### Step 1: Analyze the Code Diff
- Read all changed files in the PR/commit.
- Categorize each change:
  - `NEW_ENDPOINT` — A new route/handler was added
  - `MODIFIED_ENDPOINT` — An existing route's request/response signature changed
  - `DELETED_ENDPOINT` — An endpoint was removed
  - `NEW_ENV_VAR` — A new environment variable or config key was added
  - `NEW_TABLE_OR_SCHEMA` — A database migration or schema change was made
  - `ARCHITECTURAL_CHANGE` — A significant structural pattern change
  - `DEPENDENCY_UPGRADE` — A key dependency was upgraded

### Step 2: Map Changes to Required Documents
For each categorized change, identify the document(s) that must be created or updated:

| Change Category | Document Action Required |
|---|---|
| `NEW_ENDPOINT` | Add entry to `services/[name]/docs/api.md` |
| `MODIFIED_ENDPOINT` | Update corresponding entry in `api.md`; update version metadata |
| `DELETED_ENDPOINT` | Remove or mark as deprecated in `api.md` |
| `NEW_ENV_VAR` | Add entry to `services/[name]/docs/configuration.md` |
| `NEW_TABLE_OR_SCHEMA` | Update `services/[name]/docs/data-model.md` |
| `ARCHITECTURAL_CHANGE` | Create new ADR in `docs/adr/ADR-XXXX-title.md` |
| `DEPENDENCY_UPGRADE` | Add note to `services/[name]/docs/changelog.md` |

### Step 3: Execute Document Updates
- For each identified gap, create or update the corresponding document.
- Update the `version` and `last_updated` metadata fields.
- If creating a new ADR, generate from the ADR template (see `document_taxonomy.md`).

### Step 4: Validate Consistency
Run the Drift Detection Checklist (from `version_management_consistency.md`):
- [ ] All new endpoints documented?
- [ ] All deleted endpoints removed/deprecated in docs?
- [ ] Schemas match current implementation?
- [ ] New config variables documented?
- [ ] Changelog updated?

### Step 5: Report
Produce a **Documentation Update Report** summarizing:
- Documents created
- Documents updated
- Any open items that require human review (e.g., an architectural decision the agent cannot fully characterize)

---

## Success Criteria
All items in the Drift Detection Checklist pass. No RULE violations remain open.

## Failure Handling
If a required document cannot be completed without additional information:
1. Create a placeholder document with `status: Draft` and mark missing sections with `[TODO: Requires clarification — describe what is needed]`.
2. Flag the open item in the Documentation Update Report.
