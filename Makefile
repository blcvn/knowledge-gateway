# Makefile additions for SOL-003

# === Build ===
.PHONY: build
build:
	@echo "Building all 8 services..."
	@go build ./backend/gateway/...
	@go build ./backend/services/vnp-platform/...
	@go build ./backend/services/kg-service/...
	@go build ./backend/services/memory-service/...
	@go build ./backend/services/storage-service/...
	@go build ./backend/services/search-service/...
	@go build ./backend/services/pipeline-service/...
	@go build ./backend/services/obs-service/...
	@echo "✅ All services built"

# === Test ===
.PHONY: test
test:
	go test -count=1 -timeout 120s ./...

.PHONY: test-integration
test-integration:
	INTEGRATION_ASSUME_UP=true \
	go test -v -timeout 300s -tags integration \
		./tests/integration/sol003/...

# === Docker ===
.PHONY: up
up:
	docker compose -f deployment/docker-compose.yml up -d --wait

.PHONY: down
down:
	docker compose -f deployment/docker-compose.yml down

.PHONY: logs
logs:
	docker compose -f deployment/docker-compose.yml logs -f

.PHONY: ps
ps:
	docker compose -f deployment/docker-compose.yml ps

# === Dev ===
.PHONY: archive-old-services
archive-old-services:
	bash tests/scripts/archive-old-services.sh

# === AgentMemory Protobufs ===
.PHONY: proto-agentmemory
proto-agentmemory:
	@echo "Compiling AgentMemory protobufs..."
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       backend/api/proto/observe/v1/observe.proto \
	       backend/api/proto/memory/v1/agentmemory.proto \
	       backend/api/proto/search/v1/observe_search.proto \
	       backend/api/proto/orchestration/v1/orchestration.proto

# === Cognee Protobufs ===
.PHONY: proto-cognee
proto-cognee:
	@echo "Compiling cognee protobufs..."
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       backend/api/proto/cognee/cognify/v1/cognify.proto \
	       backend/api/proto/cognee/ingestion/v1/ingestion.proto \
	       backend/api/proto/cognee/search/v1/search.proto
