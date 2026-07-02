package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func RunMCP(ctx context.Context, cfg Config) error {
	server := &mcpServer{cfg: cfg, client: NewClient(cfg)}
	return server.serve(ctx, os.Stdin, os.Stdout, os.Stderr)
}

type mcpServer struct {
	cfg    Config
	client KGServiceClient
}

func (s *mcpServer) serve(ctx context.Context, in io.Reader, out io.Writer, log io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		msg, err := readFramedMessage(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(strings.TrimSpace(string(msg))) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.Method == "" {
			continue
		}
		if req.Method == "notifications/initialized" {
			continue
		}
		resp := s.handle(ctx, req)
		if req.ID == nil {
			continue
		}
		if err := writeFramedMessage(out, resp); err != nil {
			return err
		}
		if log != nil {
			fmt.Fprintf(log, "[mcp] handled %s\n", req.Method)
		}
	}
}

func (s *mcpServer) handle(ctx context.Context, req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "codegraph-sync",
					"version": "1.0.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			},
		}
	case "tools/list":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": toolDefinitions(s.cfg),
			},
		}
	case "tools/call":
		return s.callTool(ctx, req)
	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: "unknown method"},
		}
	}
}

func (s *mcpServer) callTool(ctx context.Context, req rpcRequest) rpcResponse {
	params, _ := req.Params["arguments"].(map[string]any)
	name, _ := req.Params["name"].(string)
	switch name {
	case "kg_semantic_search":
		return s.toolSemanticSearch(ctx, req.ID, params)
	case "kg_fulltext_search":
		return s.toolFullTextSearch(ctx, req.ID, params)
	case "kg_code_template_query":
		return s.toolTemplateQuery(ctx, req.ID, params)
	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: "unknown tool"},
		}
	}
}

func (s *mcpServer) toolSemanticSearch(ctx context.Context, id any, params map[string]any) rpcResponse {
	query, _ := params["query"].(string)
	if strings.TrimSpace(query) == "" {
		return validationResponse(id, "query is required")
	}
	topK := intFromAny(params["top_k"], s.cfg.DefaultTopK)
	if topK <= 0 {
		topK = s.cfg.DefaultTopK
	}
	req := SemanticSearchRequest{
		Query:          query,
		DomainIDs:      stringSliceFromAny(params["domain_ids"], []string{s.cfg.KGDomainID}),
		TopK:           topK,
		SemanticWeight: float64FromAny(params["semantic_weight"], 0.7),
		FTSOperator:    stringFromAny(params["fts_operator"], "all_tokens"),
	}
	resp, err := s.client.SemanticSearch(ctx, req)
	if err != nil {
		return toolError(id, err)
	}
	return toolContent(id, summarizeSearch(resp.Results), map[string]any{
		"results":        resp.Results,
		"search_time_ms": resp.SearchTimeMs,
	})
}

func (s *mcpServer) toolFullTextSearch(ctx context.Context, id any, params map[string]any) rpcResponse {
	query, _ := params["query"].(string)
	if strings.TrimSpace(query) == "" {
		return validationResponse(id, "query is required")
	}
	topK := intFromAny(params["top_k"], s.cfg.DefaultTopK)
	if topK <= 0 {
		topK = s.cfg.DefaultTopK
	}
	req := FullTextSearchRequest{
		Query:     query,
		DomainIDs: stringSliceFromAny(params["domain_ids"], []string{s.cfg.KGDomainID}),
		TopK:      topK,
		Mode:      stringFromAny(params["mode"], "all_tokens"),
		Fields:    stringSliceFromAny(params["fields"], nil),
	}
	resp, err := s.client.FullTextSearch(ctx, req)
	if err != nil {
		return toolError(id, err)
	}
	return toolContent(id, summarizeSearch(resp.Results), map[string]any{
		"results":        resp.Results,
		"search_time_ms": resp.SearchTimeMs,
	})
}

func (s *mcpServer) toolTemplateQuery(ctx context.Context, id any, params map[string]any) rpcResponse {
	templateName, _ := params["template_name"].(string)
	if strings.TrimSpace(templateName) == "" {
		return validationResponse(id, "template_name is required")
	}
	rawParams, _ := params["params"].(map[string]any)
	if rawParams == nil {
		rawParams = map[string]any{}
	}
	resp, err := s.client.TemplateQuery(ctx, s.cfg.TemplateDomainID, templateName, rawParams)
	if err != nil {
		return toolError(id, err)
	}
	return toolContent(id, summarizeTemplate(resp.Results), map[string]any{
		"results":       resp.Results,
		"query_time_ms": resp.QueryTimeMs,
	})
}

func toolDefinitions(cfg Config) []map[string]any {
	_ = cfg
	return []map[string]any{
		{
			"name":        "kg_semantic_search",
			"description": "Semantic search over the code-graph domain using the kg-service hybrid search route.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"domain_ids": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"top_k":           map[string]any{"type": "integer", "minimum": 1},
					"semantic_weight": map[string]any{"type": "number"},
					"fts_operator":    map[string]any{"type": "string"},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "kg_fulltext_search",
			"description": "Full-text search over the code-graph domain.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"domain_ids": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"top_k": map[string]any{"type": "integer", "minimum": 1},
					"mode":  map[string]any{"type": "string"},
					"fields": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "kg_code_template_query",
			"description": "Execute a persistent code-graph template for callers, callees, impact, or implements lookup.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"template_name": map[string]any{"type": "string"},
					"params":        map[string]any{"type": "object"},
				},
				"required": []string{"template_name"},
			},
		},
	}
}

func toolContent(id any, summary string, structured map[string]any) rpcResponse {
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": summary},
			},
			"structuredContent": structured,
		},
	}
}

func summarizeSearch(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}
	var b strings.Builder
	for i, result := range results {
		fmt.Fprintf(&b, "%d. %s [%s] score=%.3f\n", i+1, result.NodeID, result.NodeType, result.Score)
	}
	return strings.TrimSpace(b.String())
}

func summarizeTemplate(results []map[string]any) string {
	if len(results) == 0 {
		return "No template results."
	}
	return fmt.Sprintf("Template returned %d result rows.", len(results))
}

func validationResponse(id any, message string) rpcResponse {
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: -32602, Message: message},
	}
}

func toolError(id any, err error) rpcResponse {
	return rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: -32603, Message: err.Error()},
	}
}

type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func readFramedMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			n, convErr := strconv.Atoi(raw)
			if convErr != nil {
				return nil, convErr
			}
			contentLength = n
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	msg := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func writeFramedMessage(out io.Writer, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

func intFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}

func float64FromAny(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return fallback
	}
}

func stringFromAny(value any, fallback string) string {
	if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

func stringSliceFromAny(value any, fallback []string) []string {
	raw, ok := value.([]any)
	if !ok {
		if value == nil {
			return fallback
		}
		return fallback
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
