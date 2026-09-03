---
id: MERGE-P4-T1
title: "Cleanup: Cập nhật go.work — Xóa 40 module entries cũ"
phase: P4
service: workspace
priority: P2
status: Done
estimated: 1h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P1-T1, MERGE-P1-T2, MERGE-P1-T3, MERGE-P1-T4, MERGE-P2-T1, MERGE-P2-T2, MERGE-P2-T3, MERGE-P2-T4, MERGE-P3-T1, MERGE-P3-T2]
---

## Mục Tiêu

Cập nhật `go.work` để phản ánh architecture mới: 8 services + gateway + pkg/*. Xóa 40 module entries của các services đã được merged.

## Thay Đổi go.work

```diff
 go 1.25.0

 use (
     ./gateway
     ./pkg/forward
     ./pkg/telemetry
     ./pkg/tenant
     ./pkg/vectorstore

-    ./apps/OpenViking
-    ./apps/cognee
-    ./apps/graphiti
-    ./apps/memobase
-    ./apps/memory
-    ./apps/supermemory
-    ./apps/zep
-
-    ./services/cognee-cognify
-    ./services/cognee-ingestion
-    ./services/cognee-search
-    ./services/graphiti-ingestion
-    ./services/graphiti-knowledge
-    ./services/graphiti-search
-    ./services/graphiti-store
-    ./services/memobase-context
-    ./services/memobase-engine
-    ./services/memobase-ingestion
-    ./services/ov-admin
-    ./services/ov-crypto
-    ./services/ov-fs
-    ./services/ov-resource
-    ./services/ov-search
-    ./services/ov-session
-    ./services/sm-analytics
-    ./services/sm-auth
-    ./services/sm-connector
-    ./services/sm-document
-    ./services/sm-engine
-    ./services/sm-mcp
-    ./services/sm-memory
-    ./services/sm-profile
-    ./services/sm-project
-    ./services/sm-search
-    ./services/vnp-admin
-    ./services/vnp-dashboard
-    ./services/vnp-event
-    ./services/vnp-infra
-    ./services/vnp-observability
-    ./services/vnp-pipelines
-    ./services/vnp-search-hub
-    ./services/zep-admin
-    ./services/zep-graph
-    ./services/zep-memory
-    ./services/zep-search
-    ./services/zep-thread
-    ./services/zep-user
-    ./services/ba-knowledge-service
-    ./services/ba-knowledge-worker
-    ./services/cognee-pipeline
-    ./services/graphiti-pipeline
-    ./services/memobase-pipeline
-    ./services/zep-core
-    ./services/ov-storage
-    ./services/sm-engine
-
+    # 8 Consolidated Services
     ./services/vnp-platform
-    ./services/vnp-search-hub
+    ./services/kg-service
+    ./services/memory-service
+    ./services/storage-service
+    ./services/search-service
+    ./services/pipeline-service
+    ./services/obs-service

+    # External SDKs (used as library dependencies, NOT deployed)
     ./services/zep-go
 )
```

### go.work Sau Khi Cập Nhật

```
go 1.25.0

use (
    # Gateway (fully implemented)
    ./gateway

    # Shared packages
    ./pkg/forward
    ./pkg/telemetry
    ./pkg/tenant
    ./pkg/vectorstore

    # 8 Backend Services
    ./services/vnp-platform      # Auth + Admin + Platform
    ./services/kg-service        # Knowledge Graph
    ./services/memory-service    # Memory (Memobase + Zep + SM)
    ./services/storage-service   # Files + Crypto + Resources
    ./services/search-service    # Cross-engine Search
    ./services/pipeline-service  # Async Processing
    ./services/obs-service       # Observability + Infra

    # External SDK (dependency, not deployed service)
    ./services/zep-go
)
```

## Verification

```bash
# After updating go.work, verify workspace resolves correctly
go work sync
go build ./...
go vet ./...
```

## Handling Old Service Directories

Các service directories cũ **KHÔNG BỊ XÓA** trong task này. Chúng sẽ:
1. Không còn trong `go.work` → không compile với `./...`
2. Vẫn tồn tại trên disk như historical reference
3. Sẽ được archived trong MERGE-P4-T5

## Acceptance Criteria

- [ ] `go.work` chỉ có 8 service modules + gateway + pkg/* + zep-go
- [ ] `go work sync` passes
- [ ] `go build ./...` builds tất cả 8 services + gateway thành công
- [ ] `go vet ./...` không có errors
- [ ] `go test ./...` passes (unit tests)

## Ghi Chú

- Cần chạy `go work sync` sau khi cập nhật để đồng bộ go.work.sum
- Nếu có circular dependencies → resolve trước khi merge
