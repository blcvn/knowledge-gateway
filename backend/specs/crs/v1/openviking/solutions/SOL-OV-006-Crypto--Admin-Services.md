# SOL-OV-006 — Solution: Crypto & Admin Services

| Field | Value |
|---|---|
| **Solution ID** | SOL-OV-006 |
| **CR** | CR-OV-006 |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/ov-crypto` |

---

## 1. Giải pháp

KMS operations: DEK generation, key rotation, encryption policy management.

```go
// services/ov-crypto/internal/usecase/kms.go [NEW]
func (u *KMSUseCase) RotateKeys(ctx context.Context, tenantID string) (*RotationReport, error) {
    keys, _ := u.keyRepo.GetActive(ctx, tenantID)
    for _, key := range keys {
        newKey, _ := u.kms.GenerateDEK(ctx, tenantID, "rotation")
        u.reencryptFiles(ctx, tenantID, key, newKey)
        u.keyRepo.Archive(ctx, key.ID)
        u.keyRepo.Activate(ctx, newKey.ID)
    }
    return &RotationReport{Rotated: len(keys)}, nil
}
```

## 2. Acceptance Criteria

- [ ] Key rotation without data loss
- [ ] Old keys archived (not deleted) for audit
- [ ] Admin: GET /v1/ov/admin/key-status per tenant

