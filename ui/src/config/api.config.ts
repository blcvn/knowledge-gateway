/**
 * API Configuration — VNP Memory Console
 *
 * TASK-UI-001: Thêm flat namespace `auth`, `org`, `sdk` và các console namespaces.
 * Dùng `API_CONFIG.<namespace>` trong tất cả service files để đảm bảo consistent paths.
 */

export const API_CONFIG = {
  baseUrl:    import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',

  // ── Auth ──────────────────────────────────────────────────────────
  auth:          '/v1/auth',

  // ── Console namespaces ────────────────────────────────────────────
  dashboard:     '/v1/console/dashboard',
  sessions:      '/v1/console/sessions',
  memory:        '/v1/console/memory',
  profiles:      '/v1/console/profiles',
  adaptive:      '/v1/console/adaptive',
  governance:    '/v1/console/governance',
  observability: '/v1/console/observability',
  pipelines:     '/v1/console/pipelines',
  infra:         '/v1/console/infra',
  org:           '/v1/console/org',
  sdk:           '/v1/console/sdk',
  graph:         '/v1/console/graph',
  debugger:      '/v1/console/debugger',
  ws:            '/v1/console/ws',

  // ── Engine direct APIs (used by engine-specific low-level features) ─
  graphiti:    '/v1/graphiti',
  memobase:    '/v1/memobase',
  zep:         '/v1/zep',
  cognee:      '/v1/cognee',
  supermemory: '/v1/sm',
  openviking:  '/v1/ov',
  admin:       '/v1/admin',

  // ── Alias for backward compatibility with nested API_CONFIG.console.* ─
  /** @deprecated Use top-level namespace e.g. API_CONFIG.dashboard instead */
  console: {
    dashboard:     '/v1/console/dashboard',
    memory:        '/v1/console/memory',
    graph:         '/v1/console/graph',
    profiles:      '/v1/console/profiles',
    adaptive:      '/v1/console/adaptive',
    debugger:      '/v1/console/debugger',
    sessions:      '/v1/console/sessions',
    governance:    '/v1/console/governance',
    pipelines:     '/v1/console/pipelines',
    infra:         '/v1/console/infra',
    observability: '/v1/console/observability',
    ws:            '/v1/console/ws',
    org:           '/v1/console/org',
    sdk:           '/v1/console/sdk',
  },

  /** @deprecated Use API_CONFIG.<engine> instead */
  engines: {
    cognee:      { baseUrl: '/v1/cognee',   port: 8080 },
    graphiti:    { baseUrl: '/v1/graphiti', port: 8080 },
    zep:         { baseUrl: '/v1/zep',      port: 8080 },
    openviking:  { baseUrl: '/v1/ov',       port: 8080 },
    memobase:    { baseUrl: '/v1/memobase', port: 8080 },
    supermemory: { baseUrl: '/v1/sm',       port: 8080 },
  },

  /** @deprecated */
  gateway: {
    admin:  '/v1/admin',
    memory: '/v1/memory',
    graph:  '/v1/graph',
  },
} as const;

export type APINamespace = keyof typeof API_CONFIG;
