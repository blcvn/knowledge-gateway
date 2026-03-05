package data

import (
	"context"
	"testing"
)

func TestNATSClientPublishSubscribe(t *testing.T) {
	client := &NATSClient{
		url:    "nats://localhost:4222",
		stream: "kgs-events",
		subs:   map[string]natsSubscription{},
	}
	called := 0
	stop, err := client.Subscribe("session.close.*", func(ctx context.Context, payload []byte) {
		called++
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer stop()

	if err := client.Publish(context.Background(), "session.close.s-1", []byte(`{"session_id":"s-1"}`)); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected callback called once, got %d", called)
	}

	stop()
	if err := client.Publish(context.Background(), "session.close.s-2", []byte(`{}`)); err != nil {
		t.Fatalf("Publish failed after unsubscribe: %v", err)
	}
	if called != 1 {
		t.Fatalf("callback should not be called after unsubscribe, got=%d", called)
	}
}
