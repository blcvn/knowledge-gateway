package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for the gateway.
var (
	// RequestsTotal counts total requests by method, path, and status.
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "vnp",
		Subsystem: "gateway",
		Name:      "requests_total",
		Help:      "Total HTTP requests by method, path pattern, and status code",
	}, []string{"method", "path", "status"})

	// RequestDuration observes request latency by method and path.
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "vnp",
		Subsystem: "gateway",
		Name:      "request_duration_seconds",
		Help:      "HTTP request latency in seconds",
		Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"method", "path"})

	// ActiveConnections tracks the number of in-flight requests.
	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "vnp",
		Subsystem: "gateway",
		Name:      "active_connections",
		Help:      "Number of currently active HTTP connections",
	})

	// CircuitBreakerState tracks per-service circuit breaker state.
	CircuitBreakerState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "vnp",
		Subsystem: "gateway",
		Name:      "circuit_breaker_state",
		Help:      "Circuit breaker state per service (0=closed, 1=half-open, 2=open)",
	}, []string{"service"})

	// RateLimitRejected counts rejected requests per tenant.
	RateLimitRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "vnp",
		Subsystem: "gateway",
		Name:      "ratelimit_rejected_total",
		Help:      "Total rate-limited requests by tenant",
	}, []string{"tenant_id"})

	// ResponseSize observes response body sizes.
	ResponseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "vnp",
		Subsystem: "gateway",
		Name:      "response_size_bytes",
		Help:      "HTTP response size in bytes",
		Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B → 100MB
	}, []string{"method", "path"})
)

// Metrics returns Prometheus metrics middleware.
func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ActiveConnections.Inc()
			defer ActiveConnections.Dec()

			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()
			statusStr := strconv.Itoa(ww.statusCode)
			path := normalizePath(r.URL.Path)

			RequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
			RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
			ResponseSize.WithLabelValues(r.Method, path).Observe(float64(ww.bytesWritten))
		})
	}
}

// normalizePath reduces high-cardinality paths to route patterns.
func normalizePath(path string) string {
	// Map specific paths to their route patterns to prevent label explosion
	prefixes := []string{
		"/v1/memory", "/v1/cognee", "/v1/graphiti", "/v1/memobase",
		"/v1/ov", "/v1/zep", "/v1/sm", "/v1/admin",
		"/healthz", "/readyz", "/metrics", "/webdav",
	}
	for _, p := range prefixes {
		if len(path) >= len(p) && path[:len(p)] == p {
			return p
		}
	}
	return "/other"
}
