package grpc

import (
	"context"
)

type ConsoleGraphHandler struct{}

func NewConsoleGraphHandler() *ConsoleGraphHandler {
	return &ConsoleGraphHandler{}
}

func (h *ConsoleGraphHandler) HandleSubgraph(_ context.Context) ([]byte, error) {
	// Dummy subgraph response matching what UI expects
	return []byte(`{"nodes": [{"id": "node-1", "label": "MockEntity", "properties": {"name": "Mock"}}], "edges": []}`), nil
}

func (h *ConsoleGraphHandler) HandleGetEntity(_ context.Context) ([]byte, error) {
	return []byte(`{"id": "node-1", "label": "MockEntity", "properties": {"name": "Mock"}}`), nil
}

func (h *ConsoleGraphHandler) HandleTimeline(_ context.Context) ([]byte, error) {
	return []byte(`{"events": []}`), nil
}

func (h *ConsoleGraphHandler) HandleQuery(_ context.Context) ([]byte, error) {
	return []byte(`{"nodes": [], "edges": []}`), nil
}
