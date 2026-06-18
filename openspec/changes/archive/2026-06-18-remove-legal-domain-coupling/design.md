# Design

## Current Behavior

The service code is mostly generic, but the repo still embeds legal-domain examples as if they are part of the default platform shape:

- bootstrap startup calls a dedicated legal ontology bootstrap path
- ontology seed helpers publish legal domains, legal node types, legal relationships, legal templates, and legal status scoring
- tests assert against `luat_thue_hkd`, legal payload fields, and legal template names
- README and TDD-derived spec text describe legal ontology as the active bootstrap slice

The result is a mismatch between the intended architecture and the day-to-day repository experience.

## Problem Statement

The project should communicate one invariant clearly: core `kg-service` code knows nothing about any business domain. Domain examples may exist as optional fixtures, but they cannot be the default implementation, the default bootstrap, or the baseline language of the repository documentation.

## Goals

- Remove legal-domain coupling from bootstrap, seed data, tests, and active documentation.
- Keep all affected behavior generic and reusable for arbitrary domains.
- Preserve coverage for query templates, lifecycle/status config, search metadata, and cross-domain rules using neutral fixtures.
- Ensure OpenSpec artifacts describe legal content, if kept at all, as external sample data rather than platform baseline.

## Non-Goals

- Designing a new canonical sample business domain with rich semantics.
- Reworking backend parity scope beyond the fixture and documentation cleanup needed here.
- Deleting historical archive context that is useful for audit, unless it actively misleads current work.

## Key Decisions

### 1. Replace legal bootstrap with neutral bootstrap fixtures

The default bootstrap path should seed only generic sample data needed by tests and local development. Fixture identifiers should describe structure, not a business vertical. Examples:

- domain IDs such as `sample-registry` or `sample-policy`
- node types such as `Document`, `Section`, `Topic`
- relationship types such as `CONTAINS`, `REFERENCES`, `BELONGS_TO`

### 2. Keep domain semantics in optional fixtures, not core startup

If the team still wants a richer example domain later, it should live behind an explicit example or test-fixture boundary, not in the default bootstrap runtime. The core application should boot cleanly without legal ontology assumptions.

### 3. Rewrite tests around generic invariants

Tests should prove platform behavior:

- template registration and activation
- ACL-aware template execution
- lifecycle/status mapping
- vector metadata projection
- ingest payload persistence

They should not depend on legal naming such as `luat_thue_hkd`, `loai_van_ban`, or specific legal template titles.

### 4. Clean active documentation before archived history

User-facing and active engineering docs must be updated in this change. Archived OpenSpec material can remain for history, but active README and active changes should stop referring to legal onboarding as current scope. If archived artifacts are likely to confuse contributors, add a note that they are historical snapshots.

## Risks And Mitigations

- Replacing fixtures can break broad test coverage.
  - Mitigation: swap identifiers carefully and preserve semantic intent of each test.
- Neutral sample data may become too abstract and stop exercising status/template features.
  - Mitigation: keep generic fixtures structurally rich enough to cover lifecycle, hierarchy, and cross-domain relations.
- Historical OpenSpec references may still appear in searches.
  - Mitigation: update active docs and active changes first, then add explicit wording where archived material remains intentionally unchanged.

## Validation Strategy

- Verify startup no longer invokes legal-specific bootstrap helpers.
- Verify ontology seed tests pass using neutral sample domains and templates.
- Verify read, write, and search tests no longer depend on legal IDs or legal property names.
- Verify README and active OpenSpec changes describe the repo as domain-agnostic with no default legal domain.
