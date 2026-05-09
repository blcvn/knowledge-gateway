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

## 4. API Key Lifecycle

```
[Create] → Generate random key → Prefix "sm_" → SHA-256 hash → Store hash
  ↓
[Return raw key ONCE] → Client stores securely
  ↓
[Validate] → SHA-256(input) → Compare with stored hash → Return AuthContext
  ↓
[Revoke] → Set revoked_at → Reject future validations
```

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
