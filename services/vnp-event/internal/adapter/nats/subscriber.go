// Package nats implements NATS subscriber for cross-engine events.
// Listens to 6 engine completion events and creates UserEvent entries.
package nats

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/services/vnp-event/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/vnp-event/internal/usecase"
)

// engineSubject maps NATS subjects to EventSource values.
var engineSubjects = map[string]model.EventSource{
	"cognee.pipeline.completed":    model.SourceCognee,
	"graphiti.pipeline.completed":  model.SourceGraphiti,
	"memobase.pipeline.flush":     model.SourceMemobase,
	"ov.storage.resource.parsed":  model.SourceOpenViking,
	"zep.core.memory.enriched":    model.SourceZep,
	"sm.engine.document.saved":    model.SourceSupermemory,
}

// Subscriber listens to engine completion events.
type Subscriber struct {
	conn     *nats.Conn
	eventSvc *usecase.EventService
	subs     []*nats.Subscription
}

func NewSubscriber(conn *nats.Conn, eventSvc *usecase.EventService) *Subscriber {
	return &Subscriber{conn: conn, eventSvc: eventSvc}
}

type engineEvent struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id,omitempty"`
	EntityID string `json:"entity_id"`
	Content  string `json:"content,omitempty"`
}

// Subscribe registers listeners for all 6 engine subjects.
func (s *Subscriber) Subscribe() error {
	for subject, source := range engineSubjects {
		src := source // capture
		sub, err := s.conn.Subscribe(subject, func(msg *nats.Msg) {
			s.handleMessage(src, msg.Data)
		})
		if err != nil {
			return err
		}
		s.subs = append(s.subs, sub)
		log.Printf("vnp-event: subscribed to %s → %s", subject, source)
	}
	return nil
}

func (s *Subscriber) handleMessage(source model.EventSource, data []byte) {
	var evt engineEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		log.Printf("vnp-event: unmarshal error from %s: %v", source, err)
		return
	}

	tenantID, _ := uuid.Parse(evt.TenantID)
	userID, _ := uuid.Parse(evt.UserID)
	content := evt.Content
	if content == "" {
		content = string(source) + " pipeline completed for entity " + evt.EntityID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := s.eventSvc.CreateEvent(ctx, tenantID, userID, source, content, []string{string(source)}, time.Now())
	if err != nil {
		log.Printf("vnp-event: create event from %s failed: %v", source, err)
	}
}

// Close unsubscribes from all subjects.
func (s *Subscriber) Close() {
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
}
