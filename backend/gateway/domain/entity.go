// Package domain contains the core domain types for vnp-gateway.
// This package has ZERO external dependencies — only Go stdlib.
package domain

import "time"

// AuthContext holds authenticated identity information extracted from JWT or API Key.
type AuthContext struct {
	TenantID string   `json:"tenant_id"`
	UserID   string   `json:"user_id"`
	Roles    []string `json:"roles"`
	Scopes   []string `json:"scopes"`
	RateTier string   `json:"rate_tier"` // "free", "pro", "enterprise"
}

// TenantContext provides tenant-level configuration.
type TenantContext struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	RateTier  string    `json:"rate_tier"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// RouteTarget describes a downstream gRPC service endpoint.
type RouteTarget struct {
	Service string        `json:"service"`
	Address string        `json:"address"`
	Timeout time.Duration `json:"timeout"`
	Method  string        `json:"method"` // gRPC method path
}

// ProtocolType identifies the inbound protocol.
type ProtocolType int

const (
	ProtocolREST      ProtocolType = iota // HTTP REST via chi/v5
	ProtocolGRPC                          // gRPC-Web proxy
	ProtocolMCP                           // MCP SSE/HTTP Streamable
	ProtocolWebDAV                        // WebDAV proxy to ov-fs
	ProtocolWebSocket                     // WebSocket streaming
)

// String returns the human-readable protocol name.
func (p ProtocolType) String() string {
	switch p {
	case ProtocolREST:
		return "REST"
	case ProtocolGRPC:
		return "gRPC"
	case ProtocolMCP:
		return "MCP"
	case ProtocolWebDAV:
		return "WebDAV"
	case ProtocolWebSocket:
		return "WebSocket"
	default:
		return "Unknown"
	}
}

// StoreRequest represents a unified memory storage request with auto-routing support.
type StoreRequest struct {
	Type     string            `json:"type"` // "auto","semantic","episodic","conversational","profile","procedural"
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata,omitempty"`
	SourceID string            `json:"source_id,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
}

// RouteResult holds the outcome of a routed request.
type RouteResult struct {
	ID        string `json:"id"`
	Engine    string `json:"engine"`
	Status    string `json:"status"`
	Body      []byte `json:"-"`
	LatencyMs int64  `json:"latency_ms"`
}

// RateTier constants.
const (
	RateTierFree       = "free"
	RateTierPro        = "pro"
	RateTierEnterprise = "enterprise"
)

// MemoryType constants for auto-routing classification.
const (
	MemoryTypeSemantic       = "semantic"
	MemoryTypeEpisodic       = "episodic"
	MemoryTypeConversational = "conversational"
	MemoryTypeProfile        = "profile"
	MemoryTypeProcedural     = "procedural"
	MemoryTypeAuto           = "auto"
)

// ForwardRequest encapsulates an HTTP request being forwarded to a downstream service.
// The service uses Path and HTTPMethod to route to the correct internal handler.
type ForwardRequest struct {
	Path        string            `json:"path"`
	HTTPMethod  string            `json:"http_method"`
	Body        []byte            `json:"body,omitempty"`
	PathParams  map[string]string `json:"path_params,omitempty"`
	QueryParams map[string]string `json:"query_params,omitempty"`
}
