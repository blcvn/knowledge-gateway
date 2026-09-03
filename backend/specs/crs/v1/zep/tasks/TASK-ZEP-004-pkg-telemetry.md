# TASK-ZEP-004 — pkg/telemetry: Anonymous Telemetry (Opt-Out)

**Task ID:** TASK-ZEP-004  
**Wave:** 1 (Foundation)  
**Solution:** [SOL-ZEP-009](../solutions/SOL-ZEP-009-Resilience-Observability.md)  
**Depends on:** —  
**Ước tính:** 1h  
**Priority:** Medium

**Trạng thái:** ✅ Implemented  
**Ghi chú:** shared/pkg/telemetry: 2 .go (OTel tracer + metrics)  
---

## Mục tiêu

Tạo `pkg/telemetry/tracker.go` — anonymous usage tracking với opt-out. Không gửi PII. Project ID được hash trước khi gửi.

---

## Công việc cụ thể

### 1. Tạo `pkg/telemetry/tracker.go`

```go
package telemetry

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// TelemetryConfig cấu hình telemetry tracker
type TelemetryConfig struct {
    Disabled  bool   // true = opt-out, all Track() calls are no-ops
    ProjectID string // sẽ được SHA-256 hash trước khi gửi (no PII)
    Version   string // service version
    Endpoint  string // telemetry collection endpoint
}

// TelemetryEvent là event được gửi đi (no PII)
type TelemetryEvent struct {
    Event     string    `json:"event"`
    ProjectID string    `json:"project_id"` // SHA-256 hash, anonymous
    Version   string    `json:"version"`
    Timestamp time.Time `json:"ts"`
}

// Tracker gửi anonymous telemetry events
type Tracker struct {
    cfg    TelemetryConfig
    client *http.Client
    // hashed project ID (computed once at construction)
    hashedProjectID string
}

// NewTracker tạo Tracker mới.
// Nếu cfg.Disabled = true, tất cả Track() calls đều là no-ops.
func NewTracker(cfg TelemetryConfig) *Tracker { ... }

// Track gửi một anonymous event.
// Fire-and-forget (goroutine), không block request processing.
// No-op nếu Disabled = true.
//
// Events chuẩn:
//   "service_start"
//   "api_request"
//   "graph_extraction"
//   "search_query"
func (t *Tracker) Track(ctx context.Context, event string) { ... }
```

### 2. Tạo `pkg/telemetry/tracker_test.go`

Test cases:
- `TestTracker_DisabledIsNoop`: Track() khi Disabled=true → không gửi HTTP request
- `TestTracker_HashesProjectID`: event.project_id là SHA-256 hash, không phải raw ID
- `TestTracker_FireAndForget`: Track() trả về ngay (không block)
- `TestTracker_NoPIIInPayload`: payload JSON không chứa raw project ID

---

## Acceptance Criteria

- [ ] `go build ./pkg/telemetry/...` không có lỗi
- [ ] `go test ./pkg/telemetry/...` 100% pass
- [ ] `cfg.Disabled = true` → Track() là no-op (0 HTTP requests)
- [ ] `event.project_id` luôn là SHA-256 hash (không bao giờ là raw ID)
- [ ] Track() không block — goroutine ngay lập tức
- [ ] HTTP request timeout 5s (không treo indefinitely)

---

## Files tạo ra

```
pkg/telemetry/
├── tracker.go
└── tracker_test.go
```

## Sau khi hoàn thành

Chạy: `go build ./pkg/telemetry/... && go test ./pkg/telemetry/...`
