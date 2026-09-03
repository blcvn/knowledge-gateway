module vnp-memory/services/graphiti-store

go 1.25.0

require (
	github.com/nats-io/nats.go v1.52.0
	github.com/neo4j/neo4j-go-driver/v5 v5.28.4
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
	vnp-memory/pkg/graph v0.0.0
)

require (
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
)

replace vnp-memory/pkg/graph => ../../shared/pkg/graph
