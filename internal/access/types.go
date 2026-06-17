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
	CreatedAt       time.Time
	RevokedAt       *time.Time
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

type GrantCreateRequest struct {
	GrantorAppID    string     `json:"grantor_app_id"`
	GranteeTenantID string     `json:"grantee_tenant_id"`
	GranteeAppID    string     `json:"grantee_app_id"`
	ScopeType       string     `json:"scope_type"`
	ScopeValue      string     `json:"scope_value"`
	Permission      string     `json:"permission"`
	ExpiresAt       *time.Time `json:"expires_at"`
}

type GrantListFilter struct {
	GrantorTenantID string
	GranteeTenantID string
}

type GrantResponse struct {
	ID              string     `json:"id"`
	GrantorTenantID string     `json:"grantor_tenant_id"`
	GrantorAppID    string     `json:"grantor_app_id,omitempty"`
	GranteeTenantID string     `json:"grantee_tenant_id"`
	GranteeAppID    string     `json:"grantee_app_id,omitempty"`
	ScopeType       string     `json:"scope_type"`
	ScopeValue      string     `json:"scope_value,omitempty"`
	Permission      string     `json:"permission"`
	Status          string     `json:"status"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
}

type AuditLogEntry struct {
	ID                    string         `json:"id"`
	ResourceOwnerTenantID string         `json:"resource_owner_tenant_id"`
	ResourceOwnerAppID    string         `json:"resource_owner_app_id,omitempty"`
	RequesterTenantID     string         `json:"requester_tenant_id"`
	RequesterAppID        string         `json:"requester_app_id"`
	Action                string         `json:"action"`
	ResourceType          string         `json:"resource_type"`
	ResourceID            string         `json:"resource_id,omitempty"`
	Outcome               string         `json:"outcome"`
	Reason                string         `json:"reason,omitempty"`
	ScopeType             string         `json:"scope_type,omitempty"`
	ScopeValue            string         `json:"scope_value,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
}

type AuditFilter struct {
	ResourceOwnerTenantID string
}
