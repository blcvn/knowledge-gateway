package event

import (
	// "github.com/nats-io/nats.go"
)

type NatsSubscriber struct {
	// nc *nats.Conn
	// js nats.JetStreamContext
}

func NewNatsSubscriber() *NatsSubscriber {
	return &NatsSubscriber{}
}

func (s *NatsSubscriber) Subscribe() error {
	// s.js.Subscribe("ov.crypto.key.rotated", s.handleKeyRotated)
	// s.js.Subscribe("ov.session.memory.extracted", s.handleMemoryExtracted)
	return nil
}

func (s *NatsSubscriber) handleKeyRotated( /* m *nats.Msg */ ) {
	// Parse payload, then trigger re-wrap logic via usecase
}

func (s *NatsSubscriber) handleMemoryExtracted( /* m *nats.Msg */ ) {
	// Parse payload, then trigger WriteFile usecase
}
