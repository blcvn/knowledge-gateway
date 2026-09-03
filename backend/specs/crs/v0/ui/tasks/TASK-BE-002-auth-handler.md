# TASK-BE-002 — Auth Handler: login / logout / refresh / me

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-002 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-002 §2.1, §2.2](../solutions/SOL-002-Auth-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | TASK-BE-001 |
| **Estimated** | 3h |

---

## Context

Gateway cần handler xử lý 4 auth endpoints. Dùng bcrypt verify password, RS256 JWT sign/verify (key đã có trong gateway config), và PostgreSQL để lưu refresh_tokens.

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/auth_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` (đăng ký routes) |

---

## Implementation

### File: `gateway/internal/adapter/handler/auth_handler.go`

```go
package handler

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "time"

    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    // Import gateway's JWT manager và DB pool
)

type AuthHandler struct {
    db     *sql.DB
    jwtMgr JWTManager  // RS256 sign/verify interface
}

// POST /v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httputil.Error(w, "Invalid request body", "INVALID_REQUEST", 400)
        return
    }

    // 1. Look up user by email
    var user struct {
        ID           string
        Name         string
        Email        string
        PasswordHash string
        Role         string
        TenantID     string
        AvatarURL    *string
    }
    err := h.db.QueryRowContext(r.Context(),
        `SELECT id, name, email, password_hash, role, tenant_id, avatar_url
         FROM users WHERE email = $1 AND is_active = true`,
        req.Email,
    ).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash,
        &user.Role, &user.TenantID, &user.AvatarURL)
    if err != nil {
        httputil.Error(w, "Invalid credentials", "AUTH_INVALID_CREDENTIALS", 401)
        return
    }

    // 2. Verify password (bcrypt)
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        httputil.Error(w, "Invalid credentials", "AUTH_INVALID_CREDENTIALS", 401)
        return
    }

    // 3. Sign JWT access token (RS256)
    accessToken, _, expiresAt, err := h.jwtMgr.Sign(user.ID, user.Role, []string{"console:*"})
    if err != nil {
        httputil.Error(w, "Token signing failed", "INTERNAL_ERROR", 500)
        return
    }

    // 4. Generate refresh token (random UUID, store SHA-256 hash)
    rawRefresh := uuid.New().String()
    hash := sha256.Sum256([]byte(rawRefresh))
    tokenHash := hex.EncodeToString(hash[:])
    expiresRefresh := time.Now().UTC().Add(30 * 24 * time.Hour) // 30 days

    h.db.ExecContext(r.Context(),
        `INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
         VALUES ($1, $2, $3)`,
        user.ID, tokenHash, expiresRefresh,
    )

    // 5. Response
    httputil.JSON(w, 200, map[string]any{
        "access_token":  accessToken,
        "refresh_token": rawRefresh,
        "expires_in":    int(time.Until(expiresAt).Seconds()),
        "token_type":    "Bearer",
        "user": map[string]any{
            "id":         user.ID,
            "name":       user.Name,
            "email":      user.Email,
            "role":       user.Role,
            "tenant_id":  user.TenantID,
            "avatar_url": user.AvatarURL,
        },
    })
}

// POST /v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    var req struct {
        RefreshToken string `json:"refresh_token"`
    }
    _ = json.NewDecoder(r.Body).Decode(&req)

    if req.RefreshToken != "" {
        hash := sha256.Sum256([]byte(req.RefreshToken))
        tokenHash := hex.EncodeToString(hash[:])
        h.db.ExecContext(r.Context(),
            `UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`,
            tokenHash,
        )
    }
    w.WriteHeader(204)
}

// POST /v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
    var req struct {
        RefreshToken string `json:"refresh_token"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
        httputil.Error(w, "refresh_token required", "INVALID_REQUEST", 400)
        return
    }

    hash := sha256.Sum256([]byte(req.RefreshToken))
    tokenHash := hex.EncodeToString(hash[:])

    var userID, role string
    var expiresAt time.Time
    var revoked bool
    err := h.db.QueryRowContext(r.Context(),
        `SELECT rt.user_id, u.role, rt.expires_at, rt.revoked
         FROM refresh_tokens rt JOIN users u ON rt.user_id = u.id
         WHERE rt.token_hash = $1`,
        tokenHash,
    ).Scan(&userID, &role, &expiresAt, &revoked)

    if err != nil || revoked || time.Now().After(expiresAt) {
        httputil.Error(w, "Invalid or expired refresh token", "AUTH_TOKEN_INVALID", 401)
        return
    }

    accessToken, _, exp, err := h.jwtMgr.Sign(userID, role, []string{"console:*"})
    if err != nil {
        httputil.Error(w, "Token signing failed", "INTERNAL_ERROR", 500)
        return
    }

    httputil.JSON(w, 200, map[string]any{
        "access_token": accessToken,
        "expires_in":   int(time.Until(exp).Seconds()),
    })
}

// GET /v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
    // authMiddleware đã inject userID vào context
    userID := authctx.UserID(r.Context())

    var user struct {
        ID        string
        Name      string
        Email     string
        Role      string
        TenantID  string
        AvatarURL *string
    }
    err := h.db.QueryRowContext(r.Context(),
        `SELECT id, name, email, role, tenant_id, avatar_url
         FROM users WHERE id = $1`,
        userID,
    ).Scan(&user.ID, &user.Name, &user.Email, &user.Role, &user.TenantID, &user.AvatarURL)
    if err != nil {
        httputil.Error(w, "User not found", "NOT_FOUND", 404)
        return
    }

    httputil.JSON(w, 200, map[string]any{
        "id":         user.ID,
        "name":       user.Name,
        "email":      user.Email,
        "role":       user.Role,
        "tenant_id":  user.TenantID,
        "avatar_url": user.AvatarURL,
    })
}
```

### Routes registration (thêm vào router.go)

```go
// Auth routes (không cần auth middleware)
mux.HandleFunc("POST /v1/auth/login",   authHandler.Login)
mux.HandleFunc("POST /v1/auth/logout",  authHandler.Logout)
mux.HandleFunc("POST /v1/auth/refresh", authHandler.Refresh)

// Auth routes (cần auth middleware)
mux.HandleFunc("GET /v1/auth/me", authMiddleware(authHandler.Me))
```

---

## Verification

```bash
# Login
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@vnp.io","password":"securepassword"}'
# Expected: 200 với access_token, refresh_token, user object

# Me
curl http://localhost:8080/v1/auth/me \
  -H "Authorization: Bearer <access_token>"
# Expected: 200 với user object

# Refresh
curl -X POST http://localhost:8080/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
# Expected: 200 với new access_token

# Logout
curl -X POST http://localhost:8080/v1/auth/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
# Expected: 204 No Content
```
