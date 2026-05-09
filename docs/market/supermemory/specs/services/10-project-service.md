# 10 — Project Service

> **gRPC**: 9009 | **Health**: 9089

---

## 1. Purpose

Project/Space management: container tag orchestration, space membership (RBAC), document-to-space assignments, visibility control, và knowledge base indexing.

---

## 2. Clean Architecture

```
services/project-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Space, SpaceMember, ContainerTag, DocumentToSpace
│   │   ├── value_object.go     # Visibility, SpaceRole, KnowledgeBase
│   │   └── errors.go           # ErrSpaceNotFound, ErrDuplicateName
│   ├── usecase/
│   │   ├── create_space.go     # Create space + containerTag sm_project_{name}
│   │   ├── list_spaces.go      # List org spaces with membership
│   │   ├── update_space.go     # Rename, visibility, settings
│   │   ├── delete_space.go     # Cascade: unlink docs, remove members
│   │   ├── manage_members.go   # Add/remove members with roles
│   │   ├── assign_document.go  # M:M document ↔ space link
│   │   ├── list_container_tags.go # List all tags for an org
│   │   ├── port/
│   │   │   ├── input.go        # CreateSpaceUC, ManageMembersUC
│   │   │   └── output.go       # SpaceRepo, MemberRepo, ContainerTagRepo
│   │   └── dto/
│   │       └── space.go        # CreateSpaceInput, SpaceOutput
│   ├── adapter/
│   │   ├── grpc/handler.go     # ProjectServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── space.go           # Space CRUD
│   │   │       ├── member.go          # Space membership
│   │   │       ├── container_tag.go   # Container tag queries
│   │   │       └── document_space.go  # M:M link table
│   │   └── event/
│   │       └── subscriber.go  # document.created → auto-assign to space
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   ├── 001_create_spaces.up.sql
│   ├── 002_create_space_members.up.sql
│   └── 003_create_documents_to_spaces.up.sql
└── Dockerfile
```

---

## 3. Domain Model

```go
type Space struct {
    ID              string
    Name            string              // max 100 chars
    OrgID           string
    CreatedBy       string
    ContainerTag    string              // sm_project_{name} (unique per org)
    Visibility      Visibility          // public | private | unlisted
    Description     *string
    ContentTextIndex *KnowledgeBase     // JSONB knowledge base
    Metadata        map[string]any
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type Visibility string
const (
    VisibilityPublic   Visibility = "public"
    VisibilityPrivate  Visibility = "private"
    VisibilityUnlisted Visibility = "unlisted"
)

type SpaceRole string
const (
    SpaceRoleOwner  SpaceRole = "owner"
    SpaceRoleEditor SpaceRole = "editor"
    SpaceRoleViewer SpaceRole = "viewer"
)

type SpaceMember struct {
    SpaceID   string
    UserID    string
    Role      SpaceRole
    JoinedAt  time.Time
}
```

---

## 4. Container Tag Convention

```go
// Auto-generated from space name
func GenerateContainerTag(spaceName string) string {
    slug := slugify(spaceName)  // lowercase, alphanum + hyphens
    return fmt.Sprintf("sm_project_%s", slug)
}

// Default tag for unscoped operations
const DefaultContainerTag = "sm_project_default"

// Validation: max 128 chars, alphanumeric + underscore + hyphen
func ValidateContainerTag(tag string) error {
    if len(tag) > 128 { return ErrTagTooLong }
    if !containerTagRegex.MatchString(tag) { return ErrInvalidTag }
    return nil
}
```

---

## 5. gRPC Interface

```protobuf
service ProjectService {
  rpc CreateSpace(CreateSpaceRequest) returns (SpaceResponse);
  rpc GetSpace(GetSpaceRequest) returns (SpaceResponse);
  rpc ListSpaces(ListSpacesRequest) returns (ListSpacesResponse);
  rpc UpdateSpace(UpdateSpaceRequest) returns (SpaceResponse);
  rpc DeleteSpace(DeleteSpaceRequest) returns (google.protobuf.Empty);
  rpc AddMember(AddMemberRequest) returns (MemberResponse);
  rpc RemoveMember(RemoveMemberRequest) returns (google.protobuf.Empty);
  rpc ListMembers(ListMembersRequest) returns (ListMembersResponse);
  rpc ListContainerTags(ListContainerTagsRequest) returns (ContainerTagsResponse);
  rpc AssignDocumentToSpace(AssignDocumentRequest) returns (google.protobuf.Empty);
  rpc RemoveDocumentFromSpace(RemoveDocumentRequest) returns (google.protobuf.Empty);
}
```
