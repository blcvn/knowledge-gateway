package agentmemory_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    observepb "github.com/vnp-memory/api/proto/observe/v1"
)

func TestObserve_SessionLifecycle(t *testing.T) {
    h := setupHarness(t)

    // Start session
    startResp, err := h.observe.StartSession(h.ctx, &observepb.StartSessionRequest{
        TenantId: "test-tenant", Project: "test-project",
        Model: "claude-3-5-sonnet", AgentId: "test-agent",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, startResp.SessionId)
    assert.Equal(t, "active", startResp.Status)

    sessionID := startResp.SessionId

    // Record observations
    for i := 0; i < 5; i++ {
        _, err := h.observe.Observe(h.ctx, &observepb.ObserveRequest{
            SessionId: sessionID,
            HookType:  "post_tool_use",
            ToolName:  "Write",
            TenantId:  "test-tenant",
            AgentId:   "test-agent",
        })
        require.NoError(t, err)
    }

    // End session
    endResp, err := h.observe.EndSession(h.ctx, &observepb.EndSessionRequest{
        SessionId: sessionID, TenantId: "test-tenant",
    })
    require.NoError(t, err)
    assert.Equal(t, "completed", endResp.Status)
    assert.Equal(t, int32(5), endResp.ObservationCount)
}

func TestObserve_Deduplication(t *testing.T) {
    h := setupHarness(t)
    sessionID := h.newSession("dedup-test")

    req := &observepb.ObserveRequest{
        SessionId: sessionID, HookType: "post_tool_use",
        ToolName: "Read", ToolOutput: []byte(`{"output":"same content"}`),
        TenantId: "test-tenant",
    }

    resp1, err := h.observe.Observe(h.ctx, req)
    require.NoError(t, err)
    assert.False(t, resp1.Deduplicated)

    resp2, err := h.observe.Observe(h.ctx, req)
    require.NoError(t, err)
    assert.True(t, resp2.Deduplicated)  // second identical obs = deduped
}

func TestObserve_MaxObsPerSession(t *testing.T) {
    h := setupHarness(t)
    sessionID := h.newSession("limit-test")

    // Fill to max (500 by default in config)
    // For test, set MAX_OBS=5 via env before bootstrap
    // Verify 6th returns quota-exceeded error
}

func TestObserve_ListSessions(t *testing.T) {
    h := setupHarness(t)
    sid1 := h.newSession("project-a")
    sid2 := h.newSession("project-b")

    resp, err := h.observe.ListSessions(h.ctx, &observepb.ListSessionsRequest{
        TenantId: "test-tenant", Limit: 10,
    })
    require.NoError(t, err)
    sessionIDs := make([]string, len(resp.Sessions))
    for i, s := range resp.Sessions { sessionIDs[i] = s.Id }
    assert.Contains(t, sessionIDs, sid1)
    assert.Contains(t, sessionIDs, sid2)
}
