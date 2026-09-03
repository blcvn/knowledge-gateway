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
	fmt.Println("🚀 Starting Zep Search Service (Port 9065)...")
	
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO: Initialize Vector and Graph retrievers

	<-ctx.Done()
	log.Println("Shutting down Zep Search Service gracefully...")
}
