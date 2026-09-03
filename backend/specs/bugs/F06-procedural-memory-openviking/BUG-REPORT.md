# Bug Report — F06: Procedural Memory (OpenViking)

> Feature: VikingFS file operations, grep, search, session management, resource ingest
> Luồng: `apps/memory → gateway/adapter/handler/services.go (OpenVikingHandler) → storage-service / search-service`

---

## BUG-F06-001: `Search` và `CreateConnection` Forward Tới Sai Service

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/services.go:141-143`

**Mô tả:**  
`OpenVikingHandler.Search` forward tới `search-service` trong khi tất cả các operations khác đều dùng `storage-service`. Theo feature spec, OpenViking search nên đi qua `ov-search` service, không phải generic `search-service`.

```go
func (h *OpenVikingHandler) Search(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "search-service", h.logger)(w, r)  // Nên là "ov-search"
}
```

**Impact:**  
- `POST /v1/ov/search` có thể không trả về OpenViking-specific search results.

---

## BUG-F06-002: Thiếu Services `ov-*` Implementations

**Severity:** HIGH  
**File:** `services/ov-*` directories

**Mô tả:**  
Services OpenViking (`ov-admin`, `ov-crypto`, `ov-fs`, `ov-resource`, `ov-search`, `ov-session`, `ov-storage`) tồn tại như thư mục nhưng không có implementation code.

**Impact:**  
- Tất cả OpenViking/Procedural Memory operations sẽ fail với connection refused.
- `storage-service` và `search-service` được dùng làm fallback nhưng có thể không implement VikingFS protocol.

---

## BUG-F06-003: WebDAV Proxy Được Khởi Tạo Nhưng Không Được Mount

**Severity:** MEDIUM  
**File:** `gateway/cmd/main.go:263-264`

**Mô tả:**  
WebDAV proxy được tạo (`webdav.NewProxy(registry, logger)`) nhưng bị gán vào blank identifier `_ = webdavProxy`. Comment nói "Mounted at /webdav in router" nhưng không có mount nào xảy ra.

```go
webdavProxy := webdav.NewProxy(registry, logger)
_ = webdavProxy // Mounted at /webdav in router
```

**Impact:**  
- WebDAV endpoint cho OpenViking không accessible.
- OpenViking file system access qua WebDAV bị broken.
