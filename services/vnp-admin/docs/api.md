---
id: DOC-S02
service: vnp-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-admin — API Reference

## gRPC Service Definition

```protobuf
service VNPAdminService {
  rpc CreateTenant(CreateTenantRequest) returns (Tenant);
  rpc GetTenant(GetTenantRequest) returns (Tenant);
  rpc UpdateTenant(UpdateTenantRequest) returns (Tenant);
  rpc ListTenants(ListTenantsRequest) returns (ListTenantsResponse);
  rpc DeleteTenant(DeleteTenantRequest) returns (google.protobuf.Empty);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc GetAggregatedHealth(google.protobuf.Empty) returns (AggregatedHealthResponse);
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
}
```

## RPC: CreateTenant

```protobuf
message CreateTenantRequest {
  string name = 1;
  string subscription_tier = 2;  // free, pro, enterprise
  map<string, string> config = 3;
}

message Tenant {
  string id = 1;
  string name = 2;
  string subscription_tier = 3;
  map<string, string> config = 4;
  google.protobuf.Timestamp created_at = 5;
}
```

## RPC: CreateAPIKey

```protobuf
message CreateAPIKeyRequest {
  string tenant_id = 1;
  string name = 2;
  repeated string permissions = 3;
  google.protobuf.Timestamp expires_at = 4;
}

message CreateAPIKeyResponse {
  string key_id = 1;
  string raw_key = 2;     // Only returned once at creation
  string tenant_id = 3;
  string name = 4;
}
```

## NATS Events Published

| Subject | Payload |
|---------|---------|
| `vnp.admin.tenant.created` | `{tenant_id, name}` |
| `vnp.admin.key.issued` | `{key_id, tenant_id}` |
| `vnp.admin.key.revoked` | `{key_id, tenant_id}` |
