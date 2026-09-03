package event

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"vnp-memory/services/graphiti-search/internal/domain"
)

type mockCacheRepo struct {
	invalidatedGroupID string
}

func (m *mockCacheRepo) Get(ctx context.Context, key string) ([]domain.RankedResult, error) {
	return nil, nil
}
func (m *mockCacheRepo) Set(ctx context.Context, key string, val []domain.RankedResult, expiration time.Duration) error {
	return nil
}
func (m *mockCacheRepo) InvalidateGroup(ctx context.Context, groupID string) error {
	m.invalidatedGroupID = groupID
	return nil
}
func (m *mockCacheRepo) InvalidatePattern(ctx context.Context, pattern string) error {
	m.invalidatedGroupID = pattern
	return nil
}

func TestNatsSubscriber_Listen(t *testing.T) {
	// Start a mock NATS server or skip if unavailable
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS server not running, skipping test")
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Skip("JetStream not available, skipping test")
	}

	mockRepo := &mockCacheRepo{}
	sub := NewNatsSubscriber(nc, mockRepo)

	// Add stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "graphiti_search_events",
		Subjects: []string{"graphiti.episode.ingested"},
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		t.Fatalf("Failed to add stream: %v", err)
	}

	err = sub.Listen(context.Background())
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	// Publish test message
	payload := []byte(`{"group_id":"tenant-999"}`)
	_, err = js.Publish("graphiti.episode.ingested", payload)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	if mockRepo.invalidatedGroupID != "tenant-999" {
		t.Errorf("Expected cache invalidation for tenant-999, got %s", mockRepo.invalidatedGroupID)
	}
}
