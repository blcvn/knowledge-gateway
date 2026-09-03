package agentmemory

import "github.com/vnp-community/vnp-memory/gateway/internal/adapter/mcp"

func MemoryCoreTools(deps *Dependencies) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "memory_smart_search",
			Description: "Search memories and observations using hybrid BM25+vector search. Returns semantically relevant results even without exact keyword match.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"query":         {Type: "string", Description: "Natural language search query"},
					"project":       {Type: "string", Description: "Filter by project (optional)"},
					"limit":         {Type: "integer", Default: 10, Description: "Max results (1-50)"},
					"bm25_weight":   {Type: "number", Default: 0.4},
					"vector_weight": {Type: "number", Default: 0.6},
				},
				Required: []string{"query"},
			},
			Handler: proxySmartSearch(deps),
		},
		{
			Name:        "memory_save",
			Description: "Save a long-term memory that should persist across coding sessions. Use for patterns, architectural decisions, bug fixes, preferences, and workflows.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"type":         {Type: "string", Enum: []string{"pattern", "preference", "architecture", "bug", "workflow", "fact"}},
					"title":        {Type: "string", Description: "One-line summary (max 80 chars)"},
					"content":      {Type: "string", Description: "Full memory content"},
					"concepts":     {Type: "array", Items: &mcp.Property{Type: "string"}, Description: "Key concepts for retrieval (3-10 words)"},
					"files":        {Type: "array", Items: &mcp.Property{Type: "string"}},
					"project":      {Type: "string"},
					"forget_after": {Type: "string", Description: "ISO 8601 timestamp for TTL"},
				},
				Required: []string{"type", "title", "content", "concepts"},
			},
			Handler: proxyMemorySave(deps),
		},
		{
			Name:        "memory_context",
			Description: "Build a context block from relevant memories and session history, optimized for a token budget. Call at session start to inject useful context.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"query":        {Type: "string", Description: "Current task (for semantic retrieval)"},
					"project":      {Type: "string"},
					"session_id":   {Type: "string"},
					"token_budget": {Type: "integer", Default: 2000},
				},
				Required: []string{"project"},
			},
			Handler: proxyBuildContext(deps),
		},
		{
			Name:        "memory_forget",
			Description: "Permanently delete a memory with cascade (removes from all search indexes and graph). Creates audit trail.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"memory_id": {Type: "string"},
					"reason":    {Type: "string"},
				},
				Required: []string{"memory_id"},
			},
			Handler: proxyGovernanceDelete(deps),
		},
	}
}
