package event

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
)

const subjectDataIngested = "cognee.ingestion.data.ingested"

// DataIngestedEvent mirrors the event published by cognee-ingestion.
type DataIngestedEvent struct {
	DatasetID string   `json:"dataset_id"`
	TenantID  string   `json:"tenant_id"`
	EntryIDs  []string `json:"entry_ids"`
	NodeSets  []string `json:"node_sets"` // [NEW] CR-002
}

// CognifyJob represents a pipeline job queued for processing.
type CognifyJob struct {
	DatasetID string
	TenantID  string
	EntryIDs  []string
	NodeSets  []string // [NEW] CR-002 — propagated from ingestion event
	Steps     []string // for CR-006 partial pipeline
}

// Subscriber subscribes to NATS events and enqueues cognify jobs.
type Subscriber struct {
	nc       *nats.Conn
	jobQueue chan<- CognifyJob
	logger   *slog.Logger
}

// NewSubscriber creates a new NATS subscriber.
func NewSubscriber(nc *nats.Conn, jobQueue chan<- CognifyJob, logger *slog.Logger) *Subscriber {
	return &Subscriber{nc: nc, jobQueue: jobQueue, logger: logger}
}

// Subscribe registers the NATS subscription for data ingested events.
func (s *Subscriber) Subscribe(ctx context.Context) error {
	_, err := s.nc.Subscribe(subjectDataIngested, func(msg *nats.Msg) {
		s.handleDataIngested(msg)
	})
	return err
}

// handleDataIngested processes an incoming DataIngested NATS message.
func (s *Subscriber) handleDataIngested(msg *nats.Msg) {
	var evt DataIngestedEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		s.logger.Error("failed to unmarshal DataIngestedEvent", "error", err)
		return
	}

	job := CognifyJob{
		DatasetID: evt.DatasetID,
		TenantID:  evt.TenantID,
		EntryIDs:  evt.EntryIDs,
		NodeSets:  evt.NodeSets, // [NEW] propagate to pipeline
	}

	select {
	case s.jobQueue <- job:
		s.logger.Info("cognify job enqueued", "dataset_id", evt.DatasetID, "node_sets", evt.NodeSets)
	default:
		s.logger.Warn("cognify job queue full, dropping job", "dataset_id", evt.DatasetID)
	}
}
