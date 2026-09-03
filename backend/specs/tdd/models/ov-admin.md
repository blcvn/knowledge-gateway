# ov-admin — Data Models

> **Service**: `services/ov-admin`
> **Role**: OpenViking admin service — manages accounts, users, agents, and API keys for the OpenViking memory engine.

---

## Account

```go
type Account struct {
    ID              string          `json:"id"`
    Name            string          `json:"name"`
    NamespacePolicy NamespacePolicy `json:"namespace_policy"`
    Status          string          `json:"status"` // active | suspended | deleted
    Metadata        map[string]any  `json:"metadata"`
    CreatedAt       time.Time       `json:"created_at"`
    UpdatedAt       time.Time       `json:"updated_at"`
}

type NamespacePolicy struct {
    MaxUsers       int `json:"max_users"`
    StorageQuotaMB int `json:"storage_quota_mb"`
}
```

---

## User

```go
type User struct {
    ID        string         `json:"id"`
    AccountID string         `json:"account_id"`
    Name      string         `json:"name"`
    Role      Role           `json:"role"`
    Status    string         `json:"status"` // active | suspended
    Metadata  map[string]any `json:"metadata"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type Role string
// ROOT | ADMIN | USER | AGENT
```

---

## Agent

```go
type Agent struct {
    ID        string         `json:"id"`
    UserID    string         `json:"user_id"`
    AccountID string         `json:"account_id"`
    Name      string         `json:"name"`
    Config    map[string]any `json:"config"`
    Status    string         `json:"status"` // active | disabled
    CreatedAt time.Time      `json:"created_at"`
}
```

---

## APIKey

```go
type APIKey struct {
    KeyID     string     `json:"key_id"`
    AccountID string     `json:"account_id"`
    UserID    string     `json:"user_id"`
    Role      Role       `json:"role"`
    Label     string     `json:"label"`
    Prefix    string     `json:"prefix"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    CreatedAt time.Time  `json:"created_at"`
}

type ValidateResult struct {
    Valid     bool   `json:"valid"`
    AccountID string `json:"account_id"`
    UserID    string `json:"user_id"`
    Role      Role   `json:"role"`
    AgentID   string `json:"agent_id"`
}
```

---

## NamespaceURI

```go
// NamespaceURI uniquely identifies a resource in the OpenViking namespace.
// Format: viking://{account_id}/{user_id}/{agent_id}/
type NamespaceURI struct {
    AccountID string
    UserID    string
    AgentID   string
}
```

---

## Sources
- [`services/ov-admin/internal/domain/model/account.go`](../../services/ov-admin/internal/domain/model/account.go)
- [`services/ov-admin/internal/domain/model/user.go`](../../services/ov-admin/internal/domain/model/user.go)
- [`services/ov-admin/internal/domain/model/agent.go`](../../services/ov-admin/internal/domain/model/agent.go)
- [`services/ov-admin/internal/domain/model/api_key.go`](../../services/ov-admin/internal/domain/model/api_key.go)
- [`services/ov-admin/internal/domain/model/namespace.go`](../../services/ov-admin/internal/domain/model/namespace.go)
