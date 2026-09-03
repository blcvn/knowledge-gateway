---
id: DOC-S04
service: vnp-gateway
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — Data Model

> **Database**: PostgreSQL (config/keys), Redis (rate-limit/cache)

## 1. Tables

The gateway is primarily stateless. Minimal persistence is used for API key resolution and tenant configuration.

### 1.1 `api_keys` (PostgreSQL)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK | Key ID |
| `key_hash` | TEXT | UNIQUE, NOT NULL | SHA-256 hash of API key |
| `key_prefix` | VARCHAR(12) | NOT NULL | First 8 chars for display (`vnp_abc1...`) |
| `tenant_id` | UUID | FK → tenants, NOT NULL | Owning tenant |
| `name` | VARCHAR(255) | NOT NULL | Human-readable name |
| `scopes` | TEXT[] | NOT NULL | Permitted API scopes |
| `rate_tier` | VARCHAR(20) | DEFAULT 'free' | Rate limit tier |
| `expires_at` | TIMESTAMPTZ | NULL | Expiration (NULL = never) |
| `last_used_at` | TIMESTAMPTZ | NULL | Last usage timestamp |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Creation timestamp |
| `revoked_at` | TIMESTAMPTZ | NULL | Revocation timestamp |

### 1.2 `tenants` (PostgreSQL)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK | Tenant ID |
| `name` | VARCHAR(255) | NOT NULL | Organization name |
| `plan` | VARCHAR(20) | DEFAULT 'free' | Subscription plan |
| `rate_tier` | VARCHAR(20) | DEFAULT 'free' | Rate limit tier |
| `settings` | JSONB | DEFAULT '{}' | Tenant-specific settings |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | |
| `suspended_at` | TIMESTAMPTZ | NULL | Suspension timestamp |

### 1.3 `route_configs` (PostgreSQL)

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | SERIAL | PK | |
| `path_pattern` | VARCHAR(255) | UNIQUE | Route pattern (e.g., `/v1/cognee/*`) |
| `target_service` | VARCHAR(100) | NOT NULL | Target gRPC service name |
| `timeout_ms` | INT | DEFAULT 30000 | Request timeout |
| `rate_limit` | INT | NULL | Override rate limit |
| `circuit_threshold` | INT | DEFAULT 5 | Failure threshold |
| `enabled` | BOOLEAN | DEFAULT true | Route enabled flag |

## 2. Entity-Relationship Diagram

```mermaid
erDiagram
    tenants ||--o{ api_keys : "has many"
    tenants {
        uuid id PK
        varchar name
        varchar plan
        varchar rate_tier
        jsonb settings
        timestamptz created_at
    }
    api_keys {
        uuid id PK
        text key_hash UK
        uuid tenant_id FK
        varchar name
        text_arr scopes
        timestamptz expires_at
        timestamptz revoked_at
    }
    route_configs {
        serial id PK
        varchar path_pattern UK
        varchar target_service
        int timeout_ms
        boolean enabled
    }
```

### 1.4 `audit_logs` (PostgreSQL) — FEAT-011

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK | Log entry ID |
| `tenant_id` | UUID | FK → tenants, NOT NULL | Tenant scope |
| `actor` | VARCHAR(255) | NOT NULL | User/API key who performed action |
| `action` | VARCHAR(100) | NOT NULL | Action type (create, update, delete, forget) |
| `entity_type` | VARCHAR(100) | NOT NULL | Entity type (tenant, policy, memory, user) |
| `entity_id` | VARCHAR(255) | NULL | Affected entity ID |
| `engine` | VARCHAR(50) | NULL | Engine where action occurred |
| `details` | JSONB | DEFAULT '{}' | Action-specific metadata |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | Timestamp |

### 1.5 `policies` (PostgreSQL) — FEAT-011

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK | Policy ID |
| `tenant_id` | UUID | FK → tenants, NOT NULL | Tenant scope |
| `name` | VARCHAR(255) | NOT NULL | Policy name |
| `type` | VARCHAR(50) | NOT NULL | Policy type (access, retention, privacy) |
| `rego` | TEXT | NOT NULL | OPA Rego policy body |
| `enabled` | BOOLEAN | DEFAULT true | Active flag |
| `created_at` | TIMESTAMPTZ | DEFAULT now() | |
| `updated_at` | TIMESTAMPTZ | DEFAULT now() | |

## 2. Entity-Relationship Diagram

```mermaid
erDiagram
    tenants ||--o{ api_keys : "has many"
    tenants ||--o{ audit_logs : "has many"
    tenants ||--o{ policies : "has many"
    tenants {
        uuid id PK
        varchar name
        varchar plan
        varchar rate_tier
        jsonb settings
        timestamptz created_at
    }
    api_keys {
        uuid id PK
        text key_hash UK
        uuid tenant_id FK
        varchar name
        text_arr scopes
        timestamptz expires_at
        timestamptz revoked_at
    }
    route_configs {
        serial id PK
        varchar path_pattern UK
        varchar target_service
        int timeout_ms
        boolean enabled
    }
    audit_logs {
        uuid id PK
        uuid tenant_id FK
        varchar actor
        varchar action
        varchar entity_type
        jsonb details
        timestamptz created_at
    }
    policies {
        uuid id PK
        uuid tenant_id FK
        varchar name
        varchar type
        text rego
        boolean enabled
    }
```

## 3. Redis Data Structures

| Key Pattern | Type | TTL | Purpose |
|-------------|------|-----|---------|
| `rl:{tenant_id}:{endpoint}` | Sorted Set | 60s | Sliding window rate limit |
| `apikey:{prefix}` | Hash | 5min | API key cache (avoid DB lookup) |
| `health:{service}` | String | 30s | Service health cache |
| `cb:{service}` | String | — | Circuit breaker state |
| `dashboard:health` | Hash | 10s | Dashboard aggregated health cache |
| `dashboard:metrics` | Hash | 30s | Dashboard KPI metrics cache |
| `dashboard:throughput:{window}` | Hash | 30s | Per-engine throughput cache |
| `ws:sessions` | Set | — | Active WebSocket session IDs |
| `ws:channels:{session_id}` | Set | — | Subscribed channels per WS session |

## 4. Index Strategy

```sql
-- Original indexes
CREATE INDEX idx_api_keys_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_tenant ON api_keys (tenant_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_tenants_plan ON tenants (plan) WHERE suspended_at IS NULL;
CREATE INDEX idx_route_configs_pattern ON route_configs (path_pattern) WHERE enabled = true;

-- Governance indexes (v2.0.0)
CREATE INDEX idx_audit_logs_tenant_time ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor, created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs (action) WHERE entity_type IS NOT NULL;
CREATE INDEX idx_policies_tenant ON policies (tenant_id) WHERE enabled = true;
```

## 5. Migration History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2026-05-09 | Initial schema: tenants, api_keys, route_configs |
| 2.0.0 | 2026-05-13 | Add audit_logs, policies tables (SOL-002 Governance) |

