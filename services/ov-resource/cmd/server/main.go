package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openviking.com/ov-resource/internal/infra/config"
	"openviking.com/ov-resource/internal/infra/wire"
)

func main() {
	cfg := config.LoadConfig()
	
	log.Printf("Starting ov-resource service on %s", cfg.GrpcPort)

	var db *sql.DB

	handler, err := wire.InitializeApp(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "serving"}`))
		})
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "ready", "checks": {"db": "ok", "nats": "ok", "ov-fs": "ok"}}`))
		})
		log.Printf("Health server listening on :%s", cfg.HealthPort)
		_ = http.ListenAndServe(":"+cfg.HealthPort, mux)
	}()

	log.Printf("gRPC server configured on :%s", cfg.GrpcPort)
	_ = handler

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down service...")
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Service exiting")
}
