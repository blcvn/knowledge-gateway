# vnp-admin — Data Models

> **Service**: `services/vnp-admin`
> **Role**: Multi-tenant administration — tenant lifecycle, user management, API keys, OPA policies, audit logging.

---

## Tenant

```go
type Tenant struct {
    ID        uuid.UUID    `json:"id"`
    Name      string       `json:"name"`
    Plan      Plan         `json:"plan"`
    Config    TenantConfig `json:"config"`
    Active    bool         `json:"active"`
    CreatedAt time.Time    `json:"created_at"`
    UpdatedAt time.Time    `json:"updated_at"`
}

type TenantConfig struct {
    MaxAPIKeys      int             `json:"max_api_keys"`
    MaxUsers        int             `json:"max_users"`
    EnabledEngines  []string        `json:"enabled_engines"`
    RateLimitRPM    int             `json:"rate_limit_rpm"`
    StorageQuotaMB  int64           `json:"storage_quota_mb"`
    FeatureFlags    map[string]bool `json:"feature_flags,omitempty"`
}

type Plan string
// free | starter | enterprise
```

---

## User

```go
type User struct {
    ID        uuid.UUID      `json:"id"`
    TenantID  uuid.UUID      `json:"tenant_id"`
    Email     string         `json:"email"`
    Name      string         `json:"name"`
    Role      UserRole       `json:"role"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    Active    bool           `json:"active"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type UserRole string
// owner | admin | member | viewer

type BillingEntry struct {
    ID             uuid.UUID `json:"id"`
    TenantID       uuid.UUID `json:"tenant_id"`
    Period         string    `json:"period"`          // "2026-05"
    APICallCount   int64     `json:"api_call_count"`
    StorageUsedMB  int64     `json:"storage_used_mb"`
    LLMTokensUsed  int64     `json:"llm_tokens_used"`
    CreatedAt      time.Time `json:"created_at"`
}
```

---

## APIKey

```go
type APIKey struct {
    ID        uuid.UUID  `json:"id"`
    TenantID  uuid.UUID  `json:"tenant_id"`
    Name      string     `json:"name"`
    KeyHash   string     `json:"-"`              // SHA-256 hash, never exposed
    KeyPrefix string     `json:"key_prefix"`     // First 12 chars (vnp_ prefix)
    Scope     KeyScope   `json:"scope"`
    RateLimit int        `json:"rate_limit"`     // RPM override (0 = use tenant default)
    Active    bool       `json:"active"`
    CreatedAt time.Time  `json:"created_at"`
    RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type KeyScope string
// read | read_write | admin
```

---

## Policy

```go
type Policy struct {
    ID          uuid.UUID    `json:"id"`
    TenantID    uuid.UUID    `json:"tenant_id"`
    Name        string       `json:"name"`
    Description string       `json:"description"`
    RegoCode    string       `json:"rego_code"`
    Scope       string       `json:"scope"`
    Status      PolicyStatus `json:"status"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
}

type PolicyStatus string
// active | inactive | draft
```

---

## AuditLog

```go
type AuditLog struct {
    ID           uuid.UUID      `json:"id"`
    TenantID     uuid.UUID      `json:"tenant_id"`
    UserID       string         `json:"user_id"`
    Action       AuditAction    `json:"action"`
    ResourceType string         `json:"resource_type"`
    ResourceID   string         `json:"resource_id"`
    Metadata     map[string]any `json:"metadata,omitempty"`
    IPAddress    string         `json:"ip_address,omitempty"`
    UserAgent    string         `json:"user_agent,omitempty"`
    CreatedAt    time.Time      `json:"created_at"`
}

type AuditAction string
// create | update | delete | forget | login | export

type AuditLogFilter struct {
    TenantID     uuid.UUID   `json:"tenant_id"`
    UserID       string      `json:"user_id,omitempty"`
    Action       AuditAction `json:"action,omitempty"`
    ResourceType string      `json:"resource_type,omitempty"`
    From         *time.Time  `json:"from,omitempty"`
    To           *time.Time  `json:"to,omitempty"`
    Offset       int         `json:"offset"`
    Limit        int         `json:"limit"`
}
```

---

## Sources
- [`services/vnp-admin/domain/model/tenant.go`](../../services/vnp-admin/domain/model/tenant.go)
- [`services/vnp-admin/domain/model/user.go`](../../services/vnp-admin/domain/model/user.go)
- [`services/vnp-admin/domain/model/api_key.go`](../../services/vnp-admin/domain/model/api_key.go)
- [`services/vnp-admin/domain/model/policy.go`](../../services/vnp-admin/domain/model/policy.go)
- [`services/vnp-admin/domain/model/audit_log.go`](../../services/vnp-admin/domain/model/audit_log.go)
