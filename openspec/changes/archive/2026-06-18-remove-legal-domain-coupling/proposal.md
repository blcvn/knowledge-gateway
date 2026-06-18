# Remove Legal Domain Coupling

## Why

`kg-service` is intended to be a shared, domain-agnostic platform, but the current repository still carries legal-domain assumptions in bootstrap code, ontology seed data, tests, README text, and historical spec references. Those artifacts make the project look like a legal-product implementation instead of a generic knowledge graph service.

This creates two problems:

- contributors can misread legal fixtures as required platform behavior
- future domains inherit legal naming and sample data from the core repo

## What Changes

- Remove legal-specific bootstrap and ontology seed data from the core runtime.
- Replace legal-domain fixtures in tests with neutral sample domains, node types, relationships, and templates.
- Rewrite repository documentation so the service is described as domain-agnostic without positioning legal ontology as the default sample.
- Add a cleanup pass for OpenSpec artifacts that still treat legal onboarding as part of the platform baseline.

## Capabilities

- Domain-neutral bootstrap and sample ontology fixtures
- Domain-neutral test coverage for read, write, search, and ontology behavior
- Domain-agnostic project documentation and OpenSpec guidance

## Impact

- Changes bootstrap defaults, fixture names, and repository documentation, but does not add new runtime capabilities.
- Reduces the chance that downstream implementers copy legal examples into unrelated domains.
- Makes future OpenSpec work easier to scope because the baseline repo no longer implies a preferred business domain.
