# TASK-AM-013 — Context Injection (Session Start Auto-Inject)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-013 |
| **Wave** | 2 (Integration) |
| **Component** | `services/observe-service/` + `apps/memory/configs/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-008 §2.5 |
| **Priority** | High |
| **Depends On** | TASK-AM-002, TASK-AM-008 |
| **Estimated** | 3h |

---

## Context

Auto-inject context khi `hook_type=session_start` và `AGENTMEMORY_INJECT_CONTEXT=true`. Pipeline gọi `observe-search.BuildContext` → trả về `injected_context` trong response.

---

## Target Files

| Action | File Path |
|--------|-----------|
| MODIFY | `services/observe-service/internal/observe/pipeline.go` |
| MODIFY | `apps/memory/configs/config.yaml` |
| CREATE | `services/vnp-platform/internal/usecase/admin/plugin.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### MODIFY `observe/pipeline.go` — Add context injection step

```go
// Thêm vào struct PipelineConfig
type PipelineConfig struct {
    MaxObsPerSession int
    DedupTTL         time.Duration
    InjectContext    bool          // NEW: AGENTMEMORY_INJECT_CONTEXT env
    TokenBudget      int           // NEW: default 2000
    AgentScope       string        // NEW: "shared" | "isolated"
}

// Thêm search client field vào Pipeline
type Pipeline struct {
    // ... existing fields ...
    searchClient port.ISearchClient  // [NEW] gRPC client to am-search
}

// MODIFY Execute() — thêm context injection sau step 14 khi session_start
func (p *Pipeline) Execute(ctx context.Context, req ObserveRequest) (*ObserveResponse, error) {
    // ... existing 14 steps ...

    resp := &ObserveResponse{ObservationID: raw.ID, Compressed: compressed}

    // [NEW] Step 15: Context injection on session_start
    if domain.HookType(req.HookType) == domain.HookSessionStart && p.config.InjectContext {
        ctxResp, err := p.searchClient.BuildContext(ctx, SearchContextRequest{
            TenantID:    req.TenantID,
            Project:     req.Project,
            SessionID:   req.SessionID,
            TokenBudget: p.config.TokenBudget,
        })
        if err == nil {
            resp.InjectedContext = ctxResp.Formatted
            resp.ContextTokens   = ctxResp.TotalTokens
        }
        // Non-fatal: log if context injection fails, don't block observation
        if err != nil {
            log.Warn("context injection failed", "session_id", req.SessionID, "err", err)
        }
    }

    return resp, nil
}
```

### MODIFY `apps/memory/configs/config.yaml`

```yaml
# thêm section mới
agentmemory:
  inject_context: false    # set to true to enable auto-inject
  token_budget: 2000
  agent_scope: "shared"    # "shared" | "isolated"
  
search:
  embedding_provider: "none"   # "none" | "bifrost"
  embedding_model: "text-embedding-3-small"
  embed_dims: 384
  data_dir: "${HOME}/.agentmemory/indexes"
```

### `services/vnp-platform/internal/usecase/admin/plugin.go`

```go
package admin

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "strings"
)

var pluginConfigs = map[string]string{
    "claude-code": claudeCodeConfig,
    "codex":       codexConfig,
    "opencode":    opencodeConfig,
}

var agentConfigPaths = map[string]string{
    "claude-code": "$HOME/.claude/settings.json",
    "codex":       "$HOME/.codex/settings.json",
    "opencode":    "$HOME/.opencode/settings.json",
}

const claudeCodeConfig = `{
  "mcpServers": {
    "agentmemory": {
      "type": "http",
      "url": "http://localhost:8082",
      "headers": {
        "X-Project": "{{project}}",
        "X-Agent-ID": "claude-code"
      }
    }
  },
  "hooks": {
    "PreToolUse": {
      "url": "http://localhost:8080/v1/observe",
      "method": "POST"
    },
    "PostToolUse": {
      "url": "http://localhost:8080/v1/observe",
      "method": "POST"
    },
    "Stop": {
      "url": "http://localhost:8080/v1/observe/session/end",
      "method": "POST"
    }
  }
}`

const codexConfig = `{
  "mcpServers": {
    "agentmemory": {
      "type": "http",
      "url": "http://localhost:8082",
      "headers": {
        "X-Project": "{{project}}",
        "X-Agent-ID": "codex"
      }
    }
  }
}`

const opencodeConfig = `{
  "mcpServers": {
    "agentmemory": {
      "command": "npx",
      "args": ["-y", "@agentmemory/mcp-client", "--url", "http://localhost:8082"]
    }
  }
}`

type PluginUseCase struct {
    serverURL string
}

func NewPluginUseCase(serverURL string) *PluginUseCase {
    return &PluginUseCase{serverURL: serverURL}
}

func (uc *PluginUseCase) GetConfig(ctx context.Context, agentType string) (string, error) {
    cfg, ok := pluginConfigs[agentType]
    if !ok { return "", fmt.Errorf("unknown agent type: %s", agentType) }

    project := extractProject(ctx)
    cfg = strings.ReplaceAll(cfg, "{{project}}", project)
    cfg = strings.ReplaceAll(cfg, "http://localhost:8082", uc.serverURL)
    return cfg, nil
}

type InstallPluginRequest struct {
    AgentType string `json:"agent_type"`
    Project   string `json:"project"`
}

func (uc *PluginUseCase) Install(ctx context.Context, req InstallPluginRequest) error {
    cfg, err := uc.GetConfig(ctx, req.AgentType)
    if err != nil { return err }

    path := os.ExpandEnv(agentConfigPaths[req.AgentType])
    if path == "" { return fmt.Errorf("no config path for %s", req.AgentType) }

    // Merge with existing config
    var existing map[string]any
    if data, err := os.ReadFile(path); err == nil {
        json.Unmarshal(data, &existing)
    }
    if existing == nil { existing = make(map[string]any) }

    var newCfg map[string]any
    json.Unmarshal([]byte(cfg), &newCfg)

    // Merge mcpServers
    if ms, ok := newCfg["mcpServers"]; ok { existing["mcpServers"] = ms }
    if hooks, ok := newCfg["hooks"]; ok { existing["hooks"] = hooks }

    merged, _ := json.MarshalIndent(existing, "", "  ")
    return os.WriteFile(path, merged, 0644)
}

func extractProject(ctx context.Context) string {
    if v := ctx.Value("project"); v != nil { return v.(string) }
    return "default"
}
```

### MODIFY `gateway/router.go` — Plugin routes

```go
// Agent plugin config routes (already in SOL-007 §2.8)
r.Get("/v1/admin/plugin/claude-code", h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Get("/v1/admin/plugin/codex",       h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Get("/v1/admin/plugin/opencode",    h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Post("/v1/admin/plugin/install",    h.ForwardTo("vnp-platform", "AdminService/InstallPlugin"))

// SSE stream endpoint
r.Get("/v1/stream", observeSSEHandler.ServeSSE)
```

---

## Verification

**Context injection test:**
```bash
# 1. Set env: AGENTMEMORY_INJECT_CONTEXT=true
# 2. POST /v1/observe with hook_type=session_start
# 3. Response: {observation_id, injected_context: "...", context_tokens: N}

# Plugin config test:
# GET /v1/admin/plugin/claude-code → valid JSON with mcpServers config
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `INJECT_CONTEXT=true` + `session_start` hook → `injected_context` in response | ✅ |
| `INJECT_CONTEXT=false` (default) → `injected_context = ""` | ✅ |
| Context injection failure → non-fatal, observation still saved | ✅ |
| `GET /admin/plugin/claude-code` → valid JSON with mcpServers | ✅ |
| `POST /admin/plugin/install?agent_type=claude-code` → writes config file | ✅ |
| SSE `GET /v1/stream` → server-sent events with heartbeat | ✅ |
