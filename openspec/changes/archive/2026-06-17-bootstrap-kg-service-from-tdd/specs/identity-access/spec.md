## ADDED Requirements

### Requirement: Resolve caller identity from service-managed credentials

The KG Service MUST resolve `tenant_id` and `app_id` from service-managed authentication credentials and MUST ignore caller-supplied identity values in request payloads.

#### Scenario: Active API key resolves an app identity

- **GIVEN** an app exists with an active API key
- **WHEN** the caller sends that key in the authorization header
- **THEN** the service resolves the request as the owning `tenant_id` and `app_id`
- **AND** downstream authorization uses only the resolved identity

#### Scenario: Request body identity is ignored

- **GIVEN** a caller submits `tenant_id` or `app_id` fields in the request body
- **WHEN** the service processes the request
- **THEN** those fields do not override the identity derived from credentials
- **AND** the request is evaluated using the credential owner only

#### Scenario: Revoked API key is rejected quickly

- **GIVEN** an app API key has been revoked
- **WHEN** the caller retries with the revoked key
- **THEN** the request is rejected as unauthorized
- **AND** any identity cache entry for that key is invalidated or expires within the configured short TTL

#### Scenario: `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key` invalidates the previous key

- **GIVEN** an app has an active API key
- **WHEN** an authorized admin rotates that key through `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key`
- **THEN** the service issues a new key for future use
- **AND** requests using the old key are rejected after invalidation takes effect

### Requirement: Return consistent auth and identity API responses

The KG Service MUST return consistent success and error semantics for identity and access-management endpoints.

#### Scenario: `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key` returns a new key payload

- **GIVEN** an authorized admin rotates an app key
- **WHEN** `POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key` succeeds
- **THEN** the response returns `200 OK`
- **AND** includes the new API key material and identifying metadata needed by the admin to bind it to the app

#### Scenario: Missing or invalid credentials return unauthorized errors

- **GIVEN** a caller omits the authorization header or sends an invalid key
- **WHEN** it calls a protected identity or access endpoint
- **THEN** the service returns `401 Unauthorized`
- **AND** the error body identifies authentication failure without leaking credential internals

#### Scenario: Non-admin app cannot manage another app

- **GIVEN** an authenticated caller lacks admin permission for the target tenant or app
- **WHEN** it invokes app-management or tenant-management endpoints
- **THEN** the service returns `403 Forbidden`

#### Scenario: Unknown tenant or app returns not found

- **GIVEN** the route references a tenant or app identifier that does not exist
- **WHEN** the caller invokes the corresponding management endpoint
- **THEN** the service returns `404 Not Found`

### Requirement: Enforce deny-by-default visibility

The KG Service MUST default every app to visibility over only its own owned data and explicitly shared platform or granted data.

#### Scenario: App sees its own app-owned data

- **GIVEN** an app owns private data in the KG
- **WHEN** it performs a read or search request
- **THEN** it can see data owned by that app

#### Scenario: App does not see another tenant without grant

- **GIVEN** Tenant A and Tenant B each own private data
- **WHEN** Tenant A performs read or search operations without a grant from Tenant B
- **THEN** Tenant B data is not returned

#### Scenario: Platform-public data is always visible

- **GIVEN** the platform tenant owns data marked as platform-visible
- **WHEN** any authenticated app performs read or search operations
- **THEN** that platform-visible data is included if it matches the query

### Requirement: Evaluate grants by scope and permission

The KG Service MUST evaluate access grants by grant status, expiration, scope, and permission before cross-tenant access is allowed.

#### Scenario: Expired grant no longer authorizes access

- **GIVEN** a cross-tenant grant exists with an `expires_at` in the past
- **WHEN** the grantee performs a request that depends on that grant
- **THEN** the grant is treated as inactive
- **AND** the shared data is not returned

#### Scenario: Domain-scoped grant exposes only that domain

- **GIVEN** Tenant B grants Tenant A `read` access to one domain
- **WHEN** Tenant A reads through that domain
- **THEN** Tenant A can access matching data in that domain
- **AND** data from Tenant B outside the granted domain remains hidden

#### Scenario: Write grant is required for cross-tenant writes

- **GIVEN** a caller attempts to create or update data owned outside its own tenant
- **WHEN** no active grant with `write` permission applies
- **THEN** the service rejects the request as forbidden

#### Scenario: `GET /v1/access/resolve` returns computed visibility

- **GIVEN** an authenticated app has self, platform, and grant-derived visibility
- **WHEN** it calls `GET /v1/access/resolve`
- **THEN** the response enumerates the visible owners or scopes derived from its current access state

#### Scenario: `DELETE /v1/tenants/{tenant_id}/apps/{app_id}` revokes app access immediately

- **GIVEN** an app previously had a valid key and active visibility
- **WHEN** an authorized admin deletes or revokes the app through `DELETE /v1/tenants/{tenant_id}/apps/{app_id}`
- **THEN** subsequent requests from that app are rejected
- **AND** identity cache entries for the revoked app are invalidated

### Requirement: Enforce database isolation through request-scoped RLS context

The KG Service MUST apply request-scoped tenant/app context to PostgreSQL access for protected KG tables.

#### Scenario: RLS context is set within the write transaction

- **GIVEN** a write request has been authenticated and authorized
- **WHEN** the service opens the PostgreSQL transaction
- **THEN** it sets tenant/app context for that transaction before mutating RLS-protected tables

#### Scenario: One request context does not leak into another

- **GIVEN** two requests from different apps are processed sequentially or concurrently
- **WHEN** each request reaches protected database operations
- **THEN** each request uses only its own transaction-scoped identity context
- **AND** one request cannot inherit the other's RLS context
