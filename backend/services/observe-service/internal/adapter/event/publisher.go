package event

import (
    "context"
    "encoding/json"
    "github.com/nats-io/nats.go"
)

type NATSPublisher struct {
    conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
    return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) Publish(ctx context.Context, subject string, payload any) error {
    data, err := json.Marshal(payload)
    if err != nil { return err }
    return p.conn.Publish(subject, data)
}
