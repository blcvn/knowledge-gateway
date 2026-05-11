package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"vnp-memory/services/memobase-engine/internal/infra/config"
	// "vnp-memory/services/memobase-engine/internal/infra/wire"
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Perform Startup Checks
	log.Println("Running startup checks...")
	// TODO: LLM sanity check (Bifrost max_tokens=16)
	// TODO: Embedding dimension validation (EMBEDDING_DIM)
	// TODO: PostgreSQL pgvector extension check

	// 3. Initialize dependency injection via Wire
	// server, err := wire.InitializeApp(cfg)
	// if err != nil {
	// 	log.Fatalf("Failed to initialize app: %v", err)
	// }

	// 4. Setup health check endpoints
	go func() {
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		})
		http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			// TODO: check DB + NATS + Bifrost
			w.WriteHeader(http.StatusOK)
		})
		log.Printf("Health server listening on :%d", cfg.HealthPort)
		if err := http.ListenAndServe(":9099", nil); err != nil {
			log.Fatalf("Health server failed: %v", err)
		}
	}()

	// 5. Start gRPC server
	go func() {
		// if err := server.Start(); err != nil {
		// 	log.Fatalf("gRPC server failed: %v", err)
		// }
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")
	// TODO: drain NATS consumers, finish in-flight LLM calls
}
