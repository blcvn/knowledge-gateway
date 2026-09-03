# SOL-OV-002 — Solution: Filesystem Service (VikingFS & Encryption)

| Field | Value |
|---|---|
| **Solution ID** | SOL-OV-002 |
| **CR** | CR-OV-002 |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/ov-fs` |

---

## 1. Giải pháp

VikingFS = virtual filesystem with per-file AES-256-GCM encryption, MinIO backend.

### `services/ov-fs/internal/usecase/fs.go` [MODIFY]

```go
func (u *FSUseCase) WriteFile(ctx context.Context, req *WriteRequest) (*FileHandle, error) {
    // 1. Generate DEK (Data Encryption Key)
    dek, _ := u.kms.GenerateDEK(ctx, req.TenantID, req.Path)
    
    // 2. Encrypt content with AES-256-GCM
    ciphertext, nonce, _ := encrypt(req.Content, dek)
    
    // 3. Store in MinIO with metadata
    key := fmt.Sprintf("%s/%s/%s", req.TenantID, req.UserID, req.Path)
    u.minio.PutObject(ctx, key, ciphertext, minio.PutObjectOptions{
        UserMetadata: map[string]string{
            "nonce": base64.Encode(nonce),
            "kms_key_id": dek.KeyID,
        },
    })
    
    return &FileHandle{Path: req.Path, Size: len(ciphertext)}, nil
}
```

## 2. Acceptance Criteria

- [ ] AES-256-GCM encryption per file
- [ ] DEK stored in KMS (local/Vault/Cloud)
- [ ] Tenant isolation: tenant_id in MinIO key path

