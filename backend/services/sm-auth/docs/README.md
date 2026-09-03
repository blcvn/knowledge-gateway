---
id: DOC-S01
service: sm-auth
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-auth

> **Group**: Supermemory | **gRPC Port**: 9077 | **Health Port**: 9122 | **Origin**: Supermemory

## Purpose

Authentication and authorization service for Supermemory. Manages **JWT RS256 tokens**, **API key lifecycle** (`sm_` prefix, SHA-256 hashed), **organization-level RBAC** (Owner/Admin/Member), and **organization management** with subscription tier enforcement.

### Business Capability

- **JWT Authentication**: RS256 signed tokens with org_id + user_id claims
- **API Key Management**: Generate (`sm_` prefixed) → hash (SHA-256) → store → validate → revoke
- **Organization RBAC**: Owner, Admin, Member roles with permission matrices
- **Subscription Tiers**: api_pro, api_scale, api_enterprise with feature gating
- **Waitlist Management**: Access control for new organizations
- **Usage Tracking**: API key usage events emitted to sm-analytics

## API Surface

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

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | API keys, orgs, roles persistence |
| sm-analytics | NATS pub | `sm.auth.api_key.used` → usage tracking |
| vnp-gateway | gRPC | Token/key validation on every request |

## Owner

- **Team**: VNP Memory — Supermemory
