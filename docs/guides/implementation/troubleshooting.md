# Troubleshooting

Use this guide for common consumer-side failures before escalating back to the kg-service team.

## Authentication Fails

Symptoms:

- `401 Unauthorized`
- error code `INVALID_API_KEY`

Checks:

- Confirm the request includes `Authorization: Bearer <api_key>`.
- Confirm you are using a current key rather than a rotated or revoked key.
- Confirm the key is one of the seeded bootstrap keys if you are testing locally.

If the key was revoked or rotated intentionally and this looks wrong, escalate to the kg-service team for operator follow-up.

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

Use [Integration Workflows](./integration.md) and [API Reference](./api-reference.md) to verify the correct sequence and route family.

## Forbidden Access

Symptoms:

- `403 Forbidden`
- empty or missing results where another identity can see data

Checks:

- Confirm you are using the expected tenant or app key.
- Call `GET /v1/access/resolve` to inspect visible owners.
- Confirm a cross-tenant or cross-app grant exists when shared visibility is expected.
- Re-check the domain or node owner scope.

If shared access should exist but still looks wrong after grant changes, escalate to the kg-service team for operator follow-up.

## Missing Resource

Symptoms:

- `404 Not Found`

Checks:

- Confirm the route path parameters are correct.
- Confirm the template is active before calling it.
- Confirm the node, job, grant, or domain identifier exists in visible scope.
- Confirm you are not using an inactive or draft-only template name.

## Search Or Read Results Look Wrong

Checks:

- Confirm the caller has visibility to the expected owners and domains.
- Confirm writes completed and produced the expected projected state for bootstrap flows.
- Confirm you are querying the correct `domain_id` or template name.
- Use the integrity endpoints to inspect drift or bridge gaps.

If results still look wrong after these checks, escalate to the kg-service team — it may require operator-side recovery or reconciliation.
