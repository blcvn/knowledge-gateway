// Package port defines the input interfaces for the gateway usecase layer.
// Input ports are driven BY adapters (inbound) — adapters call these methods.
package port

import (
	"context"

	"github.com/vnp-community/vnp-memory/gateway/domain"
)

// Router classifies memory content and resolves the target downstream service.
type Router interface {
	// Route classifies the store request and forwards to the appropriate engine.
	Route(ctx context.Context, req *domain.StoreRequest) (*domain.RouteResult, error)
	// Classify determines the memory type from raw content (uses LLM when type="auto").
	Classify(ctx context.Context, data []byte) (string, error)
}

// Authenticator validates credentials and produces an AuthContext.
type Authenticator interface {
	// AuthenticateJWT validates a JWT RS256 token and extracts tenant/user claims.
	AuthenticateJWT(ctx context.Context, token string) (*domain.AuthContext, error)
	// AuthenticateAPIKey resolves an API key (vnp_*) to an AuthContext via KeyStore.
	AuthenticateAPIKey(ctx context.Context, key string) (*domain.AuthContext, error)
}

// MCPHandler dispatches MCP tool calls from AI agents.
type MCPHandler interface {
	// HandleTool executes the named MCP tool with the given parameters.
	HandleTool(ctx context.Context, toolName string, params map[string]any) (any, error)
	// ListTools returns all available MCP tool definitions.
	ListTools(ctx context.Context) ([]ToolDefinition, error)
}

// RateLimiter checks and enforces per-tenant per-endpoint rate limits.
type RateLimiter interface {
	// Check returns whether the request is allowed, plus remaining quota.
	Check(ctx context.Context, tenantID, endpoint string) (allowed bool, remaining int, err error)
}

// ToolDefinition describes an MCP tool exposed to AI agents.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}
