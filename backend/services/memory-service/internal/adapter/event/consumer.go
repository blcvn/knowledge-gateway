package event

import (
    "context"
    "encoding/json"
    "time"

    "github.com/nats-io/nats.go"
    "vnp-memory/services/memory-service/internal/consolidation"
)

type EventConsumer struct {
    pipeline *consolidation.ConsolidationPipeline
}

func (c *EventConsumer) Subscribe(nc *nats.Conn) error {
    _, err := nc.Subscribe("agentmemory.session.ended", c.handleSessionEnded)
    return err
}

func (c *EventConsumer) handleSessionEnded(msg *nats.Msg) {
    var evt struct {
        SessionID        string `json:"session_id"`
        ObservationCount int    `json:"observation_count"`
    }
    json.Unmarshal(msg.Data, &evt)

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        c.pipeline.SummarizeNow(ctx, evt.SessionID)
    }()
}
