package agentmemory_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    orchpb "github.com/vnp-memory/api/proto/orchestration/v1"
)

func TestOrchestration_ActionStateMachine(t *testing.T) {
    h := setupHarness(t)

    // Create
    action, err := h.orch.CreateAction(h.ctx, &orchpb.CreateActionRequest{
        TenantId: "test-tenant", Title: "Test Action", Priority: 50,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, action.ActionId)

    // Transition: pending → active
    _, err = h.orch.UpdateAction(h.ctx, &orchpb.UpdateActionRequest{
        ActionId: action.ActionId, Status: "active",
    })
    require.NoError(t, err)

    // Transition: active → done
    _, err = h.orch.UpdateAction(h.ctx, &orchpb.UpdateActionRequest{
        ActionId: action.ActionId, Status: "done", Result: "completed successfully",
    })
    require.NoError(t, err)

    // Invalid: done → active (should fail)
    _, err = h.orch.UpdateAction(h.ctx, &orchpb.UpdateActionRequest{
        ActionId: action.ActionId, Status: "active",
    })
    assert.Error(t, err)
}

func TestOrchestration_LeaseConflict(t *testing.T) {
    h := setupHarness(t)
    action, _ := h.orch.CreateAction(h.ctx, &orchpb.CreateActionRequest{
        TenantId: "test-tenant", Title: "Shared Task",
    })

    // Agent 1 acquires lease
    lease1, err := h.orch.AcquireLease(h.ctx, &orchpb.AcquireLeaseRequest{
        ActionId: action.ActionId, AgentId: "agent-1", TtlSecs: 300,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, lease1.LeaseId)
    assert.False(t, lease1.Conflict)

    // Agent 2 tries to acquire → conflict
    lease2, err := h.orch.AcquireLease(h.ctx, &orchpb.AcquireLeaseRequest{
        ActionId: action.ActionId, AgentId: "agent-2", TtlSecs: 300,
    })
    require.NoError(t, err)
    assert.True(t, lease2.Conflict)
    assert.Equal(t, "agent-1", lease2.ConflictingAgent)

    // Release + retry
    h.orch.ReleaseLease(h.ctx, &orchpb.ReleaseLeaseRequest{LeaseId: lease1.LeaseId})
    lease3, _ := h.orch.AcquireLease(h.ctx, &orchpb.AcquireLeaseRequest{
        ActionId: action.ActionId, AgentId: "agent-2", TtlSecs: 300,
    })
    assert.False(t, lease3.Conflict)
}

func TestOrchestration_SignalRouting(t *testing.T) {
    h := setupHarness(t)

    // Send signal
    sig, err := h.orch.SendSignal(h.ctx, &orchpb.SendSignalRequest{
        TenantId: "test-tenant", FromAgent: "agent-a", ToAgent: "agent-b",
        SignalType: "handoff", Content: "Task complete, your turn",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, sig.SignalId)

    // List signals for agent-b
    signals, err := h.orch.ListSignals(h.ctx, &orchpb.ListSignalsRequest{
        TenantId: "test-tenant", AgentId: "agent-b", UnreadOnly: true,
    })
    require.NoError(t, err)
    assert.NotEmpty(t, signals.Signals)
    assert.Equal(t, sig.SignalId, signals.Signals[0].Id)
}
