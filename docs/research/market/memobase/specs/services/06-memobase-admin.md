# 06 — Memobase Admin Service

> **gRPC**: 9045 | **Health**: 9095

---

## 1. Purpose

Quản lý CRUD cho Users, Projects, Billing, và Profile Configuration. Service admin chứa logic quản trị hệ thống, tách biệt khỏi data pipeline.

---

## 2. Clean Architecture

```
services/memobase-admin/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # User, Project, Billing, ProfileConfig
│   │   ├── value_object.go     # ProjectStatus, BillingPeriod, UsageStats
│   │   ├── event.go            # UserDeletedEvent, ProjectUpdatedEvent
│   │   └── errors.go           # ErrUserNotFound, ErrProjectSuspended
│   ├── usecase/
│   │   ├── create_user.go
│   │   ├── get_user.go
│   │   ├── update_user.go
│   │   ├── delete_user.go      # CASCADE: emit UserDeleted event
│   │   ├── list_project_users.go
│   │   ├── update_profile_config.go
│   │   ├── get_profile_config.go
│   │   ├── get_billing.go
│   │   ├── get_usage.go
│   │   ├── verify_project.go   # Verify project_secret for auth
│   │   ├── status_check.go     # System health aggregation
│   │   ├── port/
│   │   │   ├── input.go
│   │   │   └── output.go       # UserRepository, ProjectRepository,
│   │   │                       #   BillingRepository, EventPublisher
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go      # memobase.admin.v1.AdminService impl
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── user_repo.go      # users table
│   │   │       ├── project_repo.go   # projects table
│   │   │       └── billing_repo.go   # billings + project_billings
│   │   └── event/
│   │       └── publisher.go    # NATS: memobase.admin.*
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Entities

```go
type User struct {
    ID               uuid.UUID
    ProjectID        string
    AdditionalFields json.RawMessage
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type ProjectStatus string
const (
    ProjectActive    ProjectStatus = "active"
    ProjectSuspended ProjectStatus = "suspended"
)

type Project struct {
    ID            string
    Secret        string
    ProfileConfig string          // YAML string
    Status        ProjectStatus
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type ProfileConfig struct {
    Language          string              `yaml:"language"`
    StrictMode        bool                `yaml:"profile_strict_mode"`
    ValidateMode      bool                `yaml:"profile_validate_mode"`
    MaxSubtopics      int                 `yaml:"max_profile_subtopics"`
    MaxSlotTokenSize  int                 `yaml:"max_pre_profile_token_size"`
    AdditionalProfiles []ProfileTopicDef  `yaml:"additional_user_profiles"`
    OverwriteProfiles  []ProfileTopicDef  `yaml:"overwrite_user_profiles"`
    EventTags          []EventTagDef      `yaml:"event_tags"`
}

type Billing struct {
    ID         uuid.UUID
    UsageLeft  int64
    NextRefill time.Time
}

type UsageStats struct {
    Date             string
    TotalInsert      int64
    TotalInputToken  int64
    TotalOutputToken int64
}
```

---

## 4. Use Case Flow: DeleteUser

```
Client → Gateway → gRPC DeleteUser(user_id)
                        │
                        ▼
        ┌──── DeleteUserUseCase ───────────────┐
        │ 1. Verify user exists                 │
        │ 2. CASCADE delete in DB:              │
        │    - general_blobs                    │
        │    - buffer_zones                     │
        │    - user_profiles                    │
        │    - user_events                      │
        │    - user_event_gists                 │
        │    - user_statuses                    │
        │ 3. Emit UserDeletedEvent (NATS)       │
        │    → ingestion: cleanup pending bufs  │
        │    → context: invalidate cache        │
        │    → event: cleanup embeddings        │
        └──────────────────────────────────────┘
```

## 5. Use Case Flow: VerifyProject (Internal)

```go
// Called by gateway during auth
func (uc *VerifyProjectUseCase) Execute(ctx context.Context,
    projectID string, token string,
) (*ProjectInfo, error) {
    project, err := uc.projectRepo.GetByID(ctx, projectID)
    if err != nil { return nil, ErrProjectNotFound }

    if project.Status == ProjectSuspended {
        return nil, ErrProjectSuspended
    }

    if !verifySecret(project.Secret, token) {
        return nil, ErrInvalidSecret
    }

    return &ProjectInfo{
        ProjectID: project.ID,
        Status:    project.Status,
        Config:    parseProfileConfig(project.ProfileConfig),
    }, nil
}
```

---

## 6. NATS Events

| Subject | Payload | Subscriber |
|---------|---------|------------|
| `memobase.admin.user.deleted` | `{user_id, project_id}` | ingestion, context, event |
| `memobase.admin.project.updated` | `{project_id, config_changed}` | engine (reload config) |

---

## 7. Configuration

```yaml
admin:
  grpc:
    port: 9045
  health:
    port: 9095
  root:
    access_token: "${ROOT_ACCESS_TOKEN}"
    default_project_id: "__root__"
  nats:
    url: "nats://nats:4222"
    stream: "memobase"
  database:
    url: "${DATABASE_URL}"
    pool_size: 15
```
