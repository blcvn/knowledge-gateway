export const API_CONFIG = {
  baseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  useMockData: import.meta.env.VITE_USE_MOCK_DATA === 'true',

  /** Direct engine proxies (used by low-level engine-specific features only) */
  engines: {
    cognee:      { baseUrl: '/v1/cognee',    port: 8080 },
    graphiti:    { baseUrl: '/v1/graphiti',   port: 8080 },
    zep:         { baseUrl: '/v1/zep',        port: 8080 },
    openviking:  { baseUrl: '/v1/ov',         port: 8080 },
    memobase:    { baseUrl: '/v1/memobase',   port: 8080 },
    supermemory: { baseUrl: '/v1/sm',         port: 8080 },
  },

  /** Consolidated Console API namespace — all UI modules should use these */
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
  },

  /** Legacy admin namespace */
  gateway: {
    admin:   '/v1/admin',
    memory:  '/v1/memory',
    graph:   '/v1/graph',
  },
} as const;
