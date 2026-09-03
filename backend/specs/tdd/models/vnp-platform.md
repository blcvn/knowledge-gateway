# vnp-platform — Data Models

> **Service**: `services/vnp-platform`
> **Role**: Consolidated platform service — absorbs `sm-auth`, `sm-project`, `sm-analytics`, `vnp-admin`, `ov-admin`, `zep-admin`.
> Manages auth, tenancy, users, project spaces, analytics, and event timelines.

---

## auth — Authentication & JWT

```go
type AuthUser struct {
    ID             string    `json:"id"`
    Email          string    `json:"email"`
    Name           string    `json:"name"`
    PasswordHash   string    `json:"-"`
    AuthProvider   string    `json:"auth_provider"`    // "email" | "google"
    AuthProviderID string    `json:"auth_provider_id"`
    Role           string    `json:"role"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}

type AuthToken struct {
    AccessToken string    `json:"access_token"`
    TokenType   string    `json:"token_type"`
    ExpiresAt   time.Time `json:"expires_at"`
    UserID      string    `json:"user_id"`
    Email       string    `json:"email"`
    Role        string    `json:"role"`
}

type Credentials struct {
    Email    string
    Password string
}

type Organization struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

type JWTClaims struct {
    TenantID string    `json:"tid"`
    UserID   string    `json:"uid"`
    Email    string    `json:"email"`
    Name     string    `json:"name"`
    Role     string    `json:"role"`
    Issuer   string    `json:"iss"`
    IssuedAt time.Time `json:"iat"`
    ExpireAt time.Time `json:"exp"`
}
```

---

## admin — Tenant & User Management

```go
type Tenant struct {
    ID             uuid.UUID         `json:"id"`
    Name           string            `json:"name"`
    Slug           string            `json:"slug"`
    Tier           SubscriptionTier  `json:"tier"`
    Status         TenantStatus      `json:"status"`
    Metadata       map[string]any    `json:"metadata,omitempty"`
    EngineAliases  map[string]string `json:"engine_aliases,omitempty"`
    CreatedAt      time.Time         `json:"created_at"`
    UpdatedAt      time.Time         `json:"updated_at"`
}

type SubscriptionTier string
// free | pro | enterprise

type TenantStatus string
// active | suspended | deleted

type User struct {
    ID        uuid.UUID      `json:"id"`
    TenantID  uuid.UUID      `json:"tenant_id"`
    Email     string         `json:"email"`
    Name      string         `json:"name"`
    Role      UserRole       `json:"role"`
    Active    bool           `json:"active"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type UserRole string
// admin | editor | viewer

type APIKey struct {
    ID          uuid.UUID  `json:"id"`
    TenantID    uuid.UUID  `json:"tenant_id"`
    Name        string     `json:"name"`
    KeyHash     string     `json:"-"`
    KeyPrefix   string     `json:"key_prefix"`
    Permissions []string   `json:"permissions"`
    Active      bool       `json:"active"`
    ExpiresAt   *time.Time `json:"expires_at,omitempty"`
    RevokedAt   *time.Time `json:"revoked_at,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}

type HealthStatus struct {
    Service   string        `json:"service"`
    Status    string        `json:"status"` // "SERVING" | "NOT_SERVING" | "UNKNOWN"
    Latency   time.Duration `json:"latency"`
    CheckedAt time.Time     `json:"checked_at"`
}
```

---

## project — Space Management

```go
type Space struct {
    ID        uuid.UUID      `json:"id"`
    TenantID  uuid.UUID      `json:"tenant_id"`
    Name      string         `json:"name"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type ContainerTag struct {
    ID      uuid.UUID `json:"id"`
    SpaceID uuid.UUID `json:"space_id"`
    Name    string    `json:"name"`
    Color   string    `json:"color"`
}

type Membership struct {
    ID       uuid.UUID `json:"id"`
    SpaceID  uuid.UUID `json:"space_id"`
    UserID   uuid.UUID `json:"user_id"`
    Role     string    `json:"role"` // owner | editor | viewer
    JoinedAt time.Time `json:"joined_at"`
}
```

---

## analytics — Usage Tracking

```go
type UsageRecord struct {
    ID       uuid.UUID `json:"id"`
    TenantID uuid.UUID `json:"tenant_id"`
    Engine   string    `json:"engine"`
    Endpoint string    `json:"endpoint"`
    Tokens   int64     `json:"tokens"`    // LLM tokens consumed
    Requests int64     `json:"requests"`  // API call count
    Period   string    `json:"period"`    // daily | monthly
    Date     time.Time `json:"date"`
}
```

---

## event — Timeline

```go
type UserEvent struct {
    ID        uuid.UUID      `json:"id"`
    TenantID  uuid.UUID      `json:"tenant_id"`
    UserID    uuid.UUID      `json:"user_id"`
    Engine    string         `json:"engine"`  // cognee | graphiti | memobase | openviking | zep | supermemory
    Type      EventType      `json:"type"`    // ingestion | search | memory | profile | admin
    Action    string         `json:"action"`  // created | updated | deleted | searched | ingested
    Payload   map[string]any `json:"payload,omitempty"`
    GistText  string         `json:"gist_text,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
}

type EventType string
// ingestion | search | memory | profile | admin

type Timeline struct {
    TenantID uuid.UUID   `json:"tenant_id"`
    UserID   uuid.UUID   `json:"user_id"`
    Events   []UserEvent `json:"events"`
    Total    int         `json:"total"`
}
```

---

## Sources
- [`services/vnp-platform/internal/domain/auth/entity.go`](../../services/vnp-platform/internal/domain/auth/entity.go)
- [`services/vnp-platform/internal/domain/admin/entity.go`](../../services/vnp-platform/internal/domain/admin/entity.go)
- [`services/vnp-platform/internal/domain/project/entity.go`](../../services/vnp-platform/internal/domain/project/entity.go)
- [`services/vnp-platform/internal/domain/analytics/entity.go`](../../services/vnp-platform/internal/domain/analytics/entity.go)
- [`services/vnp-platform/internal/domain/event/entity.go`](../../services/vnp-platform/internal/domain/event/entity.go)
