package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blcvn/ba-shared-libs/pkg/infrastructure/queue"
	"github.com/go-redis/redis/v8"
)

func main() {
	// Initialize Redis for Queue/Events
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	redisQueue := queue.NewRedisQueue(redisClient)
	_ = redisQueue

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Knowledge Worker started. Waiting for events...")

	go func() {
		// Mock consumer loop
		for {
			// In real impl: queue.Consume("knowledge_events", handler)
			time.Sleep(5 * time.Second)
			// log.Println("Heartbeat: Worker operational")
		}
	}()

	<-sigs
	log.Println("Shutting down Knowledge Worker...")
}
