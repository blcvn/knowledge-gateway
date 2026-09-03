// Package connector defines domain entities for external data connectors.
//
// Absorbed from: sm-connector (MERGE-P2-T4)
package connector

import "time"

// ConnectorType is the type of external data source.
type ConnectorType string

const (
	ConnectorGitHub ConnectorType = "github"
	ConnectorNotion ConnectorType = "notion"
	ConnectorGDrive ConnectorType = "gdrive"
	ConnectorSlack  ConnectorType = "slack"
	ConnectorWeb    ConnectorType = "web"
)

// Connector represents an external data source configuration.
type Connector struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	Name          string         `json:"name"`
	Type          ConnectorType  `json:"type"`
	Config        map[string]any `json:"config,omitempty"`
	SyncFrequency string         `json:"sync_frequency"` // "hourly"|"daily"|"weekly"|"manual"
	LastSyncAt    *time.Time     `json:"last_sync_at,omitempty"`
	Status        string         `json:"status"` // "active"|"paused"|"error"
	CreatedAt     time.Time      `json:"created_at"`
}

// SyncJob represents an async sync execution.
type SyncJob struct {
	ID          string     `json:"id"`
	ConnectorID string     `json:"connector_id"`
	Status      string     `json:"status"` // "pending"|"running"|"completed"|"failed"
	ItemsSynced int        `json:"items_synced"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}
