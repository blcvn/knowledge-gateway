# CR-000 — Frontend Data Layer Migration: Mock → Real API

| Field | Value |
|---|---|
| **CR ID** | CR-000 |
| **Title** | Frontend Data Layer Migration: Mock → Real Backend API |
| **Type** | Architecture Change |
| **Priority** | P0 — Critical |
| **Status** | ✅ Implemented |
| **Created** | 2026-06-16 |
| **Module** | All UI modules (10 feature areas) |

---

## 1. Tổng quan

Hiện tại toàn bộ dữ liệu hiển thị trên Console UI (VNP Memory) đang được lấy từ **mock data cứng trong source code** (`/ui/src/mock/*.ts`). Khi `VITE_USE_MOCK_DATA=true`, mọi hook đều trả về dữ liệu fake thay vì gọi API thực.

CR này định nghĩa toàn bộ các thay đổi cần thực hiện để **loại bỏ mock data** và **kết nối frontend hoàn toàn với backend API**, lấy dữ liệu từ database thực.

---

## 2. Phạm vi thay đổi

| CR | Module | File Mock | Service | Hooks |
|---|---|---|---|---|
| CR-001 | Auth | `auth.ts` (services) | `POST /v1/auth/*` | `authService` |
| CR-002 | Dashboard | `dashboard.mock.ts` | `dashboard.service.ts` | `useDashboard.ts` |
| CR-003 | Sessions Explorer | `session.mock.ts` | `session.service.ts` | `useSessions.ts` |
| CR-004 | Memory Explorer | `memory.mock.ts` | `memory.service.ts` | `useMemory.ts` |
| CR-005 | Adaptive Memory | `adaptive.mock.ts` + hook inline | `adaptive.service.ts` | `useAdaptiveMemory.ts` |
| CR-006 | User Profiles | `profile.mock.ts` | `profile.service.ts` | `useProfiles.ts` |
| CR-007 | Governance Center | `governance.mock.ts` | `governance.service.ts` | `useGovernance.ts` |
| CR-008 | Observability | `observability.mock.ts` | `observability.service.ts` | `useObservability.ts` |
| CR-009 | Pipelines Monitor | `pipeline.mock.ts` | `pipeline.service.ts` | `usePipelines.ts` |
| CR-010 | Infrastructure Health | `infrastructure.mock.ts` | `infrastructure.service.ts` | `useInfrastructure.ts` |
| CR-011 | Organization Settings & API SDK | `useOrganizationSettings.ts` (inline) + `useApiSdk.ts` (inline) | N/A → mới | `useOrganizationSettings.ts`, `useApiSdk.ts` |

---

## 3. Trạng thái hiện tại

### Cơ chế mock hiện tại

Tất cả các hooks đều sử dụng pattern:

```typescript
const useMock = API_CONFIG.useMockData;  // VITE_USE_MOCK_DATA=true

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: useMock
      ? () => Promise.resolve(dashboardMock.kpis)   // ← mock data
      : () => dashboardService.getMetrics(),          // ← API thực
  });
}
```

Biến `useMockData` được đọc một lần khi module load (`const useMock = API_CONFIG.useMockData`) → **không thể toggle runtime**.

### Vấn đề

1. **Dữ liệu không thực**: Console hiển thị số liệu cứng, không phản ánh trạng thái hệ thống thực
2. **Dead code**: Mock data chiếm ~12KB code không cần thiết ở production
3. **Auth bypass**: `authService` (services/auth.ts) vẫn dùng mock validation hoàn toàn — không gọi backend
4. **Org Settings / API SDK**: Dữ liệu thành viên, API keys, webhooks, rate limits bị hardcode trong hooks (`useOrganizationSettings.ts`, `useApiSdk.ts`)
5. **Type mismatch nguy cơ**: Một số kiểu trả về trong service dùng `unknown` — cần align với backend schema thực

---

## 4. Kiến trúc đích

### 4.1 Tắt mock mode

Đặt `VITE_USE_MOCK_DATA=false` (hoặc xóa biến env) ở môi trường production/staging.

### 4.2 Backend endpoints cần implement

Toàn bộ endpoints đã được định nghĩa trong `api.config.ts` namespace `/v1/console/*`. Backend cần đảm bảo trả về đúng schema theo TypeScript types tại `/ui/src/types/`.

### 4.3 Luồng dữ liệu đích

```
Browser → apiClient (fetch) → Bearer Token + x-tenant-id
    │
    → vnp-gateway :8080 /v1/console/*
         │
         → Console Handlers (Go)
              │
              ├── dashboard/* → Query engines health + metrics từ DB/Redis
              ├── sessions/*  → PostgreSQL sessions table
              ├── memory/*    → Search hub (cross-engine gRPC fan-out)
              ├── profiles/*  → Memobase service (PostgreSQL)
              ├── adaptive/*  → Supermemory service
              ├── governance/* → PostgreSQL tenants/policies/audit
              ├── observability/* → OpenTelemetry/Prometheus data
              ├── pipelines/* → NATS JetStream queue metrics
              ├── infra/*     → Service registry health checks
              └── graph/*     → Graphiti (Neo4j)
```

---

## 5. Ưu tiên thực hiện

| Thứ tự | CR | Lý do ưu tiên |
|---|---|---|
| 1 | CR-001 (Auth) | Blocker cho tất cả các CR khác — cần token thực |
| 2 | CR-002 (Dashboard) | Landing page, critical path UX |
| 3 | CR-003 (Sessions) | Core feature — session management |
| 4 | CR-004 (Memory) | Core feature — memory search |
| 5 | CR-006 (User Profiles) | High business value |
| 6 | CR-007 (Governance) | Compliance requirement |
| 7 | CR-005 (Adaptive) | Secondary feature |
| 8 | CR-008 (Observability) | Ops monitoring |
| 9 | CR-009 (Pipelines) | Ops monitoring |
| 10 | CR-010 (Infrastructure) | Ops monitoring |
| 11 | CR-011 (Org Settings / API SDK) | Admin feature |

---

## 6. Điều kiện hoàn thành toàn bộ migration

- [ ] `VITE_USE_MOCK_DATA` không còn được dùng trong production build
- [ ] Tất cả hooks không còn import từ `../mock/*`
- [ ] `services/auth.ts` gọi real backend authentication endpoint
- [ ] Tất cả services xử lý lỗi và loading states đúng cách
- [ ] E2E test với backend thực vượt qua
- [ ] Không còn hardcoded mock data trong bất kỳ hook nào

---

## 7. References

- Backend API config: [`/ui/src/config/api.config.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/config/api.config.ts)
- API client: [`/ui/src/lib/api-client.ts`](file:///Users/binhnt/Work/blockchain/vnp-memory/ui/src/lib/api-client.ts)
- PRD Section 7 (Console Features): [`/docs/product/PRD.md`](file:///Users/binhnt/Work/blockchain/vnp-memory/docs/product/PRD.md)
- CR chi tiết: `CR-001` đến `CR-011` trong cùng thư mục này
