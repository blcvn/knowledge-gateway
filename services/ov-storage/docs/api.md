# ov-storage — API Reference

> **Service**: `ov-storage`  
> **gRPC Port**: 9051

---

## gRPC Service Definitions

`ov-storage` is a unified binary that exposes three distinct gRPC services on the same port, reflecting its consolidated nature.

### 1. `OvFsService` (File System)
```protobuf
service OvFsService {
  rpc ReadFile(ReadFileRequest) returns (ReadFileResponse);
  rpc WriteFile(WriteFileRequest) returns (WriteFileResponse);
  rpc DeleteFile(DeleteFileRequest) returns (google.protobuf.Empty);
  rpc MkDir(MkDirRequest) returns (google.protobuf.Empty);
  rpc ListDir(ListDirRequest) returns (ListDirResponse);
  rpc Tree(TreeRequest) returns (TreeResponse);
  rpc Grep(GrepRequest) returns (GrepResponse);
  rpc Glob(GlobRequest) returns (GlobResponse);
  rpc Move(MoveRequest) returns (google.protobuf.Empty);
  rpc GetRelations(GetRelationsRequest) returns (RelationsResponse);
}
```

### 2. `OvCryptoService` (Cryptography)
```protobuf
service OvCryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);
  rpc RotateKey(RotateKeyRequest) returns (RotateKeyResponse);
  rpc GetKeyStatus(KeyStatusRequest) returns (KeyStatus);
}
```

### 3. `OvResourceService` (Resource Ingestion)
```protobuf
service OvResourceService {
  rpc Ingest(IngestRequest) returns (IngestResponse);
  rpc Parse(ParseRequest) returns (ParseResponse);
  rpc Watch(WatchRequest) returns (stream WatchEvent);
  rpc Refresh(RefreshRequest) returns (RefreshResponse);
}
```

## Authentication

All endpoints require valid JWT or API key via Gateway. Tenant isolation enforced via `x-tenant-id` gRPC metadata.
