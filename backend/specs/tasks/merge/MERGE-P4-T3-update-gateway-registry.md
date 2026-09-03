---
id: MERGE-P4-T3
title: "Gateway: Cập nhật Service Registry — Map 48 → 8 service endpoints"
phase: P4
service: vnp-gateway
priority: P2
status: Done
estimated: 4h
created: 2026-06-11
updated: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P4-T1]
---

## Mục Tiêu

Cập nhật gateway để route tất cả requests đến 7 backend services thay vì 47 individual services.

> **⚠️ Quan trọng:** `pkg/` và `gateway/` code **KHÔNG thay đổi logic** — chỉ thay đổi **service name strings** trong `ForwardToService()` calls và `defaultServiceAddresses()` map. Không thêm bất kỳ file hay struct mới nào.

## Files Cần Thay Đổi

```
gateway/
├── infra/config/config.go       # Thay đổi defaultServiceAddresses() map
├── adapter/handler/services.go  # Thay đổi ForwardToService service name strings
└── adapter/handler/console.go   # Thay đổi ForwardToService service name strings
```

> **Không thay đổi:** `gateway/adapter/client/`, `gateway/domain/`, `gateway/usecase/`, `pkg/` — tất cả logic gRPC, auth, rate limiting, circuit breaker giữ nguyên.

---

## 1. Cập nhật `gateway/infra/config/config.go`

**Chỉ thay đổi hàm `defaultServiceAddresses()`** — toàn bộ cấu trúc `Config` struct giữ nguyên.

```diff
-// defaultServiceAddresses returns the standard gRPC addresses for all 35 services.
+// defaultServiceAddresses returns the standard gRPC addresses for all 7 consolidated services (SOL-003).
 func defaultServiceAddresses() map[string]string {
     return map[string]string{
-        "cognee-ingestion":  "cognee-ingestion:9011",
-        "cognee-cognify":    "cognee-cognify:9012",
-        "cognee-search":     "cognee-search:9013",
-        "graphiti-ingestion": "graphiti-ingestion:9021",
-        "graphiti-search":    "graphiti-search:9022",
-        "graphiti-knowledge": "graphiti-knowledge:9023",
-        "graphiti-store":     "graphiti-store:9024",
-        "memobase-ingestion": "memobase-ingestion:9031",
-        "memobase-engine":    "memobase-engine:9032",
-        "memobase-context":   "memobase-context:9033",
-        "vnp-event":          "vnp-event:9041",
-        "vnp-search-hub":     "vnp-search-hub:9042",
-        "vnp-admin":          "vnp-admin:9050",
-        "ov-fs":              "ov-fs:9051",
-        "ov-search":          "ov-search:9052",
-        "ov-session":         "ov-session:9053",
-        "ov-resource":        "ov-resource:9054",
-        "ov-crypto":          "ov-crypto:9055",
-        "ov-admin":           "ov-admin:9056",
-        "zep-user":           "zep-user:9061",
-        "zep-thread":         "zep-thread:9062",
-        "zep-memory":         "zep-memory:9063",
-        "zep-graph":          "zep-graph:9064",
-        "zep-search":         "zep-search:9065",
-        "zep-admin":          "zep-admin:9066",
-        "sm-document":        "sm-document:9071",
-        "sm-memory":          "sm-memory:9072",
-        "sm-search":          "sm-search:9073",
-        "sm-profile":         "sm-profile:9074",
-        "sm-connector":       "sm-connector:9075",
-        "sm-mcp":             "sm-mcp:9076",
-        "sm-auth":            "sm-auth:9077",
-        "sm-analytics":       "sm-analytics:9078",
-        "sm-project":         "sm-project:9079",
-        "vnp-dashboard":      "vnp-dashboard:9043",
-        "vnp-pipelines":      "vnp-pipelines:9044",
-        "vnp-infra":          "vnp-infra:9045",
-        "vnp-observability":  "vnp-observability:9046",
-        "sm-engine":          "sm-engine:9080",
-        "zep-core":           "zep-core:9067",
+        // SOL-003: 7 consolidated backend services
+        "vnp-platform":     "vnp-platform:9090",
+        "kg-service":       "kg-service:9090",
+        "memory-service":   "memory-service:9090",
+        "storage-service":  "storage-service:9090",
+        "search-service":   "search-service:9090",
+        "pipeline-service": "pipeline-service:9090",
+        "obs-service":      "obs-service:9090",
     }
 }
```

---

## 2. Cập nhật `gateway/adapter/handler/services.go`

Thay tất cả `ForwardToService` service name strings (không thay đổi logic):

```diff
 // CogneeHandler
 func (h *CogneeHandler) CreateDataset(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "cognee-ingestion", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *CogneeHandler) UploadData(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "cognee-ingestion", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *CogneeHandler) Cognify(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "cognee-cognify", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *CogneeHandler) Search(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "cognee-search", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 // GraphitiHandler
 func (h *GraphitiHandler) IngestEpisode(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "graphiti-ingestion", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphitiHandler) Search(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "graphiti-search", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphitiHandler) GetNode(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphitiHandler) GetEdge(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 // MemobaseHandler
 func (h *MemobaseHandler) InsertBlob(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "memobase-ingestion", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *MemobaseHandler) Flush(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "memobase-ingestion", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *MemobaseHandler) GetContext(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *MemobaseHandler) GetProfiles(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *MemobaseHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 // OpenVikingHandler
 func (h *OpenVikingHandler) ReadFile(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) WriteFile(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) Tree(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) Grep(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-fs", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) Search(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-search", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-session", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) AddMessage(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-session", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) CommitSession(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-session", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *OpenVikingHandler) Ingest(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "ov-resource", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 // ZepHandler
 func (h *ZepHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-user", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) GetUser(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-user", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-user", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) PutMemory(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-memory", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-memory", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) GraphSearch(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-search", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) SessionSearch(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-search", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) AddFact(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-graph", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ZepHandler) SetOntology(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "zep-graph", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 // SMHandler
 func (h *SMHandler) CreateDocument(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-document", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SMHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-document", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SMHandler) CreateMemory(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-memory", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SMHandler) Search(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-search", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *SMHandler) RAG(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-search", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *SMHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-profile", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SMHandler) CreateConnection(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-connector", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *SMHandler) SyncConnection(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-connector", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *SMHandler) CreateSpace(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "sm-project", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 // AdminHandler
 func (h *AdminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *AdminHandler) IssueAPIKey(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *AdminHandler) Health(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *AdminHandler) Metrics(w http.ResponseWriter, r *http.Request) {
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }
```

---

## 3. Cập nhật `gateway/adapter/handler/console.go`

File `console.go` có **nhiều handler hơn** được mô tả ban đầu. Cần thay tất cả:

```diff
 // DashboardHandler
 func (h *DashboardHandler) Health(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-dashboard", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *DashboardHandler) Metrics(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-dashboard", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *DashboardHandler) Throughput(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-dashboard", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *DashboardHandler) Heatmap(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-dashboard", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 // ExplorerHandler
 func (h *ExplorerHandler) Search(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-search-hub", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *ExplorerHandler) GetMemory(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-search-hub", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *ExplorerHandler) GetNeighbors(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-search-hub", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *ExplorerHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-memory", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 // GraphHandler
 func (h *GraphHandler) Subgraph(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphHandler) GetEntity(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphHandler) Timeline(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphHandler) GetOntology(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "cognee-search", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphHandler) UpdateOntology(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "cognee-search", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 func (h *GraphHandler) Query(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "graphiti-store", h.logger)(w, r)
+    ForwardToService(h.registry, "kg-service", h.logger)(w, r)
 }

 // ProfileHandler
 func (h *ProfileHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ProfileHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *ProfileHandler) GetContext(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ProfileHandler) GetBuffers(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "memobase-ingestion", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ProfileHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *ProfileHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 // AdaptiveHandler
 func (h *AdaptiveHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-memory", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *AdaptiveHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-memory", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *AdaptiveHandler) ListConnectors(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-connector", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *AdaptiveHandler) CreateConnector(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-connector", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *AdaptiveHandler) SyncConnector(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-connector", h.logger)(w, r)
+    ForwardToService(h.registry, "search-service", h.logger)(w, r)
 }

 func (h *AdaptiveHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-engine", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *AdaptiveHandler) GetForgetRules(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-engine", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *AdaptiveHandler) UpdateForgetRules(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "sm-engine", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 // DebuggerHandler
 func (h *DebuggerHandler) CreateTrace(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-search-hub", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *DebuggerHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *DebuggerHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 // SessionHandler
 func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "zep-core", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "zep-core", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SessionHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "zep-core", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SessionHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SessionHandler) GetWorkingMemory(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "ov-session", h.logger)(w, r)
+    ForwardToService(h.registry, "storage-service", h.logger)(w, r)
 }

 func (h *SessionHandler) GetUserSummary(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "memobase-context", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 func (h *SessionHandler) ListLiveSessions(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "zep-core", h.logger)(w, r)
+    ForwardToService(h.registry, "memory-service", h.logger)(w, r)
 }

 // GovernanceHandler
 func (h *GovernanceHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) SearchAudit(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-admin", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) GDPRForget(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 func (h *GovernanceHandler) GDPRForgetPreview(w http.ResponseWriter, r *http.Request) {
     if !requireSuperAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-event", h.logger)(w, r)
+    ForwardToService(h.registry, "vnp-platform", h.logger)(w, r)
 }

 // PipelineHandler
 func (h *PipelineHandler) Status(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-pipelines", h.logger)(w, r)
+    ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
 }

 func (h *PipelineHandler) GetEngine(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-pipelines", h.logger)(w, r)
+    ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
 }

 func (h *PipelineHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-pipelines", h.logger)(w, r)
+    ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
 }

 func (h *PipelineHandler) GetJob(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-pipelines", h.logger)(w, r)
+    ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
 }

 func (h *PipelineHandler) Queues(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-pipelines", h.logger)(w, r)
+    ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
 }

 func (h *PipelineHandler) Workers(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-pipelines", h.logger)(w, r)
+    ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
 }

 func (h *PipelineHandler) Templates(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-pipelines", h.logger)(w, r)
+    ForwardToService(h.registry, "pipeline-service", h.logger)(w, r)
 }

 // InfraHandler
 func (h *InfraHandler) Topology(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-infra", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *InfraHandler) ListServices(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-infra", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *InfraHandler) GetService(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-infra", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *InfraHandler) Databases(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-infra", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *InfraHandler) Resources(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-infra", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *InfraHandler) Deployments(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-infra", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 // ObservabilityHandler
 func (h *ObservabilityHandler) Metrics(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-observability", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *ObservabilityHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-observability", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *ObservabilityHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-observability", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *ObservabilityHandler) Errors(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-observability", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }

 func (h *ObservabilityHandler) Costs(w http.ResponseWriter, r *http.Request) {
     if !requireAdmin(w, r) { return }
-    ForwardToService(h.registry, "vnp-observability", h.logger)(w, r)
+    ForwardToService(h.registry, "obs-service", h.logger)(w, r)
 }
```

---

## Route-to-Service Mapping Table (Complete)

| Route Pattern | Handler | Service Mới |
|---------------|---------|-------------|
| POST /v1/auth/* | AuthHandler | vnp-platform |
| POST /v1/admin/* | AdminHandler | vnp-platform |
| GET /v1/admin/* | AdminHandler | vnp-platform |
| POST /v1/cognee/* | CogneeHandler | kg-service |
| POST /v1/graphiti/* | GraphitiHandler | kg-service |
| GET /v1/graphiti/* | GraphitiHandler | kg-service |
| POST /v1/memobase/*/blobs | MemobaseHandler | memory-service |
| POST /v1/memobase/*/flush | MemobaseHandler | memory-service |
| GET /v1/memobase/*/context | MemobaseHandler | memory-service |
| GET /v1/memobase/*/profiles | MemobaseHandler | memory-service |
| GET /v1/memobase/*/events | MemobaseHandler | vnp-platform |
| GET/PUT/DELETE /v1/ov/files/* | OpenVikingHandler | storage-service |
| GET /v1/ov/tree/* | OpenVikingHandler | storage-service |
| POST /v1/ov/grep | OpenVikingHandler | storage-service |
| POST /v1/ov/search | OpenVikingHandler | search-service |
| POST /v1/ov/sessions/* | OpenVikingHandler | storage-service |
| POST /v1/ov/resources/* | OpenVikingHandler | storage-service |
| POST /v1/zep/* | ZepHandler | memory-service |
| GET /v1/zep/* | ZepHandler | memory-service |
| PATCH /v1/zep/* | ZepHandler | memory-service |
| POST /v1/sm/memories | SMHandler | memory-service |
| POST /v1/sm/rag | SMHandler | search-service |
| POST /v1/sm/search | SMHandler | search-service |
| GET /v1/sm/profiles/* | SMHandler | memory-service |
| POST /v1/sm/documents | SMHandler | memory-service |
| GET /v1/sm/documents/* | SMHandler | memory-service |
| POST /v1/sm/connections/* | SMHandler | search-service |
| POST /v1/sm/projects/spaces | SMHandler | vnp-platform |
| GET /v1/console/dashboard/* | DashboardHandler | vnp-platform |
| POST /v1/console/memory/search | ExplorerHandler | search-service |
| GET /v1/console/memory/* | ExplorerHandler | search-service |
| POST /v1/console/graph/* | GraphHandler | kg-service |
| GET /v1/console/graph/* | GraphHandler | kg-service |
| GET /v1/console/profiles/* | ProfileHandler | memory-service |
| PUT /v1/console/profiles/* | ProfileHandler | memory-service |
| GET /v1/console/profiles/*/events | ProfileHandler | vnp-platform |
| GET /v1/console/profiles/*/buffers | ProfileHandler | memory-service |
| GET /v1/console/adaptive/memories* | AdaptiveHandler | search-service |
| GET/POST /v1/console/adaptive/connectors* | AdaptiveHandler | search-service |
| GET /v1/console/adaptive/analytics | AdaptiveHandler | obs-service |
| GET/PUT /v1/console/adaptive/forget-rules | AdaptiveHandler | obs-service |
| POST /v1/console/debugger/* | DebuggerHandler | obs-service |
| GET /v1/console/debugger/* | DebuggerHandler | obs-service |
| GET /v1/console/sessions* | SessionHandler | memory-service |
| GET /v1/console/sessions/*/working-memory | SessionHandler | storage-service |
| GET /v1/console/governance/* | GovernanceHandler | vnp-platform |
| POST /v1/console/governance/* | GovernanceHandler | vnp-platform |
| GET /v1/console/pipelines/* | PipelineHandler | pipeline-service |
| GET /v1/console/infra/* | InfraHandler | obs-service |
| GET /v1/console/observability/* | ObservabilityHandler | obs-service |

---

## Acceptance Criteria

- [ ] `go build ./gateway/...` passes (không thay đổi logic, chỉ string constants)
- [ ] `go test ./gateway/...` passes
- [ ] `gateway/infra/config/config.go` — `defaultServiceAddresses()` chỉ có 7 entries
- [ ] `gateway/adapter/handler/services.go` — không còn service name nào trong danh sách cũ
- [ ] `gateway/adapter/handler/console.go` — không còn service name nào trong danh sách cũ
- [ ] `curl -s http://localhost:8080/v1/admin/health` → reaches vnp-platform
- [ ] `curl -s http://localhost:8080/v1/graphiti/search` → reaches kg-service
- [ ] `curl -s http://localhost:8080/v1/memobase/users/test/context` → reaches memory-service
- [ ] `curl -s http://localhost:8080/v1/ov/files/test.txt` → reaches storage-service
- [ ] `curl -s http://localhost:8080/v1/sm/search` → reaches search-service
- [ ] `curl -s http://localhost:8080/v1/console/pipelines/status` → reaches pipeline-service
- [ ] `curl -s http://localhost:8080/v1/console/observability/metrics` → reaches obs-service
- [ ] Gateway circuit breaker works khi backend service down

## Ghi Chú

- **Gateway code không thay đổi** — chỉ thay đổi string literals (service names)
- `pkg/forward`, `pkg/telemetry`, `pkg/tenant`, `pkg/vectorstore` → **giữ nguyên hoàn toàn**
- `gateway/adapter/client/registry.go` → **giữ nguyên** (gRPC connection logic không đổi)
- Sau task này, tất cả request flows qua 7 backends thay vì 47
- Cần restart gateway sau khi cập nhật
