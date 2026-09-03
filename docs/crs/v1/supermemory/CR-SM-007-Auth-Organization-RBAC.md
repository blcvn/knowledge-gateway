# Change Request: CR-SM-007 — Auth Service & Organization Management

**CR ID:** CR-SM-007  
**Component:** `services/auth-service` [UPGRADE SERVICE]  
**Priority:** High  
**Status:** In Progress
**Reference:** Supermemory PRD §4.3, SRS §2.8, specs/services/08-auth-service.md

---

## 1. Mô tả

Nâng cấp **Auth Service** của VNP Memory để hỗ trợ đầy đủ:

1. **Dual Auth**: JWT (RS256) cho web sessions + API Key (`sm_` prefix) cho programmatic access.
2. **Organization Management**: Multi-user organizations với invitation system.
3. **RBAC hoàn chỉnh**: 4 roles (owner, admin, editor, viewer) với permission matrix chi tiết.
4. **OAuth2 Server**: Cung cấp OAuth2 authorization server để MCP clients authenticate.
5. **API Key với `sm_` prefix**: Format chuẩn, hash bằng SHA-256 trước khi lưu DB.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện dùng API Key đơn giản, thiếu RBAC granular.
- Thiếu Organization model (multi-user, multi-tenant).
- Chưa có OAuth2 authorization server để hỗ trợ MCP OAuth flow.
- API keys chưa có format chuẩn (`sm_` prefix).

---

## 3. Thay đổi đề xuất

### 3.1. [UPGRADE] `services/auth-service/` (Port gRPC: 9007)

### 3.2. API Key Format

```go
// Format: sm_<32 random bytes base62>
// Example: sm_a7Kj3mN2pQ9xR4vW8yB5cD0eF6gH1iL
// Storage: SHA-256 hash trong DB, plaintext chỉ trả về 1 lần khi tạo

func GenerateAPIKey() (plaintext, hash string) {
    raw := make([]byte, 32)
    crypto_rand.Read(raw)
    plaintext = "sm_" + base62.Encode(raw)
    hash = sha256Hex(plaintext)
    return
}
```

### 3.3. RBAC Permission Matrix

| Permission | Owner | Admin | Editor | Viewer |
|-----------|-------|-------|--------|--------|
| `document:create` | ✅ | ✅ | ✅ | ❌ |
| `document:delete` | ✅ | ✅ | ✅ | ❌ |
| `memory:forget` | ✅ | ✅ | ✅ | ❌ |
| `search:execute` | ✅ | ✅ | ✅ | ✅ |
| `connection:create` | ✅ | ✅ | ✅ | ❌ |
| `settings:write` | ✅ | ❌ | ❌ | ❌ |
| `member:manage` | ✅ | ✅ | ❌ | ❌ |
| `apikey:manage` | ✅ | ✅ | ❌ | ❌ |
| `analytics:read` | ✅ | ✅ | ❌ | ✅ |

### 3.4. Organization API

```
POST /api/v1/auth/organizations          - Tạo organization
GET  /api/v1/auth/organizations/:id      - Lấy thông tin org
POST /api/v1/auth/organizations/:id/members   - Thêm member (với role)
DELETE /api/v1/auth/organizations/:id/members/:userId - Xóa member
```

### 3.5. OAuth2 Authorization Server

Cung cấp OAuth2 flow chuẩn để MCP clients tự authenticate:
- `GET  /.well-known/oauth-authorization-server` — Metadata discovery
- `GET  /api/v1/auth/oauth/authorize` — Authorization endpoint
- `POST /api/v1/auth/oauth/token` — Token exchange

### 3.6. Token Validation Cache

Validate API key: cache kết quả trong Redis 5 phút để giảm DB queries (hot path).

---

## 4. Acceptance Criteria

- [ ] Tạo API key → nhận plaintext `sm_xxx` chỉ 1 lần, sau đó không xem lại được.
- [ ] Request với API key của `viewer` tới `DELETE /documents/:id` → bị từ chối `403 Forbidden`.
- [ ] MCP client thực hiện OAuth flow → nhận access token hợp lệ.
- [ ] Member `editor` không thể thay đổi Organization settings.
- [ ] Token validation sử dụng Redis cache: gọi 100 req/s không tăng tải DB.
