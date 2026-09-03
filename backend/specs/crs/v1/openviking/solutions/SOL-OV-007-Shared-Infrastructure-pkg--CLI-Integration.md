# SOL-OV-007 — Solution: Shared Infrastructure (pkg/) & CLI Integration

| Field | Value |
|---|---|
| **Solution ID** | SOL-OV-007 |
| **CR** | CR-OV-007 |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `shared/pkg` |

---

## 1. Giải pháp

Shared packages: VikingFS interface, encryption helpers, CLI tool for dev operations.

```go
// shared/pkg/vikingfs/interface.go [NEW]
type VikingFS interface {
    Read(ctx context.Context, tenantID, path string) ([]byte, error)
    Write(ctx context.Context, tenantID, path string, data []byte) error
    Delete(ctx context.Context, tenantID, path string) error
    List(ctx context.Context, tenantID, dir string) ([]FileInfo, error)
    Search(ctx context.Context, tenantID, query string) ([]SearchResult, error)
}
```

CLI: `vnp-ov grep <pattern> <path>`, `vnp-ov read <path>`, `vnp-ov write <path> <content>`

## 2. Acceptance Criteria

- [ ] VikingFS interface implemented by ov-fs adapter
- [ ] CLI tool usable for dev/debugging
- [ ] All OpenViking services use shared interface

