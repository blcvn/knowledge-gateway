module github.com/vnp-community/vnp-memory/apps/OpenViking

go 1.25.0

replace github.com/vnp-community/vnp-memory/gateway => ../../gateway

replace github.com/vnp-community/vnp-memory/services/ov-admin => ../../services/ov-admin

replace github.com/vnp-community/vnp-memory/services/ov-crypto => ../../services/ov-crypto

replace github.com/vnp-community/vnp-memory/services/ov-fs => ../../services/ov-fs

replace github.com/vnp-community/vnp-memory/services/ov-resource => ../../services/ov-resource

replace github.com/vnp-community/vnp-memory/services/ov-search => ../../services/ov-search

replace github.com/vnp-community/vnp-memory/services/ov-session => ../../services/ov-session

require (
	golang.org/x/sync v0.20.0
	google.golang.org/grpc v1.81.0
	vnp-memory/shared/pkg/telemetry v0.0.0-00010101000000-000000000000
	vnp-memory/shared/pkg/tenant v0.0.0-00010101000000-000000000000
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace vnp-memory/shared/pkg/telemetry => ../../shared/pkg/telemetry

replace vnp-memory/shared/pkg/tenant => ../../shared/pkg/tenant
