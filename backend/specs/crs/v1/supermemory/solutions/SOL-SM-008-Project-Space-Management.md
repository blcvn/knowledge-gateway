# Solution: SOL-SM-008 — Project & Space Management

**CR ID:** CR-SM-008  
**Solution ID:** SOL-SM-008  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/project-service/` để quản lý Projects/Spaces như namespace phân tách. Tích hợp với NATS event `document.created` để auto-assign documents vào spaces theo container tags. Thư mục này thay thế `project/` trong `vnp-platform` hiện có.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| Project domain | `services/vnp-platform/internal/domain/project/` | Minimal, chủ yếu là models cơ bản |
| Container tags | Documents, memories dùng `container_tags TEXT[]` | Đã có nhưng không được quản lý tập trung |
| `sm_project_default` | Convention | Chưa enforce |

### Gap phân tích

- Không có Space CRUD API
- Không có auto-generate container tag từ tên space
- Không có SpaceMember RBAC
- Không có tập trung quản lý container tags
- Không có Document-Space many-to-many binding

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service Mới

```
services/project-service/
├── internal/
│   ├── domain/
│   │   ├── space.go           # Space, SpaceMember entities
│   │   ├── slug.go            # ContainerTag generation
│   │   └── repository.go      # SpaceRepository port
│   ├── usecase/
│   │   ├── create_space.go    # Create + generate containerTag
│   │   ├── list_spaces.go     # List by OrgID
│   │   ├── delete_space.go    # Delete + handle docs
│   │   ├── add_member.go      # Space membership management
│   │   └── list_container_tags.go  # Flat list all tags
│   ├── adapter/
│   │   ├── grpc/              # ProjectService gRPC server
│   │   └── subscriber/
│   │       └── document_events.go  # NATS: document.created → assign to space
│   └── infra/
│       └── postgres/
│           └── space_repo.go
```

### 3.2. Domain Model

```go
// services/project-service/internal/domain/space.go

package domain

import "time"

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

type Space struct {
    ID           string
    Name         string         // max 100 chars
    OrgID        string
    CreatedBy    string
    ContainerTag string         // "sm_project_{slug}" — unique per org
    Visibility   Visibility
    Emoji        *string        // Optional emoji icon
    Description  *string
    Metadata     map[string]any
    MemberCount  int
    DocCount     int
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type SpaceMember struct {
    SpaceID  string
    UserID   string
    Role     SpaceRole
    JoinedAt time.Time
}

// Document-Space binding
type SpaceDocument struct {
    SpaceID    string
    DocumentID string
    AddedAt    time.Time
}
```

### 3.3. Container Tag Generation

```go
// services/project-service/internal/domain/slug.go

package domain

import (
    "regexp"
    "strings"
    "unicode"
    "golang.org/x/text/unicode/norm"
)

const (
    DefaultContainerTag    = "sm_project_default"
    ContainerTagPrefix     = "sm_project_"
    MaxContainerTagLength  = 128
)

// GenerateContainerTag tạo tag từ tên space
// "My Research Project" → "sm_project_my-research-project"
// "Dự Án VNP Memory"   → "sm_project_du-an-vnp-memory"
func GenerateContainerTag(spaceName string) string {
    // 1. Unicode normalization (NFD → decompose accents)
    normalized := norm.NFD.String(spaceName)

    // 2. Remove non-ASCII characters (accents, diacritics)
    var sb strings.Builder
    for _, r := range normalized {
        if unicode.Is(unicode.Mn, r) { continue } // Skip combining marks
        if r < 128 { sb.WriteRune(r) }
    }

    // 3. Lowercase
    slug := strings.ToLower(sb.String())

    // 4. Replace non-alphanumeric with hyphens
    re := regexp.MustCompile(`[^a-z0-9]+`)
    slug = re.ReplaceAllString(slug, "-")

    // 5. Trim leading/trailing hyphens
    slug = strings.Trim(slug, "-")

    // 6. Apply prefix
    tag := ContainerTagPrefix + slug

    // 7. Enforce max length
    if len(tag) > MaxContainerTagLength {
        tag = tag[:MaxContainerTagLength]
        tag = strings.TrimRight(tag, "-")
    }

    return tag
}

// ValidateContainerTag checks format constraints
func ValidateContainerTag(tag string) error {
    if len(tag) > MaxContainerTagLength {
        return ErrTagTooLong
    }
    re := regexp.MustCompile(`^[a-z0-9_-]+$`)
    if !re.MatchString(tag) {
        return ErrInvalidTagFormat
    }
    return nil
}
```

### 3.4. Create Space Use Case

```go
// services/project-service/internal/usecase/create_space.go

type CreateSpaceUseCase struct {
    repo      SpaceRepository
    publisher EventPublisher
}

func (uc *CreateSpaceUseCase) Execute(ctx context.Context, req CreateSpaceRequest) (*Space, error) {
    // 1. Generate container tag
    containerTag := GenerateContainerTag(req.Name)

    // 2. Handle conflicts: append suffix nếu đã tồn tại trong org
    if existing, _ := uc.repo.FindByContainerTag(ctx, req.OrgID, containerTag); existing != nil {
        containerTag = containerTag + "-" + generateShortID(6)
    }

    // 3. Create space
    space := &Space{
        Name:         req.Name,
        OrgID:        req.OrgID,
        CreatedBy:    req.UserID,
        ContainerTag: containerTag,
        Visibility:   VisibilityPrivate, // Default private
        Emoji:        req.Emoji,
        Description:  req.Description,
    }

    if err := uc.repo.Create(ctx, space); err != nil {
        return nil, err
    }

    // 4. Auto-add creator as owner
    uc.repo.AddMember(ctx, &SpaceMember{
        SpaceID:  space.ID,
        UserID:   req.UserID,
        Role:     SpaceRoleOwner,
        JoinedAt: time.Now(),
    })

    // 5. Publish event
    uc.publisher.Publish(ctx, "space.created", SpaceCreatedEvent{
        SpaceID:      space.ID,
        ContainerTag: space.ContainerTag,
        OrgID:        space.OrgID,
    })

    return space, nil
}
```

### 3.5. Delete Space Use Case

```go
// services/project-service/internal/usecase/delete_space.go

type DeleteSpaceOption string

const (
    DeleteOptionMoveDocs   DeleteSpaceOption = "move_to_default"
    DeleteOptionDeleteDocs DeleteSpaceOption = "delete_docs"
)

func (uc *DeleteSpaceUseCase) Execute(ctx context.Context, spaceID string, option DeleteSpaceOption) error {
    space, err := uc.repo.Get(ctx, spaceID)
    if err != nil { return err }

    switch option {
    case DeleteOptionMoveDocs:
        // Move docs to sm_project_default
        uc.docClient.BulkUpdateContainerTags(ctx, BulkUpdateTagsRequest{
            FromTag: space.ContainerTag,
            ToTag:   DefaultContainerTag,
            OrgID:   space.OrgID,
        })

    case DeleteOptionDeleteDocs:
        // Cascade delete all documents in this space
        uc.docClient.BulkDeleteByContainerTag(ctx, BulkDeleteByTagRequest{
            ContainerTag: space.ContainerTag,
            OrgID:        space.OrgID,
        })
    }

    // Delete space + members + space_documents
    return uc.repo.Delete(ctx, spaceID)
}
```

### 3.6. Document-Space Auto-Assignment

```go
// services/project-service/internal/adapter/subscriber/document_events.go

type DocumentEventSubscriber struct {
    nats      NATSClient
    spaceRepo SpaceRepository
}

// Subscribe document.created → auto-assign document vào spaces theo containerTags
func (s *DocumentEventSubscriber) Start(ctx context.Context) {
    s.nats.Subscribe(ctx, "document.created", func(msg DocumentCreatedEvent) {
        for _, tag := range msg.ContainerTags {
            space, err := s.spaceRepo.FindByContainerTag(ctx, msg.OrgID, tag)
            if err != nil { continue }

            // Tạo SpaceDocument binding
            s.spaceRepo.AddDocument(ctx, &SpaceDocument{
                SpaceID:    space.ID,
                DocumentID: msg.DocumentID,
                AddedAt:    time.Now(),
            })
        }
    })
}
```

### 3.7. SpaceRole Permission Check

```go
// services/project-service/internal/domain/space.go

// SpaceRole permissions (scoped to space, không phải org-level RBAC)
func CanAddDocumentToSpace(role SpaceRole) bool {
    return role == SpaceRoleOwner || role == SpaceRoleEditor
}

func CanManageMembers(role SpaceRole) bool {
    return role == SpaceRoleOwner
}

// Middleware: check space member role trước khi cho phép thêm document
func (h *SpaceHandler) checkSpacePermission(spaceID, userID string, check func(SpaceRole) bool) error {
    member, err := h.spaceRepo.GetMember(ctx, spaceID, userID)
    if err != nil { return ErrNotSpaceMember }
    if !check(member.Role) { return ErrForbidden }
    return nil
}
```

---

## 4. Database Schema

```sql
CREATE TABLE spaces (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL CHECK (length(name) <= 100),
    org_id        UUID NOT NULL,
    created_by    UUID NOT NULL,
    container_tag TEXT NOT NULL,
    visibility    TEXT NOT NULL DEFAULT 'private',
    emoji         TEXT,
    description   TEXT,
    metadata      JSONB DEFAULT '{}',
    member_count  INT DEFAULT 1,
    doc_count     INT DEFAULT 0,
    created_at    TIMESTAMPTZ DEFAULT now(),
    updated_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE (org_id, container_tag)
);

CREATE TABLE space_members (
    space_id  UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL,
    role      TEXT NOT NULL DEFAULT 'viewer',
    joined_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (space_id, user_id)
);

CREATE TABLE space_documents (
    space_id    UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    document_id UUID NOT NULL,
    added_at    TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (space_id, document_id)
);

-- Indexes
CREATE INDEX idx_spaces_org ON spaces(org_id);
CREATE INDEX idx_space_members_user ON space_members(user_id);
CREATE INDEX idx_space_docs_doc ON space_documents(document_id);
```

---

## 5. API Endpoints (Gateway)

```go
// gateway/adapter/handler/space_handler.go

func (h *SpaceHandler) Register(mux *http.ServeMux) {
    // Project/Space CRUD
    mux.HandleFunc("GET /api/v1/projects", h.ListProjects)
    mux.HandleFunc("POST /api/v1/projects", h.CreateProject)
    mux.HandleFunc("GET /api/v1/projects/{id}", h.GetProject)
    mux.HandleFunc("PATCH /api/v1/projects/{id}", h.UpdateProject)
    mux.HandleFunc("DELETE /api/v1/projects/{id}", h.DeleteProject) // ?option=move_to_default|delete_docs

    // Member management
    mux.HandleFunc("GET /api/v1/projects/{id}/members", h.ListMembers)
    mux.HandleFunc("POST /api/v1/projects/{id}/members", h.AddMember)
    mux.HandleFunc("DELETE /api/v1/projects/{id}/members/{userId}", h.RemoveMember)

    // Container tags
    mux.HandleFunc("GET /api/v1/container-tags/list", h.ListContainerTags)
}
```

**Create project request:**
```json
{
  "name": "VNP Memory",
  "emoji": "🧠",
  "description": "Knowledge gateway project",
  "visibility": "private"
}
```

**Create project response:**
```json
{
  "id": "space_abc123",
  "name": "VNP Memory",
  "containerTag": "sm_project_vnp-memory",
  "emoji": "🧠",
  "visibility": "private",
  "memberCount": 1,
  "docCount": 0,
  "createdAt": "2026-06-17T00:00:00Z"
}
```

**List container tags:**
```json
{
  "tags": [
    "sm_project_default",
    "sm_project_vnp-memory",
    "sm_project_research",
    "sm_project_client-acme"
  ]
}
```

---

## 6. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + slug generator | 1 ngày |
| **P2** | DB schema + Space CRUD | 1 ngày |
| **P3** | Space membership (add/remove + role check) | 1 ngày |
| **P4** | Delete space (move/delete docs option) | 1 ngày |
| **P5** | Document-Space auto-assignment (NATS subscriber) | 1 ngày |
| **P6** | Container tags listing API | 0.5 ngày |
| **P7** | Gateway integration + REST handlers | 1 ngày |
| **P8** | Tests + Acceptance Criteria | 1 ngày |

**Tổng:** ~7.5 ngày (Wave 1 — cùng với CR-SM-007)

---

## 7. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| "My Research" → `sm_project_my-research` | GenerateContainerTag() slugify |
| doc với tag `sm_project_my-research` → trong project search | Space-Document binding + search filter |
| Xóa project → docs move to default | DeleteSpaceUseCase với option=move_to_default |
| MCP `listProjects` → đúng danh sách | ListSpaces → containerTags → MCP tool |
| Viewer không thể thêm document | CanAddDocumentToSpace(SpaceRoleViewer) = false |
