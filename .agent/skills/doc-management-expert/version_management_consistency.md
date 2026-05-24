# Version Management & Consistency

## Document Versioning Strategy

### Versioning in Lockstep with Code
- All documentation is versioned alongside the codebase using Git.
- Every document carries a metadata header:
```markdown
---
version: 1.3.0
last_updated: YYYY-MM-DD
updated_by: [author/agent-name]
status: [Draft | Review | Approved | Deprecated]
---
```
- Document versions match the product/service semantic version they describe.
- When a new product release is tagged (e.g., `v2.1.0`), all associated docs must be reviewed and their metadata updated before the release is finalized.

### Changelog Maintenance
- `docs/releases/vX.Y.Z.md` (product-level) and `services/[name]/docs/changelog.md` (service-level) are updated for every release.
- Changelog entries follow the **Keep a Changelog** format:
  - `Added` — New features or documents
  - `Changed` — Modified behavior or updated documents
  - `Deprecated` — Features or documents being phased out
  - `Removed` — Deleted features
  - `Fixed` — Bug fixes

---

## Code ↔ Documentation Consistency

### The Consistency Contract
The following pairs must always be in sync:

| Code Artifact | Required Document | Consistency Check |
|---|---|---|
| New API endpoint added | `api.md` in service docs | Endpoint documented with schema? |
| New environment variable | `configuration.md` | Variable listed with description and default? |
| New database table/schema | `data-model.md` | Entity and relationships documented? |
| New service created | `services/[name]/docs/README.md` | Service overview exists? |
| Architectural pattern changed | `docs/adr/` | ADR created for the decision? |
| Service deployed to production | `runbook.md` | Runbook complete and reviewed? |
| Dependency version upgraded | `changelog.md` | Upgrade noted with migration impact? |

### Drift Detection: Consistency Audit Checklist
Run this audit whenever a significant code change is merged:
- [ ] Are all new API endpoints documented in `api.md`?
- [ ] Are all deleted endpoints removed from documentation?
- [ ] Do all documented request/response schemas match the current code implementation?
- [ ] Are all new configuration variables in `configuration.md`?
- [ ] Has `changelog.md` been updated with a description of the change?
- [ ] Is the document `version` metadata updated to match the current release?
- [ ] If an architectural decision was made, is there a corresponding ADR?
