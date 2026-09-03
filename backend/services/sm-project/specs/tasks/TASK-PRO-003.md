---
id: TASK-PRO-003
title: Data Models & Repositories
service: sm-project
status: Done
priority: P0
created: 2026-05-11
---

# Data Models & Repositories

## Objective
Implement the storage and persistence adapters.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-project
title: Technical Design — sm-project
service: sm-project
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-project

> **Group**: Supermemory | **gRPC Port**: 9079 | **Health Port**: 9124

## 1. Service Overview

Spaces (containers), container tags, membership management, visibility settings. Spaces are the primary scoping unit — documents and memories belong to spaces.

## 2. Domain Layer

- **Space**: id, name, description, org_id, owner_id, container_tag, visibility (public|private|unlisted), is_experimental, content_text_index (JSONB KnowledgeBase), index_size, metadata, created_at, updated_at
- **SpaceMember**: space_id, user_id, role (owner|admin|editor|viewer), metadata, created_at, updated_at
- **ContainerTag**: tag string, space association, is_experimental, is_nova
- **Visibility**: enum — public | private | unlisted
- **SpaceRole**: enum — owner | admin | editor | viewer
- **DeleteResult**: confirmation required, cascade info

## 3. gRPC API

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

## 4. Data Model

### Tables
- `spaces`: id(PK), name, description, org_id, owner_id, container_tag(UNIQUE per org), visibility, is_experimental(BOOL), content_text_index(JSONB), index_size, metadata(JSONB), created_at, updated_at
- `spaces_to_members`: space_id(FK), user_id — composite PK, role, metadata(JSONB), created_at, updated_at

### Key Indexes
- `idx_space_org` (org_id) — list by org
- `idx_space_container_tag` UNIQUE (org_id, container_tag) — tag lookup
- `idx_member_user` (user_id) — user's spaces

## 5. Permissions Matrix & RBAC Algorithms

| Action | Owner | Admin | Editor | Viewer |
|--------|-------|-------|--------|--------|
| Read space | ✅ | ✅ | ✅ | ✅ |
| Add documents | ✅ | ✅ | ✅ | ❌ |
| Delete documents | ✅ | ✅ | ❌ | ❌ |
| Manage members | ✅ | ✅ | ❌ | ❌ |
| Delete space | ✅ | ❌ | ❌ | ❌ |
| Change settings | ✅ | ✅ | ❌ | ❌ |

### RBAC Resolution Algorithm
For any action `A` by user `U` on space `S`:
1. If `S.visibility == public`, grant `Read` to any authenticated user.
2. Lookup `Role = spaces_to_members(S.id, U.id)`.
3. If `Role` is undefined, lookup `OrgRole` from JWT context. If `OrgRole == Owner`, grant `Admin` equivalent permissions.
4. Verify `Role` satisfies the minimum required level for `A` from the Permissions Matrix.
5. If modifying members, ensure a user cannot escalate a target beyond their own role level.

## 6. Observability

- **Metrics**: space_created_total, space_deleted_total, member_added_total
- **Health**: gRPC + HTTP /healthz on port 9124

---

> **Next Steps**: FEAT-001 (Space CRUD), FEAT-002 (Membership RBAC), FEAT-003 (Container Tag Management)

## Task Specs Registry

_To be populated during implementation._

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| TASK-PRO-001 | Implement Domain Models | Pending | P0 |
| TASK-PRO-002 | Implement Usecases | Pending | P0 |
| TASK-PRO-003 | Implement Adapters and Repositories | Pending | P0 |
| TASK-PRO-004 | Infrastructure and Telemetry setup | Pending | P1 |

```

## Acceptance Criteria
- [x] Database schema / migrations created.
- [x] Repository implementations accurately query the data models.
