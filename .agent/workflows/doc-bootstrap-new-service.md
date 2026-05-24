# Workflow: New Service Documentation Bootstrap

> **Trigger:** Whenever a new service or functional module is scaffolded/created, this workflow MUST be executed to produce the mandatory bootstrap documentation set before any feature development begins.

---

## Agents Involved
- **Documentation Management Agent** — Leads this workflow, creates all required documents.
- **Golang Backend Agent / Frontend Agent** — Consulted for architecture and API contract details.

---

## Steps

### Step 1: Gather Service Context
Collect the following information before creating any documents:
- **Service Name:** (e.g., `payment-service`, `auth-service`)
- **Service Purpose:** What business capability does this service own?
- **Owners:** Team or individual responsible.
- **Technology Stack:** Language, framework, database.
- **External Dependencies:** Other services or third-party APIs this service calls.
- **Exposed APIs:** List of endpoints this service provides (even if not fully designed yet).

### Step 2: Create Service Documentation Tree
Create the directory structure and all mandatory files:

```
services/[service-name]/docs/
├── README.md             ← Step 3
├── api.md                ← Step 4
├── architecture.md       ← Step 5
├── data-model.md         ← Step 6
├── configuration.md      ← Step 7
├── runbook.md            ← Step 8
└── changelog.md          ← Step 9
```

### Step 3: Write `README.md`
Must include:
- Service name and one-paragraph description
- Business capability owned
- Links to `api.md`, `architecture.md`, `runbook.md`
- Tech stack summary
- Owner / team contact

### Step 4: Write `api.md`
- List all known/planned endpoints, even if marked `[Planned]`.
- For each endpoint: method, path, auth requirement, request/response schema skeleton.
- Mark unfinalized sections with `[Draft]`.

### Step 5: Write `architecture.md`
- Describe the internal structure (layers: handler → usecase → repository).
- Document key design decisions with rationale.
- Include a simple component diagram if possible.

### Step 6: Write `data-model.md`
- List all database tables/collections owned by this service.
- Include entity fields, types, and relationships.
- Mark `[TBD]` for entities not yet designed.

### Step 7: Write `configuration.md`
- List all known environment variables.
- For each: name, description, type, default, required/optional.
- Add a section for feature flags if applicable.

### Step 8: Write `runbook.md`
- Include stubs for: startup procedure, health check, common error resolution, rollback.
- These may be `[Draft]` initially but MUST be completed before production deployment (enforced by RULE-002).

### Step 9: Initialize `changelog.md`
- Create the file with an initial entry:
```markdown
## [Unreleased]
### Added
- Initial service scaffolding and bootstrap documentation.
```

### Step 10: Register Service in Product-Level Docs
- Add the service to `docs/product/architecture.md` (system topology diagram/list).
- Add a brief entry and link to the service's `README.md` in the product-level `docs/product/README.md`.

---

## Success Criteria
- All 7 files in `services/[name]/docs/` exist with valid metadata headers.
- Service is registered in product-level architecture docs.
- No RULE-001 or RULE-002 violations.

## Failure Handling
- If context for a section is unavailable, create the file with `status: Draft` and mark gaps with `[TODO]`.
- Surface all `[TODO]` items in a summary report to the human owner.
