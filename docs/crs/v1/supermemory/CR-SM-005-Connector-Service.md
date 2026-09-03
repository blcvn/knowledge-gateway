# Change Request: CR-SM-005 — External Connector Service

**CR ID:** CR-SM-005  
**Component:** `services/connector-service` [NEW SERVICE]  
**Priority:** High  
**Status:** In Progress
**Reference:** Supermemory PRD §3.5, SRS §2.6, specs/services/06-connector-service.md

---

## 1. Mô tả

Xây dựng **Connector Service** — đồng bộ dữ liệu tự động từ các nguồn bên ngoài vào VNP Memory:

1. **Provider Support**: Google Drive, Gmail, Notion, OneDrive, GitHub, Web Crawler.
2. **OAuth2 Flow**: Toàn bộ vòng đời OAuth (generate URL → callback → store token → refresh).
3. **Scheduled Sync**: Cron job chạy mỗi 4 giờ để sync cập nhật.
4. **Webhook Support**: Real-time notifications từ Google Drive và Notion.
5. **Document Limit**: Giới hạn tối đa 10,000 tài liệu mỗi connection.
6. **Token Encryption**: OAuth tokens được mã hóa AES-GCM tại rest.
7. **Custom OAuth Keys**: Enterprise có thể dùng OAuth keys riêng.

---

## 2. Vấn đề hiện tại

- VNP Memory hiện tại không có tích hợp với nguồn dữ liệu bên ngoài (Google Drive, Notion, v.v.).
- Không có cơ chế tự động đồng bộ định kỳ.
- Thiếu hỗ trợ OAuth2 flow hoàn chỉnh (state CSRF, token refresh, re-auth).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/connector-service/` (Port gRPC: 9005)

### 3.2. Connection Lifecycle

```
1. CreateConnection(provider, redirectURL, containerTags, documentLimit)
   → Generate OAuth URL với stateToken (CSRF)
   → Trả về {authLink, connectionID}

2. User authorize trên provider

3. OAuth Callback (CompleteOAuth)
   → Validate stateToken
   → Exchange code → access + refresh tokens
   → Mã hóa tokens AES-GCM → lưu DB
   → Trigger initial sync (async)

4. Sync (Cron 4h / Webhook / Manual)
   → Refresh token nếu hết hạn
   → Fetch danh sách tài liệu từ provider
   → Với mỗi doc mới/cập nhật: gọi Document Service.CreateDocument()
   → Áp dụng containerTags từ connection config
   → Enforce documentLimit (max 10,000)
   → Log kết quả sync
   → Publish "connection.synced"
```

### 3.3. Provider Adapters

| Provider | API | Sync Strategy |
|----------|-----|--------------|
| **Google Drive** | Google Drive API v3 | Files list + webhook changes |
| **Gmail** | Gmail API | Messages list (last N) |
| **Notion** | Notion API | Pages list, blocks export |
| **OneDrive** | Microsoft Graph API | Drive items list |
| **GitHub** | GitHub REST API | Repo content export |
| **Web Crawler** | HTTP + goquery | URL list crawl |

### 3.4. Domain Model

```go
type Connection struct {
    ID             string
    OrgID          string
    Provider       Provider      // google_drive | gmail | notion | onedrive | github | web
    Status         ConnectionStatus
    AccessToken    []byte        // AES-GCM encrypted
    RefreshToken   []byte        // AES-GCM encrypted
    TokenExpiresAt *time.Time
    DocumentLimit  int           // Default 10,000
    ContainerTags  []string      // Applied to all imported docs
    CustomKey      *ConnectionConfig // Enterprise: custom OAuth keys
    LastSyncAt     *time.Time
    DocumentCount  int
}
```

### 3.5. API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/api/v1/connections/:provider` | Tạo connection, nhận OAuth URL |
| `GET` | `/api/v1/connections` | Liệt kê connections của org |
| `DELETE` | `/api/v1/connections/:id` | Xóa connection + cleanup |
| `POST` | `/api/v1/connections/:id/sync` | Manual trigger sync |
| `GET` | `/api/v1/connections/:id/history` | Lịch sử sync |

---

## 4. Acceptance Criteria

- [ ] Kết nối Google Drive → nhận OAuth link → authorize → tài liệu trong Drive tự động được ingest.
- [ ] Sau 4 giờ, cron job chạy và sync thêm tài liệu mới trong Drive.
- [ ] Xóa connection → tài liệu đã import KHÔNG bị xóa (chỉ ngắt đồng bộ).
- [ ] OAuth token được mã hóa: đọc trực tiếp từ DB không thấy plaintext token.
- [ ] Connection vượt quá 10,000 documents → hệ thống dừng import và log cảnh báo.
- [ ] Custom OAuth keys hoạt động với Enterprise account.
