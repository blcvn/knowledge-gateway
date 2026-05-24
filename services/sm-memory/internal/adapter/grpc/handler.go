// Package grpc provides stub handlers for sm-memory console endpoints.
// Returns mock data matching UI's adaptive.ts types.
package grpc

import (
	"context"
	"encoding/json"
	"time"
)

// SmMemoryHandler provides stub console endpoints for adaptive memory management.
type SmMemoryHandler struct{}

func NewSmMemoryHandler() *SmMemoryHandler {
	return &SmMemoryHandler{}
}

type AdaptiveMemory struct {
	ID           string `json:"id"`
	Content      string `json:"content"`
	MemoryType   string `json:"memory_type"`
	VersionCount int    `json:"version_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type MemoryVersion struct {
	ID            string `json:"id"`
	MemoryID      string `json:"memory_id"`
	Content       string `json:"content"`
	VersionNumber int    `json:"version_number"`
	IsLatest      bool   `json:"is_latest"`
	Diff          string `json:"diff,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// ListMemories returns stub adaptive memories.
func (h *SmMemoryHandler) ListMemories(_ context.Context) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := []AdaptiveMemory{
		{ID: "mem-001", Content: "User prefers dark mode interfaces", MemoryType: "static", VersionCount: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "mem-002", Content: "User is working on a Go microservices project", MemoryType: "dynamic", VersionCount: 3, CreatedAt: now, UpdatedAt: now},
		{ID: "mem-003", Content: "User's timezone is UTC+7", MemoryType: "static", VersionCount: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "mem-004", Content: "User frequently asks about gRPC and protobuf", MemoryType: "dynamic", VersionCount: 5, CreatedAt: now, UpdatedAt: now},
	}
	return json.Marshal(data)
}

// GetVersions returns stub version history for a memory.
func (h *SmMemoryHandler) GetVersions(_ context.Context, memoryID string) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	data := []MemoryVersion{
		{ID: "v-001", MemoryID: memoryID, Content: "User is working on a Go project", VersionNumber: 1, IsLatest: false, CreatedAt: now},
		{ID: "v-002", MemoryID: memoryID, Content: "User is working on a Go microservices project", VersionNumber: 2, IsLatest: false, Diff: "+microservices", CreatedAt: now},
		{ID: "v-003", MemoryID: memoryID, Content: "User is working on a Go microservices project with gRPC", VersionNumber: 3, IsLatest: true, Diff: "+with gRPC", CreatedAt: now},
	}
	return json.Marshal(data)
}
