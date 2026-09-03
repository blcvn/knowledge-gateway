// Thêm vào func Bootstrap() theo thứ tự dependency:

func Bootstrap(ctx context.Context, cfg *config.Config) error {
    // ... existing init (DB, NATS, etc.) ...

    // [NEW] Wave 1: Foundation services
    InitPrivacyPackage()                           // pkg/privacy (no-op, package init)
    InitObserveSearch(reg, db, nc, cfg)            // am-search (#37)
    InitObserveService(reg, db, nc, cfg)           // am-observe (#36)
    InitAgentMemoryLifecycle(reg, db, nc, cfg)     // extend memory-service with AgentMemory

    // [NEW] Wave 2: Integration
    InitConsolidationPipeline(reg, db, nc, cfg)    // consolidation in memory-service
    // Note: MCP tools auto-register via tool_registry.go

    // [NEW] Wave 3: Orchestration
    InitOrchestration(reg, db, nc, cfg)            // am-orchestration (#38)

    // [NEW] Wave 4: Governance (extends existing services)
    InitGovernanceAudit(reg, db, nc, cfg)
    InitHealthMonitor(db, nc, cfg)

    // Register SSE handler for gateway
    gateway.RegisterSSEHandler(observeSSEBroker)

    return nil
}
