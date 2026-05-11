# Task: Infrastructure Layer - Bootstrap & Configuration (TASK-ADM-007)

**Status:** DONE

## Description
Implement configuration management, dependency injection, and application bootstrap.

## Requirements
- Implement configuration loading in `internal/infra/config/config.go` with variables: `GRPC_PORT`, `HEALTH_PORT`, `LOG_LEVEL`, `OTEL_ENDPOINT`, `DB_DSN`, `DB_MAX_CONNECTIONS`, `AUTH_MODE`, `DEV_ACCOUNT_ID`, `DEV_USER_ID`, `OV_CRYPTO_ADDR`, `HEALTH_CHECK_TIMEOUT_MS`, `OV_FS_ADDR`, `OV_SEARCH_ADDR`, `OV_SESSION_ADDR`, `OV_RESOURCE_ADDR`.
- Setup Wire dependency injection in `internal/infra/wire/wire.go` to wire Domain, Usecase, Adapter, and Infra layers.
- Implement the main entrypoint `cmd/server/main.go`.
- Configure the gRPC server on `GRPC_PORT` (default 9056) and HTTP `/healthz` and `/readyz` probes on `HEALTH_PORT` (default 9109).
- Ensure graceful shutdown mechanisms are in place via SIGTERM (15s timeout).
