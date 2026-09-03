---
id: DOC-S01
service: sm-project
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-project

> **Group**: Supermemory | **gRPC Port**: 9079 | **Health Port**: 9124 | **Origin**: Supermemory

## Purpose

Spaces (containers) and membership management. A **Space** is the primary organizational unit in Supermemory — documents and memories are scoped to spaces via container tags. Manages space CRUD, tag management, member RBAC (owner/admin/editor/viewer), visibility settings, and knowledge base indexing.

### Business Capability

- **Space CRUD**: Create/list/delete spaces with container tags and visibility settings
- **Container Tags**: Flexible tagging system for scoping documents and memories
- **Membership Management**: Role-based access (owner/admin/editor/viewer) per space
- **Visibility**: public, private, unlisted spaces
- **Experimental Spaces**: Feature-flagged spaces for testing new capabilities
- **Knowledge Base Index**: Content text index per space for fast lookups

## API Surface

```protobuf
service SmProjectService {
  rpc CreateSpace(CreateSpaceRequest) returns (Space);
  rpc GetSpace(GetSpaceRequest) returns (Space);
  rpc DeleteSpace(DeleteSpaceRequest) returns (DeleteSpaceResponse);
  rpc ListSpaces(ListSpacesRequest) returns (ListSpacesResponse);
  rpc ListContainerTags(ListTagsRequest) returns (ListTagsResponse);
  rpc AddMember(AddMemberRequest) returns (google.protobuf.Empty);
  rpc RemoveMember(RemoveMemberRequest) returns (google.protobuf.Empty);
  rpc ManageTags(ManageTagsRequest) returns (TagsResponse);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v3/projects` | List all spaces/projects |
| POST | `/v3/projects` | Create space |
| DELETE | `/v3/projects/:projectId` | Delete space |
| GET | `/v3/container-tags/list` | List all container tags |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Spaces, members, tags persistence |
| sm-document | gRPC | Documents scoped to spaces |
| sm-memory | gRPC | Memories scoped to spaces |

## Owner

- **Team**: VNP Memory — Supermemory
