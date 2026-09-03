# TASK-UI-004 — Cập nhật `store/useStore.ts`: Thêm `tenant_id` vào UserProfile

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-004 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-002 §4.1](../solutions/SOL-002-Auth-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | TASK-UI-002 |
| **Estimated** | 0.5h |

---

## Context

`UserProfile` trong Zustand store hiện chưa có `tenant_id`. Khi login thành công, store phải được update với đầy đủ thông tin user bao gồm `tenant_id` để các component khác có thể đọc.

---

## Goal

- Thêm `tenant_id: string` vào interface `UserProfile`
- Thêm `avatar_url?: string` nếu chưa có
- Cập nhật login flow để gọi `setUser` với đầy đủ fields

---

## Target Files

| Action | File Path |
|---|---|
| MODIFY | `ui/src/store/useStore.ts` (hoặc file tương đương chứa UserProfile type) |

---

## Implementation

### Bước 1: Cập nhật `UserProfile` interface

Tìm interface `UserProfile` trong store và thêm fields:

```typescript
export interface UserProfile {
  id: string;
  name: string;
  email: string;
  role: string;
  tenant_id: string;   // NEW — required for multi-tenant API calls
  avatar_url?: string; // NEW — optional avatar URL from backend
}
```

### Bước 2: Cập nhật Login flow

Tìm nơi gọi login (thường trong `LoginPage.tsx` hoặc `useAuth.ts`) và đảm bảo `setUser` nhận đủ fields:

```typescript
// Trong Login handler:
const response = await authService.login(email, password);
useStore.getState().setUser({
  id:         response.user.id,
  name:       response.user.name,
  email:      response.user.email,
  role:       response.user.role,
  tenant_id:  response.user.tenant_id,  // NEW
  avatar_url: response.user.avatar_url, // NEW
});
```

### Bước 3: Cập nhật `getMe` flow (nếu có session restore)

Nếu app có logic restore session khi reload (gọi `GET /v1/auth/me`):

```typescript
const user = await authService.getMe();
useStore.getState().setUser({
  id:         user.id,
  name:       user.name,
  email:      user.email,
  role:       user.role,
  tenant_id:  user.tenant_id,
  avatar_url: user.avatar_url,
});
```

---

## Verification

```bash
cd ui
npx tsc --noEmit
```

Manual test: Login → kiểm tra store state trong React DevTools có đủ `id`, `name`, `email`, `role`, `tenant_id`.
