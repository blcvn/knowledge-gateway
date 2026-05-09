---
id: DOC-S04
service: vnp-gateway
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — Data Model

> **Database**: Redis (cache/rate-limit) — no primary SQL database

## Redis Keys

### Rate Limiting

| Key Pattern | Type | TTL | Description |
|-------------|------|-----|-------------|
| `rl:{tenant_id}:{endpoint}:{window}` | SORTED SET | Window size | Sliding window rate limiter |
| `rl:quota:{tenant_id}` | HASH | 1h | Tenant quota counters |

### Circuit Breaker

| Key Pattern | Type | TTL | Description |
|-------------|------|-----|-------------|
| `cb:{service_name}:state` | STRING | — | Circuit state: closed / open / half-open |
| `cb:{service_name}:failures` | INT | Reset period | Failure count |
| `cb:{service_name}:last_failure` | STRING | — | Last failure timestamp |

### Session Cache

| Key Pattern | Type | TTL | Description |
|-------------|------|-----|-------------|
| `session:{session_id}` | HASH | 30m | Cached session metadata |
| `auth:{api_key_prefix}` | HASH | 5m | Cached API key validation result |

### MCP Server State

| Key Pattern | Type | TTL | Description |
|-------------|------|-----|-------------|
| `mcp:session:{session_id}` | HASH | 30m | MCP session state |
| `mcp:tools:{tenant_id}` | SET | 10m | Available tool manifest per tenant |

## Internal State (In-Memory)

| Component | Structure | Description |
|-----------|----------|-------------|
| Route Table | `map[string]ServiceRoute` | Path → gRPC target mapping |
| Service Registry | `[]ServiceEndpoint` | All downstream service endpoints |
| Health Cache | `map[string]HealthStatus` | Last-known health per service (5s TTL) |

## No SQL Tables

The gateway is stateless by design — all persistent data is managed by downstream services. Redis is used only for ephemeral state (rate limits, circuit breakers, session cache).
