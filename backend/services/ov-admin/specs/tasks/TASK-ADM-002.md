# Task: Infrastructure Layer - PostgreSQL Repositories (TASK-ADM-002)

**Status:** DONE

## Description
Implement the PostgreSQL persistence layer for `ov-admin` to manage accounts, users, and API keys.

## Requirements
- Create migration scripts for the PostgreSQL database (`ov_accounts`, `ov_users`, `ov_agents`).
- Apply correct constraints, defaults, and indices (`idx_users_account`, `idx_users_role`, `idx_agents_user`, `idx_agents_account`).
- Implement `AccountRepository` in `internal/infra/persistence/account_repo.go`.
- Implement `UserRepository` in `internal/infra/persistence/user_repo.go`.
- Implement `APIKeyRepository` in `internal/infra/persistence/api_key_repo.go`.
- Ensure all queries are scoped by `account_id` for tenant isolation.
