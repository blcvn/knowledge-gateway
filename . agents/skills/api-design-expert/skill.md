---
skill_id: SKILL-003
version: 1.0.0
status: active
priority: P2
group: Kiến trúc & Thiết kế
created_at: 2026-04-24
---

# SKILL-003 · API Design & Integration Patterns

## Mô tả

Thiết kế và tích hợp các API — REST, gRPC, GraphQL. Đảm bảo API contract nhất quán, an toàn, và có versioning rõ ràng giữa các services trong pipeline.

## Agents sử dụng

- `knowledge-graph-agent`
- `ui-schema-generator-agent`
- `qa-pipeline-agent`

---

## Năng lực cốt lõi

### 1. RESTful API Design

```
Nguyên tắc Resource Naming:
├── Dùng danh từ số nhiều: /documents, /jobs, /schemas
├── Nested resources: /documents/{id}/requirements
├── Query parameters cho filtering: /documents?status=active&project_id=xxx
├── Không dùng verbs trong URL: ❌ /createDocument → ✅ POST /documents
└── Consistent ID format: UUID v4

HTTP Semantics:
├── GET     → Đọc resource (idempotent, cacheable)
├── POST    → Tạo resource hoặc trigger action
├── PUT     → Replace toàn bộ resource (idempotent)
├── PATCH   → Partial update (idempotent)
└── DELETE  → Xóa resource (idempotent)

Standard Response Codes:
├── 200 OK            → Success (GET, PATCH, PUT)
├── 201 Created       → Resource created (POST)
├── 204 No Content    → Success, no body (DELETE)
├── 400 Bad Request   → Invalid input format
├── 401 Unauthorized  → Not authenticated
├── 403 Forbidden     → Authenticated but no permission
├── 404 Not Found     → Resource doesn't exist
├── 409 Conflict      → Duplicate resource
├── 422 Unprocessable → Valid format, invalid business logic
└── 429 Too Many Req  → Rate limit exceeded
```

```go
// Standard error response format
type ErrorResponse struct {
    Code    string            `json:"code"`    // "VALIDATION_ERROR"
    Message string            `json:"message"` // Human-readable
    Details map[string]string `json:"details,omitempty"` // Field-level errors
    TraceID string            `json:"trace_id"` // For debugging
}

// Standard success response format
type Response[T any] struct {
    Data    T      `json:"data"`
    Meta    *Meta  `json:"meta,omitempty"`
}

type Meta struct {
    Total   int    `json:"total,omitempty"`
    Page    int    `json:"page,omitempty"`
    PerPage int    `json:"per_page,omitempty"`
}
```

### 2. gRPC / Protocol Buffers

```protobuf
// Service definition chuẩn cho Knowledge Gateway
syntax = "proto3";
package knowledge.v1;

service KnowledgeService {
  // Unary RPC
  rpc CreateDocument(CreateDocumentRequest) returns (Document);
  rpc GetDocument(GetDocumentRequest) returns (Document);
  
  // Server streaming — kết quả pipeline từng stage
  rpc StreamPipelineProgress(StreamProgressRequest) returns (stream ProgressEvent);
  
  // Bidirectional streaming — interactive extraction
  rpc InteractiveExtract(stream ExtractRequest) returns (stream ExtractResponse);
}

message Document {
  string id = 1;
  string project_id = 2;
  string title = 3;
  string content = 4;
  DocumentStatus status = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}

enum DocumentStatus {
  DOCUMENT_STATUS_UNSPECIFIED = 0;
  DOCUMENT_STATUS_PENDING = 1;
  DOCUMENT_STATUS_PROCESSING = 2;
  DOCUMENT_STATUS_COMPLETED = 3;
  DOCUMENT_STATUS_FAILED = 4;
}
```

```go
// gRPC client với retry và timeout
func NewKnowledgeClient(addr string) (pb.KnowledgeServiceClient, error) {
    conn, err := grpc.NewClient(addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(
            grpc_retry.WithMax(3),
            grpc_retry.WithBackoff(grpc_retry.BackoffExponential(100*time.Millisecond)),
            grpc_retry.WithCodes(codes.Unavailable, codes.DeadlineExceeded),
        )),
        grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
    )
    return pb.NewKnowledgeServiceClient(conn), err
}
```

### 3. API Versioning

```
Chiến lược versioning cho Knowledge Gateway:
├── URL versioning: /v1/, /v2/ (recommended — explicit, easy to route)
├── Breaking changes → bump major version (/v1 → /v2)
├── Non-breaking additions → same version + deprecation notice
└── Old version support: 6 months deprecation period

Breaking changes (cần version bump):
- Removing or renaming fields
- Changing field types
- Changing HTTP method for an endpoint
- Changing response structure

Non-breaking changes (same version):
- Adding new optional fields
- Adding new endpoints
- Adding new enum values (careful with exhaustive switches)
- Adding new optional query parameters
```

### 4. Rate Limiting

```go
// Token bucket rate limiter
type RateLimiter struct {
    limiter *rate.Limiter
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(rps), burst),
    }
}

// Rate limit middleware (Gin)
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !limiter.limiter.Allow() {
            c.Header("X-RateLimit-Limit", "100")
            c.Header("Retry-After", "1")
            c.JSON(http.StatusTooManyRequests, ErrorResponse{
                Code:    "RATE_LIMIT_EXCEEDED",
                Message: "Too many requests. Please retry after 1 second.",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}

// Per-project rate limits (Redis-backed sliding window)
func (r *RedisRateLimiter) Allow(ctx context.Context, projectID string) (bool, error) {
    key := fmt.Sprintf("ratelimit:%s:%d", projectID, time.Now().Unix())
    count, err := r.redis.Incr(ctx, key).Result()
    r.redis.Expire(ctx, key, time.Minute)
    return count <= r.limit, err
}
```

### 5. API Gateway Pattern

```yaml
# Nginx API Gateway config
upstream knowledge-service {
    server localhost:8080;
    keepalive 32;
}

server {
    location /v1/documents {
        # Authentication delegation
        auth_request /auth/validate;
        auth_request_set $user_id $upstream_http_x_user_id;
        
        # Request transformation
        proxy_set_header X-User-ID $user_id;
        proxy_set_header X-Request-ID $request_id;
        
        # Rate limiting
        limit_req zone=per_project burst=20 nodelay;
        
        proxy_pass http://knowledge-service;
    }
}
```

### 6. OpenAPI Specification

```yaml
# Template cho Knowledge Gateway API spec
openapi: 3.1.0
info:
  title: Knowledge Gateway API
  version: v1.0.0
  description: API for PRD-to-UI pipeline automation

servers:
  - url: https://api.knowledge-gateway.io/v1
    description: Production
  - url: http://localhost:8080/v1
    description: Development

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  schemas:
    Document:
      type: object
      required: [id, project_id, title, status]
      properties:
        id:
          type: string
          format: uuid
        project_id:
          type: string
          format: uuid
        title:
          type: string
          maxLength: 500
        status:
          type: string
          enum: [pending, processing, completed, failed]
        created_at:
          type: string
          format: date-time

security:
  - BearerAuth: []
```

---

## Checklist

- [ ] Tất cả endpoints đã có OpenAPI spec
- [ ] Error responses theo chuẩn `{code, message, details, trace_id}`
- [ ] Rate limiting đã implement cho tất cả public endpoints
- [ ] gRPC services đã có retry interceptor
- [ ] API versioning strategy đã được ghi vào ADR
- [ ] Breaking vs non-breaking changes đã được phân loại
- [ ] Request/response logging với trace ID
- [ ] API contract tests (contract testing với Pact hoặc manual)
