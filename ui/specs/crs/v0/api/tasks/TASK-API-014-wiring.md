# TASK-API-014 — Wiring: App Setup + Mock Cleanup + Old Services Migration

**Task ID:** TASK-API-014
**Status:** ✅ COMPLETED — 2026-06-17
**Sprint:** 4 — Final Integration
**Solution:** All API-SOL-001 → API-SOL-012
**Depends on:** TASK-API-001 → TASK-API-013
**Ước tính:** 3h
**Priority:** P0 — Critical (cuối cùng để hoàn thành migration)

---

## Mục tiêu

Kết nối toàn bộ các clients/hooks mới vào app, xóa sạch mock data và các services cũ không còn cần thiết.

---

## Công việc cụ thể

### 1. Cập nhật `ui/src/main.tsx` — Providers

```typescript
import { StrictMode }         from 'react';
import { createRoot }         from 'react-dom/client';
import { RouterProvider }     from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools }  from '@tanstack/react-query-devtools';

import { queryClient }  from './api/queryClient';
import { AuthProvider } from './contexts/AuthContext';
import { router }       from './router';
import './index.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RouterProvider router={router} />
      </AuthProvider>
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  </StrictMode>,
);
```

### 2. Migrate hooks imports trong pages

Chạy các lệnh tìm kiếm và cập nhật:

```bash
# Tìm tất cả files dùng services cũ
grep -r "from.*services/auth\|from.*services/dashboard\|from.*services/session\|from.*services/memory" \
  ui/src --include="*.ts" --include="*.tsx" -l

# Tìm files import hooks cũ
grep -r "from.*hooks/useDashboard\|from.*hooks/useSessions\|from.*hooks/useMemory" \
  ui/src --include="*.ts" --include="*.tsx" -l
```

Với mỗi file tìm được, cập nhật import paths:

| Cũ | Mới |
|---|---|
| `../services/auth` | `../api/clients/auth.client` |
| `../services/dashboard.service` | (dùng hook) |
| `../services/session.service` | (dùng hook) |
| `../services/memory.service` | (dùng hook) |
| `../hooks/useDashboard` | `../api/hooks/useDashboard` |
| `../hooks/useSessions` | `../api/hooks/useSessions` |
| `../hooks/useMemory` | `../api/hooks/useMemory` |

### 3. Xóa toàn bộ mock files

```bash
# List mock files cần xóa
ls ui/src/mock/

# Xóa sau khi đã verify không còn ai import
rm ui/src/mock/dashboard.mock.ts
rm ui/src/mock/session.mock.ts
rm ui/src/mock/memory.mock.ts
# xóa tất cả mock files còn lại trong thư mục mock/
```

### 4. Xóa `useMockData` flag trong `api.config.ts`

```typescript
// ui/src/config/api.config.ts — TRƯỚC
export const API_CONFIG = {
  useMockData: true,   // ← xóa dòng này
  baseUrl: '/v1',
  // ...
};

// SAU
export const API_CONFIG = {
  baseUrl: '/v1',
  auth:    '/v1/auth',
  console: '/v1/console',
} as const;
```

### 5. Xóa các `const useMock = API_CONFIG.useMockData` patterns

```bash
grep -r "useMock\|API_CONFIG.useMockData\|useMockData" ui/src --include="*.ts" --include="*.tsx"
# Xóa từng occurrence và remove conditional mock branch
```

### 6. Kiểm tra Protected Routes

Đảm bảo route guard dùng `useAuth()` từ `AuthContext` thay vì check localStorage trực tiếp:

```typescript
// ui/src/components/ProtectedRoute.tsx
import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) return <LoadingSpinner />;
  if (!isAuthenticated) return <Navigate to="/login" replace />;

  return <>{children}</>;
}
```

### 7. Cập nhật `vite.config.ts` — API proxy (dev environment)

```typescript
// ui/vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },
  // ...
});
```

### 8. Thêm `.env.local` cho development

```bash
# ui/.env.local
VITE_API_BASE_URL=http://localhost:8080
```

---

## Validation Script

```bash
#!/bin/bash
# ui/scripts/validate-no-mock.sh
echo "Checking for mock imports..."

MOCK_COUNT=$(grep -r "mock\|useMockData\|useMock" ui/src --include="*.ts" --include="*.tsx" | grep -v "node_modules" | wc -l)

if [ "$MOCK_COUNT" -gt 0 ]; then
  echo "❌ Found $MOCK_COUNT mock references:"
  grep -r "mock\|useMockData\|useMock" ui/src --include="*.ts" --include="*.tsx"
  exit 1
else
  echo "✅ No mock references found"
fi

echo "Running TypeScript check..."
cd ui && npx tsc --noEmit && echo "✅ TypeScript OK" || echo "❌ TypeScript errors"
```

---

## Files chỉnh sửa / xóa

```
ui/src/
├── main.tsx                   ← MODIFY (add QueryClientProvider + AuthProvider)
├── config/api.config.ts       ← MODIFY (remove useMockData flag)
├── components/
│   └── ProtectedRoute.tsx     ← MODIFY (use AuthContext)
├── mock/                      ← DELETE (entire directory)
│   ├── dashboard.mock.ts
│   ├── session.mock.ts
│   └── memory.mock.ts
└── vite.config.ts             ← MODIFY (add proxy)
ui/.env.local                  ← CREATE
ui/scripts/validate-no-mock.sh ← CREATE
```

---

## Acceptance Criteria

- [x] App khởi động với `npm run dev` không lỗi
- [x] `POST /v1/auth/login` → login thành công, navigate về dashboard
- [x] Dashboard hiển thị dữ liệu thực (không còn hardcoded numbers)
- [x] Sessions list load từ API với pagination hoạt động
- [x] Memory search trả về kết quả thực từ backend
- [x] `bash ui/scripts/validate-no-mock.sh` → "✅ No mock references found"
- [x] `cd ui && npx tsc --noEmit` → 0 errors
- [x] Logout → redirect `/login`, token bị xóa khỏi localStorage
- [x] Page reload → `useCurrentUser()` restore session từ localStorage token
- [x] 401 response → auto-refresh → retry request gốc (không logout)
- [x] `src/mock/` directory không còn tồn tại

---

## Sau khi hoàn thành

```bash
cd ui
npm run dev

# Manual test checklist:
# 1. Login với real credentials
# 2. Xem Dashboard → thấy real metrics
# 3. Xem Sessions → thấy real sessions từ DB
# 4. Search Memory → thấy kết quả từ backend
# 5. Logout → redirect login, localStorage cleared
# 6. Refresh browser khi đã login → vẫn còn session
```
