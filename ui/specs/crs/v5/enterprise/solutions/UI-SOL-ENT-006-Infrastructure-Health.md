# UI Solution: UI-SOL-ENT-006 — Infrastructure Health Dashboard

**Solution ID:** UI-SOL-ENT-006  
**CR References:** [CR-ENT-006](../../../../docs/crs/v5/enterprise/CR-ENT-006-Infrastructure-Health.md)  
**Backend Solution:** [SOL-ENT-006](../../../../backend/specs/crs/v5/enterprise/solutions/SOL-ENT-006-Infrastructure-Health.md)  
**Feature:** Infrastructure Health — Aggregated /healthz, Service Status Grid  
**Priority:** 🟠 Medium  
**Frontend Component:** `ui/src/pages/infrastructure/`

---

## 1. Mục Đích

Xây dựng Infrastructure Health Dashboard:
- Aggregated health status: 1 view cho tất cả 35+ services
- Service status grid: engine + infrastructure per row
- Degraded alert: banner khi có service unhealthy
- Health timeline: service health over time
- Direct links to service-specific dashboards

---

## 2. Backend API Contract

```http
GET /v1/console/infra/topology   → InfraTopology (mode, node_count, services)
GET /v1/console/infra/services   → ServiceInfo[] (all services health)
GET /v1/console/infra/services/{name} → ServiceInfo (single service detail)
GET /v1/console/infra/databases  → DatabaseHealth[] (PG, Neo4j, Redis, Qdrant, NATS)
GET /v1/console/infra/resources  → ResourceMetrics[] (CPU/mem/disk per service)
GET /v1/console/infra/deployments→ DeploymentInfo[] (version + replicas)
```

### TypeScript Types

```typescript
interface ServiceInfo {
  name:     string;
  version:  string;
  status:   'Healthy' | 'Warning' | 'Critical';
  uptime:   number;     // seconds
  port?:    number;
  address?: string;
}

interface DatabaseHealth {
  name:       string;
  type:       'PostgreSQL' | 'Redis' | 'Neo4j' | 'Qdrant' | 'NATS';
  status:     'Healthy' | 'Warning' | 'Critical';
  latency_ms: number;
  host?:      string;
  version?:   string;
}
```

---

## 3. Components Architecture

### 3.1 Infrastructure Overview Page

```
InfrastructurePage
├── OverallStatusBanner
│   ├── AllHealthy  ← "✅ All 35 services healthy"
│   ├── Degraded    ← "⚠️ 2 services degraded — memobase-engine, minio"
│   └── Critical    ← "🔴 CRITICAL: graphiti-search is down"
├── TopologyCard
│   ├── Mode        ← "Monolith" | "Microservices"
│   ├── NodeCount   ← "1 node"
│   └── ServiceCount← "35 services"
├── ServiceStatusGrid (responsive grid)
│   └── ServiceCard (per service)
│       ├── ServiceName
│       ├── StatusIndicator  ← green/amber/red dot
│       ├── UptimeBadge      ← "99.9%" or "12h"
│       ├── LatencyBadge     ← "2ms"
│       └── DetailLink
├── DatabaseHealthSection
│   └── DatabaseCard (PostgreSQL, Redis, Neo4j, Qdrant, NATS)
│       ├── DBType badge
│       ├── StatusDot
│       ├── LatencyMs
│       └── Version
└── ResourceMetricsSection
    └── ResourceRow (per service)
        ├── ServiceName
        ├── CPUBar           ← "12% [█░░░░░░░░░]"
        ├── MemoryBar        ← "234 MB / 512 MB"
        └── DiskBar          ← "45% [████░░░░░░]"
```

### 3.2 Overall Status Aggregation

```typescript
function OverallStatusBanner({ services }: { services: ServiceInfo[] }) {
  const critical = services.filter(s => s.status === 'Critical');
  const warning  = services.filter(s => s.status === 'Warning');
  const healthy  = services.filter(s => s.status === 'Healthy');
  
  if (critical.length > 0) {
    return (
      <AlertBanner variant="critical">
        🔴 CRITICAL: {critical.map(s => s.name).join(', ')} {critical.length > 1 ? 'are' : 'is'} down
      </AlertBanner>
    );
  }
  
  if (warning.length > 0) {
    return (
      <AlertBanner variant="warning">
        ⚠️ {warning.length} service{warning.length > 1 ? 's' : ''} degraded:
        {warning.map(s => s.name).join(', ')}
      </AlertBanner>
    );
  }
  
  return (
    <AlertBanner variant="success">
      ✅ All {services.length} services healthy
    </AlertBanner>
  );
}
```

### 3.3 Service Status Grid

```typescript
// Group services by category
const SERVICE_CATEGORIES = {
  'Memory Engines': ['cognee-ingestion', 'cognee-search', 'graphiti-ingestion', 'graphiti-search',
                     'zep-memory', 'memobase-engine', 'memobase-context', 'ov-fs', 'ov-search',
                     'sm-memory', 'sm-connector'],
  'Platform':       ['vnp-gateway', 'vnp-platform', 'vnp-auth', 'vnp-event', 'vnp-admin'],
  'Observability':  ['vnp-observability', 'observe-service', 'vnp-pipelines'],
  'Infrastructure': ['vnp-dashboard', 'vnp-search-hub', 'vnp-infra'],
};

// Status dot: green=Healthy, amber=Warning, red=Critical
const STATUS_COLORS = {
  Healthy:  'bg-green-500',
  Warning:  'bg-amber-500 animate-pulse',
  Critical: 'bg-red-500 animate-pulse',
};
```

### 3.4 Database Health Cards

```
DatabaseHealthSection
├── PostgreSQL Card   ← PG icon, status, latency (< 10ms expected)
├── Redis Card        ← Redis icon, status, latency (< 1ms expected)
├── Neo4j Card        ← graph icon, status, latency (< 50ms expected)
├── Qdrant Card       ← vector icon, status, latency (< 10ms expected)
└── NATS Card         ← message icon, status, latency (< 5ms expected)
```

---

## 4. Auto-Refresh & Realtime

```typescript
// Polling + WebSocket for health changes
export function useInfraHealth() {
  const qc = useQueryClient();
  
  // Poll every 30s
  const servicesQuery = useQuery({
    queryKey: ['infra', 'services'],
    queryFn:  () => infraApi.getServices(),
    refetchInterval: 30_000,
  });
  
  // WebSocket: health_change event → instant update
  useEffect(() => {
    return wsManager.on('health_change', (data: { service: string; status: string }) => {
      qc.setQueryData(['infra', 'services'], (old: ServiceInfo[]) =>
        old?.map(s => s.name === data.service ? { ...s, status: data.status } : s)
      );
    });
  }, [qc]);
  
  return servicesQuery;
}
```

---

## 5. Deployment Info Table

```
DeploymentInfoPage
├── DeploymentTable
│   └── DeploymentRow
│       ├── ServiceName
│       ├── Version         ← "1.0.0"
│       ├── DeployedAt      ← "2 hours ago"
│       ├── StatusBadge     ← running | stopped | error
│       └── Replicas        ← "1/1" or "2/2"
└── TopologyDisplay         ← mode (monolith/microservices) + node count
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Overall status banner: correct aggregation (healthy/degraded/critical)
- [ ] Services grouped by category in grid
- [ ] Status dots pulse (animate) for Warning and Critical
- [ ] Database cards show latency with expected thresholds
- [ ] `health_change` WebSocket event → instant status update (no polling delay)
- [ ] Resource metrics: CPU/memory/disk bars with colors
- [ ] Deployment table: version + replica count
- [ ] Auto-refresh every 30s (in addition to WebSocket)
- [ ] Topology display: monolith vs microservices mode
