// Package redis implements the task queue using Redis Lists.
//
// This reimplements the ba-shared-libs/pkg/queue interface internally,
// eliminating the external dependency. (Recommendation Option 2 from spec)
// (MERGE-P3-T1)
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TaskJob is a task payload in the queue.
type TaskJob struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	EnqueuedAt time.Time     `json:"enqueued_at"`
	Attempts  int             `json:"attempts"`
}

// HandlerFunc is a worker task handler.
type HandlerFunc func(ctx context.Context, job TaskJob) error

// Consumer is a Redis-backed task queue consumer.
type Consumer struct {
	client      *redis.Client
	handlers    map[string]HandlerFunc
	concurrency int
	pollMs      int
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// NewConsumer creates a Consumer.
func NewConsumer(cfg RedisConfig, concurrency int) *Consumer {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if concurrency <= 0 {
		concurrency = 5
	}
	return &Consumer{
		client:      rdb,
		handlers:    make(map[string]HandlerFunc),
		concurrency: concurrency,
		pollMs:      500,
	}
}

// RegisterHandler registers a task type handler.
func (c *Consumer) RegisterHandler(taskType string, fn HandlerFunc) {
	c.handlers[taskType] = fn
}

// Start begins consuming from all registered task type queues.
func (c *Consumer) Start(ctx context.Context) error {
	if len(c.handlers) == 0 {
		return fmt.Errorf("no handlers registered")
	}

	sem := make(chan struct{}, c.concurrency)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		for taskType := range c.handlers {
			queueKey := "queue:" + taskType
			data, err := c.client.BLPop(ctx, time.Duration(c.pollMs)*time.Millisecond, queueKey).Result()
			if err != nil || len(data) < 2 {
				continue
			}

			var job TaskJob
			if err := json.Unmarshal([]byte(data[1]), &job); err != nil {
				continue
			}

			handler, ok := c.handlers[job.Type]
			if !ok {
				continue
			}

			sem <- struct{}{}
			go func(h HandlerFunc, j TaskJob) {
				defer func() { <-sem }()
				if err := h(ctx, j); err != nil {
					// Requeue with backoff on failure
					j.Attempts++
					if j.Attempts < 3 {
						payload, _ := json.Marshal(j)
						_ = c.client.RPush(ctx, "queue:"+j.Type, payload).Err()
					}
				}
			}(handler, job)
		}
	}
}

// Queue implements port.Queue using Redis Lists.
type Queue struct {
	client *redis.Client
}

// NewQueue creates a Queue.
func NewQueue(cfg RedisConfig) *Queue {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &Queue{client: rdb}
}

// Push enqueues a raw payload to a named queue.
func (q *Queue) Push(ctx context.Context, queueName string, payload []byte) error {
	job := TaskJob{
		ID:         fmt.Sprintf("%d", time.Now().UnixNano()),
		Payload:    payload,
		EnqueuedAt: time.Now(),
	}
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.client.RPush(ctx, "queue:"+queueName, data).Err()
}

// Pop pops a payload from a named queue (blocking with timeout).
func (q *Queue) Pop(ctx context.Context, queueName string, timeoutMs int) ([]byte, error) {
	result, err := q.client.BLPop(ctx, time.Duration(timeoutMs)*time.Millisecond, "queue:"+queueName).Result()
	if err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, nil
	}
	return []byte(result[1]), nil
}

// Size returns the current queue depth.
func (q *Queue) Size(ctx context.Context, queueName string) (int64, error) {
	return q.client.LLen(ctx, "queue:"+queueName).Result()
}
