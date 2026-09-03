package event

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
	"vnp-memory/services/graphiti-search/internal/usecase"
)

type NatsSubscriber struct {
	nc   *nats.Conn
	repo usecase.CacheRepo
}

func NewNatsSubscriber(nc *nats.Conn, repo usecase.CacheRepo) *NatsSubscriber {
	return &NatsSubscriber{
		nc:   nc,
		repo: repo,
	}
}

func (s *NatsSubscriber) Listen(ctx context.Context) error {
	_, err := s.nc.Subscribe("graphiti.episode.ingested", func(msg *nats.Msg) {
		groupID := string(msg.Data) 
		if groupID != "" {
			err := s.repo.InvalidateGroup(context.Background(), groupID)
			if err != nil {
				log.Printf("Failed to invalidate cache for group %s: %v", groupID, err)
			} else {
				log.Printf("Invalidated cache for group %s", groupID)
			}
		}
	})
	return err
}
