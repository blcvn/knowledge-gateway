package event

import (
	"context"
	"fmt"
	"log"
	"time"

	domain "github.com/blcvn/ba-shared-libs/pkg/domain"
	"github.com/google/uuid"
)

// SimpleEventEmitter is a simple in-memory event emitter
// In production, this would publish to a message queue (Redis, Kafka, etc.)
type SimpleEventEmitter struct {
	handlers []domain.EventHandler
	events   chan *domain.PlannerEvent
}

func NewSimpleEventEmitter() *SimpleEventEmitter {
	return &SimpleEventEmitter{
		handlers: make([]domain.EventHandler, 0),
		events:   make(chan *domain.PlannerEvent, 100), // Buffer size 100
	}
}

func (e *SimpleEventEmitter) RegisterHandler(handler domain.EventHandler) {
	e.handlers = append(e.handlers, handler)
}

// Start begins processing events in a background goroutine
func (e *SimpleEventEmitter) Start(ctx context.Context) {
	go func() {
		log.Println("[EVENT] Starting event processor...")
		for {
			select {
			case <-ctx.Done():
				log.Println("[EVENT] Stopping event processor...")
				return
			case event := <-e.events:
				e.processEvent(event)
			}
		}
	}()
}

func (e *SimpleEventEmitter) processEvent(event *domain.PlannerEvent) {
	log.Printf("[EVENT] Processing event: type=%s, document_id=%s, tier=%d",
		event.Type, event.DocumentID, event.Tier)

	// Dispatch to all registered handlers
	for _, handler := range e.handlers {
		// Recovery for each handler to prevent panic affecting others
		func(h domain.EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[EVENT] Panic in handler: %v", r)
				}
			}()
			if err := h.Handle(event); err != nil {
				log.Printf("[EVENT] Handler error: %v", err)
			}
		}(handler)
	}
}

func (e *SimpleEventEmitter) Emit(event *domain.PlannerEvent) error {
	// Generate ID if not set
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Push to channel (non-blocking if full)
	select {
	case e.events <- event:
		log.Printf("[EVENT] Emitted event (queued): type=%s, document_id=%s", event.Type, event.DocumentID)
	default:
		log.Printf("[EVENT] Event queue full, dropping event: type=%s", event.Type)
		return fmt.Errorf("event queue full")
	}

	return nil
}
