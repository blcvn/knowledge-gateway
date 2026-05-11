---
id: TASK-AUT-001
title: Domain Models & Core Algorithms
service: sm-auth
status: Done
priority: P0
created: 2026-05-11
---

# Domain Models & Core Algorithms

## Objective
Implement the core domain entities, value objects, and algorithms.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-auth
title: Technical Design — sm-auth
service: sm-auth
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-auth

> **Group**: Supermemory | **gRPC Port**: 9077 | **Health Port**: 9122

## 1. Service Overview

Authentication + authorization: JWT RS256, API key management (sm_ prefix, SHA-256), organization RBAC (Owner/Admin/Member), subscription tier enforcement, waitlist management.

## 2. Domain Layer

- **AuthContext**: org_id, user_id, role, permissions[], api_key_id (optional)
- **APIKey**: id, org_id, name, key_hash (SHA-256), prefix (`sm_`), permissions[], expires_at, revoked_at, last_used_at, created_at
- **Organization**: id, name, subscription_tier (api_pro|api_scale|api_enterprise), metadata
- **Role**: enum — Owner | Admin | Member
- **SubscriptionStatus**: plan_id, status (active|cancelled|past_due)
- **WaitlistEntry**: org_id, in_waitlist, access_granted, created_at

## 3. gRPC API

```protobuf
service SmAuthService {
  rpc ValidateToken(ValidateTokenRequest) returns (AuthContext);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (AuthContext);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc GetOrganization(GetOrgRequest) returns (Organization);
  rpc CheckWaitlistStatus(WaitlistRequest) returns (WaitlistStatus);
}
```

## 4. Auth & Cryptographic Algorithms

### API Key Hashing (Argon2id / SHA-256)
- **Generation**: Generate 32 bytes of secure random entropy (`crypto/rand`).
- **Encoding**: Base58 encode to generate a clean string, append `sm_` prefix.
- **Hashing**: Use SHA-256 (or Argon2id) before storing in PostgreSQL. The raw key is never stored.
- **Verification**: `hash(input_key)` matched against `stored_hash` using constant-time comparison to prevent timing attacks.

### JWT RS256 Verification
- Service downloads public JWKS from the IdP.
- Token signature is verified using `RS256` (RSA Signature with SHA-256).
- Validates `exp` (expiration), `iss` (issuer), and `aud` (audience).
- Decodes `org_id` and RBAC roles directly from the JWT claims to construct `AuthContext`.

## 5. NATS Events

| Direction | Subject | Payload |
|-----------|---------|---------|
| Publish | `sm.auth.api_key.used` | `{key_id, org_id, request_type, timestamp}` |

## 6. Data Model

### Tables
- `api_keys`: id(PK), org_id, name, key_hash(VARCHAR(64)), prefix, permissions(TEXT[]), expires_at, revoked_at, last_used_at, created_at
- `organizations`: id(PK), name, subscription_tier, metadata(JSONB), created_at, updated_at
- `org_members`: org_id(FK), user_id, role, created_at — composite PK
- `waitlist`: org_id(PK), in_waitlist, access_granted, created_at

## 7. Observability

- **Metrics**: auth_validate_total, api_key_create_total, api_key_revoke_total, auth_failures_total
- **Health**: gRPC + HTTP /healthz on port 9122

---

> **Next Steps**: FEAT-001 (JWT Validation), FEAT-002 (API Key CRUD), FEAT-003 (RBAC), SEC-001 (Key Rotation)

## Task Specs Registry

_To be populated during implementation._

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| TASK-AUT-001 | Implement Domain Models | Pending | P0 |
| TASK-AUT-002 | Implement Usecases | Pending | P0 |
| TASK-AUT-003 | Implement Adapters and Repositories | Pending | P0 |
| TASK-AUT-004 | Infrastructure and Telemetry setup | Pending | P1 |

```

## Acceptance Criteria
- [x] Domain models compile and have no external dependencies.
- [x] Core algorithms are fully implemented and unit tested.
