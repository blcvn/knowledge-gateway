package agentmemory_test

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    observepb "github.com/vnp-memory/api/proto/observe/v1"
    memorypb  "github.com/vnp-memory/api/proto/memory/v1"
    searchpb  "github.com/vnp-memory/api/proto/search/v1"
)

// TestE2E_CompleteCodingSession simulates a full agent coding session
// and validates the end-to-end flow from observation → memory → search.
func TestE2E_CompleteCodingSession(t *testing.T) {
    h := setupHarness(t)

    // === Phase 1: Start Session ===
    startResp, err := h.observe.StartSession(h.ctx, &observepb.StartSessionRequest{
        TenantId: "e2e-tenant", Project: "auth-service",
        Model: "claude-3-5-sonnet", AgentId: "e2e-agent",
    })
    require.NoError(t, err)
    sessionID := startResp.SessionId

    // === Phase 2: Simulate coding observations ===
    tools := []struct {
        tool   string
        output map[string]any
    }{
        {"Read", map[string]any{"file": "auth.go", "content": "JWT validation"}},
        {"Write", map[string]any{"file": "auth.go", "diff": "added RS256 validation"}},
        {"Bash", map[string]any{"command": "go test ./...", "exit": 0}},
    }

    for _, tool := range tools {
        out, _ := json.Marshal(tool.output)
        _, err = h.observe.Observe(h.ctx, &observepb.ObserveRequest{
            SessionId:  sessionID,
            HookType:   "post_tool_use",
            ToolName:   tool.tool,
            ToolOutput: out,
            TenantId:   "e2e-tenant",
            AgentId:    "e2e-agent",
        })
        require.NoError(t, err)
    }

    // === Phase 3: Save memory from session ===
    memResp, err := h.memory.RememberAgent(h.ctx, &memorypb.RememberAgentRequest{
        TenantId:  "e2e-tenant",
        Project:   "auth-service",
        Type:      "architecture",
        Title:     "JWT validation uses RS256",
        Content:   "The auth-service validates JWTs with RS256 algorithm using JOSE library",
        Concepts:  []string{"jwt", "rs256", "jose", "auth"},
        SessionId: sessionID,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, memResp.MemoryId)

    // Wait a moment for indexing
    time.Sleep(100 * time.Millisecond)

    // === Phase 4: Search should return this memory ===
    searchResp, err := h.search.SmartSearch(h.ctx, &searchpb.SmartSearchRequest{
        Query:    "jwt authentication",
        TenantId: "e2e-tenant",
        Limit:    5,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, searchResp.Results)

    // === Phase 5: End session ===
    endResp, err := h.observe.EndSession(h.ctx, &observepb.EndSessionRequest{
        SessionId: sessionID, TenantId: "e2e-tenant",
    })
    require.NoError(t, err)
    assert.Equal(t, "completed", endResp.Status)
    assert.Equal(t, int32(3), endResp.ObservationCount)
}
