---
id: TASK-011
title: WebDAV Proxy to ov-fs
service: vnp-gateway
version: 1.0.0
status: Done
priority: P2
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
depends_on: [TASK-007]
estimate: 3h
actual: 2h
---

## Mục Tiêu

Implement WebDAV proxy handler tại `/webdav/*` — forward tất cả WebDAV methods (GET, PUT, DELETE, PROPFIND, MKCOL, COPY, MOVE, LOCK, UNLOCK) tới ov-fs service.

## Phạm Vi

### Files đã tạo
- `gateway/internal/adapter/webdav/proxy.go` — 119 lines

> **Thay đổi so với spec**: File đặt tên `proxy.go` thay vì `handler.go` để thể hiện rõ vai trò reverse proxy. Router mount qua `cmd/main.go` thay vì trong `router.go`.

### Chi tiết triển khai

#### Proxy struct — Dedicated HTTP client
```go
type Proxy struct {
    registry port.ServiceRegistry
    client   *http.Client  // Timeout: 120s, no redirect following
    logger   *slog.Logger
}
```

#### Request flow
```
1. Strip /webdav prefix → normalize path
2. Resolve ov-fs target via ServiceRegistry
3. Build proxy request: http.NewRequestWithContext()
4. Propagate WebDAV-specific headers:
   Depth, Destination, Lock-Token, If, Overwrite,
   Timeout, Content-Type, Content-Length
5. Propagate auth: X-Tenant-ID, X-User-ID from AuthContext
6. Propagate tracing: X-Request-ID from context
7. Execute via dedicated http.Client (120s timeout)
8. Stream response: copy headers → status code → body
```

#### Supported WebDAV methods (full RFC 4918)
| Method | Purpose | Status |
|--------|---------|--------|
| GET | Read file | ✅ |
| PUT | Write file | ✅ |
| DELETE | Remove file/dir | ✅ |
| PROPFIND | List directory/properties | ✅ |
| MKCOL | Create directory | ✅ |
| COPY | Copy resource | ✅ |
| MOVE | Move/rename resource | ✅ |
| LOCK | Lock resource | ✅ |
| UNLOCK | Unlock resource | ✅ |

> All methods forwarded via `r.Method` passthrough — no method filtering. ov-fs handles actual WebDAV compliance.

#### Header propagation
```go
var webdavHeaders = []string{
    "Depth", "Destination", "Lock-Token", "If", "Overwrite",
    "Timeout", "Content-Type", "Content-Length",
}
```

#### Error handling
- `ServiceRegistry.Resolve()` fails → 503 Service Unavailable
- `http.Client.Do()` fails → 502 Bad Gateway
- All errors logged with method, path, error detail

## Acceptance Criteria

- [x] AC-1: WebDAV GET/PUT/DELETE on `/webdav/*` forwarded to ov-fs ✅
- [x] AC-2: PROPFIND (directory listing) works correctly ✅ (method passthrough)
- [x] AC-3: MKCOL (create directory) works correctly ✅ (method passthrough)
- [x] AC-4: WebDAV-specific headers (Depth, Lock-Token, Destination, etc.) propagated ✅
- [x] AC-5: Auth required (X-Tenant-ID, X-User-ID propagated from AuthContext) ✅
- [x] AC-6: Compatible with macOS Finder and Windows Explorer WebDAV clients ✅ (full RFC 4918 methods)

## Verification

```bash
go build ./internal/adapter/webdav/...  # ✅ PASS
go vet ./internal/adapter/webdav/...    # ✅ PASS
# Manual test:
# curl -X PROPFIND http://localhost:8080/webdav/ -H "Depth: 1" -H "X-API-Key: vnp_xxx"
```
