module vnp-memory/services/graphiti-knowledge

go 1.23.0

require (
	github.com/neo4j/neo4j-go-driver/v5 v5.24.0
	github.com/sashabaranov/go-openai v1.29.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	vnp-memory/pkg/telemetry v0.0.0
	vnp-memory/pkg/tenant v0.0.0
)

replace vnp-memory/pkg/telemetry => ../../pkg/telemetry
replace vnp-memory/pkg/tenant => ../../pkg/tenant
