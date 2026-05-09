// Package mcp provides the Model Context Protocol server for AI agent integration.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// Server implements MCP protocol with SSE and HTTP Streamable transports.
type Server struct {
	registry port.ServiceRegistry
	tools    map[string]Tool
	logger   *slog.Logger
	sessions sync.Map // sessionID → *Session
}

// Tool defines an MCP tool exposed to AI agents.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Handler     ToolHandler    `json:"-"`
}

// ToolHandler processes a tool call.
type ToolHandler func(ctx context.Context, params map[string]any) (any, error)

// Session tracks an active MCP client session.
type Session struct {
	ID        string
	CreatedAt time.Time
	outCh     chan []byte
}

// NewServer creates a new MCP server with all 16 tools registered.
func NewServer(registry port.ServiceRegistry, logger *slog.Logger) *Server {
	s := &Server{
		registry: registry,
		tools:    make(map[string]Tool),
		logger:   logger,
	}
	s.registerTools()
	return s
}

// registerTools registers all 16 MCP tools.
func (s *Server) registerTools() {
	tools := []Tool{
		{Name: "memory_store", Description: "Store a memory with auto-classification and routing to the appropriate engine", InputSchema: objectSchema("content", "type", "metadata"), Handler: s.forwardTool("cognee-ingestion")},
		{Name: "memory_recall", Description: "Cross-engine semantic recall combining results from all memory engines", InputSchema: objectSchema("query", "engines", "limit"), Handler: s.forwardTool("vnp-search-hub")},
		{Name: "memory_search", Description: "Semantic search across the knowledge graph", InputSchema: objectSchema("query", "strategy", "limit"), Handler: s.forwardTool("cognee-search")},
		{Name: "memory_timeline", Description: "Query temporal events for a user", InputSchema: objectSchema("user_id", "from", "to"), Handler: s.forwardTool("vnp-event")},
		{Name: "memory_profile", Description: "Get user profile from memory context", InputSchema: objectSchema("user_id"), Handler: s.forwardTool("memobase-context")},
		{Name: "memory_forget", Description: "Delete memory across all engines (cascading)", InputSchema: objectSchema("target_id", "engines"), Handler: s.forwardTool("vnp-event")},
		{Name: "graph_query", Description: "Query the knowledge graph with filters", InputSchema: objectSchema("query", "filters"), Handler: s.forwardTool("graphiti-store")},
		{Name: "ov_read_file", Description: "Read a file from the context database", InputSchema: objectSchema("path"), Handler: s.forwardTool("ov-fs")},
		{Name: "ov_write_file", Description: "Write content to a file in the context database", InputSchema: objectSchema("path", "content"), Handler: s.forwardTool("ov-fs")},
		{Name: "ov_search", Description: "Hierarchical semantic search across files and context", InputSchema: objectSchema("query", "context_level"), Handler: s.forwardTool("ov-search")},
		{Name: "ov_list_dir", Description: "List directory contents in the context database", InputSchema: objectSchema("path"), Handler: s.forwardTool("ov-fs")},
		{Name: "ov_grep", Description: "Search file contents with regex pattern", InputSchema: objectSchema("pattern", "path"), Handler: s.forwardTool("ov-fs")},
		{Name: "ov_tree", Description: "Show directory tree structure", InputSchema: objectSchema("path", "depth"), Handler: s.forwardTool("ov-fs")},
		{Name: "ov_session_commit", Description: "Commit an editing session to persistent storage", InputSchema: objectSchema("session_id"), Handler: s.forwardTool("ov-session")},
		{Name: "ov_ingest", Description: "Ingest a resource into the context database", InputSchema: objectSchema("path", "type"), Handler: s.forwardTool("ov-resource")},
		{Name: "ov_delete", Description: "Delete a file or resource from the context database", InputSchema: objectSchema("path"), Handler: s.forwardTool("ov-fs")},
	}

	for _, t := range tools {
		s.tools[t.Name] = t
	}
	s.logger.Info("MCP tools registered", "count", len(s.tools))
}

// forwardTool creates a handler that forwards the tool call to a downstream gRPC service.
func (s *Server) forwardTool(service string) ToolHandler {
	return func(ctx context.Context, params map[string]any) (any, error) {
		target, err := s.registry.Resolve(service)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", service, err)
		}

		body, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}

		resp, err := s.registry.Forward(ctx, target, body)
		if err != nil {
			return nil, err
		}

		var result any
		if err := json.Unmarshal(resp, &result); err != nil {
			return string(resp), nil
		}
		return result, nil
	}
}

// HandleSSE handles Server-Sent Events transport for MCP.
func (s *Server) HandleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	session := &Session{
		ID:        fmt.Sprintf("mcp-%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
		outCh:     make(chan []byte, 64),
	}
	s.sessions.Store(session.ID, session)
	defer s.sessions.Delete(session.ID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-MCP-Session-ID", session.ID)

	// Send endpoint event
	endpointEvent := fmt.Sprintf("event: endpoint\ndata: /mcp/message?session_id=%s\n\n", session.ID)
	fmt.Fprint(w, endpointEvent)
	flusher.Flush()

	s.logger.Info("MCP SSE session started", "session_id", session.ID)

	for {
		select {
		case <-r.Context().Done():
			s.logger.Info("MCP SSE session ended", "session_id", session.ID)
			return
		case msg := <-session.outCh:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

// HandleMessage handles JSON-RPC 2.0 messages for MCP.
func (s *Server) HandleMessage(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, 0, -32700, "parse error")
		return
	}
	defer r.Body.Close()

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "tools/list":
		s.handleToolsList(w, req)
	case "tools/call":
		s.handleToolsCall(w, r.Context(), req)
	case "ping":
		writeJSONRPCResult(w, req.ID, map[string]string{})
	default:
		writeJSONRPCError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, req JSONRPCRequest) {
	writeJSONRPCResult(w, req.ID, map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "vnp-gateway",
			"version": "1.0.0",
		},
	})
}

func (s *Server) handleToolsList(w http.ResponseWriter, req JSONRPCRequest) {
	toolList := make([]map[string]any, 0, len(s.tools))
	for _, t := range s.tools {
		toolList = append(toolList, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	writeJSONRPCResult(w, req.ID, map[string]any{"tools": toolList})
}

func (s *Server) handleToolsCall(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	b, _ := json.Marshal(req.Params)
	json.Unmarshal(b, &params)

	tool, ok := s.tools[params.Name]
	if !ok {
		writeJSONRPCError(w, req.ID, -32602, "unknown tool: "+params.Name)
		return
	}

	s.logger.Info("MCP tool call", "tool", params.Name)

	result, err := tool.Handler(ctx, params.Arguments)
	if err != nil {
		s.logger.Error("MCP tool error", "tool", params.Name, "error", err)
		writeJSONRPCResult(w, req.ID, map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Error: " + err.Error()},
			},
			"isError": true,
		})
		return
	}

	text, _ := json.Marshal(result)
	writeJSONRPCResult(w, req.ID, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(text)},
		},
	})
}

// --- JSON-RPC types ---

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

func writeJSONRPCResult(w http.ResponseWriter, id int, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func writeJSONRPCError(w http.ResponseWriter, id int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": message},
	})
}

// objectSchema builds a simple JSON Schema object for MCP tool input.
func objectSchema(fields ...string) map[string]any {
	props := make(map[string]any, len(fields))
	for _, f := range fields {
		props[f] = map[string]any{"type": "string"}
	}
	required := []string{}
	if len(fields) > 0 {
		required = fields[:1] // first field is required
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

// Handler returns the HTTP handler for the MCP server (to be mounted on its own port).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sse", s.HandleSSE)
	mux.HandleFunc("POST /message", s.HandleMessage)
	// Also support /mcp/ prefix
	mux.HandleFunc("GET /mcp/sse", s.HandleSSE)
	mux.HandleFunc("POST /mcp/message", s.HandleMessage)
	return mux
}
