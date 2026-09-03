# TASK-UI-016 — Xóa tất cả file `src/mock/*.ts` và `useMock` ternary

| Field | Value |
|---|---|
| **Task ID** | TASK-UI-016 |
| **Layer** | Frontend — TypeScript |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-001 §2 + §3](../solutions/SOL-001-Migration-Strategy.md) |
| **Priority** | 🔴 P0 (sau khi TASK-UI-005→015 hoàn thành) |
| **Depends On** | TASK-UI-005, TASK-UI-006, TASK-UI-007, TASK-UI-009, TASK-UI-010, TASK-UI-011, TASK-UI-012, TASK-UI-013, TASK-UI-014, TASK-UI-015 |
| **Estimated** | 1h |

---

## Context

Sau khi tất cả hooks đã được refactor sang gọi API thực, cần:
1. Xóa toàn bộ file mock (`/ui/src/mock/*.ts`)
2. Xóa biến `useMock` / `VITE_USE_MOCK_DATA` khỏi hooks
3. Xóa env var `VITE_USE_MOCK_DATA` khỏi `.env` files

---

## Goal

- Zero references đến mock data files
- Không còn `useMock` ternary trong bất kỳ hook nào
- `VITE_USE_MOCK_DATA` không còn trong production `.env`

---

## Target Files

| Action | File Path |
|---|---|
| DELETE | `ui/src/mock/dashboard.mock.ts` |
| DELETE | `ui/src/mock/session.mock.ts` |
| DELETE | `ui/src/mock/memory.mock.ts` |
| DELETE | `ui/src/mock/adaptive.mock.ts` |
| DELETE | `ui/src/mock/profile.mock.ts` |
| DELETE | `ui/src/mock/governance.mock.ts` |
| DELETE | `ui/src/mock/observability.mock.ts` |
| DELETE | `ui/src/mock/pipeline.mock.ts` |
| DELETE | `ui/src/mock/infrastructure.mock.ts` |
| MODIFY | `ui/src/config/api.config.ts` (xóa `useMockData` field) |
| MODIFY | `ui/.env.production` (xóa hoặc comment `VITE_USE_MOCK_DATA`) |

---

## Implementation

### Bước 1: Verify không còn references

```bash
# Kiểm tra trước khi xóa
grep -r "from.*mock/" ui/src/hooks/ ui/src/components/ ui/src/app/
grep -r "from.*mock/" ui/src/services/
# Output phải trống hoàn toàn
```

### Bước 2: Xóa mock files

```bash
rm ui/src/mock/dashboard.mock.ts
rm ui/src/mock/session.mock.ts
rm ui/src/mock/memory.mock.ts
rm ui/src/mock/adaptive.mock.ts
rm ui/src/mock/profile.mock.ts
rm ui/src/mock/governance.mock.ts
rm ui/src/mock/observability.mock.ts
rm ui/src/mock/pipeline.mock.ts
rm ui/src/mock/infrastructure.mock.ts

# Nếu còn file nào khác trong thư mục mock:
ls ui/src/mock/  # Kiểm tra thư mục có empty chưa
```

### Bước 3: Xóa `useMock` trong `api.config.ts`

Loại bỏ field `useMockData` khỏi `API_CONFIG`:

```typescript
// TRƯỚC
export const API_CONFIG = {
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  useMockData: import.meta.env.VITE_USE_MOCK_DATA === 'true',  // XÓA DÒNG NÀY
  // ...
};

// SAU
export const API_CONFIG = {
  baseUrl: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  // ...
};
```

### Bước 4: Xóa env var

```bash
# Trong .env.production — xóa hoặc comment dòng này:
# VITE_USE_MOCK_DATA=false

# Trong .env.local (dev) — giữ comment để document:
# VITE_USE_MOCK_DATA=true  # Deprecated — no longer used
```

### Bước 5: Chạy validation script

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

INLINE_MOCK=$(grep -rn "const mock[A-Z]" ui/src/hooks/ 2>/dev/null)
if [ -n "$INLINE_MOCK" ]; then
  echo "❌ FAIL: Found inline mock objects in hooks:"
  echo "$INLINE_MOCK"
  exit 1
fi

USE_MOCK_REF=$(grep -rn "useMock\|useMockData" ui/src/hooks/ 2>/dev/null)
if [ -n "$USE_MOCK_REF" ]; then
  echo "❌ FAIL: Found useMock references:"
  echo "$USE_MOCK_REF"
  exit 1
fi

echo "✅ PASS: No mock data found in hooks or components"
```

---

## Verification

```bash
cd ui
npx tsc --noEmit         # No type errors
npm run build            # Production build thành công
bash verify-no-mock.sh   # Script pass
```

**Expected output của validation script**:
```
=== Checking for remaining mock imports ===
✅ PASS: No mock data found in hooks or components
```
