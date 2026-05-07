package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
)

type NATSHandler func(context.Context, []byte)

type NATSClient struct {
	url    string
	stream string
	log    *log.Helper

	mu sync.RWMutex
	nc *nats.Conn
}

func NewNATSClientFromConfig(cfg *conf.Data_NATS, logger *log.Helper) (*NATSClient, error) {
	if cfg == nil || strings.TrimSpace(cfg.GetUrl()) == "" {
		return nil, nil
	}
	if logger == nil {
		logger = log.NewHelper(log.DefaultLogger)
	}

	url := strings.TrimSpace(cfg.GetUrl())
	nc, err := nats.Connect(url,
		nats.Timeout(3*time.Second),
		nats.ReconnectWait(500*time.Millisecond),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(_ *nats.Conn, disconnectErr error) {
			if disconnectErr != nil {
				logger.Warnf("nats disconnected from %s: %v", url, disconnectErr)
				return
			}
			logger.Warnf("nats disconnected from %s", url)
		}),
	)
	if err != nil {
		return nil, err
	}

	return &NATSClient{
		url:    url,
		stream: strings.TrimSpace(cfg.GetStream()),
		log:    logger,
		nc:     nc,
	}, nil
}

func (c *NATSClient) Publish(ctx context.Context, subject string, payload []byte) error {
	if c == nil {
		return nil
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mu.RLock()
	nc := c.nc
	c.mu.RUnlock()
	if nc == nil || nc.IsClosed() {
		return errors.New("nats connection is closed")
	}

	return nc.Publish(subject, payload)
}

func (c *NATSClient) Subscribe(subject string, handler NATSHandler) (func(), error) {
	if c == nil {
		return func() {}, nil
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if handler == nil {
		return nil, fmt.Errorf("handler is required")
	}

	c.mu.RLock()
	nc := c.nc
	c.mu.RUnlock()
	if nc == nil || nc.IsClosed() {
		return nil, errors.New("nats connection is closed")
	}

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		handler(context.Background(), msg.Data)
	})
	if err != nil {
		return nil, err
	}

	return func() {
		_ = sub.Unsubscribe()
	}, nil
}

func (c *NATSClient) Ping(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	nc := c.nc
	c.mu.RUnlock()
	if nc == nil || nc.IsClosed() {
		return errors.New("nats connection is closed")
	}

	timeout := 2 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			timeout = remaining
		}
	}
	return nc.FlushTimeout(timeout)
}

func (c *NATSClient) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	nc := c.nc
	c.nc = nil
	c.mu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nil
	}
	err := nc.Drain()
	nc.Close()
	return err
}
