# Troubleshooting

Use this guide for common consumer-side failures before escalating to operator runbooks.

## Authentication Fails

Symptoms:

- `401 Unauthorized`
- error code `INVALID_API_KEY`

Checks:

- Confirm the request includes `Authorization: Bearer <api_key>`.
- Confirm you are using a current key rather than a rotated or revoked key.
- Confirm the key is one of the seeded bootstrap keys if you are testing locally.

If the key was revoked or rotated intentionally, use [API Key Revocation Response](../operations/api-key-revocation-response.md) only when operator follow-up is needed.

## Malformed JSON

Symptoms:

- `400 Bad Request`
- message like `Malformed JSON body`

Checks:

- Confirm the body is valid JSON.
- Confirm `Content-Type: application/json` is set on JSON requests.
- Confirm you did not leave trailing commas or send form-encoded payloads to JSON endpoints.

## Validation Fails

Symptoms:

- `422 Unprocessable Entity`
- error code `VALIDATION_FAILED`

Checks:

- Confirm required fields are present.
- Confirm field types match the endpoint contract.
- Confirm ontology artifacts exist before write requests depend on them.
- Confirm template parameters match the active template schema.

Use [Integration Workflows](./integration.md) and [API Reference](../api/README.md) to verify the correct sequence and route family.

## Control Plane Is Not Ready

Symptoms:

- `503 Service Unavailable`
- error code `SERVICE_UNAVAILABLE`
- message like `Control plane is not ready`

Checks:

- Confirm tenant and app provisioning completed successfully in the relationship DB-backed access plane.
- Confirm ontology domains, node types, relationship types, templates, and status configs exist in relationship DB before the first write.
- Confirm you are using the API key returned by the same app create flow, not a copied or stale key.
- Confirm Redis is only serving cache state; it should not be the only place where tenant/app or ontology data exist.
- If the service recently restarted, verify the durable rows still exist and the app was not depending on process-local memory.

This status means the service cannot prove the durable control-plane state needed for a safe write. It is not a caller payload error.

## Forbidden Access

Symptoms:

- `403 Forbidden`
- empty or missing results where another identity can see data

Checks:

- Confirm you are using the expected tenant or app key.
- Call `GET /v1/access/resolve` to inspect visible owners.
- Confirm the domain is actually writable by that tenant or app.
- For shared-domain writes, confirm a cross-tenant `write` or `admin` grant exists for the
  relevant owner and scope.
- Re-check the domain or node owner scope.
- If the write failed with `503 Service Unavailable`, follow the control-plane checks above before
  treating it as a payload or permission issue.

If shared access should exist but still looks wrong after grant changes, operator follow-up may involve [Grant Incident Response](../operations/grant-incident-response.md).

## Missing Resource

Symptoms:

- `404 Not Found`

Checks:

- Confirm the route path parameters are correct.
- For tenant/app routes, prefer the seeded tenant UUID over the slug. For example, use
  `11111111-1111-1111-1111-111111111111` instead of `test-alpha`.
- Call `GET /v1/access/resolve` first so you know which tenant and app the bearer token resolved
  to.
- For write failures against a visible domain, confirm you are not targeting a platform-owned or
  foreign-owned domain without an explicit grant.
- For tenant-owned write bootstraps, confirm the domain was created under the tenant that will own
  the writes.
- Check the logs:
  - `route miss ...` means the router did not match the request.
  - `access route debug ...` means the route matched but the handler saw an empty `PathValue` or
    the service returned `ErrNotFound`.
- Confirm the template is active before calling it.
- Confirm the node, job, grant, or domain identifier exists in visible scope.
- Confirm you are not using an inactive or draft-only template name.

## Search Or Read Results Look Wrong

Checks:

- Confirm the caller has visibility to the expected owners and domains.
- Confirm writes completed and produced the expected projected state for bootstrap flows.
- Confirm the difference between visible domains and writable domains when shared data is
  intentionally read-only.
- Confirm you are querying the correct `domain_id` or template name.
- Use the integrity endpoints to inspect drift or bridge gaps.

Operator follow-up may require [Replica Recovery](../operations/replica-recovery.md) or [Reconciliation Incident Handling](../operations/reconciliation-incident-handling.md).
