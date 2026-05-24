// Package forward provides the ForwardService gRPC server implementation.
// This is a server-side router that receives forwarded requests from the gateway
// and dispatches them to the correct internal handler based on the HTTP path.
package forward

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ForwardRequest matches the proto ForwardRequest message.
type ForwardRequest struct {
	Method      string            `json:"method"`
	Body        []byte            `json:"body"`
	Path        string            `json:"path"`
	HTTPMethod  string            `json:"http_method"`
	PathParams  map[string]string `json:"path_params"`
	QueryParams map[string]string `json:"query_params"`
}

// ForwardResponse matches the proto ForwardResponse message.
type ForwardResponse struct {
	Body       []byte `json:"body"`
	StatusCode int32  `json:"status_code"`
	Error      string `json:"error,omitempty"`
}

// HandlerFunc is the signature for internal route handlers.
type HandlerFunc func(ctx context.Context, body []byte, params map[string]string) ([]byte, error)

// Router dispatches forwarded requests to registered handlers based on HTTP path pattern.
type Router struct {
	routes map[string]HandlerFunc // "METHOD /pattern" -> handler
	logger *slog.Logger
}

// NewRouter creates a new forward service router.
func NewRouter(logger *slog.Logger) *Router {
	return &Router{
		routes: make(map[string]HandlerFunc),
		logger: logger,
	}
}

// Handle registers a handler for a given HTTP method and path pattern.
// Pattern supports simple suffix matching: "/v1/console/dashboard/*" matches anything
// starting with that prefix.
func (r *Router) Handle(method, pattern string, handler HandlerFunc) {
	key := method + " " + pattern
	r.routes[key] = handler
	r.logger.Debug("registered forward route", "key", key)
}

// match finds the best matching handler for the given method and path.
func (r *Router) match(method, path string) (HandlerFunc, map[string]string) {
	// Exact match first
	key := method + " " + path
	if h, ok := r.routes[key]; ok {
		return h, nil
	}

	// Prefix match with wildcard patterns
	bestLen := 0
	var bestHandler HandlerFunc
	for pattern, handler := range r.routes {
		parts := strings.SplitN(pattern, " ", 2)
		if len(parts) != 2 {
			continue
		}
		pMethod, pPath := parts[0], parts[1]
		if pMethod != method {
			continue
		}
		// Check if pattern ends with /* (wildcard)
		if strings.HasSuffix(pPath, "/*") {
			prefix := strings.TrimSuffix(pPath, "/*")
			if strings.HasPrefix(path, prefix) && len(prefix) > bestLen {
				bestLen = len(prefix)
				bestHandler = handler
			}
		}
	}

	return bestHandler, nil
}

// Forward implements the ForwardService.Forward RPC.
// This method is called by the gateway's gRPC client via conn.Invoke().
func (r *Router) Forward(ctx context.Context, req []byte) ([]byte, error) {
	// Extract routing metadata from gRPC context
	md, _ := metadata.FromIncomingContext(ctx)
	path := getMetadataValue(md, "x-forward-path")
	method := getMetadataValue(md, "x-forward-method")

	r.logger.Debug("forward received", "path", path, "method", method, "body_size", len(req))

	// Find matching handler
	handler, params := r.match(method, path)
	if handler == nil {
		r.logger.Warn("no handler found", "path", path, "method", method)
		return json.Marshal(map[string]interface{}{
			"error": fmt.Sprintf("no handler for %s %s", method, path),
		})
	}

	// Execute handler
	resp, err := handler(ctx, req, params)
	if err != nil {
		r.logger.Error("handler error", "path", path, "error", err)
		return json.Marshal(map[string]interface{}{
			"error": err.Error(),
		})
	}

	return resp, nil
}

// RegisterForwardService registers the router as a gRPC service handler.
// This uses the generic service registration pattern compatible with the
// ForwardService.Forward RPC method path.
func RegisterForwardService(server *grpc.Server, router *Router) {
	// Register a generic service descriptor that handles the Forward RPC
	sd := grpc.ServiceDesc{
		ServiceName: "vnp.gateway.forward.v1.ForwardService",
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Forward",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					var req []byte
					if err := dec(&req); err != nil {
						return nil, err
					}
					if interceptor == nil {
						resp, err := router.Forward(ctx, req)
						return resp, err
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/vnp.gateway.forward.v1.ForwardService/Forward",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return router.Forward(ctx, req.([]byte))
					}
					return interceptor(ctx, req, info, handler)
				},
			},
		},
		Streams: []grpc.StreamDesc{},
	}
	server.RegisterService(&sd, nil)
}

func getMetadataValue(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}
