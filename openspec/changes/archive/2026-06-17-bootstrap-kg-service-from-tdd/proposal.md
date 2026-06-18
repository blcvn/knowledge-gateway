# Bootstrap KG Service From TDD

> Historical archive note: this change captured the initial bootstrap plan at the time, including legal-domain seed onboarding as a sample configuration track. It is not the current baseline for `kg-service`; see `remove-legal-domain-coupling` for the domain-neutral default direction.

## Why

`docs/KG_Service_TDD_v1.md` defines the end-state architecture, data model, APIs, synchronization model, and phased rollout for KG Service, but the repository does not yet contain an implementation plan in executable OpenSpec form. The team needs a shared change package that translates the TDD into scoped deliverables, implementation sequencing, and acceptance criteria before code work begins.

## What Changes

- Create an implementation program for the domain-agnostic KG Service described in the TDD.
- Sequence delivery around the four rollout phases already defined in the design:
  - Phase A: identity, tenancy, access control, RLS, write path, and query-template compiler foundation.
  - Phase B: graph/vector sync, ACL propagation, read/search APIs, and legal ontology seed data through the ontology APIs.
  - Phase C: access sharing workflows, MCP surface, and auditability.
  - Phase D: reconciliation, rate limits, performance hardening, security validation, and domain-agnostic proof with a non-legal domain.
- Define the core service capability requirements needed to keep implementation aligned with the TDD invariants.

## Capabilities

- Multi-tenant identity and access enforcement
- Ontology-driven domain configuration
- CQRS-style write, read, and search data planes
- ACL-synchronized graph/vector replicas
- Controlled read/query execution through registered templates only
- MCP and admin operational surfaces

## Impact

- Establishes the first OpenSpec structure for this repository.
- Gives implementation teams a concrete backlog and acceptance path without changing runtime code yet.
- Reduces ambiguity around what must be built as core platform behavior versus what belongs in seed ontology/configuration.
