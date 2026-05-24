// Package grpc provides stub handlers for sm-engine console endpoints.
// Returns mock data matching UI's adaptive.ts types.
package grpc

import (
	"context"
	"encoding/json"
)

// SmEngineHandler provides stub console endpoints for adaptive analytics and forget rules.
type SmEngineHandler struct{}

func NewSmEngineHandler() *SmEngineHandler {
	return &SmEngineHandler{}
}

type AdaptiveAnalytics struct {
	CreationRate       float64 `json:"creation_rate"`
	DeletionRate       float64 `json:"deletion_rate"`
	ContradictionCount int     `json:"contradiction_count"`
	ConnectorSyncCount int     `json:"connector_sync_count"`
	StorageUsageBytes  int64   `json:"storage_usage_bytes"`
}

type ForgetRule struct {
	ID                      string `json:"id"`
	MemoryType              string `json:"memory_type"`
	ForgetAfter             string `json:"forget_after"`
	NoiseFilter             bool   `json:"noise_filter"`
	ContradictionResolution string `json:"contradiction_resolution"`
}

// GetAnalytics returns stub analytics data.
func (h *SmEngineHandler) GetAnalytics(_ context.Context) ([]byte, error) {
	data := AdaptiveAnalytics{
		CreationRate:       12.5,
		DeletionRate:       2.3,
		ContradictionCount: 7,
		ConnectorSyncCount: 34,
		StorageUsageBytes:  1073741824, // 1GB
	}
	return json.Marshal(data)
}

// GetForgetRules returns stub forget rules.
func (h *SmEngineHandler) GetForgetRules(_ context.Context) ([]byte, error) {
	data := []ForgetRule{
		{ID: "rule-001", MemoryType: "dynamic", ForgetAfter: "90d", NoiseFilter: true, ContradictionResolution: "keep_latest"},
		{ID: "rule-002", MemoryType: "static", ForgetAfter: "365d", NoiseFilter: false, ContradictionResolution: "keep_both"},
	}
	return json.Marshal(data)
}

// UpdateForgetRules updates stub forget rules (returns same input).
func (h *SmEngineHandler) UpdateForgetRules(_ context.Context, rules []ForgetRule) ([]byte, error) {
	return json.Marshal(rules)
}
