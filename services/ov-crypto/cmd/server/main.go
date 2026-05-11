package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Configure slog
	// Load config
	// Connect to DB
	// Connect to NATS
	// Init KMS Provider
	// Init Repository
	// Init Usecases
	// Init gRPC Handler
	// Setup OTel & Prometheus

	log.Println("Starting ov-crypto service...")

	// Listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down ov-crypto service gracefully...")
}
