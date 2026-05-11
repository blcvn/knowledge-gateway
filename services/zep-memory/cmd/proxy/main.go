package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("🚀 Starting Zep Memory Forwarding Proxy (Legacy Port 9062)...")
	fmt.Println("⚠️  WARNING: This service is DEPRECATED. All traffic is proxied to zep-core (Port 9061).")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. Establish gRPC connection to `zep-core`
	// 2. Start local gRPC server listening on the legacy port
	// 3. Register `MemoryServiceProxy`

	<-ctx.Done()
	log.Println("Shutting down Zep Memory Proxy gracefully...")
}
