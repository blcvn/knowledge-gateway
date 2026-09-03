# TASK-PLAT-003 — JWT RS256 Middleware

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-003 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-PLAT-001](../solutions/SOL-PLAT-001-Auth-API-Key-JWT.md) §2.3 |
| **Component** | `gateway/internal/infra/middleware/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Mục tiêu

Upgrade gateway auth middleware từ HMAC → RSA-2048 RS256. Implement JWKSHandler cho `GET /.well-known/jwks.json`.

---

## Công việc cụ thể

### 1. Modify `gateway/internal/infra/middleware/jwt_rsa.go` [MODIFY]

```go
package middleware

import (
    "crypto/rsa"
    "encoding/json"
    "fmt"
    "math/big"
    "net/http"
    "github.com/golang-jwt/jwt/v5"
)

// VNPClaims — RS256 JWT payload
type VNPClaims struct {
    jwt.RegisteredClaims
    TenantID string   `json:"tenant_id"`
    UserID   string   `json:"user_id,omitempty"`
    Roles    []string `json:"roles"`
    Scopes   []string `json:"scopes,omitempty"`
    RateTier string   `json:"rate_tier"` // "free" | "pro" | "enterprise"
}

// JWTValidator validates RS256 tokens using rotating RSA public keys
type JWTValidator struct {
    publicKeys []*rsa.PublicKey // support multiple active keys (rotation window)
}

func NewJWTValidator(publicKeys ...*rsa.PublicKey) *JWTValidator {
    return &JWTValidator{publicKeys: publicKeys}
}

func (v *JWTValidator) Validate(tokenStr string) (*VNPClaims, error) {
    var lastErr error
    for _, pubKey := range v.publicKeys {
        pk := pubKey
        token, err := jwt.ParseWithClaims(tokenStr, &VNPClaims{}, func(t *jwt.Token) (interface{}, error) {
            if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
            }
            return pk, nil
        }, jwt.WithValidMethods([]string{"RS256"}))
        if err != nil {
            lastErr = err
            continue
        }
        if claims, ok := token.Claims.(*VNPClaims); ok && token.Valid {
            return claims, nil
        }
    }
    return nil, lastErr
}

// JWKSet represents the JSON Web Key Set for public key discovery
type JWKSet struct {
    Keys []JWK `json:"keys"`
}

type JWK struct {
    Kty string `json:"kty"` // "RSA"
    Use string `json:"use"` // "sig"
    Alg string `json:"alg"` // "RS256"
    Kid string `json:"kid"` // Key ID
    N   string `json:"n"`   // modulus (base64url)
    E   string `json:"e"`   // exponent (base64url)
}

// JWKSHandler serves GET /.well-known/jwks.json
func JWKSHandler(publicKeys []*rsa.PublicKey) http.HandlerFunc {
    jwks := buildJWKS(publicKeys)
    payload, _ := json.Marshal(jwks)
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Cache-Control", "public, max-age=3600")
        w.Write(payload)
    }
}

func buildJWKS(publicKeys []*rsa.PublicKey) JWKSet {
    keys := make([]JWK, 0, len(publicKeys))
    for i, pk := range publicKeys {
        keys = append(keys, JWK{
            Kty: "RSA",
            Use: "sig",
            Alg: "RS256",
            Kid: fmt.Sprintf("key-%d", i+1),
            N:   base64.RawURLEncoding.EncodeToString(pk.N.Bytes()),
            E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pk.E)).Bytes()),
        })
    }
    return JWKSet{Keys: keys}
}
```

### 2. Modify `gateway/adapter/handler/router.go` [MODIFY] — register JWKS endpoint

```go
// In SetupRoutes() or equivalent:
// Public routes (no auth):
mux.Get("/.well-known/jwks.json", middleware.JWKSHandler(cfg.Auth.RSAPublicKeys))
```

### 3. Modify `gateway/internal/infra/config/config.go` [MODIFY] — RSA key loading

```go
type AuthConfig struct {
    // existing...
    JWTPublicKeyPath  string // path to RSA public key PEM file
    JWTPrivateKeyPath string // path to RSA private key PEM file (platform service only)
    RSAPublicKeys     []*rsa.PublicKey // loaded at startup
    DevMode           bool
}

// LoadRSAPublicKey loads PEM public key file
func LoadRSAPublicKey(path string) (*rsa.PublicKey, error) {
    pemData, err := os.ReadFile(path)
    if err != nil { return nil, err }
    block, _ := pem.Decode(pemData)
    pub, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil { return nil, err }
    rsaPub, ok := pub.(*rsa.PublicKey)
    if !ok { return nil, fmt.Errorf("not an RSA public key") }
    return rsaPub, nil
}
```

### 4. Unit test `gateway/internal/infra/middleware/jwt_rsa_test.go` [NEW]

```go
func TestJWTValidator_ValidToken(t *testing.T) {
    // Generate RSA keypair for test
    privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
    validator := NewJWTValidator(&privateKey.PublicKey)

    // Sign a test token
    claims := &VNPClaims{
        RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
        TenantID: "test-tenant",
        Roles:    []string{"admin"},
        RateTier: "pro",
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    signed, _ := token.SignedString(privateKey)

    result, err := validator.Validate(signed)
    assert.NoError(t, err)
    assert.Equal(t, "test-tenant", result.TenantID)
}

func TestJWTValidator_WrongAlgorithm_Rejected(t *testing.T) {
    // HMAC-signed token should be rejected by RS256 validator
    hmacToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "x"})
    signed, _ := hmacToken.SignedString([]byte("secret"))

    privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
    validator := NewJWTValidator(&privateKey.PublicKey)
    _, err := validator.Validate(signed)
    assert.Error(t, err)
}
```

---

## Acceptance Criteria

- [ ] `JWTValidator.Validate()` accepts RS256 tokens, rejects HMAC tokens
- [ ] `GET /.well-known/jwks.json` returns valid JWK Set JSON
- [ ] Key rotation: JWTValidator accepts tokens from multiple active public keys
- [ ] Expired JWT → validation error
- [ ] `go test ./gateway/internal/infra/middleware/...` passes

## Files

```
gateway/internal/infra/middleware/jwt_rsa.go       [MODIFY]
gateway/internal/infra/middleware/jwt_rsa_test.go  [NEW]
gateway/internal/infra/config/config.go            [MODIFY — RSA key loading]
gateway/adapter/handler/router.go                  [MODIFY — register JWKS route]
```
