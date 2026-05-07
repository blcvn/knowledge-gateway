module github.com/blcvn/backend/services/ba-knowledge-service

go 1.24.0

require (
	github.com/blcvn/ba-shared-libs/proto v0.0.0
	github.com/blcvn/kratos-proto/go/ai-proxy v1.0.0
	github.com/blcvn/kratos-proto/go/prompt v1.0.0
	github.com/google/uuid v1.6.0
	github.com/neo4j/neo4j-go-driver/v5 v5.14.0
	github.com/redis/go-redis/v9 v9.17.2
	github.com/stretchr/testify v1.11.1
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.7 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.opentelemetry.io/otel/metric v1.40.0 // indirect
	go.opentelemetry.io/otel/sdk v1.40.0 // indirect
	go.opentelemetry.io/otel/trace v1.40.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409 // indirect
)

replace github.com/blcvn/ba-shared-libs/pkg => ../../ba-shared-libs/pkg

replace github.com/blcvn/ba-shared-libs/proto => ../../ba-shared-libs/proto
