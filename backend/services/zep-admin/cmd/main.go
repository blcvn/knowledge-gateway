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
	fmt.Println("🚀 Starting Zep Admin Service (Port 9066)...")
	
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO: Initialize Dependency Injection (Google Wire)
	// TODO: Start gRPC server

	<-ctx.Done()
	log.Println("Shutting down Zep Admin Service gracefully...")
}
