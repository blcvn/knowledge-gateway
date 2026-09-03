// Package grpc provides stub handlers for sm-connector console endpoints.
// Returns mock data matching UI's adaptive.ts types.
package grpc

import (
	"context"
	"encoding/json"
	"time"
)

// SmConnectorHandler provides stub console endpoints for connector management.
type SmConnectorHandler struct{}

func NewSmConnectorHandler() *SmConnectorHandler {
	return &SmConnectorHandler{}
}

type ExternalConnector struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	LastSync      string `json:"last_sync"`
	DocumentCount int    `json:"document_count"`
	SyncFrequency string `json:"sync_frequency"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

// ListConnectors returns stub external connectors.
func (h *SmConnectorHandler) ListConnectors(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := []ExternalConnector{
		{ID: "conn-001", Type: "google_drive", Status: "Connected", LastSync: now, DocumentCount: 142, SyncFrequency: "1h"},
		{ID: "conn-002", Type: "notion", Status: "Connected", LastSync: now, DocumentCount: 89, SyncFrequency: "30m"},
		{ID: "conn-003", Type: "github", Status: "Error", LastSync: now, DocumentCount: 0, SyncFrequency: "2h", ErrorMessage: "Rate limit exceeded"},
	}
	return json.Marshal(data)
}

// CreateConnector creates a stub connector.
func (h *SmConnectorHandler) CreateConnector(_ context.Context, connectorType, syncFrequency string) ([]byte, error) {
	data := ExternalConnector{
		ID:            "conn-new-001",
		Type:          connectorType,
		Status:        "Disconnected",
		LastSync:      "",
		DocumentCount: 0,
		SyncFrequency: syncFrequency,
	}
	return json.Marshal(data)
}

// SyncConnector triggers a stub sync.
func (h *SmConnectorHandler) SyncConnector(_ context.Context, _ string) ([]byte, error) {
	return json.Marshal(map[string]string{"status": "syncing"})
}
