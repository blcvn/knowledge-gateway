# Integration Workflows

This guide explains the normal order of operations for integrating with the current bootstrap service. Use [API Reference](../api/README.md) for exact endpoint details and payload shapes.

## 1. Authenticate First

- Use `Authorization: Bearer <api_key>` on every `/v1/*` request.
- Start with `GET /v1/access/resolve` to verify the caller identity and visibility scope.
- Expect `401` when the API key is missing or invalid.
- Expect `429` if the bootstrap rate limiter rejects the caller.

## 2. Onboard A Tenant And App

Recommended order:

1. Create a tenant with `POST /v1/tenants`.
2. Read it back with `GET /v1/tenants/{tenant_id}`.
3. Create an app with `POST /v1/tenants/{tenant_id}/apps`.
4. Store the returned API key immediately.
5. Rotate the key later with `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key` when needed.
6. Use `GET /v1/tenants/{tenant_id}/apps` and `GET /v1/access/resolve` to verify the app is active and visible.

Notes:

- The app create and rotate-key flows are the only times plaintext API keys are returned.
- The bootstrap repo also ships seeded keys for local testing; those are conveniences, not a production pattern.
- JSON body fields named `tenant_id` or `app_id` are not trusted from clients because middleware strips them before handlers run.
- If you want a shorter operational checklist, use [Tenant And App Setup](./tenant-app-setup.md).

## 3. Model Ontology Before Writing Data

For a new domain, configure ontology first:

1. Create the domain with `POST /v1/tenants/{tenant_id}/ontology/domains`.
2. Add node types with `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types`.
3. Add relationship types with `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types`.
4. Add query templates with `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates`.
5. Activate templates with `PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate`.
6. Add lifecycle and authority rules with `POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config`.
7. Review the result with `GET /v1/tenants/{tenant_id}/ontology/effective` and `GET /v1/ontology/domains/{domain_id}`.

Why this order matters:

- write validation depends on ontology definitions
- template execution depends on active templates
- status cascade and authority behavior depend on status-field configuration

## 4. Write Data

Common write flows:

- Create nodes with `POST /v1/kg/write/nodes`
- Update nodes with `PUT /v1/kg/write/nodes/{id}`
- Delete nodes softly with `DELETE /v1/kg/write/nodes/{id}`
- Create relationships with `POST /v1/kg/write/relationships`
- Queue document ingestion with `POST /v1/kg/write/ingest/document`
- Check ingest progress with `GET /v1/kg/write/ingest/jobs/{job_id}`

Write-path expectations:

- node creation and document ingest currently return `202 Accepted`
- relationship creation returns `201 Created`
- writes are still subject to visibility and ontology validation
- downstream projection behavior is bootstrap-oriented today, so treat async status as local evaluation behavior rather than a finalized production SLA

## 5. Read And Search Data

Recommended read flow:

1. List active templates with `GET /v1/kg/read/templates?domain_id=...`.
2. Execute a template with `POST /v1/kg/read/template/{domain_id}/{template_name}`.
3. Fetch a known node directly with `GET /v1/kg/read/nodes/{id}` when you need full node details.

Recommended search flow:

1. Run semantic retrieval with `POST /v1/kg/search/semantic`.
2. Use `POST /v1/kg/search/rag` when the caller wants retrieval oriented toward an answer or synthesized context payload.
3. Restrict `domain_ids` when you want narrower search scope.

Visibility rules to remember:

- read and search results are filtered by the caller's visible owners and domains
- deleted content is excluded from normal search results
- if a domain or node is not visible, the caller may see empty results or authorization failure depending on the endpoint

## 6. Share Access Across Tenants Or Apps

Use the access endpoints when data must be shared:

1. Create a grant with `POST /v1/access/grants`.
2. Inspect grants with `GET /v1/access/grants`.
3. Revoke a grant with `DELETE /v1/access/grants/{id}`.
4. Review access outcomes with `GET /v1/access/audit`.
5. Re-check effective visibility with `GET /v1/access/resolve`.

This is the cleanest way to validate whether a consumer should be able to read or search another owner's projected data.

## 7. Validate Projection Health During Evaluation

For bootstrap evaluation and local demos:

- Use `GET /v1/kg/integrity/tenant/{tenant_id}` for tenant-level drift summary.
- Use `GET /v1/kg/integrity/missing-bridges?tenant_id=...` to inspect bridge gaps.

These routes are useful when writes succeed but read/search behavior does not look right.

If you are validating a deployed environment, pair these requests with the deployment flow in [Deployment Guide](./deployment.md) and the repeatable checks in [Testing Guide](./testing.md).

## Bootstrap Examples You Can Use Immediately

The bootstrap seed includes active templates in `sample-policy`, including:

- `action-guide`
- `topic-routing`
- `reference-check`
- `obligation-summary`
- `schedule-trace`

That makes `sample-policy` the easiest domain for end-to-end local evaluation before creating your own domain.
