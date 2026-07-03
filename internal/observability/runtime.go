package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"kg-service/internal/config"
)

type Runtime struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
	Propagator     propagation.TextMapPropagator
}

func NewRuntime(cfg config.Config) *Runtime {
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Observability.TraceSamplingRatio))
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sampler))
	meterProvider := metric.NewMeterProvider()
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagator)

	return &Runtime{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		Propagator:     propagator,
	}
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var err error
	if r.TracerProvider != nil {
		err = r.TracerProvider.Shutdown(ctx)
	}
	if r.MeterProvider != nil {
		if meterErr := r.MeterProvider.Shutdown(ctx); err == nil {
			err = meterErr
		}
	}
	return err
}
