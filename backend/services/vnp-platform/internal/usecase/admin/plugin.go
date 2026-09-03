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
	if !ok {
		return "", fmt.Errorf("unknown agent type: %s", agentType)
	}

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
	if err != nil {
		return err
	}

	path := os.ExpandEnv(agentConfigPaths[req.AgentType])
	if path == "" {
		return fmt.Errorf("no config path for %s", req.AgentType)
	}

	// Merge with existing config
	var existing map[string]any
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &existing)
	}
	if existing == nil {
		existing = make(map[string]any)
	}

	var newCfg map[string]any
	json.Unmarshal([]byte(cfg), &newCfg)

	// Merge mcpServers
	if ms, ok := newCfg["mcpServers"]; ok {
		existing["mcpServers"] = ms
	}
	if hooks, ok := newCfg["hooks"]; ok {
		existing["hooks"] = hooks
	}

	merged, _ := json.MarshalIndent(existing, "", "  ")
	return os.WriteFile(path, merged, 0644)
}

func extractProject(ctx context.Context) string {
	if v := ctx.Value("project"); v != nil {
		return v.(string)
	}
	return "default"
}
