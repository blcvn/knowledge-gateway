module vnp-memory/services/ov-admin

go 1.23.0

require (
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
	vnp-memory/shared/pkg/telemetry v0.0.0
	vnp-memory/shared/pkg/tenant v0.0.0
)

replace vnp-memory/shared/pkg/telemetry => ../../shared/pkg/telemetry
replace vnp-memory/shared/pkg/tenant => ../../shared/pkg/tenant
