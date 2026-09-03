package agentmemory

import (
	"github.com/vnp-community/vnp-memory/gateway/internal/adapter/mcp"
	memorypb "github.com/vnp-memory/api/proto/memory/v1"
	observepb "github.com/vnp-memory/api/proto/observe/v1"
	orchpb "github.com/vnp-memory/api/proto/orchestration/v1"
	searchpb "github.com/vnp-memory/api/proto/search/v1"
)

// RegisterAllAgentMemoryTools registers 37 new agentmemory tools
func RegisterAllAgentMemoryTools(reg *mcp.ToolRegistry, deps *Dependencies) {
	reg.RegisterAll(MemoryCoreTools(deps)...)
	reg.RegisterAll(SessionTools(deps)...)
	reg.RegisterAll(ObserveTools(deps)...)
	reg.RegisterAll(GovernanceTools(deps)...)
	reg.RegisterAll(GraphTools(deps)...)
	reg.RegisterAll(OrchestrationTools(deps)...)
	reg.RegisterAll(SignalTools(deps)...)
	reg.RegisterAll(ReplaySlotTools(deps)...)
	reg.RegisterAll(AdminTools(deps)...)
}

type Dependencies struct {
	ObserveClient       observepb.ObserveServiceClient
	MemoryClient        memorypb.AgentMemoryServiceClient
	SearchClient        searchpb.ObserveSearchServiceClient
	OrchestrationClient orchpb.OrchestrationServiceClient
	AdminClient         interface{}
}
