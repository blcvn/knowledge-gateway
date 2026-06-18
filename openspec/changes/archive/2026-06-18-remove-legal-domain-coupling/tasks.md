# Tasks

## Milestone: `internal/bootstrap`

- [x] Remove the legal-only bootstrap entrypoint from startup wiring.
- [x] Replace legal bootstrap fixtures with neutral sample ontology bootstrap data, or disable default domain seeding when not required.
- [x] Rename helper functions, logs, and comments so bootstrap terminology is generic.

## Milestone: `internal/ontology`

- [x] Replace legal domains, templates, and status-config seed data in [internal/ontology/seed.go](/Users/anhdt/vnpay/knowledge/kg-service/internal/ontology/seed.go) with neutral sample fixtures.
- [x] Update ontology tests to validate generic fixture behavior instead of legal fixture names.
- [x] Preserve coverage for template activation count, status mapping, and cross-domain rules after the fixture swap.

## Milestone: `internal/read`, `internal/write`, and `internal/search`

- [x] Replace legal domain IDs and legal field names in tests with neutral sample identifiers.
- [x] Keep ingest, template execution, and search assertions focused on generic behavior rather than legal terminology.
- [x] Confirm search projection metadata remains domain-agnostic while still covering authority and status semantics.

## Milestone: Documentation And OpenSpec

- [x] Rewrite [README.md](/Users/anhdt/vnpay/knowledge/kg-service/README.md) so bootstrap scope and examples are domain-neutral.
- [x] Revise [docs/KG_Service_TDD_v1.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/KG_Service_TDD_v1.md) to remove legal product framing from active platform guidance.
- [x] Update active OpenSpec artifacts that still describe legal onboarding as part of the current baseline.
- [x] Add clarifying wording to archived material only where needed to avoid confusion about current project scope.

## Milestone: Validation

- [x] Run the affected test suites for bootstrap, ontology, read, write, and search flows after fixture cleanup.
- [x] Review repository-wide search results to confirm active code and docs no longer treat legal ontology as the default domain.
