# CR-004: Response Schema Contracts

**CR ID**: CR-004-response-schema-contracts  
**Status**: Open  
**Priority**: 🟠 Medium  
**Target Components**: All backend services (schema validation)  
**Frontend Source**: `ui/src/types/*.ts`  
**Created**: 2026-06-18

---

## Problem

The frontend uses strongly-typed TypeScript interfaces for every API response. The backend services were built independently and may return responses that do not exactly match these type contracts. Any field name mismatch, missing field, or wrong type will cause the frontend to silently fail or display incorrect data.

This CR documents the **exact response schemas** the backend must conform to for each domain, based on the TypeScript source (`ui/src/types/*.ts`).

---

## 1. Dashboard API Contracts (`vnp-dashboard`)

### `GET /v1/console/dashboard/metrics` → `KPIData`

```json
{
  "activeAgents":        0,
  "recallLatencyP50Ms":  0,
  "recallLatencyP95Ms":  0,
  "contextSavingsPct":   0.0,
  "graphNodesTotal":     0,
  "graphEdgesTotal":     0,
  "graphGrowth24h":      0.0,
  "errorRatePct":        0.0,
  "activeSessions":      0,
  "activeProfiles":      0,
  "memoryVersions":      0
}
```

### `GET /v1/console/dashboard/health` → `EngineHealth[]`

```json
[
  {
    "name":          "cognee",
    "role":          "string",
    "status":        "Healthy | Warning | Critical",
    "latencyP50Ms":  0,
    "latencyP95Ms":  0,
    "queueDepth":    0,
    "uptimeSeconds": 0,
    "lastCheck":     "ISO 8601"
  }
]
```

> **Note**: `status` must be one of `"Healthy"`, `"Warning"`, `"Critical"` — capital first letter, not lowercase.

### `GET /v1/console/dashboard/throughput` → `ThroughputData`

```json
{
  "window":  "1h",
  "engines": {
    "cognee": {
      "ingestPerSec":              0.0,
      "recallPerSec":              0.0,
      "embedPerSec":               0.0,
      "profileExtractionsPerSec":  0.0,
      "queueBacklog":              0
    }
  }
}
```

---

## 2. Memory Explorer Contracts (`vnp-search-hub`, `sm-memory`)

### `POST /v1/console/memory/search` → `MemorySearchResult`

```json
{
  "results": [
    {
      "id":               "graphiti:ep_abc",
      "engine":           "graphiti",
      "memoryType":       "episodic",
      "title":            "string",
      "summary":          "string",
      "content":          "string",
      "score":            0.95,
      "entities":         ["string"],
      "sourceSessions":   ["string"],
      "temporalValidity": { "from": "ISO 8601 | null", "to": "ISO 8601 | null" },
      "policyTags":       ["string"],
      "versionChain":     "string | null",
      "metadata":         {}
    }
  ],
  "total":     100,
  "facets": {
    "byEngine": { "cognee": 5, "graphiti": 10 },
    "byType":   { "episodic": 10 }
  },
  "latencyMs": 120
}
```

> **Critical**: Memory `id` uses the format `"engine:local_id"` (e.g., `"graphiti:ep_abc123"`). The frontend URL-encodes this for path params using `encodeURIComponent()`.

### `GET /v1/console/memory/{id}/neighbors`

Must accept URL-encoded IDs (e.g., `graphiti%3Aep_abc`) and return `MemorySearchResult`.

Must support query params:
- `strategy` = `semantic | graph | temporal` (default: `semantic`)
- `limit` = integer (default: `10`)

---

## 3. Graph Studio Contracts (`graphiti-store`, `cognee-search`)

### `GET /v1/console/graph/ontology` → `OntologySchema`

```json
{
  "classes":       ["Entity", "Event"],
  "relationships": ["MENTIONS", "HAPPENED_AT"],
  "properties":    { "Entity": ["name", "type"] }
}
```

### `POST /v1/console/graph/subgraph` → `SubgraphData`

```json
{
  "nodes": [
    { "id": "n1", "label": "string", "type": "string", "properties": {} }
  ],
  "edges": [
    { "id": "e1", "source": "n1", "target": "n2", "type": "string", "properties": {} }
  ]
}
```

---

## 4. User Profiles Contracts (`memobase-context`, `memobase-ingestion`, `vnp-event`)

### `GET /v1/console/profiles` → `UserProfile[]`

```json
[
  {
    "user_id":    "string",
    "profiles": [
      { "topic": "string", "sub_topic": "string", "content": "string", "confidence": 0.9 }
    ],
    "created_at": "ISO 8601",
    "updated_at": "ISO 8601"
  }
]
```

### `GET /v1/console/profiles/{user_id}/buffers` → `BufferZone`

```json
{
  "user_id":         "string",
  "buffer_type":     "string",
  "token_count":     0,
  "token_threshold": 1000,
  "idle_timeout":    "30m",
  "last_flush":      "ISO 8601",
  "flush_count":     0
}
```

### `GET /v1/console/profiles/{user_id}/context` → `ContextAssembly`

```json
{
  "user_id":                "string",
  "context_string":         "string",
  "token_count":            0,
  "profile_section_tokens": 0,
  "event_section_tokens":   0,
  "latency_ms":             0
}
```

### `GET /v1/console/profiles/{user_id}/events` → `UserEvent[]`

```json
[
  {
    "id":         "string",
    "user_id":    "string",
    "gist":       "string",
    "tags":       ["string"],
    "created_at": "ISO 8601",
    "embedding":  [0.1, 0.2]
  }
]
```

---

## 5. Governance Contracts (`vnp-admin`, `vnp-event`)

### `GET /v1/console/governance/policies` → `Policy[]`

```json
[
  {
    "id":          "string",
    "name":        "string",
    "description": "string",
    "rego_code":   "package mem.policy\n...",
    "scope":       "string",
    "enabled":     true,
    "tenant_id":   "string",
    "created_at":  "ISO 8601"
  }
]
```

### `POST /v1/console/governance/gdpr/forget/preview` → `GDPRPreviewResponse`

```json
{
  "user_id":              "string",
  "estimated_items":      42,
  "breakdown_by_engine":  { "graphiti": 10, "cognee": 5 },
  "warnings":             ["string"]
}
```

---

## 6. Observability Contracts (`vnp-observability`)

### `GET /v1/console/observability/metrics` → `MetricsResponse`

```json
{
  "latency":    [{ "timestamp": "ISO 8601", "value": 120, "label": "p95" }],
  "error_rate": [{ "timestamp": "ISO 8601", "value": 0.02 }],
  "throughput": [{ "timestamp": "ISO 8601", "value": 1500 }]
}
```

### `GET /v1/console/observability/traces` → `TraceSpan[]`

Supports query filters: `service`, `status`, `operation`, `from`, `to`

```json
[
  {
    "trace_id":   "string",
    "span_id":    "string",
    "name":       "string",
    "operation":  "string",
    "service":    "string",
    "duration_ms": 120,
    "status":     "ok | slow | error",
    "timestamp":  "ISO 8601"
  }
]
```

### `GET /v1/console/observability/errors` → `ErrorEntry[]`

Supports `?service=xxx` filter.

```json
[
  {
    "id":             "string",
    "message":        "string",
    "service":        "string",
    "count":          5,
    "timestamp":      "ISO 8601",
    "lastOccurrence": "ISO 8601",
    "stack":          "string"
  }
]
```

### `GET /v1/console/observability/costs` → `CostEntry[]`

```json
[
  {
    "model":         "gpt-4o",
    "engine":        "cognee",
    "tokens_input":  1000,
    "tokens_output": 500,
    "cost_usd":      0.02,
    "date":          "ISO 8601"
  }
]
```

---

## 7. Infrastructure Contracts (`vnp-infra`)

### `GET /v1/console/infra/topology` → `InfraTopology`

```json
{
  "mode":        "monolith | microservices",
  "node_count":  18,
  "services":    ["cognee-ingestion", "graphiti-store"],
  "deployed_at": "ISO 8601"
}
```

### `GET /v1/console/infra/databases` → `DatabaseHealth[]`

```json
[
  {
    "name":       "postgres",
    "type":       "PostgreSQL | Redis | Neo4j | Qdrant | NATS",
    "status":     "Healthy | Warning | Critical",
    "latency_ms": 2,
    "host":       "localhost",
    "version":    "15.3"
  }
]
```

---

## 8. Error Response Contract (All Services)

The frontend parses errors using:

```typescript
interface ApiErrorResponse {
  message: string;
  code:    string;
  status:  number;
  details?: Record<string, unknown>;
}
```

**All services must return errors in this shape.** The current backend format uses:

```json
{ "error": { "code": "...", "message": "...", "details": [], "request_id": "..." } }
```

> ⚠️ **Mismatch**: Backend wraps in `"error": {}`, frontend expects flat `{ message, code, status }`. This needs alignment — either the gateway transforms the error wrapper before responding, or the frontend's error parser is updated. **Recommend: gateway-level error normalisation.**

---

## Acceptance Criteria

- [ ] All `HealthStatus` values returned as `"Healthy"`, `"Warning"`, `"Critical"` (not lowercase)
- [ ] Memory IDs use format `"engine:local_id"` (colon-separated)
- [ ] `MemorySearchResult` includes `facets.byEngine` and `facets.byType`
- [ ] `PaginatedResponse` includes both snake_case and camelCase alias fields
- [ ] `MetricsResponse` returns arrays for `latency`, `error_rate`, `throughput`
- [ ] Error responses are normalised to `{ message, code, status }` shape (or frontend parser updated)
- [ ] `OntologySchema` returns `classes`, `relationships`, and `properties` object
