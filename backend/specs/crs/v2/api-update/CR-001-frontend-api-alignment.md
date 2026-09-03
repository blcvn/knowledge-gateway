# Change Request: Frontend API Alignment (v2)

**CR ID**: CR-001-frontend-api-alignment
**Status**: Draft
**Target Component**: `vnp-gateway`, `sm-auth`, `vnp-admin`
**Created**: 2026-06-18

## 1. Overview
After cross-referencing the Backend API Specifications (`specs/backend-api-specs.md`) with the Frontend API Specifications (`ui/specs/frontend-backend-api-specs.md`), several missing endpoints and parameter enhancements have been identified. The backend must implement these missing APIs to fully support the frontend application.

## 2. Missing APIs \u0026 Enhancements

### 2.1. Authentication API (`/v1/auth/*`)
The frontend application requires a dedicated authentication API for session management (login, logout, token refresh, and user profile retrieval). Currently, the gateway only documents standard JWT/API key validation.

**Required Endpoints:**
| Method | Path | Request Body | Description |
|--------|------|--------------|-------------|
| `POST` | `/v1/auth/login` | `{ "email": "", "password": "" }` | Authenticates a user and returns `access_token`, `refresh_token`, and user details (incl. `tenant_id`). |
| `POST` | `/v1/auth/logout` | `{ "refresh_token": "" }` | Logs out the current user and invalidates the session. |
| `GET` | `/v1/auth/me` | - | Retrieves the current authenticated user's profile (`AuthUser`). |
| `POST` | `/v1/auth/refresh` | `{ "refresh_token": "" }` | Refreshes the access token and returns a new set of tokens (`RefreshResponse`). |

**Implementation Notes:**
- Gateway needs to route `/v1/auth/*` to the appropriate auth service (e.g., `sm-auth`).
- `GET /v1/auth/me` should use the `X-Tenant-ID` and user ID from the `Authorization` header to fetch the profile.

### 2.2. Organization \u0026 SDK Management APIs (`/v1/console/org/*` \u0026 `/v1/console/sdk/*`)
The frontend includes sections for managing organization settings, members, SDK API keys, rate limits, and webhooks. These console routes are completely missing from the gateway's current routing table.

**Required Endpoints:**

**Org API:**
| Method | Path | Request Body | Description |
|--------|------|--------------|-------------|
| `GET` | `/v1/console/org/settings` | - | Retrieves organization settings. |
| `PUT` | `/v1/console/org/settings` | `Partial<OrgSettings>` | Updates organization settings. |
| `GET` | `/v1/console/org/members` | - | Lists organization members. |
| `GET` | `/v1/console/org/roles` | - | Lists available organization roles. |

**SDK API:**
| Method | Path | Request Body | Description |
|--------|------|--------------|-------------|
| `GET` | `/v1/console/sdk/keys` | - | Lists SDK API keys (raw key should be masked). |
| `POST` | `/v1/console/sdk/keys` | `CreateKeyPayload` | Creates a new API key. Returns the `raw_key` only once. |
| `DELETE` | `/v1/console/sdk/keys/{id}` | - | Revokes/Deletes an API key. |
| `GET` | `/v1/console/sdk/rate-limits` | - | Retrieves rate limit configurations. |
| `GET` | `/v1/console/sdk/webhooks` | - | Lists configured webhooks. |
| `POST` | `/v1/console/sdk/webhooks` | `CreateWebhookPayload` | Creates a new webhook. |
| `DELETE` | `/v1/console/sdk/webhooks/{id}` | - | Deletes a webhook. |

**Implementation Notes:**
- These routes should be added to `gateway/internal/adapter/handler/router.go`.
- They will likely be handled by a new `OrgHandler` and `SDKHandler` which interact with `vnp-admin` or the governance services.

### 2.3. Session API Query Parameters Enhancement
The frontend depends on extensive filtering and pagination capabilities for listing sessions. The current backend specification lists `GET /v1/console/sessions` but does not explicitly document or guarantee support for these parameters.

**Required Enhancements for `GET /v1/console/sessions`:**
Must explicitly parse and support the following query parameters:
- `status` (e.g., active, ended)
- `user_id`
- `agent_id`
- `search` (text search across session metadata)
- `sort`
- `page`
- `page_size`

## 3. Action Items
1. **Update Gateway Router**: Add `/v1/auth/*`, `/v1/console/org/*`, and `/v1/console/sdk/*` routes to `gateway/internal/adapter/handler/router.go`.
2. **Implement Handlers**: Create matching handler methods and wire them to backend gRPC services (`sm-auth`, `vnp-admin`).
3. **Enhance Session Handler**: Update `session.ListSessions` handler to properly extract query parameters and pass them down to the fetching service.
4. **Update API Documentation**: Once implemented, update `gateway/docs/api.md` to include these new endpoints.
