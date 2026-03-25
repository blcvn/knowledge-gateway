package kafka

import (
	"context"
	"encoding/json"
	"log"

	"kgs-platform/internal/conf"

	"github.com/segmentio/kafka-go"
)

// DocumentIngestedEvent is published by ai-orchestrator after parsing
type DocumentIngestedEvent struct {
	DocID      string         `json:"docId"`
	AppID      string         `json:"appId"`
	DocType    string         `json:"docType"`    // PRD|SRS|UI|TESTCASE
	NodeType   string         `json:"nodeType"`   // KG node type label
	Properties map[string]any `json:"properties"`
	ParentID   string         `json:"parentId,omitempty"`
	EdgeType   string         `json:"edgeType,omitempty"`
}

type GraphUsecase interface {
	CreateNode(ctx context.Context, appID, label string, props map[string]any) (map[string]any, error)
	CreateEdge(ctx context.Context, appID, relationType, sourceNodeID, targetNodeID string, props map[string]any) (map[string]any, error)
}

type Consumer struct {
	reader *kafka.Reader
	graph  GraphUsecase
}

func NewConsumer(c *conf.Data, graph GraphUsecase) *Consumer {
	brokers := []string{"localhost:9092"}
	topic := "document.ingested"
	if c.Kafka != nil {
		if len(c.Kafka.Brokers) > 0 {
			brokers = c.Kafka.Brokers
		}
		if c.Kafka.TopicDocumentIngested != "" {
			topic = c.Kafka.TopicDocumentIngested
		}
	}
	
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: "knowledge-service",
		}),
		graph: graph,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	go func() {
		for {
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("kafka read error: %v", err)
				continue
			}
			var event DocumentIngestedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				continue
			}
			c.handle(ctx, event)
		}
	}()
}

func (c *Consumer) handle(ctx context.Context, e DocumentIngestedEvent) {
	nodeRes, err := c.graph.CreateNode(ctx, e.AppID, e.NodeType, e.Properties)
	if err != nil {
		log.Printf("CreateNode error: %v", err)
		return
	}
	
	nodeID, ok := nodeRes["id"].(string)
	if !ok {
		// fallback if node doesn't return ID directly
		nodeID = e.DocID
	}

	if e.ParentID != "" && e.EdgeType != "" {
		if _, err := c.graph.CreateEdge(ctx, e.AppID, e.EdgeType, e.ParentID, nodeID, nil); err != nil {
			log.Printf("CreateEdge error: %v", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
