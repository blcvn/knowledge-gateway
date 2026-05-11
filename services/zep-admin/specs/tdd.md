# Technical Design Document: Zep Admin Service

## 1. System Architecture

The `zep-admin` service follows the 4-layer Clean Architecture, focusing on tenant isolation, project metadata, API keys, and health aggregation.

```text
zep-admin/
├── internal/
│   ├── domain/
│   │   ├── project/     # Project entities, Tenant ID
│   │   └── auth/        # API keys, Argon2id hashing
│   ├── usecase/
│   │   ├── project/     # Project lifecycle management
│   │   ├── auth/        # Key generation and validation
│   │   └── health/      # Cross-service health aggregation
│   ├── adapter/
│   │   ├── grpc/        # gRPC Handlers (ZepAdminService)
│   │   └── broker/      # NATS Publisher
│   └── infra/
│       ├── postgres/    # PostgreSQL for projects and keys
│       ├── config/      # Viper configuration
│       └── metrics/     # Prometheus/OTel
```

## 2. Component Design

### 2.1 Domain Layer
- **Project Entity**: Contains `uuid`, `name`, `config`, timestamps.
- **ApiKey Entity**: Contains `key_hash` (Argon2id), `project_uuid`, `role`.
- **Interfaces**: `ProjectRepository`, `ApiKeyRepository`, `EventPublisher`.

### 2.2 Usecase Layer
- **CreateProject**: Creates a project in DB, publishes `zep.admin.project.created` to NATS.
- **DeleteProject**: Deletes project, publishes `zep.admin.project.deleted` for cascade removal.
- **GenerateApiKey**: Generates raw token, hashes with Argon2id, stores hash, returns raw token.

### 2.3 Adapter Layer
- **gRPC Handlers**: Implements `ZepAdminService`.
- **NATS Publisher**: Emits system lifecycle events.

### 2.4 Infrastructure Layer
- **Database**: PostgreSQL with `admin` schema.
- **Observability**: OpenTelemetry tracing, Prometheus metrics (`zep_admin_projects_total`, `zep_admin_api_calls_total`).

## 3. Data Models

```sql
CREATE TABLE projects (
    uuid UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    deleted_at BIGINT
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    project_uuid UUID REFERENCES projects(uuid) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL,
    created_at BIGINT NOT NULL,
    expires_at BIGINT
);
```
