# Core Skills — API Design & Integration Patterns

## RESTful API Design

### Resource Naming & URL Structure
```
# Correct — noun-based, hierarchical
GET    /v1/requirements                    # List all requirements
GET    /v1/requirements/{id}              # Get one requirement
POST   /v1/requirements                    # Create requirement
PUT    /v1/requirements/{id}              # Replace requirement
PATCH  /v1/requirements/{id}              # Partial update
DELETE /v1/requirements/{id}              # Delete requirement

GET    /v1/requirements/{id}/screens      # Get screens linked to a requirement
POST   /v1/pipeline/runs                  # Trigger a pipeline run (action as resource)

# Wrong — verb-based, inconsistent
POST   /v1/createRequirement
GET    /v1/getRequirementById/{id}
POST   /v1/runPipeline
```

### Standard Error Response Format
```json
{
  "error": {
    "code": "REQUIREMENT_NOT_FOUND",
    "message": "Requirement with id 'req-123' was not found.",
    "details": [
      { "field": "id", "issue": "No requirement exists with this identifier." }
    ],
    "request_id": "req-abc-456",
    "timestamp": "2026-04-20T11:30:00Z"
  }
}
```

### HTTP Status Code Guide
| Code | When to Use |
|---|---|
| 200 OK | Successful GET, PATCH, DELETE |
| 201 Created | Successful POST that creates a resource |
| 202 Accepted | Async operation started (e.g., pipeline run triggered) |
| 400 Bad Request | Client sent invalid data (validation error) |
| 401 Unauthorized | Missing or invalid authentication token |
| 403 Forbidden | Authenticated but not authorized for this action |
| 404 Not Found | Resource does not exist |
| 409 Conflict | Conflict with current state (e.g., duplicate resource) |
| 422 Unprocessable | Syntactically valid but semantically wrong |
| 429 Too Many Requests | Rate limit exceeded |
| 500 Internal Server Error | Unexpected server failure |

## API Versioning Strategy
- **URL Path Versioning:** `/v1/...` — explicit, easy to route, recommended for this platform.
- **Deprecation Policy:** A version is supported for a minimum of 6 months after a new version is released. Deprecation is announced via `Deprecation` and `Sunset` response headers.
- **Backward Compatibility Rules:**
  - ✅ Safe: Adding new optional fields to responses
  - ✅ Safe: Adding new optional query parameters
  - ❌ Breaking: Removing fields from responses
  - ❌ Breaking: Changing field types
  - ❌ Breaking: Making optional fields required

## OpenAPI Specification
Every API endpoint MUST be documented in an OpenAPI 3.1 specification before implementation:
```yaml
paths:
  /v1/requirements/{id}:
    get:
      summary: Get a requirement by ID
      operationId: getRequirementById
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Requirement found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Requirement'
        '404':
          $ref: '#/components/responses/NotFound'
```
