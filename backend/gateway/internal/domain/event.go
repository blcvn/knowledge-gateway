package domain

import "time"

// RequestReceived is published when the gateway receives any inbound request.
type RequestReceived struct {
	RequestID  string    `json:"request_id"`
	TenantID   string    `json:"tenant_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Protocol   string    `json:"protocol"`
	RemoteAddr string    `json:"remote_addr"`
	ReceivedAt time.Time `json:"received_at"`
}

// RequestRouted is published after successful downstream dispatch.
type RequestRouted struct {
	RequestID  string `json:"request_id"`
	TenantID   string `json:"tenant_id"`
	Target     string `json:"target"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
}

// AuthFailed is published on authentication failure for security monitoring.
type AuthFailed struct {
	RequestID  string    `json:"request_id"`
	Reason     string    `json:"reason"`
	RemoteAddr string    `json:"remote_addr"`
	Timestamp  time.Time `json:"timestamp"`
}

// RateLimitExceeded is published when a tenant exceeds their rate limit.
type RateLimitExceeded struct {
	RequestID string    `json:"request_id"`
	TenantID  string    `json:"tenant_id"`
	Endpoint  string    `json:"endpoint"`
	Tier      string    `json:"tier"`
	Timestamp time.Time `json:"timestamp"`
}

// CircuitOpened is published when a downstream circuit breaker transitions to open.
type CircuitOpened struct {
	Service   string    `json:"service"`
	Failures  int       `json:"failures"`
	OpenedAt  time.Time `json:"opened_at"`
}

// NATS subject constants for gateway events.
const (
	SubjectRequestReceived   = "gateway.request.received"
	SubjectRequestRouted     = "gateway.request.routed"
	SubjectAuthFailed        = "gateway.auth.failed"
	SubjectRateLimitExceeded = "gateway.ratelimit.exceeded"
	SubjectCircuitOpened     = "gateway.circuit.opened"
)
