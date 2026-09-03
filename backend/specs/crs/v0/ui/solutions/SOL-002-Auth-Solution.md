# SOL-002 — Solution: Auth API (CR-001)

| Field | Value |
|---|---|
| **Solution ID** | SOL-002 |
| **CR** | [CR-001 — Authentication](../CR-001-AUTH.md) |
| **Architecture ref** | §3.1 Gateway Domain Model · §5.2 Admin Domain · §7 Tech Stack |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Implemented** | 2026-06-17 |

---

## 1. Phân tích kiến trúc

Theo `architecture.md §3.1`, Gateway đã có sẵn domain model:
- `AuthContext` — JWT/API Key identity (TenantID, UserID, Roles, Scopes, RateTier)
- Middleware pipeline: `Recovery → RequestID → Logger → CORS → **Auth** → RateLimit → Metrics → Timeout`

Theo `§5.2 Admin Domain`:
```go
User   — ID, TenantID, Email, Role(admin|editor|viewer)
APIKey — ID, TenantID, KeyHash(SHA-256), Permissions, ExpiresAt
```

Auth được xử lý qua `vnp-platform/internal/domain/auth` — JWT RS256 + API Key SHA-256.

---

## 2. Giải pháp Backend

### 2.1 Handler layer (`gateway/internal/adapter/handler/`)

Tạo file `auth_handler.go`:

```go
package handler

// POST /v1/auth/login
// POST /v1/auth/logout
// POST /v1/auth/refresh
// GET  /v1/auth/me
type AuthHandler struct {
    authUC AuthUseCase
}
```

Route registration trong `gateway/internal/infra/server/router.go`:
```go
mux.HandleFunc("POST /v1/auth/login",   authHandler.Login)
mux.HandleFunc("POST /v1/auth/logout",  authHandler.Logout)
mux.HandleFunc("POST /v1/auth/refresh", authHandler.Refresh)
mux.HandleFunc("GET  /v1/auth/me",      authMiddleware(authHandler.Me))
```

### 2.2 Use case layer (`gateway/internal/usecase/`)

```go
// Extend AuthUC (đã có AuthUseCase port) với login/logout/refresh
type AuthUseCase interface {
    Login(ctx, email, password string) (*LoginResult, error)
    Logout(ctx, refreshToken string) error
    Refresh(ctx, refreshToken string) (*TokenPair, error)
    GetCurrentUser(ctx, userID string) (*User, error)
}

type LoginResult struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    TokenType    string `json:"token_type"`
    User         User   `json:"user"`
}

type User struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Role     string `json:"role"`
    TenantID string `json:"tenant_id"`
}
```

### 2.3 Database schema — PostgreSQL

Bổ sung vào `vnp-platform` migrations:

```sql
-- Bảng users (User entity trong Admin domain)
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT,                         -- bcrypt, NULL nếu SSO only
    role          TEXT DEFAULT 'viewer'
                  CHECK (role IN ('admin','editor','viewer')),
    avatar_url    TEXT,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Refresh token store
CREATE TABLE refresh_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    TEXT NOT NULL UNIQUE,         -- SHA-256 của refresh token
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_users_tenant ON users(tenant_id);
```

### 2.4 JWT RS256 implementation

Gateway đã có JWT infrastructure. Login handler:

```go
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    // decode body...

    result, err := h.authUC.Login(r.Context(), req.Email, req.Password)
    if err != nil {
        // Map to 401 AppError
        httputil.Error(w, AppError{Code: "AUTH_INVALID_CREDENTIALS", Status: 401})
        return
    }
    httputil.JSON(w, 200, result)
}
```

---

## 3. Giải pháp Frontend

### 3.1 Thay thế `services/auth.ts`

```typescript
import { apiClient } from '../lib/api-client';

const BASE = '/v1/auth';

export interface LoginResponse {
    access_token: string;
    refresh_token: string;
    expires_in: number;
    token_type: 'Bearer';
    user: { id: string; name: string; email: string; role: string; tenant_id: string };
}

export const authService = {
    login: (email: string, password: string) =>
        apiClient.post<LoginResponse>(`${BASE}/login`, { email, password }).then(res => {
            localStorage.setItem('access_token', res.access_token);
            localStorage.setItem('refresh_token', res.refresh_token);
            localStorage.setItem('tenant_id', res.user.tenant_id);
            return res;
        }),

    logout: () => {
        const rt = localStorage.getItem('refresh_token');
        return apiClient.post<void>(`${BASE}/logout`, { refresh_token: rt }).finally(() => {
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            localStorage.removeItem('tenant_id');
        });
    },

    getMe: () => apiClient.get<LoginResponse['user']>(`${BASE}/me`),

    refresh: () => {
        const rt = localStorage.getItem('refresh_token');
        return apiClient.post<{ access_token: string; expires_in: number }>(
            `${BASE}/refresh`, { refresh_token: rt }
        ).then(res => {
            localStorage.setItem('access_token', res.access_token);
            return res;
        });
    },
};
```

### 3.2 Thêm 401 auto-refresh vào `api-client.ts`

```typescript
// Trong fetchWrapper — sau khi nhận response.status 401
if (response.status === 401 && attempt === 0) {
    try {
        await authService.refresh();
        attempt++;
        continue; // retry với token mới
    } catch {
        localStorage.clear();
        window.location.href = '/login';
        throw new AppError('Session expired', 'AUTH_EXPIRED', 401);
    }
}
```

---

## 4. Luồng dữ liệu đầy đủ

```
Browser → POST /v1/auth/login
    → Gateway AuthHandler.Login (không qua Auth middleware)
    → AuthUC.Login → bcrypt.CompareHash(password, user.password_hash)
    → OK → sign JWT RS256 (RS_PRIVATE_KEY env var)
          → store refresh_token_hash in refresh_tokens table
    → Response: { access_token, refresh_token, user }

Browser → GET /v1/console/dashboard/metrics
    → Authorization: Bearer <JWT>
    → Auth middleware → jwt.ParseRS256 → TenantID injected
    → ConsoleHandler.GetMetrics → ...
```

---

## 4.1 Cập nhật `useStore.ts` — Sync UserProfile với AuthUser (CR-001 §3.2)

Sau khi login thành công, `UserProfile` trong Zustand store phải được populate với dữ liệu thực từ API:

```typescript
// ui/src/store/useStore.ts — thêm tenant_id vào UserProfile
export interface UserProfile {
  id: string;
  name: string;
  email: string;
  role: string;
  tenant_id: string;  // NEW — required for multi-tenant
  avatar_url?: string;
}

// Action setUser khi login:
setUser: (user: UserProfile) => set({ currentUser: user }),

// Trong Login component/hook:
const response = await authService.login(email, password);
useStore.getState().setUser({
  id:         response.user.id,
  name:       response.user.name,
  email:      response.user.email,
  role:       response.user.role,
  tenant_id:  response.user.tenant_id,  // ← NEW
  avatar_url: response.user.avatar_url,
});
```

## 4.2 Cập nhật `api.config.ts` — Auth namespace (CR-001 §3.3)

```typescript
// ui/src/config/api.config.ts
export const API_CONFIG = {
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  useMockData: import.meta.env.VITE_USE_MOCK_DATA === 'true',

  // Namespaces
  auth:         '/v1/auth',           // NEW
  console:      '/v1/console',
  dashboard:    '/v1/console/dashboard',
  sessions:     '/v1/console/sessions',
  memory:       '/v1/console/memory',
  profiles:     '/v1/console/profiles',
  adaptive:     '/v1/console/adaptive',
  governance:   '/v1/console/governance',
  observability: '/v1/console/observability',
  pipelines:    '/v1/console/pipelines',
  infra:        '/v1/console/infra',
  org:          '/v1/console/org',
  sdk:          '/v1/console/sdk',
} as const;
```

---

## 5. Dev Mode Override

Khi `VNP_MEMORY_AUTH_DEV_MODE=true` (§9.3 config), Gateway skip JWT validation và inject default dev tenant. Frontend vẫn phải gọi `/v1/auth/login` nhưng backend sẽ accept mọi credential.
