---
id: DOC-S02
service: sm-project
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-project — API Reference

## gRPC Service Definition

```protobuf
service SmProjectService {
  rpc CreateSpace(CreateSpaceRequest) returns (Space);
  rpc AddToSpace(AddToSpaceRequest) returns (Empty);
  rpc ListSpaces(ListSpacesRequest) returns (ListSpacesResponse);
  rpc ManageTags(ManageTagsRequest) returns (TagsResponse);
}
```

## RPCs: CreateSpace, AddToSpace, ListSpaces, ManageTags

## NATS Events

None — entity management service.
