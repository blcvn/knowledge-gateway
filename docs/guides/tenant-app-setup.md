# Tenant And App Setup

Use this guide when you need to onboard a new tenant/app pair and prepare it for KG work.

## 1. Create The Tenant

Call `POST /v1/tenants` with:

- `slug`
- `name`
- `tier`

Keep the slug stable because it is the human-friendly identifier used in documentation and scripts.

## 2. Create The App

Call `POST /v1/tenants/{tenant_id}/apps`.

The response may include a plaintext API key. Store it securely because it is shown only at creation or rotation time.

## 3. Verify Identity

Use the new key with:

```bash
curl -s \
  -H "Authorization: Bearer <api_key>" \
  http://127.0.0.1:8082/v1/access/resolve
```

This confirms the tenant and app identity the service derived from the token.

## 4. Create A Domain

Define a domain with:

- node types
- relationship types
- query templates
- lifecycle/status config when needed

The safe order is:

1. create domain
2. define schema
3. add templates
4. activate templates
5. add status/lifecycle config

## 5. Add Sharing Grants

Use `POST /v1/access/grants` when another tenant or app must read, search, or write the domain.

Common patterns:

- `scope_type=domain` for a whole domain
- `scope_type=node_type` for a narrower slice
- `scope_type=all` for platform-wide access

## 6. Validate Visibility

Re-check:

- `GET /v1/access/resolve`
- `GET /v1/tenants/{tenant_id}/ontology/effective`
- `GET /v1/kg/read/templates?domain_id=...`

If a caller cannot see something expected, the usual causes are:

- the app key is wrong or revoked
- the grant is missing
- the template is inactive
- the domain is not visible to that tenant/app

