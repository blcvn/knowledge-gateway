# Change Request: CR-SM-008 — Project & Space Management

**CR ID:** CR-SM-008  
**Component:** `services/project-service` [NEW SERVICE]  
**Priority:** Medium  
**Status:** In Progress
**Reference:** Supermemory PRD §4.2, SRS §2.10, specs/services/10-project-service.md

---

## 1. Mô tả

Xây dựng **Project Service** — quản lý Projects/Spaces như namespace phân tách cho memories:

1. **Space CRUD**: Tạo, đọc, cập nhật, xóa spaces (projects).
2. **Container Tag Convention**: Auto-generate `sm_project_{name}` từ tên space.
3. **Space Membership**: Thêm/xóa members với SpaceRole (owner, editor, viewer).
4. **Document-Space Binding**: Gán document vào nhiều space (many-to-many).
5. **Container Tag Listing**: Liệt kê tất cả container tags cho MCP project selection.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có khái niệm "Space" hay "Project" để phân tách memory theo chủ đề/project.
- Thiếu namespace isolation giữa các user hoặc các project khác nhau.
- Container tags chưa được quản lý tập trung.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/project-service/` (Port gRPC: 9009)

### 3.2. Domain Model

```go
type Space struct {
    ID           string
    Name         string          // max 100 chars
    OrgID        string
    CreatedBy    string
    ContainerTag string          // "sm_project_{slug}" — unique per org
    Visibility   Visibility      // public | private | unlisted
    Description  *string
    Metadata     map[string]any
    CreatedAt    time.Time
}

type SpaceMember struct {
    SpaceID  string
    UserID   string
    Role     SpaceRole   // owner | editor | viewer
    JoinedAt time.Time
}
```

### 3.3. Container Tag Convention

```go
// Auto-generated slug từ tên space
func GenerateContainerTag(spaceName string) string {
    slug := slugify(spaceName)  // lowercase, alphanum + hyphens
    return fmt.Sprintf("sm_project_%s", slug)
}

const DefaultContainerTag = "sm_project_default"
// Validation: max 128 chars, alphanumeric + underscore + hyphen
```

### 3.4. Document-Space Many-to-Many

- Mỗi document có thể thuộc nhiều space.
- Khi tạo document với `containerTags`, auto-assign vào các spaces tương ứng.
- Subscribe event `document.created` để auto-assign.

### 3.5. API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/projects` | Liệt kê projects của org |
| `POST` | `/api/v1/projects` | Tạo project mới `{name, emoji?}` |
| `DELETE` | `/api/v1/projects/:id` | Xóa project (move docs hoặc delete all) |
| `GET` | `/api/v1/container-tags/list` | Flat list tất cả container tags |

**Create project response:**
```json
{
  "id": "space_123",
  "name": "VNP Memory",
  "containerTag": "sm_project_vnp-memory",
  "emoji": "🧠"
}
```

---

## 4. Acceptance Criteria

- [ ] Tạo project "My Research" → containerTag tự động là `sm_project_my-research`.
- [ ] Tạo document với `containerTags: ["sm_project_my-research"]` → document xuất hiện trong project search.
- [ ] Xóa project với option "move docs to default" → docs còn lại trong `sm_project_default`.
- [ ] MCP tool `listProjects` hiển thị đúng danh sách projects.
- [ ] Member `viewer` của space không thể thêm document vào space đó.
