package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("🚀 Starting Zep Core Service (Port 9061)...")
	
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO: Initialize Dependency Injection (Google Wire)
	// TODO: Start gRPC server
	// TODO: Connect to PostgreSQL and NATS

	<-ctx.Done()
	log.Println("Shutting down Zep Core Service gracefully...")
}
