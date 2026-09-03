# Solution: Frontend Mock-to-API Migration Strategy

| Field | Value |
|---|---|
| **Solution ID** | SOL-001 |
| **Liên quan đến CRs** | CR-001 đến CR-011 |
| **Loại** | Architecture Decision |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Implemented** | 2026-06-17 |

---

## 1. Tổng quan

Tài liệu này mô tả **chiến lược kỹ thuật** để thực hiện toàn bộ 11 Change Requests — chuyển frontend từ mock data sang real API một cách an toàn, có thể kiểm tra, không gây downtime.

---

## 2. Nguyên tắc

### 2.1 Không break UI trong quá trình migration

Mỗi CR được thực hiện **module-by-module** theo thứ tự ưu tiên. Trong quá trình transition, các module chưa migrate vẫn có thể dùng mock (flag `VITE_USE_MOCK_DATA`) trong dev environment.

### 2.2 Backend-first

Mỗi backend endpoint phải **hoàn thành và tested** trước khi frontend chuyển sang dùng nó. Thứ tự:
```
1. Backend implements endpoint
2. Backend viết integration test
3. Frontend xóa mock, chuyển sang service call
4. Frontend viết smoke test với endpoint thực (staging env)
```

### 2.3 Typesafe contract

TypeScript types trong `ui/src/types/*.ts` là **contract chung** giữa frontend và backend. Backend phải serialize response theo đúng casing và shape của TypeScript interface.

---

## 3. Pattern chuẩn để migrate một module

### Bước 1: Backend implement endpoint (Go)

```go
// Ví dụ: handler cho GET /v1/console/dashboard/metrics
func (h *ConsoleHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    tenantID := middleware.TenantFromContext(ctx)

    metrics, err := h.dashboardSvc.GetKPIs(ctx, tenantID)
    if err != nil {
        httputil.Error(w, err)
        return
    }
    httputil.JSON(w, http.StatusOK, metrics)
}
```

Response phải match TypeScript interface (dùng camelCase JSON tags):

```go
type KPIData struct {
    ActiveAgents       int     `json:"activeAgents"`
    RecallLatencyP50Ms float64 `json:"recallLatencyP50Ms"`
    RecallLatencyP95Ms float64 `json:"recallLatencyP95Ms"`
    // ...
}
```

### Bước 2: Frontend xóa mock import

```typescript
// TRƯỚC
import { dashboardMock } from '../mock/dashboard.mock';
const useMock = API_CONFIG.useMockData;

export function useMetrics() {
  return useQuery({
    queryFn: useMock
      ? () => Promise.resolve(dashboardMock.kpis)
      : () => dashboardService.getMetrics(),
  });
}
```

```typescript
// SAU — sạch, không còn mock
import { dashboardService } from '../services/dashboard.service';

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: () => dashboardService.getMetrics(),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 60 * 1000,
  });
}
```

### Bước 3: Xóa file mock (sau khi toàn module đã migrate)

```bash
rm ui/src/mock/dashboard.mock.ts
# Verify không còn import nào trỏ vào file này
grep -r "dashboard.mock" ui/src/  # Phải không có kết quả
```

---

## 4. Xử lý lỗi chuẩn

### 4.1 Loading skeleton

Mọi component sử dụng API query phải có loading state:

```tsx
const { data, isLoading, isError, error } = useMetrics();

if (isLoading) return <DashboardSkeleton />;
if (isError) return <ErrorCard message={error.message} />;
return <DashboardContent data={data} />;
```

### 4.2 Retry & Error toast

`api-client.ts` đã implement exponential backoff cho GET requests (3 retries mặc định). Với lỗi không retry được (4xx), React Query sẽ `onError` → toast notification.

### 4.3 Stale data

Khi backend trả về lỗi sau khi đã có data cached, React Query giữ data cũ và hiển thị warning banner thay vì xóa trắng UI.

---

## 5. Auth token flow

```
1. User login → POST /v1/auth/login
2. Backend trả về { access_token, refresh_token, user }
3. Frontend lưu:
   localStorage['access_token']  = access_token
   localStorage['refresh_token'] = refresh_token
   localStorage['tenant_id']     = user.tenant_id
4. api-client.ts inject vào mọi request:
   Authorization: Bearer <access_token>
   x-tenant-id: <tenant_id>
5. Khi nhận 401 → auto refresh → retry
6. Refresh thất bại → redirect /login
```

---

## 6. Environment configuration

### Development (mock mode — tạm thời)
```env
# ui/.env.local
VITE_API_BASE_URL=http://localhost:8080
VITE_USE_MOCK_DATA=true
```

### Staging (mixed — từng module sau khi backend ready)
```env
# ui/.env.staging
VITE_API_BASE_URL=https://staging.vnp-memory.io
VITE_USE_MOCK_DATA=false
```

### Production
```env
# ui/.env.production
VITE_API_BASE_URL=https://api.vnp-memory.io
# VITE_USE_MOCK_DATA không set = false
```

---

## 7. Checklist migration mỗi module

```
[ ] Backend endpoint implement + integration test
[ ] TypeScript types khớp với Go struct (casing, fields)
[ ] Hook xóa mock import, chỉ gọi service
[ ] Loading skeleton hiển thị đúng
[ ] Error state hiển thị đúng (message từ AppError)
[ ] File mock xóa nếu không còn reference nào
[ ] Smoke test trên staging environment
[ ] Code review: không còn bất kỳ "mock" nào trong file
```

---

## 8. Validation script

Sau khi hoàn thành migration, chạy script này để verify:

```bash
#!/bin/bash
# verify-no-mock.sh
echo "=== Checking for remaining mock imports ==="

MOCK_REFS=$(grep -r "from.*mock/" ui/src/hooks/ ui/src/app/ 2>/dev/null)
if [ -n "$MOCK_REFS" ]; then
  echo "❌ FAIL: Found mock imports:"
  echo "$MOCK_REFS"
  exit 1
fi

INLINE_MOCK=$(grep -r "const mock" ui/src/hooks/ 2>/dev/null)
if [ -n "$INLINE_MOCK" ]; then
  echo "❌ FAIL: Found inline mock objects in hooks:"
  echo "$INLINE_MOCK"
  exit 1
fi

echo "✅ PASS: No mock data found in hooks or components"
```
