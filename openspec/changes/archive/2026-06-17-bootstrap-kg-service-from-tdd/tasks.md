# Tasks

> Historical archive note: completed legal-seed tasks below reflect the original bootstrap program snapshot. They should not be interpreted as the active default scope of the shared `kg-service` repository.

## Milestone: `migrations/`

### Phase A Foundation

- [x] Create migration for table `tenants` with platform sentinel seed support.
- [x] Create migration for table `apps` with API key hash/prefix indexes.
- [x] Create migration for table `access_grants` with scope, permission, status, and expiration constraints.
- [x] Create migration for table `access_audit_log` with monthly partitioning strategy.
- [x] Create migration for table `domains`.
- [x] Create migration for table `ontology_versions`.
- [x] Create migration for table `node_type_schemas`.
- [x] Create migration for table `rel_type_schemas`.
- [x] Create migration for table `cross_domain_rel_rules`.
- [x] Create migration for table `domain_query_templates`.
- [x] Create migration for table `domain_status_field_configs`.
- [x] Create migration for table `kg_nodes`.
- [x] Create migration for table `kg_relationships`.
- [x] Create migration for table `kg_outbox_events`.
- [x] Enable row-level security on `kg_nodes`.
- [x] Enable row-level security on `kg_relationships`.
- [x] Seed the platform sentinel tenant row and baseline tenant/app fixtures for integration tests.

## Milestone: `internal/access`

### Phase A Foundation (`specs/identity-access`)

- [x] Implement `IdentityResolver` credential lookup against `apps`.
- [x] Implement Redis cache for `apikey:{hash}` with short TTL and revoke invalidation.
- [x] Implement `AccessResolver` visibility calculation for self, tenant-wide, platform, and grant-derived visibility.
- [x] Implement Redis cache for `acl:{tenant_id}:{app_id}` with targeted invalidation on grant changes.
- [x] Implement request-context identity model shared by HTTP, workers, and audit flows.
- [x] Implement auth middleware helpers that ignore caller-supplied `tenant_id` and `app_id` body fields.

### Phase C Sharing And Auditability (`specs/admin-mcp-observability`)

- [x] Implement access-grant creation service with scope validation and `expires_at` enforcement for cross-tenant `write` and `admin` grants.
- [x] Implement access-grant listing service for grantor/grantee filters.
- [x] Implement access-grant revoke service with immediate cache invalidation hooks.
- [x] Implement audit log writer for read, search, write, grant-create, and grant-revoke actions.
- [x] Implement audit retrieval service for owner-scoped history.

## Milestone: `internal/ontology`

### Phase A Foundation (`specs/ontology-plane`)

- [x] Implement domain repository for registration, lookup, visibility, and version retrieval.
- [x] Implement effective ontology resolution across platform-owned, tenant-owned, and shared domains.
- [x] Implement node-type schema lookup and property validation.
- [x] Implement relationship-type schema lookup and direction/type validation.
- [x] Implement cross-domain bridge validation and rule expansion.
- [x] Implement Query Pattern DSL validator for shape, params, hop count, and allowed filter constructs.
- [x] Implement status-field configuration repository and reader APIs for lifecycle and ranking metadata.

### Phase B Legal Seed Onboarding

- [x] Prepare bootstrap flow for seeding legal domains via ontology APIs.
- [x] Prepare bootstrap flow for seeding legal node types and relationship types.
- [x] Register the five initial legal query templates through ontology APIs.
- [x] Activate the legal query templates through the template activation API.
- [x] Validate that each legal template executes successfully through the generic read route.

## Milestone: `internal/write`

### Phase A Foundation (`specs/write-path`)

- [x] Implement transaction-scoped `SET LOCAL app.tenant_id` and `SET LOCAL app.app_id` handling.
- [x] Implement `WriteService` node create flow with domain validation, ownership metadata, visibility, status extraction, and outbox emission.
- [x] Implement `WriteService` node update flow with domain revalidation and outbox emission.
- [x] Implement `WriteService` node delete flow with soft-delete semantics and outbox emission.
- [x] Implement `WriteService` relationship create flow with schema validation and outbox emission.
- [x] Implement rule-driven bridge relationship creation during node writes.
- [x] Implement external-ref persistence and uniqueness handling.

## Milestone: `internal/read`

### Phase B Read Templates (`specs/read-templates`)

- [x] Implement `QueryTemplateCompiler` runtime compilation from stored DSL to graph queries.
- [x] Inject ACL predicates at the start node match.
- [x] Inject ACL predicates at every traversal hop.
- [x] Implement parameter schema validation for required and typed read parameters.
- [x] Implement lifecycle-aware hop filtering from `domain_status_field_configs`.
- [x] Implement graph timeout and max-row safeguards.
- [x] Implement `ReadService` execution against the selected graph backend.
- [x] Implement read audit logging for allow and deny outcomes.

## Milestone: `internal/search`

### Phase B Semantic Search (`specs/semantic-search`)

- [x] Create and document vector projection schema for collection `kg_vectors`.
- [x] Implement payload mapping for `node_id`, `node_type`, `domain_id`, `owner_tenant_id`, `owner_app_id`, `acl_visible_to`, `is_deleted`, `status_value`, `authority_score`, and `domain_props`.
- [x] Implement embedding generation for projected searchable content.
- [x] Implement `SearchService` ACL filtering.
- [x] Implement `SearchService` deletion-state filtering.
- [x] Implement `SearchService` explicit domain filtering.
- [x] Implement lifecycle-aware search filtering only when all targeted domains are configured.
- [x] Implement authority-score return path for downstream reranking.

## Milestone: `internal/workers`

### Phase B Sync And Consistency (`specs/sync-consistency`)

- [x] Implement outbox polling or stream publisher for pending `kg_outbox_events`.
- [x] Implement shared worker runtime with retry and dead-letter/error status handling.
- [x] Implement `GraphSyncWorker` node upsert handler.
- [x] Implement `GraphSyncWorker` relationship upsert handler.
- [x] Implement `GraphSyncWorker` ACL recomputation handler for grant changes.
- [x] Implement `GraphSyncWorker` status cascade handler from configured cascade rules.
- [x] Implement `VectorSyncWorker` embedding upsert handler.
- [x] Implement `VectorSyncWorker` ACL payload update handler.
- [x] Implement `VectorSyncWorker` status and authority payload mapping.
- [x] Implement `AccessSyncWorker` Redis cache invalidation for grant create/revoke.
- [x] Implement `AccessSyncWorker` graph fanout orchestration for ACL refresh.
- [x] Implement `AccessSyncWorker` vector fanout orchestration for ACL refresh.
- [x] Validate that seeded legal content still respects ACL constraints after graph/vector projection.

### Phase D Hardening (`specs/sync-consistency`)

- [x] Implement scheduled reconciliation job comparing PostgreSQL `kg_nodes` and `kg_relationships` against graph projections.
- [x] Implement scheduled reconciliation job comparing PostgreSQL `kg_nodes` against Qdrant payloads.
- [x] Implement reconciliation result persistence or reporting surface for drift metrics.

## Milestone: `internal/http`

### Phase A Foundation

- [x] Initialize the Go service workspace, configuration model, environment loading, and bootstrap wiring for PostgreSQL and Redis.
- [x] Define shared HTTP success envelope types and serializers aligned with `specs/api-conventions`.
- [x] Define shared HTTP error envelope types with stable `code`, `message`, and `details` fields.
- [x] Define shared status-code mapping rules for `400`, `401`, `403`, `404`, `422`, timeout-class errors, and documented `5xx` responses.
- [x] Implement `POST /v1/tenants`.
- [x] Implement `GET /v1/tenants/{tenant_id}`.
- [x] Implement `PUT /v1/tenants/{tenant_id}`.
- [x] Implement `DELETE /v1/tenants/{tenant_id}`.
- [x] Implement `POST /v1/tenants/{tenant_id}/apps`.
- [x] Implement `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key`.
- [x] Implement `GET /v1/tenants/{tenant_id}/apps`.
- [x] Implement `DELETE /v1/tenants/{tenant_id}/apps/{app_id}` with immediate API-key cache invalidation.
- [x] Implement `GET /v1/access/resolve`.
- [x] Implement `POST /v1/tenants/{tenant_id}/ontology/domains`.
- [x] Implement `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types`.
- [x] Implement `GET /v1/tenants/{tenant_id}/ontology/effective`.
- [x] Implement `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates`.
- [x] Implement `PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate`.
- [x] Implement `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config`.
- [x] Implement `GET /v1/ontology/domains/{domain_id}`.
- [x] Implement `POST /v1/kg/write/nodes`.
- [x] Implement `PUT /v1/kg/write/nodes/{id}`.
- [x] Implement `DELETE /v1/kg/write/nodes/{id}`.
- [x] Implement `POST /v1/kg/write/relationships`.
- [x] Implement `POST /v1/kg/write/ingest/document`.
- [x] Implement `GET /v1/kg/write/ingest/jobs/{job_id}`.

### Phase B Read And Search

- [x] Implement `POST /v1/kg/read/template/{domain_id}/{template_name}`.
- [x] Implement `GET /v1/kg/read/templates?domain_id=...`.
- [x] Implement `GET /v1/kg/read/nodes/{id}`.
- [x] Implement `POST /v1/kg/search/semantic`.
- [x] Implement `POST /v1/kg/search/rag`.
- [x] Normalize list-response envelopes and pagination/filter parsing for read-template, grant-list, app-list, and audit-list endpoints.

### Phase C Sharing, Integrity, And MCP

- [x] Implement `POST /v1/access/grants`.
- [x] Implement `GET /v1/access/grants?grantor_tenant_id=...&grantee_tenant_id=...`.
- [x] Implement `DELETE /v1/access/grants/{id}`.
- [x] Implement `GET /v1/access/audit?resource_owner_tenant_id=...`.
- [x] Implement `GET /v1/kg/integrity/tenant/{tenant_id}`.
- [x] Implement `GET /v1/kg/integrity/missing-bridges?tenant_id=...`.
- [x] Implement MCP transport over HTTP+SSE.
- [x] Implement MCP capability `kg_list_templates`.
- [x] Implement MCP capability for template execution.
- [x] Implement MCP capability for semantic search.
- [x] Implement MCP capability for graph-RAG retrieval.
- [x] Implement MCP capability for ontology inspection.
- [x] Implement MCP capability for access-resolution or visibility introspection.
- [x] Implement MCP capability for integrity or health inspection as defined by the TDD tool set.
- [x] Normalize MCP tool success and validation-error mapping to match `specs/api-conventions`.

### Phase D Hardening

- [x] Implement rate limiting for REST endpoints by tenant tier.
- [x] Implement rate limiting for MCP operations by tenant tier.

## Milestone: `tests/integration`

### Phase A Foundation

- [x] Add integration tests for active key resolution, revoked key rejection, own-data visibility, platform-visible data, expired grants, and request-context isolation.
- [x] Add tests for effective ontology composition, unknown domain rejection, template registration, raw-Cypher rejection, traversal-depth enforcement, and missing bridge validation.
- [x] Add integration tests for write authorization, schema validation rejection, atomic outbox creation, bridge relationship creation, soft delete handling, and external-ref persistence behavior.
- [x] Add contract tests for shared success envelope shape and shared error envelope shape across representative endpoints.

### Phase B Read, Search, And Sync

- [x] Add tests for inactive template rejection, inaccessible start-node filtering, inaccessible hop filtering, missing parameter rejection, type mismatch rejection, lifecycle-filter no-op behavior, and timeout/row-cap enforcement.
- [x] Add tests for ACL filtering, deleted-node filtering, domain scoping, mixed-domain lifecycle handling, authority metadata return, and search over all visible domains when no filter is supplied.
- [x] Add tests for successful projection, retry behavior, status cascade execution, grant-create visibility propagation, revoke enforcement, and ACL fanout consistency across Redis/graph/vector stores.
- [x] Add contract tests for pagination/filter semantics on list-style endpoints and validation error mapping for malformed filters.

### Phase C Sharing, MCP, And Auditability

- [x] Add end-to-end tests for grant creation, grant expiry, revoke propagation under the target SLA, integrity endpoint behavior, MCP parity with REST ACL behavior, and audit trail generation.

### Phase D Hardening

- [ ] Run performance validation for `POST /v1/kg/read/template/{domain_id}/{template_name}` against the TDD latency objective.
- [ ] Run performance validation for `POST /v1/kg/search/semantic` against the TDD latency objective.
- [ ] Run performance validation for write-to-sync visibility latency across PostgreSQL to graph/vector replicas.
- [ ] Run performance validation for the GraphRAG pipeline against the TDD latency objective.
- [x] Execute security validation for raw-query prevention on ontology and read APIs.
- [x] Execute security validation for RLS isolation across concurrent tenant requests.
- [x] Execute security validation for ACL propagation and revoke enforcement timing.
- [x] Execute security validation for cross-tenant privilege-escalation attempts on grant and write flows.
- [x] Onboard at least one non-legal sample domain using ontology APIs only, without core service code changes.

## Milestone: `docs/operations`

### Phase D Hardening (`specs/sync-consistency`, `specs/admin-mcp-observability`)

- [x] Document runbook for replica recovery.
- [x] Document runbook for reconciliation incident handling.
- [x] Document runbook for grant incident response.
- [x] Document runbook for ontology rollout and version rollback.
- [x] Document runbook for API key revocation response.
