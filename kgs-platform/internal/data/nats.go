package data

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"kgs-platform/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

type NATSHandler func(context.Context, []byte)

type natsSubscription struct {
	id      string
	subject string
	handler NATSHandler
}

type NATSClient struct {
	url    string
	stream string
	log    *log.Helper

	mu   sync.RWMutex
	subs map[string]natsSubscription
}

func NewNATSClientFromConfig(cfg *conf.Data_NATS, logger *log.Helper) (*NATSClient, error) {
	if cfg == nil || strings.TrimSpace(cfg.GetUrl()) == "" {
		return nil, nil
	}
	return &NATSClient{
		url:    strings.TrimSpace(cfg.GetUrl()),
		stream: strings.TrimSpace(cfg.GetStream()),
		log:    logger,
		subs:   map[string]natsSubscription{},
	}, nil
}

func (c *NATSClient) Publish(ctx context.Context, subject string, payload []byte) error {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, sub := range c.subs {
		if subjectMatch(sub.subject, subject) {
			sub.handler(ctx, payload)
		}
	}
	return nil
}

func (c *NATSClient) Subscribe(subject string, handler NATSHandler) (func(), error) {
	if c == nil {
		return func() {}, nil
	}
	if strings.TrimSpace(subject) == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if handler == nil {
		return nil, fmt.Errorf("handler is required")
	}

	id := strings.TrimSpace(subject) + fmt.Sprintf("-%p", handler)
	sub := natsSubscription{
		id:      id,
		subject: subject,
		handler: handler,
	}

	c.mu.Lock()
	c.subs[id] = sub
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		delete(c.subs, id)
		c.mu.Unlock()
	}, nil
}

func (c *NATSClient) Ping(ctx context.Context) error {
	if c == nil {
		return nil
	}
	raw := strings.TrimSpace(c.url)
	if raw == "" {
		return fmt.Errorf("nats url is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "nats://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid nats url %q: %w", c.url, err)
	}
	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("invalid nats url %q", c.url)
	}
	if !strings.Contains(host, ":") {
		host += ":4222"
	}
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return fmt.Errorf("nats tcp check failed %s: %w", host, err)
	}
	_ = conn.Close()
	return nil
}

func subjectMatch(pattern, subject string) bool {
	pattern = strings.TrimSpace(pattern)
	subject = strings.TrimSpace(subject)
	if pattern == subject {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(subject, prefix)
	}
	return false
}
