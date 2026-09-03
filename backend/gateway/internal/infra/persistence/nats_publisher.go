package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSPublisher implements port.EventPublisher using NATS JetStream.
type NATSPublisher struct {
	js     nats.JetStreamContext
	conn   *nats.Conn
	logger *slog.Logger
}

// NewNATSPublisher creates a new NATS JetStream event publisher.
func NewNATSPublisher(url string, credsFile string, logger *slog.Logger) (*NATSPublisher, func(), error) {
	opts := []nats.Option{
		nats.Name("vnp-gateway"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(60),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", "error", err)
			}
		}),
	}

	if credsFile != "" {
		opts = append(opts, nats.UserCredentials(credsFile))
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("nats connect: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("jetstream context: %w", err)
	}

	// Ensure stream exists
	streamName := "GATEWAY"
	_, err = js.StreamInfo(streamName)
	if err != nil {
		// Create the stream
		_, err = js.AddStream(&nats.StreamConfig{
			Name:       streamName,
			Subjects:   []string{"gateway.>"},
			Retention:  nats.LimitsPolicy,
			MaxAge:     7 * 24 * time.Hour, // 7 days retention
			MaxMsgs:    -1,
			MaxBytes:   1 << 30, // 1 GB
			Discard:    nats.DiscardOld,
			Storage:    nats.FileStorage,
			Duplicates: 2 * time.Minute,
		})
		if err != nil {
			conn.Close()
			return nil, nil, fmt.Errorf("create stream: %w", err)
		}
		logger.Info("NATS JetStream stream created", "name", streamName)
	}

	cleanup := func() {
		logger.Info("closing NATS connection")
		conn.Close()
	}

	logger.Info("NATS publisher initialized", "url", url, "stream", streamName)
	return &NATSPublisher{js: js, conn: conn, logger: logger}, cleanup, nil
}

// Publish sends an event to the specified NATS subject via JetStream.
func (p *NATSPublisher) Publish(ctx context.Context, subject string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Publish with timeout from context
	_, err = p.js.Publish(subject, data, nats.Context(ctx))
	if err != nil {
		p.logger.Error("publish failed",
			"subject", subject,
			"error", err,
			"data_len", len(data),
		)
		return fmt.Errorf("publish to %s: %w", subject, err)
	}

	p.logger.Debug("event published", "subject", subject, "bytes", len(data))
	return nil
}

// Close drains and closes the NATS connection.
func (p *NATSPublisher) Close() {
	if p.conn != nil {
		p.conn.Drain()
		p.conn.Close()
	}
}
