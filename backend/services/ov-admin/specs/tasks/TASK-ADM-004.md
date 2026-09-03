# Task: Usecase Layer - Account & User CRUD (TASK-ADM-004)

**Status:** DONE

## Description
Implement the business logic for managing accounts, users, and namespace isolation.

## Requirements
- Implement `AccountUseCase` in `internal/usecase/account_ops.go` for Create, Get, List, and Delete operations.
- Implement `UserUseCase` in `internal/usecase/user_ops.go` for Create, Get, List, and Delete operations.
- Enforce RBAC permissions per operation (e.g., ADMIN can only manage users within their `account_id`).
- Implement namespace isolation logic (`viking://{account}/{user}/{agent}/`).
