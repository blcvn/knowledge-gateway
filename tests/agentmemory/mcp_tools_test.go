package agentmemory_test

import (
    "context"
    "encoding/json"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/vnp-memory/gateway/internal/adapter/mcp"
    agentmemorytools "github.com/vnp-memory/gateway/internal/adapter/mcp/tools/agentmemory"
)

func TestMCPTools_Count(t *testing.T) {
    reg := mcp.NewToolRegistry()
    deps := agentmemorytools.MockDependencies()
    agentmemorytools.RegisterAllAgentMemoryTools(reg, deps)
    assert.Equal(t, 53, reg.Count())
}

func TestMCPTools_NoDuplicates(t *testing.T) {
    reg := mcp.NewToolRegistry()
    deps := agentmemorytools.MockDependencies()
    agentmemorytools.RegisterAllAgentMemoryTools(reg, deps)

    names := reg.Names()
    seen := make(map[string]bool)
    for _, name := range names {
        assert.False(t, seen[name], "duplicate tool: "+name)
        seen[name] = true
    }
}

func TestMCPTools_InputSchemaValidation(t *testing.T) {
    reg := mcp.NewToolRegistry()
    deps := agentmemorytools.MockDependencies()
    agentmemorytools.RegisterAllAgentMemoryTools(reg, deps)

    requiredTools := []string{
        "memory_smart_search", "memory_save", "memory_observe",
        "memory_session_start", "memory_lease_acquire",
        "memory_signal_send", "memory_sketch_promote",
        "memory_doctor", "memory_snapshot_create",
    }
    for _, name := range requiredTools {
        tool, ok := reg.GetTool(name)
        require.True(t, ok, "missing tool: "+name)
        assert.NotEmpty(t, tool.Description, "empty description: "+name)
        assert.NotEmpty(t, tool.InputSchema.Type, "empty schema: "+name)
    }
}
