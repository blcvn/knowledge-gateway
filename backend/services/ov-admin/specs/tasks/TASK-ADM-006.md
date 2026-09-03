# Task: Adapter Layer - gRPC Handlers (TASK-ADM-006)

**Status:** DONE

## Description
Implement the gRPC API layer for `ov-admin`.

## Requirements
- Generate protobuf stubs for `OvAdminService`.
- Implement gRPC handlers in `internal/adapter/grpc/handler.go` mapping to usecase methods.
- Implement RPC endpoints: `CreateAccount`, `GetAccount`, `ListAccounts`, `DeleteAccount`, `CreateUser`, `GetUser`, `ListUsers`, `DeleteUser`, `CreateAPIKey`, `ValidateAPIKey`, `RevokeAPIKey`, `ListAPIKeys`, `GetHealth`.
- Map domain errors to appropriate gRPC status codes (`NOT_FOUND`, `ALREADY_EXISTS`, `PERMISSION_DENIED`, `UNAUTHENTICATED`, `INVALID_ARGUMENT`).
- Integrate OpenTelemetry (OTel) spans for CRUD and health aggregation.
- Integrate Prometheus metrics (`ov_admin_accounts_total`, `ov_admin_users_total`, `ov_admin_api_key_validations_total`, `ov_admin_health_check_duration_seconds`).
- Implement structured JSON logging via `slog` injecting `request_id` and `account_id` into context.
