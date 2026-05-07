---
skill_id: SKILL-013
version: 1.0.0
status: active
priority: P2
group: Security Engineering
created_at: 2026-04-24
---

# SKILL-013 · Security Engineering & Hardening

## Mô tả

Thiết kế và thực thi bảo mật toàn diện cho hệ thống — authentication, authorization, data protection, API security, và threat modeling.

## Agents sử dụng

- `qa-pipeline-agent`
- `doc-consistency-agent`

## Tài liệu liên kết

- `docs/standards/security-policy.md`

---

## Năng lực cốt lõi

### 1. Authentication & Authorization

```go
// JWT validation middleware
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenStr := extractBearerToken(c.GetHeader("Authorization"))
        if tokenStr == "" {
            c.AbortWithStatusJSON(401, ErrorResponse{Code: "MISSING_TOKEN"})
            return
        }
        
        claims, err := validateJWT(tokenStr, secret)
        if err != nil {
            c.AbortWithStatusJSON(401, ErrorResponse{Code: "INVALID_TOKEN"})
            return
        }
        
        c.Set("user_id", claims.UserID)
        c.Set("project_ids", claims.ProjectIDs)
        c.Set("roles", claims.Roles)
        c.Next()
    }
}

// RBAC middleware
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        roles := c.GetStringSlice("roles")
        if !hasPermission(roles, permission) {
            c.AbortWithStatusJSON(403, ErrorResponse{Code: "FORBIDDEN"})
            return
        }
        c.Next()
    }
}

// Permission matrix
var rolePermissions = map[string][]string{
    "admin":    {"document:read", "document:write", "document:delete", "project:manage"},
    "engineer": {"document:read", "document:write", "pipeline:trigger"},
    "viewer":   {"document:read"},
}
```

### 2. Input Validation & Sanitization

```go
// Defense in depth: validate at every layer
// Layer 1: HTTP middleware (format check)
// Layer 2: Service layer (business rules)
// Layer 3: Repository layer (SQL parameterization)

// Chống SQL injection — NEVER string concatenate SQL
// ❌ BAD
db.Raw("SELECT * FROM documents WHERE project_id = '" + projectID + "'")

// ✅ GOOD - parameterized query
db.Where("project_id = ?", projectID).Find(&documents)

// Chống XSS — sanitize HTML content
import "github.com/microcosm-cc/bluemonday"

func SanitizeHTML(input string) string {
    p := bluemonday.UGCPolicy()
    return p.Sanitize(input)
}

// Input size limits
const (
    MaxDocumentSize = 10 * 1024 * 1024  // 10MB
    MaxTitleLength  = 500
    MaxQueryLength  = 1000
)

func ValidateDocumentRequest(req *CreateDocumentRequest) error {
    if len(req.Content) > MaxDocumentSize {
        return fmt.Errorf("document too large: max %d bytes", MaxDocumentSize)
    }
    if len(req.Title) > MaxTitleLength {
        return fmt.Errorf("title too long: max %d chars", MaxTitleLength)
    }
    // Validate project_id is UUID
    if _, err := uuid.Parse(req.ProjectID); err != nil {
        return fmt.Errorf("invalid project_id format")
    }
    return nil
}
```

### 3. Secrets Management

```go
// ✅ Đọc secrets từ environment hoặc Vault — KHÔNG hardcode
// ❌ KHÔNG BAO GIỜ làm thế này:
const dbPassword = "mypassword123"

// ✅ Từ environment
type Config struct {
    DBPassword  string `env:"DB_PASSWORD,required"`
    JWTSecret   string `env:"JWT_SECRET,required"`
    Neo4jPass   string `env:"NEO4J_PASSWORD,required"`
    OpenAIKey   string `env:"OPENAI_API_KEY,required"`
}

// ✅ Từ Vault (production)
func LoadFromVault(path string) (*Config, error) {
    client, _ := vault.New(vault.WithAddress(os.Getenv("VAULT_ADDR")))
    secret, err := client.Secrets.KvV2Read(ctx, path)
    if err != nil {
        return nil, err
    }
    return mapToConfig(secret.Data.Data), nil
}

// .gitignore — phải có
// .env
// *.key
// *.pem
// **/secrets/
```

### 4. Transport Security

```yaml
# TLS configuration cho gRPC servers
tls:
  cert_file: "/etc/ssl/certs/server.crt"
  key_file: "/etc/ssl/private/server.key"
  min_version: "TLS1.2"
  cipher_suites:
    - "TLS_AES_256_GCM_SHA384"
    - "TLS_CHACHA20_POLY1305_SHA256"

# HSTS header
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
```

```go
// gRPC server với TLS
func NewGRPCServer(certFile, keyFile string) (*grpc.Server, error) {
    creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
    if err != nil {
        return nil, err
    }
    return grpc.NewServer(grpc.Creds(creds)), nil
}
```

### 5. Dependency Vulnerability Scanning

```yaml
# GitHub Actions workflow: security scanning
name: Security Scan

on: [push, pull_request]

jobs:
  govulncheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golang/govulncheck-action@v1
        with:
          go-version-input: "1.22"
          
  gosec:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Gosec
        uses: securego/gosec@master
        with:
          args: ./...
          
  trivy:
    runs-on: ubuntu-latest
    steps:
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          security-checks: 'vuln,secret'
```

### 6. Threat Modeling (STRIDE)

```markdown
## STRIDE Analysis — Knowledge Gateway

### S: Spoofing Identity
- Threat: Attacker impersonates valid user/service
- Mitigation: JWT with short expiry (15min), refresh tokens, service-to-service mTLS

### T: Tampering with Data
- Threat: PRD content tampered during pipeline
- Mitigation: Content hashing (SHA256) at each pipeline stage, DB integrity constraints

### R: Repudiation
- Threat: User denies performing action
- Mitigation: Immutable audit logs with user_id + timestamp + action

### I: Information Disclosure
- Threat: Sensitive PRD content leaked cross-project
- Mitigation: Project-scoped queries, row-level security in PostgreSQL

### D: Denial of Service
- Threat: Large document floods pipeline
- Mitigation: File size limits (10MB), rate limiting, worker queue capacity limits

### E: Elevation of Privilege
- Threat: viewer role accessing admin functions
- Mitigation: RBAC on every endpoint, permission checks in middleware
```

### 7. Security Headers

```go
// Security headers middleware
func SecurityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Permissions-Policy", "geolocation=(), microphone=()")
        
        // CORS — chỉ cho phép allowed origins
        origin := c.GetHeader("Origin")
        if isAllowedOrigin(origin) {
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Access-Control-Allow-Credentials", "true")
        }
        c.Next()
    }
}
```

### 8. Data Privacy

```go
// PII fields — không log raw, mask khi cần
type UserInfo struct {
    ID    string `json:"id"`
    Email string `json:"email" pii:"true"` // Tag để identify PII
    Name  string `json:"name"  pii:"true"`
}

// Mask PII trong logs
func MaskEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 || len(parts[0]) < 3 {
        return "***@***"
    }
    return parts[0][:2] + "***@" + parts[1]
}

// Encryption at rest cho sensitive content
func EncryptContent(plaintext, key []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

---

## Security Checklist (Pre-deployment)

- [ ] Tất cả secrets từ environment/Vault — không có hardcoded values
- [ ] `govulncheck` và `gosec` pass trong CI
- [ ] TLS 1.2+ trên tất cả external connections
- [ ] JWT validation middleware trên tất cả protected endpoints
- [ ] RBAC permissions check trên admin operations
- [ ] Input validation và size limits trên tất cả endpoints
- [ ] SQL queries 100% parameterized
- [ ] Security headers middleware active
- [ ] CORS whitelist chỉ chứa allowed origins
- [ ] STRIDE threat model đã review trước khi production
- [ ] Audit log cho tất cả write operations
- [ ] PII fields đã được identify và mask trong logs
