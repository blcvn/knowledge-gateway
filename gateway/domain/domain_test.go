package domain_test

import (
	"testing"

	"github.com/vnp-community/vnp-memory/gateway/domain"
)

func TestGatewayError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *domain.GatewayError
		want string
	}{
		{"unauthenticated", domain.ErrUnauthenticated, "[UNAUTHENTICATED] missing or invalid authentication"},
		{"forbidden", domain.ErrForbidden, "[PERMISSION_DENIED] insufficient permissions"},
		{"not_found", domain.ErrNotFound, "[NOT_FOUND] resource not found"},
		{"invalid_argument", domain.ErrInvalidArgument, "[INVALID_ARGUMENT] invalid request parameters"},
		{"rate_limited", domain.ErrRateLimited, "[RESOURCE_EXHAUSTED] rate limit exceeded"},
		{"circuit_open", domain.ErrCircuitOpen, "[UNAVAILABLE] service temporarily unavailable"},
		{"timeout", domain.ErrTimeout, "[DEADLINE_EXCEEDED] request timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGatewayError_HTTPStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  *domain.GatewayError
		want int
	}{
		{"unauthenticated → 401", domain.ErrUnauthenticated, 401},
		{"forbidden → 403", domain.ErrForbidden, 403},
		{"not_found → 404", domain.ErrNotFound, 404},
		{"invalid_argument → 400", domain.ErrInvalidArgument, 400},
		{"rate_limited → 429", domain.ErrRateLimited, 429},
		{"circuit_open → 503", domain.ErrCircuitOpen, 503},
		{"timeout → 504", domain.ErrTimeout, 504},
		{"unknown → 500", &domain.GatewayError{Code: "CUSTOM"}, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.HTTPStatusCode(); got != tt.want {
				t.Errorf("HTTPStatusCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGatewayError_WithMessage(t *testing.T) {
	original := domain.ErrUnauthenticated
	modified := original.WithMessage("token expired")

	if modified.Message != "token expired" {
		t.Errorf("WithMessage() = %q, want %q", modified.Message, "token expired")
	}
	// Original should be unchanged
	if original.Message != "missing or invalid authentication" {
		t.Errorf("original was modified: %q", original.Message)
	}
}

func TestGatewayError_WithDetails(t *testing.T) {
	err := domain.ErrInvalidArgument.WithDetails(
		domain.ErrorDetail{Field: "content", Reason: "required"},
	)

	if len(err.Details) != 1 {
		t.Fatalf("WithDetails() len = %d, want 1", len(err.Details))
	}
	if err.Details[0].Field != "content" {
		t.Errorf("Details[0].Field = %q, want %q", err.Details[0].Field, "content")
	}
}

func TestProtocolType_String(t *testing.T) {
	tests := []struct {
		p    domain.ProtocolType
		want string
	}{
		{domain.ProtocolREST, "REST"},
		{domain.ProtocolGRPC, "gRPC"},
		{domain.ProtocolMCP, "MCP"},
		{domain.ProtocolWebDAV, "WebDAV"},
		{domain.ProtocolWebSocket, "WebSocket"},
		{domain.ProtocolType(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.p.String(); got != tt.want {
			t.Errorf("ProtocolType(%d).String() = %q, want %q", tt.p, got, tt.want)
		}
	}
}

func TestStoreRequest_MemoryTypes(t *testing.T) {
	types := []string{
		domain.MemoryTypeSemantic,
		domain.MemoryTypeEpisodic,
		domain.MemoryTypeConversational,
		domain.MemoryTypeProfile,
		domain.MemoryTypeProcedural,
		domain.MemoryTypeAuto,
	}

	for _, mt := range types {
		if mt == "" {
			t.Errorf("memory type constant is empty")
		}
	}
}
