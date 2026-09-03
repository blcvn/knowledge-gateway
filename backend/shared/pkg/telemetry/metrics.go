package telemetry

// MetricsConfig documents the shared Prometheus metrics naming convention.
// Actual metric registration is done per-service using the prometheus client.
//
// SOL-ENT-005 / TASK-ENT-009
//
// Naming Convention:
//   vnp_{service}_grpc_requests_total{method, status}
//   vnp_{service}_grpc_latency_seconds{method}
//   vnp_{service}_llm_tokens_total{model, type}       — type: "prompt" | "completion"
//   vnp_{service}_llm_latency_seconds{model}
//
// Each service should use the prometheus/promauto package to register metrics.
// The gateway already registers HTTP metrics in:
//   backend/gateway/internal/infra/middleware/metrics.go
//
// Example for a gRPC service:
//
//   var grpcRequests = promauto.NewCounterVec(prometheus.CounterOpts{
//       Namespace: "vnp",
//       Subsystem: "memory_service",
//       Name:      "grpc_requests_total",
//   }, []string{"method", "status"})

// MetricsLabels contains the standard label names used across all VNP services.
var MetricsLabels = struct {
	Method     string
	Status     string
	Model      string
	TokenType  string
	Service    string
	TenantID   string
}{
	Method:    "method",
	Status:    "status",
	Model:     "model",
	TokenType: "type",
	Service:   "service",
	TenantID:  "tenant_id",
}

// ServiceMetricsConfig contains configuration for a service's Prometheus metrics.
type ServiceMetricsConfig struct {
	// Namespace is the Prometheus namespace (always "vnp").
	Namespace string
	// Subsystem is the service name used as Prometheus subsystem.
	Subsystem string
}

// DefaultConfig returns a MetricsConfig for a service.
func DefaultConfig(serviceName string) ServiceMetricsConfig {
	return ServiceMetricsConfig{
		Namespace: "vnp",
		Subsystem: serviceName,
	}
}
