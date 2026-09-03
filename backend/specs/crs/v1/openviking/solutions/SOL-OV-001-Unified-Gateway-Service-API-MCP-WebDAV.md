# SOL-OV-001 — Solution: Unified Gateway Service (API, MCP, WebDAV)

| Field | Value |
|---|---|
| **Solution ID** | SOL-OV-001 |
| **CR** | CR-OV-001 |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/ov-gateway` |

---

## 1. Giải pháp

OpenViking gateway unifies REST + MCP + WebDAV protocols on single service.

### `services/ov-gateway/internal/adapter/handler/router.go` [MODIFY]

```go
r.Post("/v1/ov/files/upload", h.UploadFile)
r.Get("/v1/ov/files/{id}", h.GetFile)
r.Get("/v1/ov/search", h.Search)
r.Get("/v1/ov/grep", h.Grep)
r.Get("/v1/ov/list", h.ListDir)
```

WebDAV mounts file namespace at `/webdav/{tenant_id}/`.

## 2. Acceptance Criteria

- [ ] REST API handles file CRUD
- [ ] MCP tools: ov_read_file, ov_write_file, ov_grep, ov_search, ov_list_dir
- [ ] WebDAV compatible with OS-level mounts

