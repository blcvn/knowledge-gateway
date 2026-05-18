# Changelog

Tất cả những thay đổi nổi bật cho dự án Frontend UI sẽ được tài liệu hóa ở đây.
Định dạng dựa trên [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [1.0.0] - 2026-05-13

### Added — MVP Modules (11/11)
- **Dashboard Overview** (T01): KPI cards, Memory Flow chart, Engine Health Grid.
- **Memory Explorer** (T02): Tìm kiếm, filter, confidence score display.
- **Agent Context Debugger** (T03): Debug RAG pipeline, test prompt, view context.
- **Governance Center** (T04): GDPR Forget, OPA Policy Editor, Audit Explorer, TTL.
- **Graph Studio** (T05): Knowledge graph visualization, timeline slider.
- **Observability & Error** (T06): System metrics, error tracking, distributed tracing.
- **Sessions Explorer** (T07): Session replay, conversation context viewer.
- **Pipelines Monitor** (T08): Pipeline stages, job status monitoring.
- **Infrastructure Health** (T09): Database, queue, compute node health.
- **API & SDK Manager** (T10): API keys, rate limits, webhook management.
- **Organization Settings** (T11): Org info, members, RBAC, billing.

### Added — Enterprise Infrastructure
- **Authentication & RBAC** (ENT-04): AuthProvider, RouteGuard, RequireRole, idle timeout.
- **API Client** (ENT-05): Fetch wrapper with tenant injection, AppError, interceptors.
- **React Query Config** (ENT-05): Smart caching, retry, optimistic update helpers.
- **Zustand Store** (ENT-05): Modular global state (theme, sidebar, tenant).
- **Error Boundary** (ENT-02): Global + module-level error catching.
- **Fallback Pages** (ENT-02): 404, 500, chunk-error, generic error pages.
- **Production Logger** (ENT-02): Environment-controlled, production-safe logging.
- **Lazy Routes** (ENT-03): Route-based code splitting for all 11 modules.
- **CI/CD Pipeline** (ENT-06): GitHub Actions (lint → test → build → E2E).
- **Vitest + Playwright** (ENT-06): Unit test + E2E test framework setup.

### Added — Persona Navigation Flows
- P1: AI Agent Developer (Flows 1-8)
- P2: Platform / DevOps Engineer (Flows 1-3, 9-12)
- P3: ML/AI Engineer (Flows 1-3, 13-15)
- P4: Enterprise Architect (Flows 1-4, 16-18)
- P5: IDE Plugin User
- P6: Framework Integrator
- P7: AI Power User (Flows 19-20)
- P8: Product Manager

### Added — Documentation
- Architecture v2.0: System architecture diagram, security model, scalability strategy.
- SOL-001 v2.0: Enterprise-grade checklist, Phase 2 roadmap.
- 43 task specifications (TASK-001 to TASK-043) — all completed.

## [Unreleased]
### Planned (Phase 2)
- WebSocket/SSE real-time integration.
- Virtual list for large datasets (@tanstack/react-virtual).
- URL search params sync for deep-linkable filters.
- Husky + lint-staged pre-commit hooks.
- Coverage threshold enforcement (>70%).
- Service Worker for offline-first cache.
