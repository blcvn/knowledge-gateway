# SOL-OV-004 — Solution: Session Service (Two-Phase Commit & Working Memory v2)

| Field | Value |
|---|---|
| **Solution ID** | SOL-OV-004 |
| **CR** | CR-OV-004 |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |
| **Component** | `services/ov-session` |

---

## 1. Giải pháp

Two-phase commit for concurrent file writes + Working Memory (L0) for active session context.

### Two-phase commit

```go
func (u *SessionUseCase) PrepareWrite(ctx context.Context, req *WriteRequest) (string, error) {
    lockID, _ := u.leaseRepo.AcquireLock(ctx, req.Path, req.SessionID)
    // Stage changes in temp location
    tempPath := fmt.Sprintf("/tmp/%s/%s", req.SessionID, req.Path)
    u.fs.Write(ctx, tempPath, req.Content)
    return lockID, nil
}

func (u *SessionUseCase) CommitWrite(ctx context.Context, lockID string) error {
    temp, _ := u.leaseRepo.GetLockedPaths(ctx, lockID)
    for _, t := range temp {
        u.fs.Move(ctx, t.TempPath, t.FinalPath)
    }
    u.leaseRepo.ReleaseLock(ctx, lockID)
    return nil
}
```

## 2. Acceptance Criteria

- [ ] Concurrent writes to same file serialized via lock
- [ ] Uncommitted writes auto-rollback after 5min TTL
- [ ] Working memory (L0) stores session-scoped context

