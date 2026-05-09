---
id: TDD-zep-admin
title: Technical Design — zep-admin
service: zep-admin
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Zep
---

# Technical Design — zep-admin

> **Group**: Zep | **gRPC Port**: 9066 | **Health Port**: 12066

## 1. Service Overview

Cross-cutting admin service. Health aggregation, project/tenant management, API key lifecycle, schema migration. Orchestrates health checks across all 5 Zep domain services.

## 2. Domain Model

- **Project**: UUID, Name, Description, OrganizationID, Settings{MaxRequestSize, RequestTimeout, RateLimitRPS, TelemetryEnabled, GraphitiEnabled}
- **APIKey**: UUID, KeyHash (SHA-256), KeyPrefix (8 chars), ProjectUUID, Name, Scopes[], ExpiresAt, LastUsedAt, RevokedAt
- **AggregatedHealth**: Status (SERVING|NOT_SERVING|DEGRADED), Services map, Timestamp
- **HealthStatus**: SERVING | NOT_SERVING | DEGRADED

## 3. gRPC API (11 RPCs)

- **Health**: AggregatedHealth (parallel check)
- **Projects**: Create, Get, List, Update, Delete
- **API Keys**: Create (raw key, shown once), Validate, List, Revoke
- **Schema**: MigrateSchema

## 4. NATS Events

| Subject | Subscribers |
|---------|-------------|
| `zep.admin.project.created` | All Zep services |
| `zep.admin.project.deleted` | All Zep services (cascade) |

## 5. Storage

PostgreSQL: projects table (settings JSONB) + api_keys table (SHA-256 hash, scopes array).

## 6. Security

- API keys: SHA-256 hashed, raw key shown once at creation
- Scopes: read, write, admin
- Revocation: immediate via revoked_at timestamp

---

> **Next Steps**: Decompose into FEAT specs in `specs/features/`.
