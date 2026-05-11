package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// InitProvider bootstraps the OpenTelemetry pipeline.
// It configures the OTLP exporter, sets up the TraceProvider, and registers global propagators.
// Returns a shutdown function that must be deferred in main().
func InitProvider(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// 1. Setup Resource (Service metadata)
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironment(getEnv("ENV", "production")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	// 2. Setup OTLP Exporter (via gRPC to OTel Collector/Jaeger)
	// Expects OTEL_EXPORTER_OTLP_ENDPOINT environment variable to be set
	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp exporter: %w", err)
	}

	// 3. Setup TracerProvider
	// Uses AlwaysSample for development. In production, this should be TraceIDRatioBased.
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// 4. Register Globals
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator to tracecontext (the W3C Trace Context format).
	// This ensures traces propagate across microservice gRPC boundaries.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("OpenTelemetry provider initialized successfully", 
		slog.String("service", serviceName),
		slog.String("env", getEnv("ENV", "production")),
	)

	// Return the shutdown function
	return tracerProvider.Shutdown, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
