# Solution: SOL-SM-007 — Auth Service & Organization RBAC

**CR ID:** CR-SM-007  
**Solution ID:** SOL-SM-007  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Nâng cấp Auth trong `vnp-platform/` để thêm Organization management, RBAC 4-role với permission matrix, `sm_` prefixed API keys, và OAuth2 Authorization Server. Tất cả thay đổi tương thích ngược với JWT RS256 hiện có.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| Auth domain | `services/vnp-platform/internal/domain/auth/` | JWT + API Key cơ bản |
| Admin domain | `services/vnp-platform/internal/domain/admin/` | Tenant, User, APIKey |
| `APIKey` entity | | ID, TenantID, KeyHash(SHA-256), Permissions, ExpiresAt |
| `User` entity | | ID, TenantID, Email, Role(admin\|editor\|viewer) |
| JWT RS256 validation | `gateway/infra/middleware/` | Đang hoạt động |

### Gap phân tích

- `User.Role` chỉ có 3 values, thiếu `owner`
- API keys không có `sm_` prefix
- Thiếu Organization model (multi-user org, invitation system)
- Thiếu RBAC permission matrix chi tiết
- Thiếu OAuth2 Authorization Server
- Thiếu Redis cache cho token validation (hot path)

---

## 3. Thiết kế Giải pháp

### 3.1. API Key Format (sm_ prefix)

```go
// services/vnp-platform/internal/domain/auth/apikey.go

package auth

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "math/big"
)

const base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// Format: sm_<32 random bytes base62>
// Example: sm_a7Kj3mN2pQ9xR4vW8yB5cD0eF6gH1iL3oP
func GenerateAPIKey() (plaintext, hash string) {
    raw := make([]byte, 32)
    rand.Read(raw)

    // Base62 encoding
    n := new(big.Int).SetBytes(raw)
    base := big.NewInt(62)
    var encoded []byte
    for n.Sign() > 0 {
        mod := new(big.Int)
        n.DivMod(n, base, mod)
        encoded = append(encoded, base62Chars[mod.Int64()])
    }
    // Pad to 43 chars (32 bytes → ~43 base62 chars)
    for len(encoded) < 43 {
        encoded = append(encoded, base62Chars[0])
    }
    // Reverse
    for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
        encoded[i], encoded[j] = encoded[j], encoded[i]
    }

    plaintext = "sm_" + string(encoded)
    h := sha256.Sum256([]byte(plaintext))
    hash = fmt.Sprintf("%x", h)
    return
}

// Validate format
func IsValidAPIKeyFormat(key string) bool {
    if len(key) < 46 { return false }       // "sm_" + 43 chars minimum
    if key[:3] != "sm_" { return false }
    for _, c := range key[3:] {
        if !strings.ContainsRune(base62Chars, c) { return false }
    }
    return true
}
```

### 3.2. RBAC Role & Permission System

```go
// services/vnp-platform/internal/domain/auth/rbac.go

package auth

type Role string

const (
    RoleOwner  Role = "owner"   // Full access
    RoleAdmin  Role = "admin"   // Most operations
    RoleEditor Role = "editor"  // Create/delete content, search
    RoleViewer Role = "viewer"  // Read-only
)

type Permission string

const (
    PermDocumentCreate   Permission = "document:create"
    PermDocumentDelete   Permission = "document:delete"
    PermMemoryForget     Permission = "memory:forget"
    PermSearchExecute    Permission = "search:execute"
    PermConnectionCreate Permission = "connection:create"
    PermSettingsWrite    Permission = "settings:write"
    PermMemberManage     Permission = "member:manage"
    PermAPIKeyManage     Permission = "apikey:manage"
    PermAnalyticsRead    Permission = "analytics:read"
)

// Permission matrix theo CR-SM-007 spec
var permissionMatrix = map[Role]map[Permission]bool{
    RoleOwner: {
        PermDocumentCreate:   true,
        PermDocumentDelete:   true,
        PermMemoryForget:     true,
        PermSearchExecute:    true,
        PermConnectionCreate: true,
        PermSettingsWrite:    true,
        PermMemberManage:     true,
        PermAPIKeyManage:     true,
        PermAnalyticsRead:    true,
    },
    RoleAdmin: {
        PermDocumentCreate:   true,
        PermDocumentDelete:   true,
        PermMemoryForget:     true,
        PermSearchExecute:    true,
        PermConnectionCreate: true,
        PermSettingsWrite:    false,
        PermMemberManage:     true,
        PermAPIKeyManage:     true,
        PermAnalyticsRead:    true,
    },
    RoleEditor: {
        PermDocumentCreate:   true,
        PermDocumentDelete:   true,
        PermMemoryForget:     true,
        PermSearchExecute:    true,
        PermConnectionCreate: true,
        PermSettingsWrite:    false,
        PermMemberManage:     false,
        PermAPIKeyManage:     false,
        PermAnalyticsRead:    false,
    },
    RoleViewer: {
        PermDocumentCreate:   false,
        PermDocumentDelete:   false,
        PermMemoryForget:     false,
        PermSearchExecute:    true,
        PermConnectionCreate: false,
        PermSettingsWrite:    false,
        PermMemberManage:     false,
        PermAPIKeyManage:     false,
        PermAnalyticsRead:    true,
    },
}

func HasPermission(role Role, perm Permission) bool {
    if perms, ok := permissionMatrix[role]; ok {
        return perms[perm]
    }
    return false
}
```

### 3.3. Organization Model

```go
// services/vnp-platform/internal/domain/admin/organization.go

package admin

import "time"

type Organization struct {
    ID          string
    Name        string
    Slug        string         // URL-friendly name, unique
    OwnerUserID string
    Plan        Plan           // free | pro | enterprise
    Settings    OrgSettings
    CreatedAt   time.Time
}

type OrgSettings struct {
    MaxMembers    int
    MaxAPIKeys    int
    MaxConnections int
    CustomOAuth   bool  // Enterprise only
}

type OrgMember struct {
    OrgID     string
    UserID    string
    Role      Role      // owner | admin | editor | viewer
    InvitedBy string
    JoinedAt  time.Time
}

type OrgInvitation struct {
    ID           string
    OrgID        string
    Email        string
    Role         Role
    Token        string      // Secure random token
    ExpiresAt    time.Time   // 7 days
    CreatedBy    string
    AcceptedAt   *time.Time
}
```

### 3.4. Organization API

```go
// gateway/adapter/handler/org_handler.go

func (h *OrgHandler) Register(mux *http.ServeMux) {
    // Org management
    mux.HandleFunc("POST /api/v1/auth/organizations", h.Create)
    mux.HandleFunc("GET /api/v1/auth/organizations/{id}", h.Get)

    // Member management (requires member:manage permission)
    mux.HandleFunc("POST /api/v1/auth/organizations/{id}/members", h.AddMember)
    mux.HandleFunc("DELETE /api/v1/auth/organizations/{id}/members/{userId}", h.RemoveMember)

    // Invitation flow
    mux.HandleFunc("POST /api/v1/auth/organizations/{id}/invitations", h.InviteMember)
    mux.HandleFunc("POST /api/v1/auth/invitations/{token}/accept", h.AcceptInvitation)
}

// RBAC middleware enforcement
func (h *OrgHandler) AddMember(w http.ResponseWriter, r *http.Request) {
    authCtx := getAuthContext(r)
    if !HasPermission(authCtx.Role, PermMemberManage) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    // ... handle
}
```

### 3.5. Token Validation Cache (Redis)

```go
// services/vnp-platform/internal/usecase/auth/validate_token.go

type ValidateTokenUseCase struct {
    apiKeyRepo APIKeyRepository
    jwtParser  JWTParser
    cache      ValidationCache  // Redis
}

// Cache key: "auth:apikey:<sha256_hash>"
// Cache TTL: 5 phút
// Cache value: {orgID, userID, role, permissions, expiresAt}

func (uc *ValidateTokenUseCase) ValidateAPIKey(ctx context.Context, plaintext string) (*AuthContext, error) {
    // 1. Validate format
    if !IsValidAPIKeyFormat(plaintext) {
        return nil, ErrInvalidKeyFormat
    }

    // 2. Hash key
    h := sha256.Sum256([]byte(plaintext))
    hash := fmt.Sprintf("%x", h)

    // 3. Check Redis cache (hot path: ~1ms)
    cacheKey := fmt.Sprintf("auth:apikey:%s", hash)
    if cached, err := uc.cache.Get(ctx, cacheKey); err == nil {
        return cached, nil
    }

    // 4. Cache miss → DB lookup
    apiKey, err := uc.apiKeyRepo.FindByHash(ctx, hash)
    if err != nil { return nil, ErrInvalidAPIKey }

    // 5. Check expiry
    if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
        return nil, ErrExpiredAPIKey
    }

    // 6. Load org member role
    member, _ := uc.orgMemberRepo.Get(ctx, apiKey.OrgID, apiKey.UserID)

    authCtx := &AuthContext{
        OrgID:  apiKey.OrgID,
        UserID: apiKey.UserID,
        Role:   member.Role,
        KeyID:  apiKey.ID,
    }

    // 7. Cache kết quả 5 phút
    uc.cache.Set(ctx, cacheKey, authCtx, 5*time.Minute)

    return authCtx, nil
}
```

### 3.6. OAuth2 Authorization Server

```go
// services/vnp-platform/internal/adapter/oauth2/server.go

// Triển khai OAuth2 Authorization Code flow chuẩn (RFC 6749)
// Dùng cho MCP clients (Claude Desktop, Cursor, etc.)

// Endpoints:
// GET  /.well-known/oauth-authorization-server  → Metadata discovery
// GET  /api/v1/auth/oauth/authorize             → Authorization endpoint
// POST /api/v1/auth/oauth/token                 → Token exchange

type OAuthServer struct {
    clientRepo   OAuthClientRepository
    codeStore    AuthCodeStore     // Redis: code → {clientID, userID, scope, expires}
    tokenSigner  TokenSigner       // RS256 JWT signer
}

// Metadata discovery response
type OAuthMetadata struct {
    Issuer                             string   `json:"issuer"`
    AuthorizationEndpoint              string   `json:"authorization_endpoint"`
    TokenEndpoint                      string   `json:"token_endpoint"`
    ResponseTypesSupported             []string `json:"response_types_supported"`
    GrantTypesSupported                []string `json:"grant_types_supported"`
    TokenEndpointAuthMethodsSupported  []string `json:"token_endpoint_auth_methods_supported"`
    CodeChallengeMethodsSupported      []string `json:"code_challenge_methods_supported"` // PKCE
}

func (s *OAuthServer) Metadata() *OAuthMetadata {
    return &OAuthMetadata{
        Issuer:                "https://api.vnpmemory.io",
        AuthorizationEndpoint: "https://api.vnpmemory.io/api/v1/auth/oauth/authorize",
        TokenEndpoint:         "https://api.vnpmemory.io/api/v1/auth/oauth/token",
        ResponseTypesSupported: []string{"code"},
        GrantTypesSupported:   []string{"authorization_code", "refresh_token"},
        CodeChallengeMethodsSupported: []string{"S256"}, // PKCE required
    }
}

// Authorization endpoint: renders login page or redirects if already authenticated
func (s *OAuthServer) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
    // Validate client_id, redirect_uri, response_type=code
    // Validate PKCE code_challenge (required for public clients like MCP)
    // Generate auth code (16 random bytes, expires 10 phút)
    // Store: code → {clientID, userID, redirectURI, codeChallenge, scope}
    // Redirect to redirect_uri?code=xxx&state=yyy
}

// Token endpoint: exchange code for access_token + refresh_token
func (s *OAuthServer) HandleToken(w http.ResponseWriter, r *http.Request) {
    // Validate grant_type=authorization_code
    // Validate code + code_verifier (PKCE)
    // Generate JWT access_token (RS256, expires 1h)
    // Generate refresh_token (expires 30d)
    // Return: {access_token, token_type: "bearer", expires_in: 3600, refresh_token}
}
```

---

## 4. Database Schema (Additions)

```sql
-- organizations table
CREATE TABLE organizations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    slug         TEXT NOT NULL UNIQUE,
    owner_id     UUID NOT NULL REFERENCES users(id),
    plan         TEXT NOT NULL DEFAULT 'free',
    settings     JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT now()
);

-- org_members table
CREATE TABLE org_members (
    org_id     UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'viewer',
    invited_by UUID,
    joined_at  TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (org_id, user_id)
);

-- org_invitations table
CREATE TABLE org_invitations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email       TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'viewer',
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_by  UUID NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- oauth2_clients table (cho MCP clients)
CREATE TABLE oauth2_clients (
    id           TEXT PRIMARY KEY,  -- client_id
    name         TEXT NOT NULL,
    secret_hash  TEXT,              -- Null for public clients (PKCE)
    redirect_uris TEXT[] NOT NULL,
    scopes       TEXT[] DEFAULT '{}',
    is_public    BOOLEAN DEFAULT true,  -- Public clients use PKCE
    org_id       UUID,
    created_at   TIMESTAMPTZ DEFAULT now()
);

-- Nâng cấp api_keys table
ALTER TABLE api_keys 
    ADD COLUMN org_id UUID REFERENCES organizations(id),
    ADD COLUMN role TEXT DEFAULT 'editor';

-- Validate sm_ prefix
ALTER TABLE api_keys 
    ADD CONSTRAINT chk_api_key_format CHECK (key_hash ~ '^[a-f0-9]{64}$');
```

---

## 5. Middleware RBAC Enforcement

```go
// gateway/infra/middleware/rbac.go

// Middleware để enforce permission trên từng route
func RequirePermission(perm auth.Permission) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authCtx := getAuthContext(r)
            if !auth.HasPermission(authCtx.Role, perm) {
                http.Error(w, `{"error":"Forbidden"}`, http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Áp dụng trong router
mux.Handle("DELETE /api/v1/documents/{id}",
    RequirePermission(auth.PermDocumentDelete)(documentHandler.Delete))
mux.Handle("POST /api/v1/connections/{provider}",
    RequirePermission(auth.PermConnectionCreate)(connHandler.Create))
```

---

## 6. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | sm_ API key generation + validation | 1 ngày |
| **P2** | RBAC permission matrix + HasPermission | 1 ngày |
| **P3** | Organization domain model + DB schema | 2 ngày |
| **P4** | Organization API (create, get, members) | 2 ngày |
| **P5** | Invitation system (email invite) | 1 ngày |
| **P6** | Redis cache cho token validation | 1 ngày |
| **P7** | RBAC middleware + route enforcement | 1 ngày |
| **P8** | OAuth2 Authorization Server | 3 ngày |
| **P9** | Tests + Acceptance Criteria | 2 ngày |

**Tổng:** ~14 ngày (Wave 1 — ưu tiên cao nhất)

---

## 7. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| Tạo API key → plaintext `sm_xxx` 1 lần | GenerateAPIKey() trả về plaintext, store hash only |
| Viewer → DELETE /documents → 403 | RequirePermission(PermDocumentDelete) middleware |
| MCP OAuth flow → access token | OAuthServer.HandleAuthorize + HandleToken |
| Editor không đổi Org settings | HasPermission(RoleEditor, PermSettingsWrite) = false |
| 100 req/s không tăng tải DB | Redis cache 5 phút cho token validation |
