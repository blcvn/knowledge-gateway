package event

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/infra/config"
	"graphiti-pipeline/internal/usecase/port"
)

type NATSPublisher struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	logger *zap.Logger
}

func NewNATSPublisher(cfg config.Config, logger *zap.Logger) port.EventPublisher {
	url := cfg.NATS.URL
	if url == "" {
		url = nats.DefaultURL
	}
	
	// Better connection options for resilience
	opts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * 1000 * 1000 * 1000), // 2 seconds
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("Disconnected from NATS", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("Reconnected to NATS", zap.String("url", nc.ConnectedUrl()))
		}),
	}
	
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		logger.Error("Failed to connect to NATS", zap.Error(err))
		return &NATSPublisher{logger: logger} // Fallback mode
	}

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("Failed to get JetStream context", zap.Error(err))
		return &NATSPublisher{nc: nc, logger: logger}
	}

	logger.Info("Connected to NATS JetStream successfully")
	return &NATSPublisher{nc: nc, js: js, logger: logger}
}

func (p *NATSPublisher) PublishEpisodeIngested(ctx context.Context, event ingestion.EpisodeIngested) error {
	if p.js == nil {
		p.logger.Warn("NATS JS not initialized. Skipping publish.")
		return nil
	}
	data, _ := json.Marshal(event)
	_, err := p.js.Publish("graphiti.episode.ingested", data)
	if err != nil {
		return fmt.Errorf("failed to publish episode ingested event: %w", err)
	}
	return nil
}

func (p *NATSPublisher) PublishEpisodeFailed(ctx context.Context, event ingestion.EpisodeFailed) error {
	if p.js == nil {
		p.logger.Warn("NATS JS not initialized. Skipping publish.")
		return nil
	}
	data, _ := json.Marshal(event)
	_, err := p.js.Publish("graphiti.episode.failed", data)
	if err != nil {
		return fmt.Errorf("failed to publish episode failed event: %w", err)
	}
	return nil
}
