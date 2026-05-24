# Document Taxonomy & Classification

## Document Scope Levels

### Level 1: Product-Level Documents (Global)
These documents apply to the entire product and are maintained centrally. All services must conform to them.

| Document Type | File/Path Convention | Purpose |
|---|---|---|
| Product Overview | `docs/product/README.md` | High-level vision, goals, stakeholders |
| Architecture Decision Records (ADR) | `docs/adr/ADR-XXXX-title.md` | Record significant technical decisions with context and rationale |
| System Architecture Diagram | `docs/product/architecture.md` | Overall system topology, component relationships |
| API Standards & Conventions | `docs/standards/api-conventions.md` | Naming, versioning, error format standards for all APIs |
| Data Model Glossary | `docs/standards/data-glossary.md` | Canonical definitions for all domain entities across services |
| Security Policy | `docs/standards/security-policy.md` | Authentication, authorization, encryption standards |
| Release Notes | `docs/releases/vX.Y.Z.md` | Per-version changelog covering all services |

### Level 2: Service-Level Documents (Per Service/Feature)
Each service or functional module maintains its own documentation sub-tree.

```
services/[service-name]/docs/
├── README.md            # Service overview, purpose, owners
├── api.md               # Full API reference (endpoints, request/response schemas)
├── architecture.md      # Internal architecture, design decisions specific to this service
├── data-model.md        # Database schema, entity relationships for this service
├── configuration.md     # Environment variables, feature flags, external dependencies
├── runbook.md           # Operational guide: how to deploy, monitor, troubleshoot, rollback
└── changelog.md         # Service-specific change history
```

## Document Type Definitions

### Architecture Decision Record (ADR)
- **When to create:** Any significant, non-obvious technical decision (technology choice, pattern adoption, migration plan).
- **Template:**
```markdown
# ADR-XXXX: [Title]
- **Date:** YYYY-MM-DD
- **Status:** [Proposed | Accepted | Deprecated | Superseded by ADR-XXXX]
## Context
[Why does this decision need to be made?]
## Decision
[What was decided?]
## Consequences
[What are the trade-offs, risks, and downstream impacts?]
```

### API Reference Document
- **When to create:** Every time a new public or internal endpoint is added.
- **Must include:** Endpoint, HTTP method, authentication requirement, request schema, response schema (success + all error codes), example request/response.

### Runbook
- **When to create:** Before a service goes to production.
- **Must include:** Start/stop procedures, health check endpoint, common error diagnosis steps, escalation contacts, rollback procedure.
