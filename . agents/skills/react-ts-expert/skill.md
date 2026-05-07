---
skill_id: SKILL-006
version: 1.0.0
status: active
priority: existing
group: Frontend Development
created_at: 2026-04-24
---

# SKILL-006 · Frontend Development — React + TypeScript

## Mô tả

Phát triển giao diện React + TypeScript với kiến trúc tổ chức code chuẩn, performance tối ưu, trải nghiệm người dùng tốt nhất và an toàn bảo mật.

## Agents sử dụng

- `ui-renderer-agent`
- `ui-schema-generator-agent`

---

## Năng lực cốt lõi

### 1. Feature-Sliced Design (FSD)

```
src/
├── app/           → App providers, global styles, router
├── pages/         → Page components (route-level)
├── widgets/       → Complex UI blocks (sidebar, header)
├── features/      → User interactions (auth, document-upload)
├── entities/      → Business entities (Document, Project, Job)
├── shared/        → Reusable UI kit, utils, API clients
│   ├── ui/        → Button, Input, Modal, etc.
│   ├── api/       → Base API client, hooks
│   └── lib/       → Pure utilities
└── index.tsx

Rules:
- Lower layer CANNOT import from upper layer
- Cross-slice imports only through public API (index.ts)
```

### 2. Data Fetching (React Query / SWR)

```tsx
// React Query pattern cho data fetching
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

// Query keys as constants
export const documentKeys = {
  all: ['documents'] as const,
  list: (projectId: string) => [...documentKeys.all, 'list', projectId] as const,
  detail: (id: string) => [...documentKeys.all, 'detail', id] as const,
}

// Typed query hook
export function useDocument(id: string) {
  return useQuery({
    queryKey: documentKeys.detail(id),
    queryFn: () => documentApi.get(id),
    staleTime: 5 * 60 * 1000,  // 5 minutes
    enabled: Boolean(id),
  })
}

// Optimistic mutation
export function useUpdateDocument() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: documentApi.update,
    onMutate: async (variables) => {
      await queryClient.cancelQueries({ queryKey: documentKeys.detail(variables.id) })
      const previous = queryClient.getQueryData(documentKeys.detail(variables.id))
      queryClient.setQueryData(documentKeys.detail(variables.id), variables)
      return { previous }
    },
    onError: (_, variables, context) => {
      queryClient.setQueryData(documentKeys.detail(variables.id), context?.previous)
    },
    onSettled: (_, __, variables) => {
      queryClient.invalidateQueries({ queryKey: documentKeys.detail(variables.id) })
    },
  })
}
```

### 3. State Management (Zustand)

```tsx
// Zustand store — simple, no boilerplate
interface PipelineStore {
  activeJobId: string | null
  progress: Record<string, number>
  setActiveJob: (id: string) => void
  updateProgress: (id: string, percent: number) => void
}

export const usePipelineStore = create<PipelineStore>((set) => ({
  activeJobId: null,
  progress: {},
  setActiveJob: (id) => set({ activeJobId: id }),
  updateProgress: (id, percent) => set((state) => ({
    progress: { ...state.progress, [id]: percent }
  })),
}))
```

### 4. TypeScript Strict Mode

```tsx
// tsconfig.json — strict settings
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "noImplicitReturns": true
  }
}

// Type-safe API client
interface ApiResponse<T> {
  data: T
  meta?: { total: number; page: number }
}

async function apiFetch<T>(url: string): Promise<T> {
  const res = await fetch(url)
  if (!res.ok) throw new ApiError(res.status, await res.json())
  const json: ApiResponse<T> = await res.json()
  return json.data
}
```

### 5. Performance (Code Splitting + Core Web Vitals)

```tsx
// Lazy loading cho route-level components
const DocumentPage = lazy(() => import('./pages/DocumentPage'))
const GraphPage = lazy(() => import('./pages/GraphPage'))

// React.memo cho expensive components
const DocumentCard = memo(({ doc }: { doc: Document }) => (
  <Card title={doc.title} status={doc.status} />
), (prev, next) => prev.doc.id === next.doc.id && prev.doc.status === next.doc.status)

// Virtual list cho large datasets
import { useVirtual } from '@tanstack/react-virtual'

// Image optimization
<img
  src={url}
  loading="lazy"
  decoding="async"
  width={800}
  height={600}
  alt="Document preview"
/>

// Core Web Vitals targets:
// LCP < 2.5s, FID < 100ms, CLS < 0.1
```

### 6. Security (XSS / CSRF Prevention)

```tsx
// ❌ KHÔNG dùng dangerouslySetInnerHTML với user content
<div dangerouslySetInnerHTML={{ __html: userContent }} />

// ✅ Sanitize trước khi render (DOMPurify)
import DOMPurify from 'dompurify'
const clean = DOMPurify.sanitize(userContent, { USE_PROFILES: { html: true } })
<div dangerouslySetInnerHTML={{ __html: clean }} />

// CSRF: dùng SameSite cookies + CSRF token header
// Axios auto-attach CSRF token
axios.defaults.headers.common['X-CSRF-Token'] = getCSRFToken()
```

---

## Checklist

- [ ] TypeScript strict mode enabled
- [ ] Feature-Sliced Design structure
- [ ] React Query cho tất cả server state
- [ ] Zustand cho UI-only state
- [ ] Route-level code splitting với Suspense
- [ ] No dangerouslySetInnerHTML với unsanitized content
- [ ] Core Web Vitals measured (Lighthouse CI)
- [ ] Error boundary tại route level
