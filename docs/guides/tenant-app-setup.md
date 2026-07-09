# Tenant And App Setup

Use this guide when you need to onboard a new tenant/app pair and prepare it for KG work.

## Local Seed Values

The local bootstrap data already includes these identities:

- Platform tenant ID: `00000000-0000-0000-0000-000000000000`
- Platform admin app ID: `00000000-0000-4000-8000-000000000001`
- Platform admin API key: `kgsk_platform_admin`
- Test Alpha tenant ID: `11111111-1111-1111-1111-111111111111`
- Test Alpha tenant slug: `test-alpha`
- Test Alpha admin app ID: `11111111-1111-4111-8111-111111111111`
- Test Alpha admin API key: `kgsk_test_alpha_admin`
- Test Alpha app ID: `11111111-aaaa-1111-aaaa-111111111111`
- Test Alpha app API key: `kgsk_test_alpha`
- Test Beta tenant ID: `22222222-2222-2222-2222-222222222222`
- Test Beta tenant slug: `test-beta`
- Test Beta app ID: `22222222-bbbb-2222-bbbb-222222222222`
- Test Beta app API key: `kgsk_test_beta`

## 1. Create The Tenant

Call `POST /v1/tenants` with:

- `slug`
- `name`
- `tier`

Keep the slug stable because it is the human-friendly identifier used in documentation and scripts.

## 2. Create The App

Call `POST /v1/tenants/{tenant_id}/apps`.

The response may include a plaintext API key. Store it securely because it is shown only at creation or rotation time.

An app created through this route is the tenant-owned write path for that tenant. If you plan to
write into a foreign-owned or platform-owned domain, you still need an explicit grant for that
owner and scope.

## 3. Verify Identity

Use the new key with:

```bash
curl -s \
  -H "Authorization: Bearer <api_key>" \
  http://127.0.0.1:8082/v1/access/resolve
```

This confirms the tenant and app identity the service derived from the token.

If the response is not what you expect, compare it with the seeded IDs above before debugging
tenant/app routes.

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

Keep the ownership model straight:

- `visible domain` means the caller can see the domain in effective ontology or resolve output
- `writable domain` means the caller can actually write into that domain
- platform-owned domains are visible by default but are not tenant-writable without a matching
  `write` or `admin` grant
- tenant-owned domains are writable by the owning tenant when the ontology schema also validates
- cross-tenant shared domains require an explicit write-capable grant before they are treated as
  writable

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

If a caller can see a domain but write attempts still fail, check whether the domain is owned by a
different tenant and whether the caller has a matching `write` or `admin` grant.

## Tenant/App 404 Checklist

When `GET /v1/tenants/{tenant_id}` or `/v1/tenants/{tenant_id}/apps` returns `404`, work through
this order:

1. Call `GET /v1/access/resolve` first and confirm the caller identity.
2. Use the seeded tenant ID, not the slug, in `/v1/tenants/{tenant_id}`.
3. Watch the server logs:
   - `route miss ...` means the router never matched the path.
   - `access route debug ...` means the route matched but the handler saw an empty path value or
     the service returned `ErrNotFound`.
4. Use a tenant-admin or platform-admin token when testing tenant management routes.
