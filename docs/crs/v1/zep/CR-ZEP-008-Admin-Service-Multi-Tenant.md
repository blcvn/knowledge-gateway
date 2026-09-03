# Change Request: CR-ZEP-008 — Admin Service & Multi-Tenant Project Management

**CR ID:** CR-ZEP-008  
**Component:** `services/admin-service` [NEW SERVICE]  
**Priority:** Medium  
**Status:** In Progress
**Reference:** Zep PRD §7.2, SRS §5.5, specs/services/07-admin-service.md

---

## 1. Mô tả

Xây dựng **Admin Service (zep-admin)** — quản lý cross-cutting concerns của VNP Memory:

1. **Health Aggregation**: Aggregate health checks từ tất cả services.
2. **Project/Tenant Management**: CRUD projects (multi-tenant namespaces).
3. **API Key Management**: Tạo, liệt kê, revoke API keys.
4. **Telemetry Control**: Enable/disable anonymous usage tracking.
5. **Shared Secret Auth**: Hỗ trợ `Authorization: Bearer <shared-secret>` cho self-hosted.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có service tập trung để quản lý multi-tenant projects.
- Thiếu aggregated health dashboard cho tất cả services.
- API key management hiện phân tán.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/admin-service/` (Port gRPC: 9046)

### 3.2. Project Management

```go
type Project struct {
    UUID      string
    Name      string
    APIKeys   []APIKey
    Settings  ProjectSettings
    CreatedAt time.Time
}

type ProjectSettings struct {
    MaxRequestSizeMB int       // default: 5
    TimeoutSeconds   int       // default: 30
    TelemetryEnabled bool      // anonymous usage tracking
}
```

### 3.3. Health Aggregation

```go
// GET /healthz → aggregate health từ all services
type HealthResponse struct {
    Status   string                    // "healthy" | "degraded" | "unhealthy"
    Services map[string]ServiceHealth  // per-service health status
    Uptime   time.Duration
}

type ServiceHealth struct {
    Status  string  // "up" | "down"
    Latency int64   // ms
}
// Fan-out gRPC health checks đến: user, thread, memory, graph, search services
```

### 3.4. API Key Management

```go
// Tạo API key cho project
// Format: "vnp_" + base62(32 random bytes)
// Storage: SHA-256 hash, plaintext trả về 1 lần

type APIKey struct {
    UUID        string
    ProjectUUID string
    Name        string        // human-friendly label
    Hash        string        // SHA-256 of plaintext key
    Prefix      string        // first 8 chars for identification
    CreatedAt   time.Time
    LastUsedAt  *time.Time
    RevokedAt   *time.Time    // nil = active
}
```

### 3.5. API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/healthz` | Aggregated health check |
| `GET` | `/api/v2/admin/projects` | List projects |
| `POST` | `/api/v2/admin/projects` | Create project |
| `GET` | `/api/v2/admin/projects/:id` | Get project |
| `DELETE` | `/api/v2/admin/projects/:id` | Delete project (cascade) |
| `POST` | `/api/v2/admin/api-keys` | Create API key |
| `GET` | `/api/v2/admin/api-keys` | List API keys |
| `DELETE` | `/api/v2/admin/api-keys/:id` | Revoke API key |

### 3.6. NATS Events (Cascade)

```
zep.admin.project.created → All services (init tenant schema)
zep.admin.project.deleted → All services (cascade delete all tenant data)
```

---

## 4. Acceptance Criteria

- [ ] `GET /healthz` trả về status của tất cả downstream services trong một response.
- [ ] Tạo project → publish NATS event → tất cả services nhận và init tenant schema.
- [ ] Delete project → cascade delete tất cả data của tenant đó qua NATS.
- [ ] Tạo API key → nhận plaintext `vnp_xxx` 1 lần duy nhất.
- [ ] Revoke API key → requests với key đó bị từ chối 401 ngay lập tức.
