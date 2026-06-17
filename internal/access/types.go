package access

import "time"

const PlatformTenantID = "00000000-0000-0000-0000-000000000000"

type Tenant struct {
	ID                   string
	Slug                 string
	Name                 string
	Status               string
	Tier                 string
	DefaultSharingPolicy string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type App struct {
	ID           string
	TenantID     string
	Slug         string
	Name         string
	Type         string
	APIKeyHash   string
	APIKeyPrefix string
	Status       string
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

type AccessGrant struct {
	ID              string
	GrantorTenantID string
	GrantorAppID    string
	GranteeTenantID string
	GranteeAppID    string
	ScopeType       string
	ScopeValue      string
	Permission      string
	Status          string
	ExpiresAt       *time.Time
}

type Identity struct {
	TenantID string `json:"tenant_id"`
	AppID    string `json:"app_id"`
	AppType  string `json:"-"`
}

type VisibleOwner struct {
	TenantID   string `json:"tenant_id"`
	AppID      string `json:"app_id,omitempty"`
	Permission string `json:"permission,omitempty"`
	ScopeType  string `json:"scope_type,omitempty"`
	ScopeValue string `json:"scope_value,omitempty"`
	Source     string `json:"source"`
}

type ResolveResponse struct {
	TenantID      string         `json:"tenant_id"`
	AppID         string         `json:"app_id"`
	VisibleOwners []VisibleOwner `json:"visible_owners"`
}

type TenantCreateRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Tier string `json:"tier"`
}

type TenantUpdateRequest struct {
	Tier                 string `json:"tier"`
	DefaultSharingPolicy string `json:"default_sharing_policy"`
}

type TenantResponse struct {
	ID                   string    `json:"id"`
	Slug                 string    `json:"slug"`
	Name                 string    `json:"name"`
	Status               string    `json:"status"`
	Tier                 string    `json:"tier"`
	DefaultSharingPolicy string    `json:"default_sharing_policy"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

type AppCreateRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type AppResponse struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id,omitempty"`
	Slug      string     `json:"slug"`
	Name      string     `json:"name,omitempty"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	APIKey    string     `json:"api_key,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type RotateKeyResponse struct {
	APIKey    string    `json:"api_key"`
	RotatedAt time.Time `json:"rotated_at"`
}
