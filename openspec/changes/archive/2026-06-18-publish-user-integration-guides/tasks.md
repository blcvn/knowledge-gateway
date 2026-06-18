# Tasks

## Milestone: `docs/guides`

- [x] Add a user-facing quickstart guide for local bootstrap usage.
- [x] Add a navigation entry that points users to quickstart, integration workflows, API reference, and operations runbooks by audience.
- [x] Keep `README.md` focused on project overview while linking out to deeper consumer documentation.

## Milestone: `auth-and-onboarding`

- [x] Document local bootstrap credentials and how to call protected endpoints with `Authorization`.
- [x] Document the recommended onboarding sequence for tenant creation, app creation, API-key rotation, and visibility resolution.
- [x] Explain which auth and bootstrap behaviors are local-development conveniences versus production-oriented expectations.

## Milestone: `ontology-and-data-flows`

- [x] Document the sequence for domain creation, node-type creation, relationship-type creation, query-template creation, template activation, and status-field configuration.
- [x] Document common write flows for nodes, relationships, and document ingest.
- [x] Document common read and search flows, including template listing, template execution, node lookup, semantic search, and RAG search.

## Milestone: `mcp-integration`

- [x] Add a user-facing guide for connecting to the MCP transport, creating a session, and posting tool messages.
- [x] Explain when a consumer should prefer MCP over direct REST integration.
- [x] Clarify that MCP and REST share the same authorization and validation model.

## Milestone: `troubleshooting`

- [x] Add an integration troubleshooting section for authentication failures, malformed JSON, validation failures, forbidden access, and missing resources.
- [x] Link consumer-facing troubleshooting to existing operations runbooks only where operator action is actually required.

## Milestone: `alignment`

- [x] Cross-link the integration guides with the published API spec so schemas stay authoritative in one place.
- [x] Add a repeatable review step that updates user-facing guides whenever onboarding or integration behavior changes materially.
