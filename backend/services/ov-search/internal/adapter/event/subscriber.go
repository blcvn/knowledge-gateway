package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"vnp-memory/ov-search/internal/usecase/dto"
	"vnp-memory/ov-search/internal/usecase/port"
)

type Subscriber struct {
	embeddingUC port.EmbeddingUseCase
	hotnessUC   port.HotnessUseCase
	js          jetstream.JetStream
	conn        *nats.Conn
}

func NewSubscriber(euc port.EmbeddingUseCase, huc port.HotnessUseCase, natsURL string) (*Subscriber, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	return &Subscriber{
		embeddingUC: euc,
		hotnessUC:   huc,
		js:          js,
		conn:        nc,
	}, nil
}

func (s *Subscriber) Start(ctx context.Context) error {
	slog.Info("Starting NATS event subscribers")

	// Consume openviking stream
	cons, err := s.js.CreateOrUpdateConsumer(ctx, "openviking", jetstream.ConsumerConfig{
		Durable:       "ov-search-consumer",
		FilterSubjects: []string{"ov.content.written", "ov.content.deleted", "ov.resource.ingested", "ov.session.committed"},
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return err
	}

	// Consume in a background goroutine
	iter, err := cons.Messages()
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := iter.Next()
			if err != nil {
				slog.Error("NATS message iteration error", "error", err)
				continue
			}

			s.handleMessage(ctx, msg)
		}
	}()

	return nil
}

func (s *Subscriber) Stop() {
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *Subscriber) handleMessage(ctx context.Context, msg jetstream.Msg) {
	subj := msg.Subject()
	var err error

	switch subj {
	case "ov.content.written":
		err = s.handleContentWritten(ctx, msg.Data())
	case "ov.content.deleted":
		err = s.handleContentDeleted(ctx, msg.Data())
	case "ov.session.committed":
		err = s.handleSessionCommitted(ctx, msg.Data())
	default:
		slog.Debug("Unhandled subject", "subject", subj)
	}

	if err != nil {
		slog.Error("Failed to process message", "subject", subj, "error", err)
		msg.Nak() // Negative ack
	} else {
		msg.Ack()
	}
}

// Payload structures (internal to adapter)
type contentWrittenPayload struct {
	AccountID string `json:"account_id"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

type sessionCommittedPayload struct {
	AccountID string   `json:"account_id"`
	Paths     []string `json:"paths"`
}

func (s *Subscriber) handleContentWritten(ctx context.Context, data []byte) error {
	var payload contentWrittenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	
	req := dto.UpsertRequest{
		Path:      payload.Path,
		AccountID: payload.AccountID,
		Content:   payload.Content,
	}
	return s.embeddingUC.Upsert(ctx, req)
}

func (s *Subscriber) handleContentDeleted(ctx context.Context, data []byte) error {
	var payload contentWrittenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	req := dto.DeleteRequest{
		Path:      payload.Path,
		AccountID: payload.AccountID,
	}
	return s.embeddingUC.Delete(ctx, req)
}

func (s *Subscriber) handleSessionCommitted(ctx context.Context, data []byte) error {
	var payload sessionCommittedPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	return s.hotnessUC.BoostSession(ctx, payload.AccountID, payload.Paths)
}
