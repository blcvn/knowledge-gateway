package event

import (
	"context"
	"encoding/json"
	"fmt"

	"openviking.com/ov-resource/internal/domain"
)

type natsPublisher struct {
	url string
}

func NewNatsPublisher(url string) *natsPublisher {
	return &natsPublisher{url: url}
}

func (p *natsPublisher) PublishResourceIngested(ctx context.Context, event domain.ResourceIngested) error {
	// Stub NATS publishing
	data, _ := json.Marshal(event)
	fmt.Printf("[NATS] Published ov.resource.ingested: %s\n", string(data))
	return nil
}
