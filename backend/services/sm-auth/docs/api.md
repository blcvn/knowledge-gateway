---
id: DOC-S02
service: sm-auth
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-auth — API Reference

## gRPC Service Definition

```protobuf
service SmAuthService {
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeRequest) returns (Empty);
  rpc CreateOrg(CreateOrgRequest) returns (Org);
  rpc GetOrg(GetOrgRequest) returns (Org);
}
```

## RPCs: CreateAPIKey, ValidateAPIKey, RevokeAPIKey, CreateOrg, GetOrg

## NATS Events

None — synchronous auth service.
