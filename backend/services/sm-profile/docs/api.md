---
id: DOC-S02
service: sm-profile
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-profile — API Reference

## gRPC Service Definition

```protobuf
service SmProfileService {
  rpc GetProfile(GetProfileRequest) returns (Profile);
  rpc UpdateProfile(UpdateProfileRequest) returns (Profile);
  rpc GetDynamicTraits(GetTraitsRequest) returns (TraitsResponse);
}
```

## RPCs: GetProfile, UpdateProfile, GetDynamicTraits

## NATS Events

Subscribed: `sm.memory.created` → update dynamic traits based on memory patterns.
