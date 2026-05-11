package main

import (
	"database/sql"
	"log"

	"github.com/vnp-memory/services/ov-session/internal/infra/config"
	"github.com/vnp-memory/services/ov-session/internal/infra/wire"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("Starting ov-session on %s", cfg.GRPCPort)

	// Mock DB connection
	db := &sql.DB{}

	handler, err := wire.InitializeApp(db)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	_ = handler
	log.Println("Service initialized successfully")
}
