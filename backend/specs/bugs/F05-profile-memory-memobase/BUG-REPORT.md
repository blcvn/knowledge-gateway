# Bug Report — F05: Profile Memory (Memobase)

> Feature: Blob insert, flush, context retrieval, profile query, event query
> Luồng: `apps/memory → gateway/adapter/handler/services.go (MemobaseHandler) → memory-service`

---

## BUG-F05-001: `GetEvents` Route Tới `vnp-platform` Thay Vì `memory-service`

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/services.go:107-109`

**Mô tả:**  
`MemobaseHandler.GetEvents` forward tới `vnp-platform` trong khi các methods khác (InsertBlob, Flush, GetContext, GetProfiles) đều forward tới `memory-service`. Feature spec cho thấy Memobase events nên được serve từ cùng một service.

```go
func (h *MemobaseHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)  // Inconsistent
}
```

**Impact:**  
- `GET /v1/memobase/users/{uid}/events` có thể trả về empty hoặc wrong data nếu `vnp-platform` không implement endpoint này.
- Inconsistency trong data routing.

---

## BUG-F05-002: Memobase Auto-Flush Logic (20 blobs) Chưa Implement

**Severity:** HIGH  
**File:** `services/memory-service/internal/usecase/memobase/`

**Mô tả:**  
Feature spec yêu cầu auto-flush khi đạt 20 blobs. `IngestUseCase` được tạo với `nil` params cho flush trigger và event publisher:

```go
mbIngest := ucmb.NewIngestUseCase(blobRepo, nil, nil, nil)
```

**Impact:**  
- Auto-flush tại 20 blobs không hoạt động — blobs tích lũy mà không bao giờ được flush thành profile.
- Profile memory sẽ không được cập nhật tự động.

---

## BUG-F05-003: Thiếu `memobase-admin`, `memobase-event` Service Implementations

**Severity:** MEDIUM  
**File:** `services/memobase-admin/`, `services/memobase-event/`

**Mô tả:**  
Các services memobase chuyên biệt (`memobase-admin`, `memobase-event`) tồn tại như thư mục nhưng không có implementation.

**Impact:**  
- Admin operations và event handling cho Memobase không hoạt động.
