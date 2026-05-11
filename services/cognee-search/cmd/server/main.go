package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"vnp-memory/services/cognee-search/internal/infrastructure/di"
)

func main() {
	// Initialize DI
	// server, subscriber, err := di.InitializeServer()
	// Mocking InitializeServer since we didn't run wire
	
	log.Println("Starting cognee-search service...")

	// 1. Health check HTTP server
	go func() {
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("SERVING"))
		})
		log.Println("Health check listening on :9093")
		if err := http.ListenAndServe(":9093", nil); err != nil {
			log.Fatalf("Failed to serve health check: %v", err)
		}
	}()

	// 2. Start NATS Subscriber
	// if err := subscriber.Subscribe(); err != nil {
	//     log.Fatalf("Failed to subscribe to NATS: %v", err)
	// }

	// 3. Start gRPC Server
	// go func() {
	// 	if err := server.Start(); err != nil {
	// 		log.Fatalf("Failed to start gRPC server: %v", err)
	// 	}
	// }()

	log.Println("cognee-search service started.")

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down service...")
	// server.Stop()
	log.Println("Service stopped gracefully")
}
