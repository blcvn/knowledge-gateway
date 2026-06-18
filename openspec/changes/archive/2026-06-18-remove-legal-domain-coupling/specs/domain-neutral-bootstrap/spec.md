# domain-neutral-bootstrap

## ADDED Requirements

### Requirement: Domain-neutral default bootstrap
The system SHALL keep the default `kg-service` bootstrap path free of business-domain-specific ontology assumptions.

#### Scenario: Service startup initializes bootstrap fixtures
- WHEN the service starts with repository-default bootstrap behavior
- THEN any seeded ontology data SHALL use neutral sample identifiers and terminology
- AND SHALL NOT require legal-domain IDs, legal template names, or legal field names to function

### Requirement: Domain-neutral repository fixtures
The repository SHALL express test fixtures and sample seed data in domain-agnostic terms so platform behavior can be reused across arbitrary domains.

#### Scenario: Tests validate template, status, and search behavior
- WHEN automated tests exercise ontology, read, write, or search flows
- THEN the assertions SHALL depend on generic platform invariants
- AND SHALL NOT rely on legal-domain fixture names to prove correctness

### Requirement: Active documentation reflects shared-platform scope
The repository SHALL describe `kg-service` as a shared domain-agnostic platform in active documentation and active OpenSpec artifacts.

#### Scenario: A contributor reads the current project documentation
- WHEN they read the README, active TDD guidance, or active OpenSpec changes
- THEN the materials SHALL present legal content, if mentioned, as optional historical or example context
- AND SHALL NOT describe legal ontology as the default scope of the core project
