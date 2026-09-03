# TASK-SM-003 — Auth: OAuth2 Authorization Server

**Task ID:** TASK-SM-003  
**Wave:** 1 (Foundation)  
**Solution:** [SOL-SM-007](../solutions/SOL-SM-007-Auth-Organization-RBAC.md)  
**Depends on:** TASK-SM-001 (JWT signer), TASK-SM-002 (Organization)  
**Ước tính:** 3h  
**Priority:** High — MCP client authentication

**Trạng thái:** ⏳ Pending  
**Ghi chú:** OAuth2 server (sm-auth) not implemented  
---

## Mục tiêu

Implement OAuth2 Authorization Code flow (RFC 6749) với PKCE (RFC 7636) cho MCP clients (Claude Desktop, Cursor):
1. `/.well-known/oauth-authorization-server` — metadata discovery
2. GET `/api/v1/auth/oauth/authorize` — authorization endpoint
3. POST `/api/v1/auth/oauth/token` — token exchange

---

## Công việc cụ thể

### 1. Tạo OAuth2 Server

**`services/vnp-platform/internal/adapter/oauth2/server.go`**

```go
type OAuthServer struct {
    clientRepo  OAuthClientRepository  // postgres
    codeStore   AuthCodeStore          // Redis: code → {clientID, userID, scope, codeChallenge, expiresAt}
    tokenSigner TokenSigner            // RS256 JWT signer (reuse existing)
}

// Metadata Discovery
func (s *OAuthServer) HandleMetadata(w http.ResponseWriter, r *http.Request)
// Response: {issuer, authorization_endpoint, token_endpoint, response_types_supported, code_challenge_methods_supported: ["S256"]}

// Authorization Endpoint
// GET /api/v1/auth/oauth/authorize?client_id=...&redirect_uri=...&code_challenge=...&code_challenge_method=S256&state=...
func (s *OAuthServer) HandleAuthorize(w http.ResponseWriter, r *http.Request)
// 1. Validate client_id từ oauth2_clients
// 2. Validate redirect_uri trong client.redirect_uris
// 3. Validate code_challenge_method = "S256" (PKCE required)
// 4. Check user session (nếu chưa login → redirect to login page)
// 5. Generate auth code: 16 random bytes hex, expires 10 phút
// 6. Redis: code → {clientID, userID, redirectURI, codeChallenge, scope}
// 7. Redirect: redirect_uri?code={code}&state={state}

// Token Endpoint
// POST /api/v1/auth/oauth/token
// Body: {grant_type, code, redirect_uri, code_verifier}
func (s *OAuthServer) HandleToken(w http.ResponseWriter, r *http.Request)
// 1. Validate grant_type = "authorization_code"
// 2. Fetch code from Redis, verify not expired
// 3. PKCE verify: SHA-256(code_verifier) == stored codeChallenge (Base64URL)
// 4. Generate access_token: RS256 JWT, expires 1h
// 5. Generate refresh_token: 32 random bytes, expires 30d
// 6. Delete code from Redis (single-use)
// 7. Return: {access_token, token_type: "bearer", expires_in: 3600, refresh_token}
```

### 2. Tạo Redis Auth Code Store

**`services/vnp-platform/internal/infra/redis/auth_code_store.go`**

```go
type AuthCodeData struct {
    ClientID      string
    UserID        string
    RedirectURI   string
    CodeChallenge string  // SHA-256 of code_verifier, base64url encoded
    Scope         string
    ExpiresAt     time.Time
}

// TTL: 10 phút
func (s *RedisAuthCodeStore) Store(ctx context.Context, code string, data AuthCodeData) error { ... }
func (s *RedisAuthCodeStore) Fetch(ctx context.Context, code string) (*AuthCodeData, error) { ... }
func (s *RedisAuthCodeStore) Delete(ctx context.Context, code string) error { ... }
```

### 3. PKCE Verification

```go
// services/vnp-platform/internal/usecase/auth/pkce.go

// VerifyPKCE: SHA-256(code_verifier) → base64url → compare with stored code_challenge
func VerifyPKCE(codeVerifier, codeChallenge string) bool {
    h := sha256.Sum256([]byte(codeVerifier))
    computed := base64.RawURLEncoding.EncodeToString(h[:])
    return computed == codeChallenge
}
```

### 4. Đăng ký Endpoints

```go
// gateway/adapter/handler/register.go (thêm vào)
mux.HandleFunc("GET /.well-known/oauth-authorization-server", oauthServer.HandleMetadata)
mux.HandleFunc("GET /api/v1/auth/oauth/authorize",            oauthServer.HandleAuthorize)
mux.HandleFunc("POST /api/v1/auth/oauth/token",               oauthServer.HandleToken)
```

### 5. Tests

- `TestPKCE_Verify_Valid`: SHA-256(verifier) encoded == challenge → true
- `TestPKCE_Verify_Invalid`: wrong verifier → false
- `TestHandleToken_CodeExpired`: expired code → 400 invalid_grant
- `TestHandleToken_WrongVerifier`: wrong code_verifier → 400 invalid_grant
- `TestHandleToken_Success`: valid code + verifier → access_token in response
- `TestHandleAuthorize_InvalidClient`: unknown client_id → 400
- `TestHandleAuthorize_PKCERequired`: no code_challenge → 400
- `TestMetadataEndpoint`: GET /.well-known/... → JSON với S256 listed

---

## Acceptance Criteria

- [ ] `go build ./...` không lỗi
- [ ] GET `/.well-known/oauth-authorization-server` → 200 JSON với đúng fields
- [ ] Auth code expires sau 10 phút (Redis TTL)
- [ ] Auth code single-use: 2nd token exchange → 400 invalid_grant
- [ ] PKCE required: no code_challenge → 400 invalid_request
- [ ] Wrong code_verifier → 400 invalid_grant
- [ ] Valid flow → access_token (RS256 JWT) + refresh_token
- [ ] `go test ./services/vnp-platform/...` pass

---

## Files tạo/sửa

```
services/vnp-platform/internal/
├── adapter/oauth2/
│   └── server.go              (NEW)
├── usecase/auth/
│   ├── pkce.go                (NEW)
│   └── pkce_test.go           (NEW)
└── infra/redis/
    └── auth_code_store.go     (NEW)

gateway/adapter/handler/
└── register.go                (MODIFY: thêm OAuth2 routes)
```

## Sau khi hoàn thành

Chạy: `go build ./... && go test ./services/vnp-platform/...`
