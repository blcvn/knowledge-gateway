package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"strings"

	"kg-service/internal/access"
)

type Handler struct {
	service *Service
	limiter access.Limiter
}

func NewHandler(service *Service, limiter ...access.Limiter) Handler {
	var rateLimiter access.Limiter
	if len(limiter) > 0 {
		rateLimiter = limiter[0]
	}
	return Handler{service: service, limiter: rateLimiter}
}

func (h Handler) Connect(w http.ResponseWriter, r *http.Request) {
	identity, _ := access.IdentityFromContext(r.Context())
	session := h.service.CreateSession(identity)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "event: session\ndata: {\"session_id\":%q}\n\n", session.SessionID)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (h Handler) Message(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["session_id"]
	identity, ok := h.service.IdentityForSession(sessionID)
	if !ok {
		writeRPCError(w, nil, -32000, "invalid session")
		return
	}
	if h.limiter != nil && !h.limiter.Allow(identity) {
		writeRPCError(w, nil, -32029, "rate limit exceeded", map[string]any{"tenant_id": identity.TenantID})
		return
	}

	var req JSONRPCRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, nil, -32600, "invalid request")
		return
	}

	switch req.Method {
	case "tools/list":
		writeRPCResult(w, req.ID, map[string]any{"tools": h.service.ListTools()})
	case "tools/call":
		name, _ := req.Params["name"].(string)
		args, _ := req.Params["arguments"].(map[string]any)
		result, rpcErr := h.service.CallTool(identity, strings.TrimSpace(name), args)
		if rpcErr != nil {
			writeRPCError(w, req.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data)
			return
		}
		writeRPCResult(w, req.ID, result)
	default:
		writeRPCError(w, req.ID, -32601, "method not found")
	}
}

func writeRPCResult(w http.ResponseWriter, id any, result any) {
	writeRPC(w, JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeRPCError(w http.ResponseWriter, id any, code int, message string, data ...map[string]any) {
	var payload map[string]any
	if len(data) > 0 {
		payload = data[0]
	}
	writeRPC(w, JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    payload,
		},
	})
}

func writeRPC(w http.ResponseWriter, resp JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
