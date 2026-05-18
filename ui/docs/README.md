# VNP Memory Console UI

> **Operating System for AI Cognition** — Enterprise-grade Control Plane for the VNP Memory ecosystem.

## Mục Đích
Cung cấp giao diện quản trị tập trung (Control Plane) cho toàn bộ hệ sinh thái VNP Memory. Hỗ trợ quản lý tenant, quan sát memory flow, debug agent context, và theo dõi knowledge graph.

## Tech Stack
| Layer | Tool |
|---|---|
| **Framework** | Vite 6 + React 18 |
| **Language** | TypeScript (strict) |
| **UI Components** | shadcn/ui + Radix UI |
| **Styling** | TailwindCSS 4 + Framer Motion |
| **Server State** | @tanstack/react-query |
| **Client State** | Zustand |
| **Auth** | Custom AuthProvider (JWT + RBAC) |
| **Graph** | React Flow |
| **Charts** | Recharts |
| **Testing** | Vitest + Playwright |
| **CI/CD** | GitHub Actions |

## Quick Start
```bash
# Cài đặt dependencies
pnpm install

# Khởi chạy development server
pnpm dev

# Build production
pnpm build

# Chạy unit tests
pnpm exec vitest run

# Chạy E2E tests
pnpm exec playwright test
```

## Cấu Trúc Dự Án
```
ui/
├── docs/                       # Tài liệu kiến trúc & thiết kế
│   ├── architecture.md         # Kiến trúc tổng quan
│   ├── navigations/            # Luồng điều hướng theo persona
│   └── screens/                # Đặc tả giao diện từng màn hình
├── specs/                      # Đặc tả kỹ thuật
│   ├── features/               # Đặc tả tính năng (FEAT-001..011)
│   ├── solutions/              # Giải pháp triển khai (SOL-001)
│   └── tasks/                  # Backlog tác vụ (TASK-001..043)
├── src/
│   ├── app/components/         # 11 MVP route-level modules
│   ├── components/             # Shared components (ErrorBoundary, Fallback)
│   ├── lib/                    # Core infrastructure (API, Auth, Logger, Query)
│   ├── store/                  # Zustand global state
│   ├── styles/                 # CSS (theme, fonts, tailwind)
│   └── __tests__/              # Test files
└── .github/workflows/          # CI/CD pipeline
```

## Tài Liệu Liên Quan
- [Architecture](docs/architecture.md): Tổng quan kiến trúc frontend (v2.0).
- [SOL-001](specs/solutions/SOL-001-ui-implementation.md): Giải pháp triển khai MVP.
- [Changelog](docs/changelog.md): Lịch sử thay đổi.
