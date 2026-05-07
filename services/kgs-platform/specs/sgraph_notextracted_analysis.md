# S_GRAPH "Not extracted yet" — Root Cause Analysis

## Problem
User clicked "Generate S_GRAPH" but the KG Readiness Indicator still shows "Not extracted yet" for the Specs-UI layer.

## Investigation Trace

### ✅ Frontend dispatch chain verified
```
Frontend → POST /api/v1/projects/{id}/kg/generate { pipeline_id: "gen_sgraph" }
  → Backend HandleKGGenerate → DispatchKGGenJob
    → goroutine → callUIKSPipeline (fire & forget)
      → POST http://ui-knowledge-service:8080/v1/projects/{id}/pipelines/run
```

### ✅ All whitelists include gen_sgraph
| Location | Variable | Has `gen_sgraph`? |
|---|---|---|
| Backend `ai_job_service.go` | `validGenPipelines` | ✅ |
| UIKS `pipeline_service.go` | `validPipelineIDs` | ✅ |
| UIKS `biz.go` | `RegisterAllRunners` | ✅ |

### ✅ Code path verified
- `GenSGraphRunner.Run()` → `fetchProjectKG()` → 3-pass LLM → `buildSGraphBatch()` → `WriteBatchScoped()`
- Nodes written with labels like `S_USER_JOURNEY`, `S_SCREEN_SPEC` → matches `graphTypeToLabels["S_GRAPH"]`

## Root Causes Identified

### 🔴 Cause 1: UIKS service not rebuilt/redeployed
The running UIKS binary is likely the **old version** before the `pipeline1.go` fix from the previous session. The `GenSGraphRunner` was integrated into Pipeline1 but the standalone `gen_sgraph` dispatch requires the UIKS to have the updated binary.

> **Action: Rebuild & redeploy UIKS service**

### 🟡 Cause 2: Pipeline execution failure (silent)
The `callUIKSPipeline` dispatches in a goroutine. If it fails (LLM timeout, KGS write error, network timeout), the error is only logged — the user gets a "queued" response but no S_* nodes are written.

Common failure modes:
- UIKS `base_url: "http://ui-knowledge-service:8080"` → Docker hostname unreachable if services not in same network
- LLM 504 timeout during 3-pass generation
- KGS gRPC connection failure

### 🟢 Cause 3: Redis cache (60s TTL) — Fixed automatically
Inventory cached for 60s — if checked immediately after gen_sgraph completes, stale cache returns "not available". This resolves itself within 60s.

## Fixes Applied This Session

### Fix 1: UIKS `validPipelineIDs` sync
```diff
// pipeline_service.go
var validPipelineIDs = map[string]bool{
    "pipeline0": true, "pipeline1": true, "pipeline2": true,
    "gen_urd": true, "gen_brd": true, "gen_srs": true, "gen_tdd": true,
-   "gen_ux": true, "gen_sgraph": true, "gen_proto": true,
+   "gen_ux": true, "gen_sgraph": true, "gen_rgraph": true, "gen_proto": true,
+   // SOL-017: Supplement + Traceability pipelines
+   "gen_prd_supplement": true, "gen_brd_supplement": true,
+   "gen_srs_supplement": true, "gen_tdd_supplement": true,
+   "gen_traceability": true,
}
```

> [!IMPORTANT]
> This fix enables `gen_rgraph` and all supplement pipelines on the UIKS side. Without it, those pipelines would return HTTP 400.

## Next Steps

1. **Rebuild UIKS**:
   ```bash
   cd services/ui-knowledge-service
   go build -o /tmp/uiks ./cmd/server
   ```

2. **Redeploy UIKS** — copy binary to the deployment target and restart the container/process

3. **Invalidate Redis cache** after redeployment:
   ```bash
   redis-cli DEL "kg_inventory:<project_id>"
   ```

4. **Monitor UIKS logs** during next `gen_sgraph` run:
   ```
   gen_sgraph start source_doc=...
   phase1/verified total=X ux=Y entity=Z biz=W
   phase2/pass1_prompt len=...
   gen_sgraph complete nodes=N
   ```
