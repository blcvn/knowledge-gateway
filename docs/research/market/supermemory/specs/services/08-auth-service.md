# 08 — Auth Service

> **gRPC**: 9007 | **Health**: 9087

---

## 1. Purpose

Authentication và authorization: JWT token issuance/validation, API key management, organization/user CRUD, RBAC enforcement, OAuth2 provider cho MCP clients.

---

## 2. Clean Architecture

```
services/auth-service/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # User, Organization, OrgMember, APIKey, Session
│   │   ├── value_object.go     # Role, Permission, PlanTier, AuthMethod
│   │   └── errors.go           # ErrInvalidCredentials, ErrBlocked, ErrExpired
│   ├── usecase/
│   │   ├── login.go            # Email/OTP/Magic link → JWT
│   │   ├── validate_token.go   # JWT verification + user lookup
│   │   ├── validate_api_key.go # API key verification → user context
│   │   ├── create_api_key.go   # Generate sm_ prefixed key
│   │   ├── manage_org.go       # Organization CRUD + member management
│   │   ├── authorize.go        # RBAC permission check
│   │   ├── oauth_server.go     # OAuth2 authorization server for MCP
│   │   ├── port/
│   │   │   ├── input.go        # LoginUC, ValidateTokenUC, CreateAPIKeyUC
│   │   │   └── output.go       # UserRepo, OrgRepo, APIKeyRepo, SessionRepo,
│   │   │                       # TokenSigner, PasswordHasher, EmailSender
│   │   └── dto/
│   │       └── auth.go         # LoginInput, TokenOutput, SessionOutput
│   ├── adapter/
│   │   ├── grpc/handler.go     # AuthServiceServer implementation
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── user.go
│   │   │       ├── organization.go
│   │   │       ├── api_key.go       # Hashed key storage
│   │   │       └── session.go
│   │   ├── token/
│   │   │   └── jwt.go              # RS256 JWT sign/verify
│   │   ├── cache/
│   │   │   └── redis.go            # Session cache, API key validation cache
│   │   ├── oauth/
│   │   │   └── server.go           # OAuth2 authorization server endpoints
│   │   └── crypto/
│   │       ├── hasher.go           # bcrypt/argon2 for passwords
│   │       └── api_key_gen.go      # sm_ + secure random generator
│   └── infra/
│       ├── config/config.go
│       └── wire/wire.go
├── migrations/
│   ├── 001_create_users.up.sql
│   ├── 002_create_organizations.up.sql
│   └── 003_create_api_keys.up.sql
└── Dockerfile
```

---

## 3. RBAC Model

```go
type Role string
const (
    RoleOwner  Role = "owner"   // Full control
    RoleAdmin  Role = "admin"   // Manage members, settings
    RoleEditor Role = "editor"  // Create, edit, delete content
    RoleViewer Role = "viewer"  // Read-only
)

type Permission string
const (
    PermDocumentCreate Permission = "document:create"
    PermDocumentRead   Permission = "document:read"
    PermDocumentDelete Permission = "document:delete"
    PermMemoryForget   Permission = "memory:forget"
    PermSearchExecute  Permission = "search:execute"
    PermConnCreate     Permission = "connection:create"
    PermConnDelete     Permission = "connection:delete"
    PermSettingsRead   Permission = "settings:read"
    PermSettingsWrite  Permission = "settings:write"
    PermMemberManage   Permission = "member:manage"
    PermAPIKeyManage   Permission = "apikey:manage"
    PermAnalyticsRead  Permission = "analytics:read"
)

// Role → Permissions mapping
var rolePermissions = map[Role][]Permission{
    RoleOwner:  allPermissions,
    RoleAdmin:  allExcept(PermSettingsWrite),  // Cannot change billing
    RoleEditor: {PermDocumentCreate, PermDocumentRead, PermDocumentDelete,
                 PermMemoryForget, PermSearchExecute, PermConnCreate},
    RoleViewer: {PermDocumentRead, PermSearchExecute, PermAnalyticsRead},
}
```

---

## 4. API Key Format

```go
// Format: sm_<32 random bytes base62>
// Example: sm_a7Kj3mN2pQ9xR4vW8yB5cD0eF6gH1iL
// Storage: SHA-256 hash in DB, plaintext returned once on creation

func GenerateAPIKey() (plaintext string, hash string) {
    raw := make([]byte, 32)
    crypto_rand.Read(raw)
    plaintext = "sm_" + base62.Encode(raw)
    hash = sha256Hex(plaintext)
    return
}
```

---

## 5. gRPC Interface

```protobuf
service AuthService {
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc GetSession(GetSessionRequest) returns (SessionResponse);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc CreateOrganization(CreateOrganizationRequest) returns (OrganizationResponse);
  rpc ListOrganizationMembers(ListMembersRequest) returns (ListMembersResponse);
  rpc AddOrganizationMember(AddMemberRequest) returns (MemberResponse);
  rpc RemoveOrganizationMember(RemoveMemberRequest) returns (google.protobuf.Empty);
  rpc Authorize(AuthorizeRequest) returns (AuthorizeResponse);
  // OAuth2 endpoints for MCP
  rpc GetOAuthMetadata(google.protobuf.Empty) returns (OAuthMetadataResponse);
  rpc Authorize(OAuthAuthorizeRequest) returns (OAuthAuthorizeResponse);
  rpc ExchangeToken(OAuthTokenRequest) returns (OAuthTokenResponse);
}
```
