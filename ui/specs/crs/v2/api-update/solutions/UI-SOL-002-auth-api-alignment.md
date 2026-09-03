# UI Solution: UI-SOL-002 — Auth API Frontend Alignment

**Solution ID:** UI-SOL-002  
**CR References:** [CR-001-auth-api](../../../../docs/crs/v2/api-update/CR-001-auth-api.md) + [CR-001-frontend-api-alignment](../../../../docs/crs/v2/api-update/CR-001-frontend-api-alignment.md)  
**Backend Solution:** [SOL-001-auth-api.md](../../../../backend/specs/crs/v2/api-update/solutions/SOL-001-auth-api.md)  
**Feature:** Auth API — Login, Logout, Refresh, Me  
**Priority:** 🔴 Critical  
**Frontend Component:** `ui/src/services/auth.ts`, `ui/src/contexts/AuthContext.tsx`

---

## 1. Mục Đích

Align frontend Auth implementation với backend Auth API contract:
- Đảm bảo `POST /v1/auth/login` request/response types chính xác
- Implement auto-refresh token on 401
- Handle Google SSO stub (throw error gracefully)
- Fix `tenant_id` extraction từ `LoginResponse`

---

## 2. Backend API Contract (source of truth)

```http
POST /v1/auth/login
{ "email": string, "password": string }
→ {
    "access_token":  string,   // JWT RS256
    "refresh_token": string,
    "expires_in":    number,   // seconds
    "token_type":    "Bearer",
    "user": {
      "id":        string,
      "name":      string,
      "email":     string,
      "role":      string,     // "admin" | "super_admin"
      "tenant_id": string,
      "avatar_url?": string
    }
  }

POST /v1/auth/refresh
{ "refresh_token": string }
→ { "access_token": string, "expires_in": number }

GET /v1/auth/me
→ AuthUser

POST /v1/auth/logout
{ "refresh_token": string }
→ void (204)
```

---

## 3. Frontend Implementation

### 3.1 Auth Service (`ui/src/services/auth.ts`)

```typescript
// CHANGES:
// 1. Fix: store tenant_id từ user object (không phải top-level)
// 2. Fix: refresh endpoint path "/v1/auth/refresh"
// 3. Add: auto-retry on 401 in api-client.ts

export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await publicFetch('/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  const data: LoginResponse = await res.json();
  
  // Store tokens
  localStorage.setItem('access_token', data.access_token);
  localStorage.setItem('refresh_token', data.refresh_token);
  localStorage.setItem('tenant_id', data.user.tenant_id);  // FIX: from user object
  
  return data;
}

export async function refreshToken(): Promise<string> {
  const refresh = localStorage.getItem('refresh_token');
  if (!refresh) throw new Error('No refresh token');
  
  const res = await publicFetch('/v1/auth/refresh', {    // FIX: correct path
    method: 'POST',
    body: JSON.stringify({ refresh_token: refresh }),
  });
  const data: RefreshResponse = await res.json();
  localStorage.setItem('access_token', data.access_token);
  return data.access_token;
}
```

### 3.2 API Client Auto-Refresh (`ui/src/lib/api-client.ts`)

```typescript
// Intercept 401 → try refresh → retry original request
async function fetchWithAuth(url: string, options: RequestInit): Promise<Response> {
  let res = await fetch(url, withAuthHeaders(options));
  
  if (res.status === 401) {
    try {
      await refreshToken();
      res = await fetch(url, withAuthHeaders(options));  // retry
    } catch {
      // Refresh failed → redirect to login
      localStorage.clear();
      window.location.href = '/login';
    }
  }
  
  return res;
}
```

### 3.3 TypeScript Types Alignment

```typescript
// ui/src/types/auth.ts — ensure exact match with backend

interface AuthUser {
  id:          string;
  name:        string;
  email:       string;
  role:        'admin' | 'super_admin';     // STRICT union type
  tenant_id:   string;
  avatar_url?: string;
}

interface LoginResponse {
  access_token:  string;
  refresh_token: string;
  expires_in:    number;
  token_type:    'Bearer';
  user:          AuthUser;
}
```

### 3.4 Google SSO Stub (graceful)

```typescript
// loginWithGoogle throws with friendly message
export async function loginWithGoogle(): Promise<never> {
  throw new Error('Google SSO is not yet available. Use email/password login.');
}
// UI shows: Toast("Google login coming soon! Please use email/password")
```

---

## 4. AuthContext Changes

```typescript
// ui/src/contexts/AuthContext.tsx
// Add: token expiry tracking + background refresh

const [tokenExpiresAt, setTokenExpiresAt] = useState<number | null>(null);

// Schedule refresh 60s before expiry
useEffect(() => {
  if (!tokenExpiresAt) return;
  const msUntilExpiry = tokenExpiresAt - Date.now();
  const timer = setTimeout(() => refreshToken(), msUntilExpiry - 60_000);
  return () => clearTimeout(timer);
}, [tokenExpiresAt]);
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] Login với email/password → tokens stored in localStorage đúng keys
- [ ] `tenant_id` extracted từ `response.user.tenant_id` (không phải top-level)
- [ ] 401 response → auto-refresh → retry original request trong `< 300ms`
- [ ] Refresh thất bại → redirect `/login` + clear localStorage
- [ ] Google login → toast "coming soon" (không crash)
- [ ] Token expiry tracking: background refresh 60s trước khi hết hạn
- [ ] `role` field strict typed: `'admin' | 'super_admin'`
