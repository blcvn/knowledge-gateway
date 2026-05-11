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
	fmt.Println("🚀 Starting Zep Graph Service (Port 9064)...")
	
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO: Connect to Neo4j
	// TODO: Start NATS background workers

	<-ctx.Done()
	log.Println("Shutting down Zep Graph Service gracefully...")
}
