package bus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

type NATSBus struct {
	server *natsserver.Server
	conn   *nats.Conn
	js     nats.JetStreamContext
	logger *slog.Logger
}

type NATSConfig struct {
	Mode     string
	URL      string
	StoreDir string
}

func NewNATSBus(cfg NATSConfig, logger *slog.Logger) (*NATSBus, error) {
	if cfg.Mode == "external" {
		return newExternalNATS(cfg, logger)
	}
	return newEmbeddedNATS(cfg, logger)
}

func newExternalNATS(cfg NATSConfig, logger *slog.Logger) (*NATSBus, error) {
	nc, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &NATSBus{conn: nc, js: js, logger: logger}, nil
}

func newEmbeddedNATS(cfg NATSConfig, logger *slog.Logger) (*NATSBus, error) {
	opts := &natsserver.Options{
		DontListen: true,
		JetStream:  true,
		StoreDir:   cfg.StoreDir,
		MaxPayload: 8 * 1024 * 1024,
	}

	srv, err := natsserver.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("create NATS server: %w", err)
	}

	go srv.Start()

	if !srv.ReadyForConnections(10 * time.Second) {
		return nil, fmt.Errorf("NATS embedded not ready in 10s")
	}

	nc, err := nats.Connect("",
		nats.InProcessServer(srv),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to embedded NATS: %w", err)
	}

	js, err := nc.JetStream(
		nats.PublishAsyncMaxPending(256),
	)
	if err != nil {
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	bus := &NATSBus{
		server: srv,
		conn:   nc,
		js:     js,
		logger: logger,
	}

	if err := bus.createStreams(); err != nil {
		return nil, err
	}

	logger.Info("NATS embedded started", "mode", "in-process", "jetstream", true)
	return bus, nil
}

func (b *NATSBus) createStreams() error {
	streams := []struct {
		Name     string
		Subjects []string
	}{
		{"cognee", []string{"cognee.>"}},
		{"graphiti", []string{"graphiti.>"}},
		{"memobase", []string{"memobase.>"}},
		{"openviking", []string{"ov.>"}},
		{"zep", []string{"zep.>"}},
		{"supermemory", []string{"sm.>"}},
		{"admin", []string{"admin.>"}},
	}

	for _, s := range streams {
		_, err := b.js.AddStream(&nats.StreamConfig{
			Name:      s.Name,
			Subjects:  s.Subjects,
			Retention: nats.WorkQueuePolicy,
			MaxAge:    24 * time.Hour,
		})
		if err != nil {
			return fmt.Errorf("create stream %s: %w", s.Name, err)
		}
	}
	return nil
}

func (b *NATSBus) Publish(ctx context.Context, subject string, data []byte) error {
	_, err := b.js.Publish(subject, data)
	return err
}

func (b *NATSBus) Subscribe(subject, consumer string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return b.js.QueueSubscribe(subject, consumer, handler,
		nats.Durable(consumer),
		nats.DeliverNew(),
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(3),
	)
}

func (b *NATSBus) Close() {
	if b.conn != nil {
		b.conn.Drain()
	}
	if b.server != nil {
		b.server.Shutdown()
	}
}
