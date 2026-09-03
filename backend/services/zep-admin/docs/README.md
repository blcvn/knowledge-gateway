---
id: DOC-S01
service: zep-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Zep Team
---

# zep-admin

> **Group**: Zep (Context Engineering) | **gRPC Port**: 9066 | **Health Port**: 12066 | **Origin**: Zep

## Purpose

Cross-cutting admin service quản lý health aggregation, project/tenant management, API key lifecycle, và schema migration. Orchestrate health checks across all 5 Zep domain services.

### Business Capability

- **Health Aggregation**: Parallel health check across 5 Zep services → aggregated status
- **Project Management**: CRUD cho projects (multi-tenant isolation unit)
- **API Key Management**: Generate, validate, revoke API keys (SHA-256 hash, prefix identification)
- **Schema Management**: Trigger migration across services
- **Telemetry Config**: Enable/disable anonymous usage telemetry

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server + NATS JetStream publisher
- **Database**: PostgreSQL (projects, api_keys tables)
- **Architecture**: 4-layer Clean Architecture

## Quick Start

```bash
make build-zep-admin
make run-zep-admin
docker compose up zep-admin
```

## API Surface

### gRPC Service

```protobuf
service AdminService {
  rpc AggregatedHealth(google.protobuf.Empty) returns (AggregatedHealthResponse);
  rpc CreateProject(CreateProjectRequest) returns (ProjectResponse);
  rpc GetProject(GetProjectRequest) returns (ProjectResponse);
  rpc ListProjects(ListProjectsRequest) returns (ProjectListResponse);
  rpc UpdateProject(UpdateProjectRequest) returns (ProjectResponse);
  rpc DeleteProject(DeleteProjectRequest) returns (google.protobuf.Empty);
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (APIKeyListResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc MigrateSchema(MigrateSchemaRequest) returns (MigrateSchemaResponse);
}
```

## NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.admin.project.created` | `{project_uuid, name, settings}` | All Zep services (init per-project config) |
| `zep.admin.project.deleted` | `{project_uuid}` | All Zep services (cascade delete) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| zep-user | gRPC Health | Health check |
| zep-thread | gRPC Health | Health check |
| zep-memory | gRPC Health | Health check |
| zep-graph | gRPC Health | Health check |
| zep-search | gRPC Health | Health check |
| PostgreSQL | SQL | Projects, API keys |
| NATS JetStream | Pub | Project lifecycle events |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Reference Design](../../../references/zep/specs/services/07-admin-service.md)

## Owner

- **Team**: VNP Memory — Zep
