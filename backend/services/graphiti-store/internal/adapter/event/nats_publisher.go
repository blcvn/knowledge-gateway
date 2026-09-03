// Package event implements the EventPublisher port via NATS JetStream.
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

const (
	// SubjectBulkSaved is the NATS subject for bulk save events.
	SubjectBulkSaved = "graphiti.store.bulk.saved"
)

// BulkSavedEvent is published when a bulk save completes.
type BulkSavedEvent struct {
	GroupID   string `json:"group_id"`
	EpisodeID string `json:"episode_id"`
	NodeCount int    `json:"node_count"`
	EdgeCount int    `json:"edge_count"`
}

// NATSPublisher publishes domain events to NATS JetStream.
type NATSPublisher struct {
	js     nats.JetStreamContext
	logger *slog.Logger
}

// NewNATSPublisher creates a publisher with the given JetStream context.
func NewNATSPublisher(js nats.JetStreamContext, logger *slog.Logger) *NATSPublisher {
	return &NATSPublisher{
		js:     js,
		logger: logger.With("adapter", "nats_publisher"),
	}
}

// PublishBulkSaved publishes an event when a bulk save completes.
func (p *NATSPublisher) PublishBulkSaved(ctx context.Context, groupID, episodeID string, nodeCount, edgeCount int) error {
	event := BulkSavedEvent{
		GroupID:   groupID,
		EpisodeID: episodeID,
		NodeCount: nodeCount,
		EdgeCount: edgeCount,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	ack, err := p.js.Publish(SubjectBulkSaved, data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", SubjectBulkSaved, err)
	}

	p.logger.Info("event published",
		"subject", SubjectBulkSaved,
		"episode_id", episodeID,
		"group_id", groupID,
		"nodes", nodeCount,
		"edges", edgeCount,
		"stream", ack.Stream,
		"sequence", ack.Sequence,
	)
	return nil
}
