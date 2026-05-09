package domain

import "fmt"

// GatewayError is the standard error type for all gateway-layer errors.
type GatewayError struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorDetail provides field-level error context.
type ErrorDetail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// Error implements the error interface.
func (e *GatewayError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithDetails returns a copy of the error with appended details.
func (e *GatewayError) WithDetails(details ...ErrorDetail) *GatewayError {
	cp := *e
	cp.Details = append(cp.Details, details...)
	return &cp
}

// WithMessage returns a copy of the error with a custom message.
func (e *GatewayError) WithMessage(msg string) *GatewayError {
	cp := *e
	cp.Message = msg
	return &cp
}

// Sentinel errors for the gateway layer.
// These map directly to gRPC status codes and HTTP status codes.
var (
	// ErrUnauthenticated — 401 / gRPC UNAUTHENTICATED
	ErrUnauthenticated = &GatewayError{Code: "UNAUTHENTICATED", Message: "missing or invalid authentication"}

	// ErrForbidden — 403 / gRPC PERMISSION_DENIED
	ErrForbidden = &GatewayError{Code: "PERMISSION_DENIED", Message: "insufficient permissions"}

	// ErrNotFound — 404 / gRPC NOT_FOUND
	ErrNotFound = &GatewayError{Code: "NOT_FOUND", Message: "resource not found"}

	// ErrInvalidArgument — 400 / gRPC INVALID_ARGUMENT
	ErrInvalidArgument = &GatewayError{Code: "INVALID_ARGUMENT", Message: "invalid request parameters"}

	// ErrRateLimited — 429 / gRPC RESOURCE_EXHAUSTED
	ErrRateLimited = &GatewayError{Code: "RESOURCE_EXHAUSTED", Message: "rate limit exceeded"}

	// ErrCircuitOpen — 503 / gRPC UNAVAILABLE
	ErrCircuitOpen = &GatewayError{Code: "UNAVAILABLE", Message: "service temporarily unavailable"}

	// ErrTimeout — 504 / gRPC DEADLINE_EXCEEDED
	ErrTimeout = &GatewayError{Code: "DEADLINE_EXCEEDED", Message: "request timeout"}
)

// HTTPStatusCode maps GatewayError codes to HTTP status codes.
func (e *GatewayError) HTTPStatusCode() int {
	switch e.Code {
	case "UNAUTHENTICATED":
		return 401
	case "PERMISSION_DENIED":
		return 403
	case "NOT_FOUND":
		return 404
	case "INVALID_ARGUMENT":
		return 400
	case "RESOURCE_EXHAUSTED":
		return 429
	case "UNAVAILABLE":
		return 503
	case "DEADLINE_EXCEEDED":
		return 504
	default:
		return 500
	}
}
