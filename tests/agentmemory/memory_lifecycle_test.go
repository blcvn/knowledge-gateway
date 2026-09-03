package agentmemory_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    memorypb "github.com/vnp-memory/api/proto/memory/v1"
)

func TestMemory_RememberAndVersioning(t *testing.T) {
    h := setupHarness(t)

    // Remember v1
    resp1, err := h.memory.RememberAgent(h.ctx, &memorypb.RememberAgentRequest{
        TenantId: "test-tenant", Project: "test-project",
        Type: "pattern", Title: "Use JOSE for JWT",
        Content: "Always use JOSE library for JWT handling",
        Concepts: []string{"jose", "jwt", "auth"},
    })
    require.NoError(t, err)
    assert.Equal(t, int32(1), resp1.Version)
    assert.Empty(t, resp1.Superseded)

    // Remember v2 (concepts overlap > 70% Jaccard)
    resp2, err := h.memory.RememberAgent(h.ctx, &memorypb.RememberAgentRequest{
        TenantId: "test-tenant", Project: "test-project",
        Type: "pattern", Title: "Updated JWT approach",
        Content: "Use JOSE with RS256 algorithm",
        Concepts: []string{"jose", "jwt", "signing"},
    })
    require.NoError(t, err)
    assert.Equal(t, int32(2), resp2.Version)
    assert.Contains(t, resp2.Superseded, resp1.MemoryId)
}

func TestMemory_NoVersioning_DifferentConcepts(t *testing.T) {
    h := setupHarness(t)

    resp1, _ := h.memory.RememberAgent(h.ctx, &memorypb.RememberAgentRequest{
        TenantId: "test-tenant", Project: "p",
        Type: "fact", Title: "Redis is fast",
        Content: "Redis is fast for caching",
        Concepts: []string{"redis", "cache", "performance"},
    })

    resp2, _ := h.memory.RememberAgent(h.ctx, &memorypb.RememberAgentRequest{
        TenantId: "test-tenant", Project: "p",
        Type: "fact", Title: "Auth with JOSE",
        Content: "Use JOSE for auth",
        Concepts: []string{"jose", "auth", "security"},
    })

    assert.Equal(t, int32(1), resp2.Version)  // no supersede
    assert.NotContains(t, resp2.Superseded, resp1.MemoryId)
}

func TestMemory_RetentionScore(t *testing.T) {
    h := setupHarness(t)

    resp, _ := h.memory.RememberAgent(h.ctx, &memorypb.RememberAgentRequest{
        TenantId: "test-tenant", Project: "p",
        Type: "fact", Title: "New fact",
        Content: "Just remembered",
        Concepts: []string{"new"},
    })

    score, err := h.memory.GetRetentionScore(h.ctx, &memorypb.GetRetentionScoreRequest{
        MemoryId: resp.MemoryId,
    })
    require.NoError(t, err)
    assert.Greater(t, score.Score, 0.0)
    assert.Equal(t, "keep", score.RecommendAction)
}

func TestMemory_Slots(t *testing.T) {
    h := setupHarness(t)

    // Write slot
    _, err := h.memory.WriteSlot(h.ctx, &memorypb.WriteSlotRequest{
        TenantId: "test-tenant", Scope: "project", Label: "tech-stack",
        Content: "Go, PostgreSQL, NATS", Mode: "replace",
    })
    require.NoError(t, err)

    // Read slot
    slot, err := h.memory.GetSlot(h.ctx, &memorypb.GetSlotRequest{
        TenantId: "test-tenant", Scope: "project", Label: "tech-stack",
    })
    require.NoError(t, err)
    assert.Equal(t, "Go, PostgreSQL, NATS", slot.Content)

    // Append
    _, err = h.memory.WriteSlot(h.ctx, &memorypb.WriteSlotRequest{
        TenantId: "test-tenant", Scope: "project", Label: "tech-stack",
        Content: "Redis", Mode: "append",
    })
    require.NoError(t, err)

    slotUpdated, _ := h.memory.GetSlot(h.ctx, &memorypb.GetSlotRequest{
        TenantId: "test-tenant", Scope: "project", Label: "tech-stack",
    })
    assert.Contains(t, slotUpdated.Content, "Redis")
    assert.Contains(t, slotUpdated.Content, "Go, PostgreSQL, NATS")
}
