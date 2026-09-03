package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	orchpb "github.com/vnp-memory/api/proto/orchestration/v1"

	"vnp-memory/services/orchestration-service/internal/domain"
	"vnp-memory/services/orchestration-service/internal/usecase/port"
)

// NATSSignalRouter implements signal routing via NATS JetStream.
// It replaces the dummy SignalService for real inter-agent communication.
//
// SOL-ENT-001 / TASK-ENT-005
//
// Signal subjects:
//   vnp.orchestration.signal.{toAgent}   — directed signal to a specific agent
//   vnp.orchestration.signal.broadcast   — broadcast to all agents in a tenant
type NATSSignalRouter struct {
	repo port.ISignalRepo
	js   nats.JetStreamContext
	conn *nats.Conn
}

// NewNATSSignalRouter creates a new NATS-backed signal router.
// Streams must be pre-created (via NATS stream config or nats-setup job).
func NewNATSSignalRouter(repo port.ISignalRepo, natsURL string) (*NATSSignalRouter, error) {
	conn, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			log.Printf("[signal-router] NATS disconnected: %v", err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("signal-router: connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("signal-router: init JetStream: %w", err)
	}

	// Ensure the orchestration stream exists
	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     "VNP_ORCHESTRATION",
		Subjects: []string{"vnp.orchestration.signal.>"},
		MaxAge:   24 * time.Hour,
		Storage:  nats.MemoryStorage,
	}); err != nil {
		// Stream may already exist — ignore error
		log.Printf("[signal-router] stream already exists or error: %v", err)
	}

	return &NATSSignalRouter{repo: repo, js: js, conn: conn}, nil
}

// Send routes an agent signal via NATS JetStream and persists it to DB.
func (r *NATSSignalRouter) Send(ctx context.Context, req *orchpb.SendSignalRequest) (*domain.Signal, error) {
	signal := domain.Signal{
		ID:         uuid.New().String(),
		TenantID:   req.TenantId,
		FromAgent:  req.FromAgent,
		ToAgent:    req.ToAgent,
		SignalType: req.SignalType,
		Content:    req.Content,
		ThreadID:   req.ThreadId,
		ReplyTo:    req.ReplyTo,
		IsRead:     false,
		ExpiresAt:  time.Now().Add(48 * time.Hour),
		CreatedAt:  time.Now(),
	}

	// 1. Persist to DB for durability + queryability
	if err := r.repo.Save(ctx, signal); err != nil {
		return nil, fmt.Errorf("signal-router: save to DB: %w", err)
	}

	// 2. Publish to NATS JetStream for realtime delivery
	subject := fmt.Sprintf("vnp.orchestration.signal.%s", req.ToAgent)
	payload, err := json.Marshal(signal)
	if err != nil {
		return &signal, nil // saved to DB; NATS failure is non-fatal
	}

	if _, err := r.js.Publish(subject, payload); err != nil {
		log.Printf("[signal-router] NATS publish failed (signal in DB): %v", err)
		// Non-fatal: signal is persisted; agent will poll DB as fallback
	}

	return &signal, nil
}

// Subscribe sets up a JetStream push consumer for an agent.
// Call this in agent service bootstrap to receive directed signals.
func (r *NATSSignalRouter) Subscribe(agentID string, handler func(signal domain.Signal)) (*nats.Subscription, error) {
	subject := fmt.Sprintf("vnp.orchestration.signal.%s", agentID)
	return r.js.Subscribe(subject, func(msg *nats.Msg) {
		var signal domain.Signal
		if err := json.Unmarshal(msg.Data, &signal); err != nil {
			log.Printf("[signal-router] malformed signal msg: %v", err)
			msg.Nak()
			return
		}
		handler(signal)
		msg.Ack()
	}, nats.Durable(fmt.Sprintf("agent-%s", agentID)))
}

// ReapExpired sweeps expired signals from the database.
func (r *NATSSignalRouter) ReapExpired(ctx context.Context) {
	if err := r.repo.DeleteExpired(ctx); err != nil {
		log.Printf("[signal-router] reap expired: %v", err)
	}
}

// DeleteExpired removes expired signals (alias for compatibility with SignalService interface).
func (r *NATSSignalRouter) DeleteExpired(ctx context.Context) {
	r.ReapExpired(ctx)
}

// Close gracefully closes the NATS connection.
func (r *NATSSignalRouter) Close() {
	if r.conn != nil {
		r.conn.Drain()
	}
}
