module vnp-memory/services/graphiti-admin

go 1.23.0

require (
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
	vnp-memory/pkg/graph v0.0.0
)

replace vnp-memory/pkg/graph => ../../shared/pkg/graph
