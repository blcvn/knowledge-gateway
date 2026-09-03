package agentmemory

import "github.com/vnp-community/vnp-memory/gateway/internal/adapter/mcp"

func OrchestrationTools(deps *Dependencies) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "memory_action_create",
			Description: "Create a task action for multi-agent coordination. Track work items with priority, dependencies, and status.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"title":          {Type: "string"},
					"description":    {Type: "string"},
					"priority":       {Type: "integer", Default: 50, Description: "0-100"},
					"agent_id":       {Type: "string"},
					"project":        {Type: "string"},
					"requires":       {Type: "array", Items: &mcp.Property{Type: "string"}},
					"conflicts_with": {Type: "array", Items: &mcp.Property{Type: "string"}},
					"tags":           {Type: "array", Items: &mcp.Property{Type: "string"}},
				},
				Required: []string{"title"},
			},
			Handler: proxyCreateAction(deps),
		},
		{
			Name:        "memory_action_list",
			Description: "List actions by status for the current project.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"status":  {Type: "string", Enum: []string{"pending", "active", "blocked", "done", "cancelled", "failed"}, Description: "Filter by status"},
					"project": {Type: "string"},
					"limit":   {Type: "integer", Default: 20},
				},
			},
			Handler: proxyListActions(deps),
		},
		{
			Name:        "memory_action_update",
			Description: "Update action status or result. Use to mark actions as done/failed/blocked.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"action_id": {Type: "string"},
					"status":    {Type: "string", Enum: []string{"active", "done", "blocked", "cancelled", "failed"}},
					"result":    {Type: "string", Description: "Outcome description"},
				},
				Required: []string{"action_id", "status"},
			},
			Handler: proxyUpdateAction(deps),
		},
		{
			Name:        "memory_lease_acquire",
			Description: "Acquire a distributed lease to prevent concurrent writes to shared state. Lease expires automatically after TTL.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"action_id": {Type: "string"},
					"agent_id":  {Type: "string"},
					"ttl_secs":  {Type: "integer", Default: 300},
				},
				Required: []string{"action_id", "agent_id"},
			},
			Handler: proxyAcquireLease(deps),
		},
		{
			Name:        "memory_lease_release",
			Description: "Release a previously acquired lease.",
			InputSchema: mcp.Schema{
				Type:       "object",
				Properties: map[string]mcp.Property{"lease_id": {Type: "string"}},
				Required:   []string{"lease_id"},
			},
			Handler: proxyReleaseLease(deps),
		},
		{
			Name:        "memory_checkpoint_create",
			Description: "Create a human-approval gate. Agent pauses until checkpoint is approved or rejected.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"title":        {Type: "string"},
					"description":  {Type: "string"},
					"action_id":    {Type: "string"},
					"expire_hours": {Type: "integer", Default: 24},
				},
				Required: []string{"title"},
			},
			Handler: proxyCreateCheckpoint(deps),
		},
		{
			Name:        "memory_checkpoint_approve",
			Description: "Approve a pending checkpoint.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"checkpoint_id": {Type: "string"},
					"approved_by":   {Type: "string"},
				},
				Required: []string{"checkpoint_id"},
			},
			Handler: proxyApproveCheckpoint(deps),
		},
		{
			Name:        "memory_sketch_create",
			Description: "Create a sketch to group related actions into a coherent work unit.",
			InputSchema: mcp.Schema{
				Type: "object",
				Properties: map[string]mcp.Property{
					"title":        {Type: "string"},
					"project":      {Type: "string"},
					"expire_hours": {Type: "integer", Default: 72},
				},
				Required: []string{"title"},
			},
			Handler: proxyCreateSketch(deps),
		},
		{
			Name:        "memory_sketch_promote",
			Description: "Promote a sketch to a Crystal — a synthesized narrative of completed actions. Uses LLM to generate insights.",
			InputSchema: mcp.Schema{
				Type:       "object",
				Properties: map[string]mcp.Property{"sketch_id": {Type: "string"}},
				Required:   []string{"sketch_id"},
			},
			Handler: proxyPromoteSketch(deps),
		},
		{
			Name:        "memory_crystal_get",
			Description: "Get a Crystal (promoted sketch) with its narrative, key outcomes, and lessons.",
			InputSchema: mcp.Schema{
				Type:       "object",
				Properties: map[string]mcp.Property{"crystal_id": {Type: "string"}},
				Required:   []string{"crystal_id"},
			},
			Handler: proxyCrystalGet(deps),
		},
	}
}
