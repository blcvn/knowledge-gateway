package observability

import (
	"strconv"
	"sync/atomic"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kg_request_total",
			Help: "Total number of KG service requests.",
		},
		[]string{"method", "status"},
	)

	requestDurationMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kg_request_duration_ms",
			Help:    "Duration of KG service requests in milliseconds.",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000},
		},
		[]string{"method", "status"},
	)

	entityWriteTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kg_entity_write_total",
			Help: "Total number of entity write operations.",
		},
		[]string{"operation", "status"},
	)

	searchDurationMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kg_search_duration_ms",
			Help:    "Duration of search operations in milliseconds.",
			Buckets: []float64{10, 25, 50, 100, 200, 500, 1000, 1500, 3000},
		},
		[]string{"search_type"},
	)

	overlayActiveGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kg_overlay_count_active",
			Help: "Current number of active overlays.",
		},
		[]string{"namespace"},
	)

	lockAcquireDurationMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kg_lock_acquire_duration_ms",
			Help:    "Duration of lock acquisition in milliseconds.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000},
		},
		[]string{"level", "status"},
	)

	overlayCounts syncMapCounter
)

type syncMapCounter struct {
	counts atomic.Pointer[map[string]int64]
}

func init() {
	m := make(map[string]int64)
	overlayCounts.counts.Store(&m)
}

func ObserveRequest(method string, started time.Time, err error) {
	status := statusCodeLabel(err)
	requestTotal.WithLabelValues(method, status).Inc()
	requestDurationMs.WithLabelValues(method, status).Observe(float64(time.Since(started).Milliseconds()))
}

func ObserveEntityWrite(operation string, err error) {
	entityWriteTotal.WithLabelValues(operation, statusLabel(err)).Inc()
}

func ObserveSearchDuration(searchType string, duration time.Duration) {
	searchDurationMs.WithLabelValues(searchType).Observe(float64(duration.Milliseconds()))
}

func IncOverlayActive(namespace string) {
	if namespace == "" {
		namespace = "unknown"
	}
	counts := cloneCounts()
	counts[namespace]++
	overlayCounts.counts.Store(&counts)
	overlayActiveGauge.WithLabelValues(namespace).Set(float64(counts[namespace]))
}

func DecOverlayActive(namespace string) {
	if namespace == "" {
		namespace = "unknown"
	}
	counts := cloneCounts()
	if counts[namespace] > 0 {
		counts[namespace]--
	}
	overlayCounts.counts.Store(&counts)
	overlayActiveGauge.WithLabelValues(namespace).Set(float64(counts[namespace]))
}

func ObserveLockAcquire(level string, duration time.Duration, err error) {
	lockAcquireDurationMs.WithLabelValues(level, statusLabel(err)).Observe(float64(duration.Milliseconds()))
}

func statusLabel(err error) string {
	if err == nil {
		return "success"
	}
	return "failure"
}

func statusCodeLabel(err error) string {
	if err == nil {
		return "200"
	}
	if e := kerrors.FromError(err); e != nil {
		return strconv.Itoa(int(e.Code))
	}
	return "500"
}

func cloneCounts() map[string]int64 {
	ptr := overlayCounts.counts.Load()
	if ptr == nil {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(*ptr))
	for k, v := range *ptr {
		out[k] = v
	}
	return out
}
