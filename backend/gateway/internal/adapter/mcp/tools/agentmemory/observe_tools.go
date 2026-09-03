package agentmemory

import "github.com/vnp-community/vnp-memory/gateway/internal/adapter/mcp"

func ObserveTools(deps *Dependencies) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "memory_observe",
			Description: "Capture a hook event from agent activity. Called automatically by agent plugins.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"session_id":  {Type: "string"},
					"hook_type":   {Type: "string", Enum: []string{"session_start", "prompt_submit", "pre_tool_use", "post_tool_use", "post_tool_failure", "session_end", "task_completed"}},
					"tool_name":   {Type: "string"},
					"tool_input":  {Type: "object"},
					"tool_output": {Type: "object"},
					"agent_id":    {Type: "string"},
				},
				Required: []string{"session_id", "hook_type"},
			},
			Handler: proxyObserve(deps),
		},
		{
			Name:        "memory_session_start",
			Description: "Start a new observation session. Returns session_id to use in subsequent observe calls.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"project":  {Type: "string"},
					"cwd":      {Type: "string"},
					"model":    {Type: "string"},
					"agent_id": {Type: "string"},
				},
				Required: []string{"project"},
			},
			Handler: proxySessionStart(deps),
		},
		{
			Name:        "memory_session_end",
			Description: "End current session. Triggers session summarization and memory consolidation.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"session_id": {Type: "string"},
				},
				Required: []string{"session_id"},
			},
			Handler: proxySessionEnd(deps),
		},
		{
			Name:        "memory_import_transcript",
			Description: "Import a text transcript as a batch of observations for the current session.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"session_id": {Type: "string"},
					"transcript": {Type: "string"},
					"format":     {Type: "string", Enum: []string{"plain", "markdown", "json"}},
				},
				Required: []string{"session_id", "transcript"},
			},
			Handler: proxyImportTranscript(deps),
		},
		{
			Name:        "memory_stream_subscribe",
			Description: "Get SSE stream URL for real-time session events. Returns URL to subscribe to.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"session_id": {Type: "string", Description: "Filter to specific session (optional)"},
				},
			},
			Handler: proxyStreamSubscribe(deps),
		},
		{
			Name:        "memory_retention_score",
			Description: "Get retention score for a memory (strength × recency × frequency). Returns recommendation: keep/review/evict.",
			InputSchema: mcp.Schema{
				Type:       "object",
				Properties: map[string]mcp.Property{"memory_id": {Type: "string"}},
				Required:   []string{"memory_id"},
			},
			Handler: proxyRetentionScore(deps),
		},
	}
}
