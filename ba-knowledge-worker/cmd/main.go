package main

import (
	"log"
	"os"
	"strconv"

	"github.com/blcvn/backend/services/ba-knowledge-worker/internal/worker"
	"github.com/blcvn/ba-shared-libs/pkg/queue"
)

func main() {
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	cfg := queue.RedisConfig{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}

	consumer := queue.NewConsumer(cfg, 10) // 10 concurrent workers

	// Register Handlers
	consumer.RegisterHandler(queue.TaskTypeIndexPRD, worker.HandleIndexPRD)
	consumer.RegisterHandler(queue.TaskTypeGenOutline, worker.HandleGenOutline)

	log.Println("Starting Knowledge Worker...")
	if err := consumer.Start(); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
