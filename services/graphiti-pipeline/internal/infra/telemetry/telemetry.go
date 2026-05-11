package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func NewTracer() trace.Tracer {
	return otel.Tracer("graphiti-pipeline")
}

func NewMetrics() error {
	return nil
}
