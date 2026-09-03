# CR-001 — Authentication: Mock → Real Backend Auth API

| Field | Value |
|---|---|
| **CR ID** | CR-001 |
| **Title** | Authentication Service: Thay thế mock auth bằng real API |
| **Type** | Feature Implementation |
| **Priority** | P0 — Critical (Blocker cho tất cả CRs khác) |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | Auth |
| **Files thay đổi** | `ui/src/services/auth.ts` |

---

## 1. Hiện trạng

File [`ui/src/services/auth.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/services/auth.ts) hiện tại là **hoàn toàn mock** — không gọi bất kỳ HTTP request nào đến backend:

```typescript
// HIỆN TẠI — Mock hoàn toàn
export const authService = {
  async login(email: string, password: string) {
    await delay(1000);  // ← fake delay

    if (email === 'admin@vnp-memory.io' && password === 'admin') {
      return {
        token: 'mock-jwt-token-123456789',   // ← fake JWT
        user: { id: 'usr_1', name: 'Admin User', ... }
      };
    }
    throw new Error('Invalid email or password. Hint: admin@vnp-memory.io / admin');
  },

  async loginWithGoogle() {
    await delay(1200);
    return { token: 'mock-jwt-token-google-sso', ... };  // ← fake SSO
  },

  async register(name, email, password) {
    await delay(1500);
    return { token: 'mock-jwt-token-new-user', ... };  // ← fake registration
  }
};
```

**Vấn đề:**
- Token fake không thể dùng để gọi API thực → mọi request khác đều sẽ fail (401)
- Không có session validation thực
- Google SSO chỉ là placeholder
- `api-client.ts` đọc token từ `localStorage.getItem('access_token')` — token này cần là JWT thực từ backend

---

## 2. Backend API cần implement

### 2.1 Endpoints xác thực

| Method | Path | Mô tả |
|---|---|---|
| `POST` | `/v1/auth/login` | Email/password login |
| `POST` | `/v1/auth/logout` | Invalidate session |
| `POST` | `/v1/auth/refresh` | Refresh access token |
| `GET` | `/v1/auth/me` | Lấy thông tin user hiện tại |
| `POST` | `/v1/auth/google` | Google OAuth2 SSO (tương lai) |

> **Note**: Theo PRD §6.2, backend hỗ trợ **API Key (SHA-256)** và **JWT RS256**. Console UI nên dùng JWT RS256 cho browser session.

### 2.2 Request/Response schema

**POST /v1/auth/login**
```json
// Request
{
  "email": "admin@example.com",
  "password": "securepassword"
}

// Response 200
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "user": {
    "id": "usr_abc123",
    "name": "Admin User",
    "email": "admin@example.com",
    "role": "admin",
    "tenant_id": "tenant_xyz",
    "avatar_url": "https://..."
  }
}

// Response 401
{
  "message": "Invalid credentials",
  "code": "AUTH_INVALID_CREDENTIALS",
  "status": 401
}
```

**GET /v1/auth/me**
```json
// Response 200
{
  "id": "usr_abc123",
  "name": "Admin User",
  "email": "admin@example.com",
  "role": "admin",
  "tenant_id": "tenant_xyz",
  "avatar_url": "https://..."
}
```

**POST /v1/auth/refresh**
```json
// Request
{ "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..." }

// Response 200
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600
}
```

### 2.3 Database schema

Backend cần bảng `users` và `sessions` trong PostgreSQL:

```sql
-- Users table (nếu chưa có)
CREATE TABLE users (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        TEXT NOT NULL,
  email       TEXT UNIQUE NOT NULL,
  password_hash TEXT,                    -- bcrypt hash
  role        TEXT DEFAULT 'user',       -- 'admin', 'user', 'viewer'
  tenant_id   UUID NOT NULL,
  avatar_url  TEXT,
  created_at  TIMESTAMPTZ DEFAULT NOW(),
  updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Sessions table (cho JWT rotation)
CREATE TABLE sessions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
  refresh_token TEXT UNIQUE NOT NULL,
  expires_at    TIMESTAMPTZ NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 3. Frontend thay đổi

### 3.1 Cập nhật `auth.ts`

```typescript
// MỚI — Gọi API thực
import { apiClient } from '../lib/api-client';

const BASE = '/v1/auth';

export interface AuthUser {
  id: string;
  name: string;
  email: string;
  role: string;
  tenant_id: string;
  avatar_url?: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
  user: AuthUser;
}

export const authService = {
  async login(email: string, password: string): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>(`${BASE}/login`, { email, password });
    // Lưu tokens
    localStorage.setItem('access_token', response.access_token);
    localStorage.setItem('refresh_token', response.refresh_token);
    if (response.user.tenant_id) {
      localStorage.setItem('tenant_id', response.user.tenant_id);
    }
    return response;
  },

  async logout(): Promise<void> {
    const refreshToken = localStorage.getItem('refresh_token');
    await apiClient.post<void>(`${BASE}/logout`, { refresh_token: refreshToken });
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('tenant_id');
  },

  async getMe(): Promise<AuthUser> {
    return apiClient.get<AuthUser>(`${BASE}/me`);
  },

  async refreshToken(): Promise<{ access_token: string; expires_in: number }> {
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) throw new Error('No refresh token');
    const response = await apiClient.post<{ access_token: string; expires_in: number }>(
      `${BASE}/refresh`,
      { refresh_token: refreshToken }
    );
    localStorage.setItem('access_token', response.access_token);
    return response;
  },
};
```

### 3.2 Cập nhật type trong `useStore.ts`

Đồng bộ `UserProfile` trong store với `AuthUser` từ API:

```typescript
// Thêm tenant_id vào UserProfile
export interface UserProfile {
  id: string;
  name: string;
  email: string;
  role: string;
  tenant_id?: string;  // NEW
  avatar?: string;
}
```

### 3.3 Cập nhật `api.config.ts`

```typescript
// Thêm auth namespace
export const API_CONFIG = {
  // ... existing ...
  auth: '/v1/auth',  // NEW
} as const;
```

### 3.4 Token refresh interceptor

Thêm auto-refresh vào `api-client.ts` khi nhận 401:

```typescript
// Trong fetchWrapper, sau khi nhận 401:
if (response.status === 401) {
  try {
    await authService.refreshToken();
    // Retry request với token mới
    return fetchWrapper<T>(url, options);
  } catch {
    // Redirect to login
    window.location.href = '/login';
    throw new AppError('Session expired', 'AUTH_EXPIRED', 401);
  }
}
```

---

## 4. Điều kiện hoàn thành (Definition of Done)

- [ ] `POST /v1/auth/login` trả về JWT thực từ backend
- [ ] `GET /v1/auth/me` trả về thông tin user từ database
- [ ] `POST /v1/auth/refresh` refresh token hoạt động
- [ ] Token được lưu đúng trong localStorage (key: `access_token`, `refresh_token`, `tenant_id`)
- [ ] `api-client.ts` tự động gửi `Authorization: Bearer <token>` và `x-tenant-id`
- [ ] Login page không còn chấp nhận hardcoded credentials `admin/admin`
- [ ] Auth flow hoạt động end-to-end: login → gọi API protected → logout

---

## 5. Notes

> **Dev Mode**: Backend có `AUTH_DEV_MODE=true` cho phép skip auth ở localhost. Frontend có thể tạm thời dùng mode này trong quá trình dev, nhưng phải tắt ở staging/production.

> **Google SSO**: Có thể implement sau — cần backend OAuth2 callback endpoint. Phase 1 chỉ cần email/password.
