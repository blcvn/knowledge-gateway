package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/services/memobase-context/usecase/port"
)

type NatsSubscriber struct {
	nc    *nats.Conn
	cache port.ProfileCache
}

func NewNatsSubscriber(nc *nats.Conn, cache port.ProfileCache) *NatsSubscriber {
	return &NatsSubscriber{nc: nc, cache: cache}
}

func (s *NatsSubscriber) Start() error {
	_, err := s.nc.Subscribe("memobase.profile.changed", func(m *nats.Msg) {
		var payload struct {
			UserID    string `json:"user_id"`
			ProjectID string `json:"project_id"`
		}
		if err := json.Unmarshal(m.Data, &payload); err == nil {
			_ = s.cache.DeleteProfiles(context.Background(), payload.UserID, payload.ProjectID)
			slog.Info("Invalidated profile cache via NATS event", "user_id", payload.UserID, "project_id", payload.ProjectID)
		}
	})
	return err
}
