package main

import (
	"log"
	"time"

	"vnp-memory/apps/zep/internal/app"
	"vnp-memory/apps/zep/internal/config"
)

func main() {
	log.Println("Initializing Zep Monolith...")

	// 1. Load Unified Configuration
	cfg := config.LoadConfig()

	// 2. Initialize Supervisor with a 10s shutdown timeout
	supervisor := app.NewSupervisor(10 * time.Second)

	// 3. Register internal Zep services (TASK-003)
	// Following the order: Infra dependencies -> Internal Services -> Gateway
	supervisor.Register(app.NewZepServiceWrapper("zep-user", cfg.UserPort))
	supervisor.Register(app.NewZepServiceWrapper("zep-thread", cfg.ThreadPort))
	supervisor.Register(app.NewZepServiceWrapper("zep-memory", cfg.MemoryPort))
	supervisor.Register(app.NewZepServiceWrapper("zep-graph", cfg.GraphPort))
	supervisor.Register(app.NewZepServiceWrapper("zep-search", cfg.SearchPort))
	supervisor.Register(app.NewZepServiceWrapper("zep-admin", cfg.AdminPort))

	// 4. Register Gateway last (TASK-004)
	supervisor.Register(app.NewGatewayWrapper(cfg.GatewayPort))

	// 5. Start all components
	if err := supervisor.StartAll(); err != nil {
		log.Fatalf("Failed to start all components: %v", err)
	}

	log.Println("Zep Monolith is fully operational.")

	// 6. Wait for termination signal and stop gracefully
	supervisor.WaitAndStop()
}
