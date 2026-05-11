package nats

import (
	"context"
	"log"

	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type Subscriber struct {
	// nats connection here
	cache port.CacheStore
}

func NewSubscriber(cache port.CacheStore) *Subscriber {
	return &Subscriber{
		cache: cache,
	}
}

func (s *Subscriber) Subscribe() error {
	// Mock nats subscription to "cognee.pipeline.completed"
	// When received, we would invalidate cache
	
	go func() {
		log.Println("Subscribed to cognee.pipeline.completed for cache invalidation")
		// Simulate msg received
		// s.cache.Invalidate(context.Background(), "search:*")
	}()

	return nil
}
