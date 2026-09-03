# Change Request: CR-SM-004 — User Profile Service

**CR ID:** CR-SM-004  
**Component:** `services/profile-service` [NEW SERVICE]  
**Priority:** High  
**Status:** In Progress
**Reference:** Supermemory PRD §3.2, SRS §2.5, specs/services/05-profile-service.md  
**Target latency:** < 100ms (p95)

---

## 1. Mô tả

Xây dựng **Profile Service** — tự động duy trì hồ sơ người dùng từ tất cả memories tích lũy:

1. **Static Profile**: Thực tế ổn định lâu dài (role, sở thích cốt lõi, thông tin cá nhân).
2. **Dynamic Profile**: Ngữ cảnh gần đây (dự án hiện tại, hoạt động gần đây).
3. **Profile + Search Combo**: Một lần gọi API lấy cả profile lẫn kết quả search (< 100ms).
4. **Redis Caching**: Cache profile với TTL 5 phút, invalidate khi có memory mới.
5. **Deduplication**: Ưu tiên Static > Dynamic > Search Results.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có khái niệm "User Profile" riêng biệt.
- AI phải tự tổng hợp profile từ search results → tốn thêm LLM call, tăng latency.
- Thiếu phân biệt rõ ràng giữa thông tin ổn định (static) và tạm thời (dynamic).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/profile-service/` (Port gRPC: 9004)

### 3.2. Domain Model

```go
type UserProfile struct {
    OrgID        string
    ContainerTag string
    Static       []string    // Long-term stable facts (isStatic=true memories)
    Dynamic      []string    // Recent activities (isStatic=false memories, sorted by UpdatedAt)
    UpdatedAt    time.Time
}

type ProfileWithSearch struct {
    Profile       UserProfile
    SearchResults []SearchResult  // Optional query results
}
```

### 3.3. Profile Build Algorithm

```go
// 1. Fetch all non-forgotten, isLatest=true memories for containerTag
// 2. Separate: Static (isStatic=true) vs Dynamic (isStatic=false, recent)
// 3. Dedup seen facts across both lists
// 4. Cache in Redis với TTL 5 phút
// 5. Invalidate on memory.created / memory.forgotten events
```

### 3.4. Event-Driven Update

- Subscribe `memory.created` → rebuild profile cho containerTag liên quan.
- Subscribe `memory.forgotten` → rebuild profile.

### 3.5. API Endpoints qua Gateway

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/api/v1/profiles` | Lấy profile cho containerTag |
| `POST` | `/api/v1/profiles/search` | Profile + search combo trong 1 call |
| `POST` | `/api/v1/profiles/rebuild` | Force rebuild profile từ memories |

**Response example:**
```json
{
  "static": [
    "User is a backend developer who prefers Go",
    "User works at ACME Corp as a senior engineer"
  ],
  "dynamic": [
    "Currently working on VNP Memory microservices",
    "Recently exploring pgvector for vector search"
  ],
  "searchResults": [...]  // Only if query provided
}
```

### 3.6. Inject vào System Prompt

Cung cấp helper để developer inject profile vào system prompt:
```
"About the user:
Long-term facts: {static}
Current context: {dynamic}"
```

---

## 4. Acceptance Criteria

- [ ] Sau khi lưu 10 memories cho một user, `GET /profiles?containerTag=user123` trả về profile chia đúng Static/Dynamic.
- [ ] Profile API latency p95 < 100ms (với Redis cache HIT).
- [ ] Thêm memory mới → Redis cache tự động invalidate, profile lần sau đã cập nhật.
- [ ] `POST /profiles/search` với `q="python"` trả về cả profile lẫn search results liên quan.
- [ ] Không có fact nào bị duplicate giữa static và dynamic arrays.
