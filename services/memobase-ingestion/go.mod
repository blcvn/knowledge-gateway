module github.com/vnp-community/vnp-memory/services/memobase-ingestion

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/pkoukk/tiktoken-go v0.1.7 // Ensure gpt-4o encoder support
	github.com/nats-io/nats.go v1.34.1
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.1
	github.com/jackc/pgx/v5 v5.6.0
	github.com/spf13/viper v1.19.0
	github.com/google/wire v0.6.0
	go.opentelemetry.io/otel v1.27.0
	github.com/prometheus/client_golang v1.19.1
)
