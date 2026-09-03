# TASK-PLAT-009 — SSO Google OAuth Use Case & Handlers

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-009 |
| **Wave** | 2 (Auth Flows) |
| **Solution** | [SOL-PLAT-003](../solutions/SOL-PLAT-003-SSO-Google-OAuth.md) §2.1–2.4 |
| **Component** | `services/vnp-platform/internal/usecase/`, `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-003 (JWT service) |
| **Estimated** | 5h |

---

## Mục tiêu

Implement Google OAuth2 PKCE flow: `AuthorizeURL` → Google consent → `HandleCallback` → VNP JWT. User provisioning (first login) + account linking (existing email). Mobile flow: `ExchangeGoogleToken`.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/port/google_oauth.go` [NEW]

```go
package port

type GoogleOAuthClient interface {
    // AuthCodeURL generates the OAuth2 authorization URL with state param
    AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
    // Exchange exchanges authorization code for Google tokens
    Exchange(ctx context.Context, code string) (*oauth2.Token, error)
    // GetUserInfo fetches user info using access token (server-side validation)
    GetUserInfo(ctx context.Context, accessToken string) (*domain.GoogleUserInfo, error)
    // ValidateIDToken validates Google ID token (for mobile flow)
    ValidateIDToken(ctx context.Context, idToken string) (*domain.GoogleUserInfo, error)
}

type StateRepository interface {
    Set(ctx context.Context, state string, data *domain.OAuthState, ttl time.Duration) error
    Get(ctx context.Context, state string) (*domain.OAuthState, error)
    Delete(ctx context.Context, state string) error
}
```

### 2. Tạo `services/vnp-platform/internal/adapter/google/client.go` [NEW]

```go
package google

import (
    "encoding/json"
    "fmt"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

type Client struct {
    config    *oauth2.Config
    httpClient *http.Client
}

func NewClient(clientID, clientSecret, redirectURI string) *Client {
    return &Client{
        config: &oauth2.Config{
            ClientID:     clientID,
            ClientSecret: clientSecret,
            RedirectURL:  redirectURI,
            Scopes:       []string{"openid", "email", "profile"},
            Endpoint:     google.Endpoint,
        },
        httpClient: &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *Client) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
    return c.config.AuthCodeURL(state, opts...)
}

func (c *Client) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
    return c.config.Exchange(ctx, code)
}

func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*domain.GoogleUserInfo, error) {
    resp, err := c.httpClient.Get(
        "https://www.googleapis.com/oauth2/v3/userinfo?access_token=" + accessToken)
    if err != nil { return nil, err }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("google userinfo failed: %d", resp.StatusCode)
    }
    var userInfo domain.GoogleUserInfo
    if err = json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
        return nil, err
    }
    return &userInfo, nil
}

func (c *Client) ValidateIDToken(ctx context.Context, idToken string) (*domain.GoogleUserInfo, error) {
    // Server-side validation via tokeninfo endpoint
    resp, err := c.httpClient.Get(
        "https://www.googleapis.com/oauth2/v3/tokeninfo?id_token=" + idToken)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("invalid google id_token: %d", resp.StatusCode)
    }
    var userInfo domain.GoogleUserInfo
    json.NewDecoder(resp.Body).Decode(&userInfo)
    return &userInfo, nil
}
```

### 3. Tạo `services/vnp-platform/internal/usecase/sso_uc.go` [NEW]

Implement đầy đủ theo SOL-PLAT-003 §2.2 — xem solution file để copy code pattern.

Key methods:
- `AuthorizeURL(ctx, redirectURI) (string, error)` — generate state + Google URL
- `HandleCallback(ctx, code, state) (*SSOResult, error)` — verify state + exchange + upsert user + issue JWT
- `ExchangeGoogleToken(ctx, googleIDToken) (*SSOResult, error)` — mobile flow

### 4. Modify `gateway/adapter/handler/auth.go` [MODIFY] — add SSO endpoints

```go
// GET /v1/auth/sso/google/authorize
func (h *AuthHandler) SSOGoogleAuthorize(w http.ResponseWriter, r *http.Request) { ... }

// GET /v1/auth/sso/google/callback
func (h *AuthHandler) SSOGoogleCallback(w http.ResponseWriter, r *http.Request) { ... }

// POST /v1/auth/sso/google
func (h *AuthHandler) SSOGoogleExchange(w http.ResponseWriter, r *http.Request) { ... }
```

See SOL-PLAT-003 §2.3 for full handler implementations.

### 5. Modify `gateway/adapter/handler/router.go` [MODIFY] — register SSO routes

```go
// Public auth routes (no auth required)
r.Get("/v1/auth/sso/google/authorize", authH.SSOGoogleAuthorize)
r.Get("/v1/auth/sso/google/callback", authH.SSOGoogleCallback)
r.Post("/v1/auth/sso/google", authH.SSOGoogleExchange)
```

### 6. Add SSO columns migration `deployment/dev/migrations/XXX_users_sso.up.sql` [NEW]

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS provider      TEXT DEFAULT 'email';
ALTER TABLE users ADD COLUMN IF NOT EXISTS provider_id   TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url    TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider
    ON users(provider, provider_id) WHERE provider != 'email';
```

---

## Acceptance Criteria

- [ ] `GET /v1/auth/sso/google/authorize` → redirect to Google OAuth consent page
- [ ] `GET /v1/auth/sso/google/callback?code=xxx&state=yyy` → issues VNP JWT + sets httpOnly refresh cookie
- [ ] State parameter: validated (CSRF), consumed on first use (Redis 10-min TTL)
- [ ] Google token validated **server-side** via tokeninfo API (not just decoded)
- [ ] New user: provisioned with name/email/avatar, role=viewer
- [ ] Existing email: linked to existing account (no duplicate)
- [ ] `email_verified: false` from Google → 401 Unauthorized
- [ ] Mobile flow: `POST /v1/auth/sso/google` with id_token → VNP JWT
- [ ] `go build ./...` passes

## Files

```
services/vnp-platform/internal/domain/sso.go                    [NEW]
services/vnp-platform/internal/port/google_oauth.go              [NEW]
services/vnp-platform/internal/adapter/google/client.go          [NEW]
services/vnp-platform/internal/usecase/sso_uc.go                 [NEW]
gateway/adapter/handler/auth.go                                  [MODIFY — add 3 SSO endpoints]
gateway/adapter/handler/router.go                                [MODIFY — register SSO routes]
deployment/dev/migrations/XXX_users_sso.up.sql                   [NEW]
```
