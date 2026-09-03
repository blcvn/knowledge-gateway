# SOL-PLAT-003 — Solution: SSO Google OAuth Integration

| Field | Value |
|---|---|
| **Solution ID** | SOL-PLAT-003 |
| **CR** | [CR-PLAT-003](../../../../docs/crs/v3/platform/CR-PLAT-003-SSO-Google-OAuth.md) |
| **TDD ref** | [01-gateway.md §4.1-Auth-API](../../../tdd/architecture/01-gateway.md) · [backend-api-specs.md §1-Authentication](../../../tdd/backend-api-specs.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** LoginWithGoogle RPC exists; Google token validation in sm-auth not fully implemented
---

## 1. Phân tích kiến trúc

Theo TDD `01-gateway.md §4.1`, auth handler đã implement `LoginWithGoogle` tại `POST /v1/auth/sso/google`. Tuy nhiên **OAuth2 PKCE full flow chưa implement**:
- `GET /v1/auth/sso/google/authorize` redirect endpoint chưa có
- `GET /v1/auth/sso/google/callback` callback handler chưa có
- Server-side token validation với Google tokeninfo API
- User provisioning (first login) + linking (existing email)
- State parameter CSRF protection
- Refresh token rotation

Gateway hiện route SSO qua `services/vnp-platform` (auth backend). Ta cần implement ở đây.

---

## 2. Giải pháp

### 2.1 `services/vnp-platform/internal/domain/sso.go` [NEW]

```go
package domain

type OAuthState struct {
    State       string    // random 32-byte nonce (stored in Redis 10min TTL)
    RedirectURI string    // where to redirect after auth
    CreatedAt   time.Time
}

type GoogleUserInfo struct {
    Sub        string `json:"sub"`         // Google user ID
    Email      string `json:"email"`
    Name       string `json:"name"`
    Picture    string `json:"picture"`
    Verified   bool   `json:"email_verified"`
}

type SSOResult struct {
    User        User
    AccessToken  string    // VNP JWT (30min)
    RefreshToken string    // VNP refresh token (7d)
    IsNewUser   bool
}
```

### 2.2 `services/vnp-platform/internal/usecase/sso_uc.go` [NEW]

```go
package usecase

type SSOUseCase struct {
    users        port.UserRepository
    tenants      port.TenantRepository
    tokenService port.JWTService
    oauthState   port.StateRepository   // Redis
    googleClient port.GoogleOAuthClient
}

// Step 1: Generate authorize URL with PKCE + state
func (uc *SSOUseCase) AuthorizeURL(ctx context.Context, redirectURI string) (string, error) {
    // Generate random state (CSRF protection)
    stateBytes := make([]byte, 32)
    rand.Read(stateBytes)
    state := base64.RawURLEncoding.EncodeToString(stateBytes)

    // Store state in Redis (10min TTL)
    if err := uc.oauthState.Set(ctx, state, &domain.OAuthState{
        State:       state,
        RedirectURI: redirectURI,
        CreatedAt:   time.Now(),
    }, 10*time.Minute); err != nil {
        return "", err
    }

    // Build Google OAuth2 authorize URL
    url := uc.googleClient.AuthCodeURL(state,
        oauth2.AccessTypeOffline,
        oauth2.ApprovalForce,
    )
    return url, nil
}

// Step 2: Handle callback — exchange code, validate, provision/link user
func (uc *SSOUseCase) HandleCallback(ctx context.Context, code, state string) (*domain.SSOResult, error) {
    // Verify CSRF state
    savedState, err := uc.oauthState.Get(ctx, state)
    if err != nil || savedState == nil {
        return nil, ErrInvalidState
    }
    uc.oauthState.Delete(ctx, state) // consume state (one-time use)

    // Exchange code for Google tokens
    googleToken, err := uc.googleClient.Exchange(ctx, code)
    if err != nil {
        return nil, fmt.Errorf("google token exchange failed: %w", err)
    }

    // Server-side validation: fetch user info from Google tokeninfo API
    userInfo, err := uc.googleClient.GetUserInfo(ctx, googleToken.AccessToken)
    if err != nil {
        return nil, fmt.Errorf("google token validation failed: %w", err)
    }

    if !userInfo.Verified {
        return nil, ErrEmailNotVerified
    }

    // Lookup or create user
    user, isNew, err := uc.upsertUser(ctx, userInfo)
    if err != nil {
        return nil, err
    }

    // Issue VNP tokens
    accessToken, err := uc.tokenService.IssueJWT(user, 30*time.Minute)
    if err != nil {
        return nil, err
    }
    refreshToken, err := uc.tokenService.IssueRefresh(user, 7*24*time.Hour)
    if err != nil {
        return nil, err
    }

    return &domain.SSOResult{
        User:         *user,
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        IsNewUser:    isNew,
    }, nil
}

func (uc *SSOUseCase) upsertUser(ctx context.Context, info *domain.GoogleUserInfo) (*domain.User, bool, error) {
    // Check if user with same email exists
    existing, err := uc.users.FindByEmail(ctx, info.Email)
    if err != nil && !errors.Is(err, port.ErrNotFound) {
        return nil, false, err
    }

    if existing != nil {
        // Link existing account: update last_login_at, sync avatar
        existing.LastLoginAt = time.Now().UTC()
        existing.AvatarURL = info.Picture // optional sync
        uc.users.Update(ctx, existing)
        return existing, false, nil
    }

    // First login: provision new user
    user := &domain.User{
        Email:     info.Email,
        Name:      info.Name,
        AvatarURL: info.Picture,
        Provider:  "google",
        ProviderID: info.Sub,
        Role:      "viewer", // default role
        CreatedAt: time.Now().UTC(),
    }

    // Assign to default tenant or create new one
    tenant, err := uc.tenants.GetDefault(ctx)
    if err != nil {
        tenant, err = uc.tenants.CreateDefault(ctx, info.Email)
        if err != nil {
            return nil, false, err
        }
    }
    user.TenantID = tenant.ID

    if err = uc.users.Insert(ctx, user); err != nil {
        return nil, false, err
    }
    return user, true, nil
}

// Mobile flow: exchange Google ID token directly (no redirect)
func (uc *SSOUseCase) ExchangeGoogleToken(ctx context.Context, googleIDToken string) (*domain.SSOResult, error) {
    // Server-side: validate Google ID token via tokeninfo endpoint
    userInfo, err := uc.googleClient.ValidateIDToken(ctx, googleIDToken)
    if err != nil {
        return nil, fmt.Errorf("invalid google token: %w", err)
    }

    user, isNew, err := uc.upsertUser(ctx, userInfo)
    if err != nil {
        return nil, err
    }

    accessToken, _ := uc.tokenService.IssueJWT(user, 30*time.Minute)
    refreshToken, _ := uc.tokenService.IssueRefresh(user, 7*24*time.Hour)

    return &domain.SSOResult{
        User:         *user,
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        IsNewUser:    isNew,
    }, nil
}
```

### 2.3 `gateway/adapter/handler/auth.go` [MODIFY] — SSO endpoints

```go
// GET /v1/auth/sso/google/authorize
func (h *AuthHandler) SSOGoogleAuthorize(w http.ResponseWriter, r *http.Request) {
    redirectURI := r.URL.Query().Get("redirect_uri")
    if redirectURI == "" {
        redirectURI = h.config.Auth.DefaultRedirectURI
    }
    url, err := h.ssoUC.AuthorizeURL(r.Context(), redirectURI)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "sso_error", err.Error())
        return
    }
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GET /v1/auth/sso/google/callback
func (h *AuthHandler) SSOGoogleCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")
    if code == "" || state == "" {
        writeError(w, http.StatusBadRequest, "invalid_callback", "missing code or state")
        return
    }

    result, err := h.ssoUC.HandleCallback(r.Context(), code, state)
    if err != nil {
        http.Redirect(w, r, "/login?error=sso_failed", http.StatusTemporaryRedirect)
        return
    }

    // Set refresh token as httpOnly cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "vnp_refresh",
        Value:    result.RefreshToken,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   7 * 24 * 3600,
    })

    // Redirect to console with access token
    http.Redirect(w, r, fmt.Sprintf("/dashboard?token=%s", result.AccessToken),
        http.StatusTemporaryRedirect)
}

// POST /v1/auth/sso/google — mobile/SPA flow
func (h *AuthHandler) SSOGoogleExchange(w http.ResponseWriter, r *http.Request) {
    var req struct {
        IDToken string `json:"id_token"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    result, err := h.ssoUC.ExchangeGoogleToken(r.Context(), req.IDToken)
    if err != nil {
        writeError(w, http.StatusUnauthorized, "invalid_google_token", err.Error())
        return
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "access_token":  result.AccessToken,
        "refresh_token": result.RefreshToken,
        "user":          result.User,
        "is_new_user":   result.IsNewUser,
    })
}
```

### 2.4 DB Migration — `deployment/dev/migrations/xxx_users_sso.up.sql` [NEW]

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS provider      TEXT DEFAULT 'email';
ALTER TABLE users ADD COLUMN IF NOT EXISTS provider_id   TEXT;     -- Google sub
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url    TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- State store for CSRF protection (backup if Redis unavailable)
CREATE TABLE IF NOT EXISTS oauth_states (
    state       TEXT PRIMARY KEY,
    redirect_uri TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes'
);
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `services/vnp-platform/internal/domain/sso.go` | NEW | OAuthState, GoogleUserInfo, SSOResult domain types |
| `services/vnp-platform/internal/usecase/sso_uc.go` | NEW | AuthorizeURL, HandleCallback, ExchangeGoogleToken |
| `services/vnp-platform/internal/port/google_oauth.go` | NEW | GoogleOAuthClient interface |
| `services/vnp-platform/internal/adapter/google/client.go` | NEW | Google OAuth2 client implementation |
| `gateway/adapter/handler/auth.go` | MODIFY | Add SSOGoogleAuthorize, SSOGoogleCallback, SSOGoogleExchange |
| `gateway/adapter/handler/router.go` | MODIFY | Register 3 new SSO routes |
| `deployment/dev/migrations/xxx_users_sso.up.sql` | NEW | Add provider/avatar fields to users table |

---

## 4. API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/auth/sso/google/authorize` | Start OAuth flow → redirect to Google |
| `GET` | `/v1/auth/sso/google/callback` | OAuth callback → issue VNP JWT |
| `POST` | `/v1/auth/sso/google` | Exchange Google ID token → VNP JWT (mobile/SPA) |

---

## 5. Acceptance Criteria

- [ ] Full OAuth2 PKCE flow: `authorize` → Google consent → `callback` → VNP JWT
- [ ] Google token validated server-side via tokeninfo API (not just decoded)
- [ ] State parameter: random 32-byte nonce, stored in Redis, consumed on first use (CSRF protection)
- [ ] New users: auto-provisioned with name/email/avatar from Google, role=viewer
- [ ] Existing email: linked to existing account (no duplicate user created)
- [ ] Refresh token rotation: new refresh token issued on each use
- [ ] `is_new_user: true` flag in mobile flow response for onboarding UX
- [ ] `email_verified: false` from Google → reject with 401

---

## 6. Dependencies

- Google OAuth2 credentials (Client ID + Secret) in environment
- Redis for state storage (10-min TTL)
- `tokenService` (JWT RS256 from SOL-PLAT-001) already available
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI` env vars
