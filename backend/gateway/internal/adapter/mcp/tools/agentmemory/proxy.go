package agentmemory

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/vnp-community/vnp-memory/gateway/internal/adapter/mcp"

	memorypb "github.com/vnp-memory/api/proto/memory/v1"
	observepb "github.com/vnp-memory/api/proto/observe/v1"
	orchpb "github.com/vnp-memory/api/proto/orchestration/v1"
	searchpb "github.com/vnp-memory/api/proto/search/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── Memory Core Proxies ───────────────────────────────────────────────────

func proxySmartSearch(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &searchpb.SmartSearchRequest{
			Query:    getString(input, "query"),
			Project:  getString(input, "project"),
			Limit:    int32(getInt(input, "limit", 10)),
			TenantId: extractTenantID(ctx),
		}
		if bw, ok := input["bm25_weight"].(float64); ok {
			req.Weights = &searchpb.SearchWeights{Bm25: bw, Vector: getFloat(input, "vector_weight", 0.6)}
		}
		return deps.SearchClient.SmartSearch(ctx, req)
	}
}

func proxyMemorySave(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &memorypb.RememberAgentRequest{
			TenantId: extractTenantID(ctx),
			Project:  getString(input, "project"),
			Type:     getString(input, "type"),
			Title:    getString(input, "title"),
			Content:  getString(input, "content"),
			Concepts: toStringSlice(input["concepts"]),
			Files:    toStringSlice(input["files"]),
		}
		if fa, ok := input["forget_after"].(string); ok && fa != "" {
			t, _ := time.Parse(time.RFC3339, fa)
			req.ForgetAfter = timestamppb.New(t)
		}
		resp, err := deps.MemoryClient.RememberAgent(ctx, req)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"memory_id":  resp.MemoryId,
			"version":    resp.Version,
			"superseded": resp.Superseded,
		}, nil
	}
}

func proxyBuildContext(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &searchpb.ContextRequest{
			TenantId:    extractTenantID(ctx),
			Project:     getString(input, "project"),
			SessionId:   getString(input, "session_id"),
			Query:       getString(input, "query"),
			TokenBudget: int32(getInt(input, "token_budget", 2000)),
		}
		return deps.SearchClient.BuildContext(ctx, req)
	}
}

func proxyGovernanceDelete(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &memorypb.DeleteAgentMemoryRequest{
			MemoryId: getString(input, "memory_id"),
			TenantId: extractTenantID(ctx),
		}
		return deps.MemoryClient.DeleteAgentMemory(ctx, req)
	}
}

// ── Observe Proxies ───────────────────────────────────────────────────────

func proxyObserve(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &observepb.ObserveRequest{
			SessionId: getString(input, "session_id"),
			HookType:  getString(input, "hook_type"),
			ToolName:  getString(input, "tool_name"),
			AgentId:   getString(input, "agent_id"),
			TenantId:  extractTenantID(ctx),
		}
		if ti, ok := input["tool_input"]; ok {
			b, _ := json.Marshal(ti)
			req.ToolInput = b
		}
		if to, ok := input["tool_output"]; ok {
			b, _ := json.Marshal(to)
			req.ToolOutput = b
		}
		return deps.ObserveClient.Observe(ctx, req)
	}
}

func proxySessionStart(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &observepb.StartSessionRequest{
			TenantId: extractTenantID(ctx),
			Project:  getString(input, "project"),
			Cwd:      getString(input, "cwd"),
			Model:    getString(input, "model"),
			AgentId:  getString(input, "agent_id"),
		}
		resp, err := deps.ObserveClient.StartSession(ctx, req)
		if err != nil {
			return nil, err
		}
		return map[string]any{"session_id": resp.SessionId, "status": resp.Status}, nil
	}
}

func proxySessionEnd(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &observepb.EndSessionRequest{
			SessionId: getString(input, "session_id"),
			TenantId:  extractTenantID(ctx),
		}
		return deps.ObserveClient.EndSession(ctx, req)
	}
}

func proxyImportTranscript(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		// Import transcript as observation batch
		sessionID := getString(input, "session_id")
		transcript := getString(input, "transcript")
		lines := strings.Split(transcript, "\n")
		var results []string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			req := &observepb.ObserveRequest{
				SessionId: sessionID, HookType: "import",
				UserPrompt: line, TenantId: extractTenantID(ctx),
			}
			resp, err := deps.ObserveClient.Observe(ctx, req)
			if err == nil {
				results = append(results, resp.ObservationId)
			}
		}
		return map[string]any{"imported": len(results)}, nil
	}
}

func proxyStreamSubscribe(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		sessionFilter := getString(input, "session_id")
		baseURL := extractBaseURL(ctx)
		url := baseURL + "/v1/stream"
		if sessionFilter != "" {
			url += "?session_id=" + sessionFilter
		}
		return map[string]any{"sse_url": url, "method": "GET"}, nil
	}
}

func proxyRetentionScore(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &memorypb.GetRetentionScoreRequest{MemoryId: getString(input, "memory_id")}
		return deps.MemoryClient.GetRetentionScore(ctx, req)
	}
}

// ── Orchestration Proxies ─────────────────────────────────────────────────

func proxyCreateAction(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		req := &orchpb.CreateActionRequest{
			TenantId:      extractTenantID(ctx),
			Project:       getString(input, "project"),
			AgentId:       getString(input, "agent_id"),
			Title:         getString(input, "title"),
			Description:   getString(input, "description"),
			Priority:      int32(getInt(input, "priority", 50)),
			Requires:      toStringSlice(input["requires"]),
			ConflictsWith: toStringSlice(input["conflicts_with"]),
			Tags:          toStringSlice(input["tags"]),
		}
		resp, err := deps.OrchestrationClient.CreateAction(ctx, req)
		if err != nil {
			return nil, err
		}
		return map[string]any{"action_id": resp.ActionId}, nil
	}
}

func proxyListActions(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return deps.OrchestrationClient.ListActions(ctx, &orchpb.ListActionsRequest{
			TenantId: extractTenantID(ctx),
			Status:   getString(input, "status"),
			Limit:    int32(getInt(input, "limit", 20)),
		})
	}
}

func proxyUpdateAction(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return deps.OrchestrationClient.UpdateAction(ctx, &orchpb.UpdateActionRequest{
			ActionId: getString(input, "action_id"),
			Status:   getString(input, "status"),
			Result:   getString(input, "result"),
		})
	}
}

func proxyAcquireLease(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		resp, err := deps.OrchestrationClient.AcquireLease(ctx, &orchpb.AcquireLeaseRequest{
			ActionId: getString(input, "action_id"),
			AgentId:  getString(input, "agent_id"),
			TtlSecs:  int32(getInt(input, "ttl_secs", 300)),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"lease_id":          resp.LeaseId,
			"conflict":          resp.Conflict,
			"conflicting_agent": resp.ConflictingAgent,
		}, nil
	}
}

func proxyReleaseLease(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return deps.OrchestrationClient.ReleaseLease(ctx, &orchpb.ReleaseLeaseRequest{
			LeaseId: getString(input, "lease_id"),
		})
	}
}

func proxyCreateCheckpoint(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		resp, err := deps.OrchestrationClient.CreateCheckpoint(ctx, &orchpb.CreateCheckpointRequest{
			TenantId:    extractTenantID(ctx),
			Title:       getString(input, "title"),
			Description: getString(input, "description"),
			ActionId:    getString(input, "action_id"),
			ExpireSecs:  int32(getInt(input, "expire_hours", 24) * 3600),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"checkpoint_id": resp.CheckpointId}, nil
	}
}

func proxyApproveCheckpoint(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return deps.OrchestrationClient.ApproveCheckpoint(ctx, &orchpb.ApproveCheckpointRequest{
			CheckpointId: getString(input, "checkpoint_id"),
			ApprovedBy:   getString(input, "approved_by"),
		})
	}
}

func proxyCreateSketch(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		resp, err := deps.OrchestrationClient.CreateSketch(ctx, &orchpb.CreateSketchRequest{
			TenantId:    extractTenantID(ctx),
			Project:     getString(input, "project"),
			Title:       getString(input, "title"),
			ExpireHours: int32(getInt(input, "expire_hours", 72)),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"sketch_id": resp.SketchId}, nil
	}
}

func proxyPromoteSketch(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		resp, err := deps.OrchestrationClient.PromoteSketch(ctx, &orchpb.PromoteSketchRequest{
			SketchId: getString(input, "sketch_id"),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"crystal_id": resp.CrystalId}, nil
	}
}

func proxyCrystalGet(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return deps.OrchestrationClient.GetCrystal(ctx, &orchpb.GetCrystalRequest{
			CrystalId: getString(input, "crystal_id"),
		})
	}
}

// ── Signal Proxies ────────────────────────────────────────────────────────

func proxySendSignal(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		resp, err := deps.OrchestrationClient.SendSignal(ctx, &orchpb.SendSignalRequest{
			TenantId:   extractTenantID(ctx),
			FromAgent:  getString(input, "from"),
			ToAgent:    getString(input, "to"),
			SignalType: getString(input, "type"),
			Content:    getString(input, "content"),
			ReplyTo:    getString(input, "reply_to"),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"signal_id": resp.SignalId}, nil
	}
}

func proxyListSignals(deps *Dependencies) mcp.ToolHandler {
	return func(ctx context.Context, input map[string]any) (any, error) {
		return deps.OrchestrationClient.ListSignals(ctx, &orchpb.ListSignalsRequest{
			TenantId:   extractTenantID(ctx),
			AgentId:    getString(input, "agent_id"),
			UnreadOnly: getBool(input, "unread_only"),
		})
	}
}
