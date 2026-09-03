# vnp-platform — Unified Admin, Event & Auth Service

> **Service**: `vnp-platform` | **gRPC Port**: 9050 | **Health**: 9103  
> **Origin**: Consolidated from vnp-admin + vnp-event + ov-admin + zep-admin + sm-auth + sm-analytics + sm-project  
> **Status**: Proposed | **Version**: 0.1.0

---

## Purpose

Unified platform service handling all cross-engine administrative concerns: tenant management, user CRUD, API key lifecycle, health aggregation, event timeline, authentication/authorization, usage analytics, and project/space management.

## Business Capability

- **Tenant Management**: Create, update, delete tenants across all 6 engines
- **User Management**: Unified user CRUD with engine-specific metadata
- **API Key Lifecycle**: Create, revoke, rotate API keys (all engines)
- **Health Aggregation**: Cascade health checks across all services
- **Event Timeline**: Cross-domain temporal event storage and search
- **Auth/RBAC**: JWT validation, API key authentication, role-based access control
- **Usage Analytics**: Token economics, API usage tracking per org
- **Project/Spaces**: Document organization, container tags, membership

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.23+ |
| RPC | gRPC (7 service definitions on single port) |
| Database | PostgreSQL 17 + pgvector |
| Cache | Redis 7+ |
| Async | NATS JetStream |
| Observability | OTel + Prometheus + slog |

## Quick Start

```bash
cd services/vnp-platform
go run cmd/server/main.go
# gRPC: :9050 | Health: :9103
```

## Links

- [API Reference](./api.md)
- [Architecture](./architecture.md)
- [Data Model](./data-model.md)
- [Configuration](./configuration.md)
- [Runbook](./runbook.md)
- [Changelog](./changelog.md)

## Owner

Software Architect / Platform Team
