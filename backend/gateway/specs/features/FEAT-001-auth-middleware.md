---
id: FEAT-001
title: JWT RS256 + API Key Authentication Middleware
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
linked_sol: SOL-001
linked_prd: PRD §3.1 — Multi-tenant Authentication
---

## Mục Tiêu

Implement authentication middleware that validates JWT RS256 tokens or API keys on every request, extracting tenant context for downstream propagation.

## Bối Cảnh Nghiệp Vụ

Every API request must be authenticated to identify the tenant. Two auth methods are supported:
1. **JWT Bearer Token** — Standard for frontend/web clients
2. **API Key** — For server-to-server and SDK integrations

## Scope

### In Scope
- JWT RS256 validation (signature, expiry, issuer, audience)
- API Key resolution from PostgreSQL (via KeyStore interface)
- AuthContext population (tenant_id, user_id, roles, scopes, rate_tier)
- gRPC metadata propagation (x-tenant-id, x-user-id)
- Dev mode bypass (AUTH_DEV_MODE=true)

### Out of Scope
- JWT token issuance (handled by external IdP)
- API key creation (handled by vnp-admin service)
- OAuth2 flows

## Thiết Kế Kỹ Thuật

### API Contract
No new API endpoints. This is a middleware applied to all routes.

### Business Logic
```
1. Extract token from Authorization header OR X-API-Key header
2. If JWT:
   a. Parse + validate RS256 signature against public key
   b. Check expiry (exp), issuer (iss), audience (aud)
   c. Extract claims: tenant_id, user_id, roles, scopes
   d. Build AuthContext
3. If API Key:
   a. SHA-256 hash the key
   b. Lookup in KeyStore (PostgreSQL via cache)
   c. Check: not revoked, not expired
   d. Build AuthContext from stored metadata
4. If neither → 401 Unauthenticated
5. Set AuthContext in request context
6. Propagate tenant_id via gRPC metadata for downstream calls
```

### Internal Architecture
- `internal/infra/middleware/auth.go` — chi middleware function
- `internal/usecase/auth.go` — AuthenticateUseCase
- `internal/adapter/client/keystore.go` — KeyStore PostgreSQL implementation

## Acceptance Criteria
- [ ] AC-1: Given valid JWT, When request hits any endpoint, Then AuthContext is populated with tenant_id
- [ ] AC-2: Given valid API key (vnp_*), When request hits any endpoint, Then AuthContext is populated
- [ ] AC-3: Given expired JWT, When request hits any endpoint, Then return 401 with "token expired"
- [ ] AC-4: Given revoked API key, When request hits any endpoint, Then return 401 with "key revoked"
- [ ] AC-5: Given no auth header, When request hits any endpoint, Then return 401
- [ ] AC-6: Given AUTH_DEV_MODE=true, When request has no auth, Then use default dev tenant

## Test Requirements
- **Unit tests**: JWT parsing, API key hashing, AuthContext building
- **Integration tests**: Full middleware chain with mock KeyStore
- **Minimum coverage**: 80%

## Definition of Done
- [ ] Code implements all Acceptance Criteria
- [ ] Unit tests pass, coverage ≥ 80%
- [ ] Integration tests pass
- [ ] No lint errors
- [ ] No hardcoded secrets
