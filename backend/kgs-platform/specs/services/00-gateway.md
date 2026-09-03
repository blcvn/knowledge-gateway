# kgs-gateway — API Gateway Service

> **Role:** Điểm vào duy nhất cho toàn bộ KGS Platform. Xử lý authentication, rate limiting, routing và audit logging.

---

## 1. Trách Nhiệm (Single Responsibility)

`kgs-gateway` là **API Gateway** — không chứa bất kỳ business logic nào. Nhiệm vụ duy nhất là:

1. **Authenticate** request (API Key → App Context)
2. **Rate limit** theo app quota
3. **Route** request đến service phù hợp
4. **Audit log** mọi request
5. **Transform** gRPC response → HTTP/JSON cho external clients

---

## 2. Kiến Trúc Nội Tại

```
Client Request (HTTPS)
         │
         ▼
┌────────────────────────────────────────────────────────┐
│                    kgs-gateway                          │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │              Middleware Pipeline                  │  │
│  │                                                  │  │
│  │  1. TLS Termination                              │  │
│  │  2. Request ID injection (X-Request-ID)          │  │
│  │  3. API Key Extractor (Header: X-API-Key)        │  │
│  │  4. Auth Middleware → registry-service           │  │
│  │     └─ Returns: AppContext{app_id, scopes, quota}│  │
│  │  5. Rate Limiter (Redis token bucket per app_id) │  │
│  │  6. Scope Checker (required scope vs app scopes) │  │
│  │  7. Audit Logger (async write to PostgreSQL)     │  │
│  └──────────────────────────────────────────────────┘  │
│                          │                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │                   Router                          │  │
│  │                                                  │  │
│  │  /v1/apps/**          → registry-service         │  │
│  │  /v1/ontology/**      → ontology-service         │  │
│  │  /v1/graph/**         → graph-service            │  │
│  │  /v1/query/**         → query-intel-service      │  │
│  │  /v1/rules/**         → rule-engine-service      │  │
│  │  /v1/policies/**      → policy-service           │  │
│  │  /v1/search/**        → search-service           │  │
│  │  /v1/overlay/**       → overlay-service          │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────┘
```

---

## 3. Authentication Flow

### 3.1 API Key Validation

```
Request: GET /v1/graph/nodes/{id}
Header:  X-API-Key: kgs_abc123...

Gateway:
  1. Extract API Key từ header
  2. Gọi registry-service.ValidateAPIKey(key_hash)
  3. registry-service trả về:
     {
       app_id: "ba_agent",
       tenant_id: "tenant_001",
       scopes: ["graph:read", "graph:write"],
       quota: { requests_per_minute: 1000 }
     }
  4. Gateway inject vào gRPC metadata:
     x-app-id: ba_agent
     x-tenant-id: tenant_001
     x-scopes: graph:read,graph:write
     x-request-id: uuid-xxx
```

### 3.2 App Context Cache

Để tránh gọi registry-service cho mỗi request, Gateway cache App Context:
- **Cache Store:** Redis
- **Key:** `gateway:ctx:{key_hash}`
- **TTL:** 60 giây
- **Invalidation:** registry-service publish NATS event khi API Key bị revoke

---

## 4. Rate Limiting

### 4.1 Algorithm: Token Bucket (Redis-based)

```
Per app_id:
  Bucket size  = quota.requests_per_minute
  Refill rate  = quota.requests_per_minute / 60 tokens/second

Response headers:
  X-RateLimit-Limit:     1000
  X-RateLimit-Remaining: 847
  X-RateLimit-Reset:     1735000000
```

### 4.2 Rate Limit Tiers

| Quota Type | Default | Premium |
|-----------|---------|---------|
| requests_per_minute | 100 | 10,000 |
| requests_per_day | 10,000 | unlimited |
| max_nodes_per_query | 500 | 10,000 |

---

## 5. Routing Table

| HTTP Method | Path Pattern | Backend Service | Required Scope |
|-------------|-------------|----------------|----------------|
| POST | `/v1/apps` | registry-service | `admin` |
| GET | `/v1/apps` | registry-service | `admin` |
| GET | `/v1/apps/:app_id` | registry-service | `admin` |
| POST | `/v1/apps/:app_id/keys` | registry-service | `admin` |
| DELETE | `/v1/apps/:app_id/keys/:key_id` | registry-service | `admin` |
| POST | `/v1/ontology/entity-types` | ontology-service | `ontology:write` |
| GET | `/v1/ontology/entity-types` | ontology-service | `ontology:read` |
| GET | `/v1/ontology/entity-types/:name` | ontology-service | `ontology:read` |
| POST | `/v1/ontology/relation-types` | ontology-service | `ontology:write` |
| GET | `/v1/ontology/relation-types` | ontology-service | `ontology:read` |
| POST | `/v1/graph/nodes` | graph-service | `graph:write` |
| GET | `/v1/graph/nodes/:id` | graph-service | `graph:read` |
| PUT | `/v1/graph/nodes/:id` | graph-service | `graph:write` |
| DELETE | `/v1/graph/nodes/:id` | graph-service | `graph:write` |
| POST | `/v1/graph/edges` | graph-service | `graph:write` |
| GET | `/v1/graph/nodes/:id/context` | query-intel-service | `graph:read` |
| GET | `/v1/graph/nodes/:id/impact` | query-intel-service | `graph:read` |
| GET | `/v1/graph/nodes/:id/coverage` | query-intel-service | `graph:read` |
| POST | `/v1/graph/subgraph` | query-intel-service | `graph:read` |
| POST | `/v1/query` | query-intel-service | `graph:read` |
| GET | `/v1/analytics/coverage` | query-intel-service | `analytics:read` |
| GET | `/v1/analytics/traceability` | query-intel-service | `analytics:read` |
| POST | `/v1/rules` | rule-engine-service | `rules:write` |
| GET | `/v1/rules` | rule-engine-service | `rules:read` |
| GET | `/v1/rules/:id` | rule-engine-service | `rules:read` |
| POST | `/v1/policies` | policy-service | `policies:write` |
| GET | `/v1/policies` | policy-service | `policies:read` |
| POST | `/v1/search` | search-service | `graph:read` |
| POST | `/v1/overlay` | overlay-service | `graph:write` |
| POST | `/v1/overlay/:id/commit` | overlay-service | `graph:write` |

---

## 6. Audit Logging

Mọi request đều được audit log bất đồng bộ:

```json
{
  "request_id": "uuid-xxx",
  "app_id": "ba_agent",
  "tenant_id": "tenant_001",
  "method": "POST",
  "path": "/v1/graph/nodes",
  "status_code": 201,
  "latency_ms": 45,
  "timestamp": "2026-06-11T00:00:00Z",
  "user_agent": "ba-agent-service/1.0",
  "ip": "10.0.0.1"
}
```

Audit logs được ghi vào PostgreSQL table `audit_logs` qua async queue (không block request).

---

## 7. Error Responses

```json
// 401 Unauthorized
{ "error": "INVALID_API_KEY", "message": "API key is invalid or expired" }

// 403 Forbidden
{ "error": "INSUFFICIENT_SCOPE", "message": "Required scope: graph:write" }

// 429 Too Many Requests
{
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "Rate limit exceeded",
  "retry_after": 15
}

// 503 Service Unavailable
{ "error": "BACKEND_UNAVAILABLE", "message": "Service temporarily unavailable" }
```

---

## 8. Configuration

```yaml
# configs/gateway.yaml
gateway:
  port: 8080
  tls:
    enabled: true
    cert_file: /certs/tls.crt
    key_file: /certs/tls.key

  auth:
    cache_ttl: 60s

  rate_limit:
    enabled: true
    redis_addr: redis:6379
    default_rpm: 100

  backends:
    registry_service: registry-service:9001
    ontology_service: ontology-service:9002
    graph_service: graph-service:9003
    query_intel_service: query-intel-service:9004
    rule_engine_service: rule-engine-service:9005
    policy_service: policy-service:9006
    search_service: search-service:9007
    overlay_service: overlay-service:9008

  audit:
    enabled: true
    async: true
    buffer_size: 1000

  observability:
    metrics_port: 9090
    tracing_endpoint: http://jaeger:14268/api/traces
```

---

## 9. Health & Observability

| Endpoint | Mô tả |
|---------|-------|
| `GET /healthz` | Liveness probe |
| `GET /readyz` | Readiness probe (kiểm tra kết nối backends) |
| `GET /metrics` | Prometheus metrics |

### Key Metrics

- `gateway_requests_total{app_id, method, path, status}` — Tổng requests
- `gateway_request_duration_seconds{path}` — Latency histogram
- `gateway_rate_limit_blocked_total{app_id}` — Số requests bị throttle
- `gateway_auth_cache_hits_total` — Cache hit rate

---

## 10. Technology Stack

| Component | Technology |
|-----------|-----------|
| Framework | Go + Kratos HTTP |
| gRPC client | google.golang.org/grpc |
| Rate limit | Redis (go-redis + token bucket) |
| Auth cache | Redis |
| Audit store | PostgreSQL (async write) |
| Metrics | Prometheus |
| Tracing | OpenTelemetry + Jaeger |
