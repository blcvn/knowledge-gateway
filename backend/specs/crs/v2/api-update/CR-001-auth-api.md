# CR-001: Authentication API

**CR ID**: CR-001-auth-api  
**Status**: Open  
**Priority**: 🔴 Critical  
**Target Components**: `vnp-gateway` (router), `vnp-auth` (new service) or `sm-auth` (existing)  
**Frontend Source**: `ui/src/services/auth.ts`, `ui/src/services/auth-api.service.ts`  
**Created**: 2026-06-18

---

## Problem

The frontend's HTTP client (`ui/src/lib/api-client.ts`) depends on a fully functional Auth API at `/v1/auth/*`. Currently, the gateway has **no auth routes at all** — it only validates incoming JWTs/API keys via middleware but has no mechanism to issue them.

The `401` auto-refresh loop in `api-client.ts` calls `POST /v1/auth/refresh`; without this endpoint, every expired token causes an infinite error loop or logout. This is a **blocking issue** for the entire frontend.

---

## Required Endpoints

### POST `/v1/auth/login`

**Auth**: Public (no token required)

**Request Body:**
```json
{
  "email":    "string",
  "password": "string"
}
```

**Response (200):**
```json
{
  "access_token":  "string (JWT RS256)",
  "refresh_token": "string",
  "expires_in":    3600,
  "token_type":    "Bearer",
  "user": {
    "id":         "string",
    "name":       "string",
    "email":      "string",
    "role":       "admin | super_admin | user",
    "tenant_id":  "string",
    "avatar_url": "string | null"
  }
}
```

### POST `/v1/auth/logout`

**Auth**: Bearer token required

**Request Body:**
```json
{ "refresh_token": "string" }
```

**Response (200):** `204 No Content`

**Behaviour**: Invalidate refresh token server-side. Frontend always clears `localStorage` regardless of response.

### GET `/v1/auth/me`

**Auth**: Bearer token required

**Response (200):**
```json
{
  "id":         "string",
  "name":       "string",
  "email":      "string",
  "role":       "string",
  "tenant_id":  "string",
  "avatar_url": "string | null"
}
```

### POST `/v1/auth/refresh`

**Auth**: Public (refresh token passed in body)

**Request Body:**
```json
{ "refresh_token": "string" }
```

**Response (200):**
```json
{
  "access_token": "string",
  "expires_in":   3600
}
```

**Response (401):** If refresh token is expired or revoked → frontend redirects to `/login`.

---

## Implementation Notes

1. **Gateway Router**: Add routes in `gateway/internal/adapter/handler/router.go`:
   ```go
   mux.HandleFunc("POST /v1/auth/login",   authH.Login)
   mux.HandleFunc("POST /v1/auth/logout",  authH.Logout)
   mux.HandleFunc("GET /v1/auth/me",       authH.Me)
   mux.HandleFunc("POST /v1/auth/refresh", authH.Refresh)
   ```

2. **Handler**: Create `gateway/internal/adapter/handler/auth.go` with `AuthHandler` struct forwarding to `sm-auth`.

3. **Auth middleware**: `/v1/auth/login` and `/v1/auth/refresh` must be **excluded** from JWT validation middleware (they're pre-auth).

4. **Backend Service**: Forward to `sm-auth` (service already exists in `services/sm-auth/`).

---

## Acceptance Criteria

- [ ] `POST /v1/auth/login` returns `LoginResponse` with valid JWT and user details
- [ ] `POST /v1/auth/logout` invalidates the refresh token server-side
- [ ] `GET /v1/auth/me` returns current user decoded from JWT
- [ ] `POST /v1/auth/refresh` issues a new `access_token` when given valid `refresh_token`
- [ ] `POST /v1/auth/refresh` returns `401` on expired/invalid refresh token
- [ ] Login and refresh endpoints bypass JWT validation middleware
