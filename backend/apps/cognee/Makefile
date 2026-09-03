# =============================================================================
# Cognee App — Makefile
# Single binary monolith embedding cognee-ingestion, cognee-cognify, cognee-search
# =============================================================================

APP_NAME    := cognee-app
BINARY      := bin/$(APP_NAME)
DOCKER_TAG  := $(APP_NAME):latest

.PHONY: build run test lint docker up down clean help

## build: Compile the cognee-app binary
build:
	@echo "→ Building $(APP_NAME)..."
	go build -o $(BINARY) ./cmd/cognee/
	@echo "✓ Built $(BINARY) ($$(ls -lh $(BINARY) | awk '{print $$5}'))"

## run: Build and run locally with dev mode
run: build
	@echo "→ Starting $(APP_NAME) (dev mode)..."
	AUTH_DEV_MODE=true \
	POSTGRES_DSN=postgresql://postgres:password@localhost:5432/cognee \
	NATS_URL=nats://localhost:4222 \
	NEO4J_URI=bolt://localhost:7687 \
	NEO4J_PASSWORD=password \
	QDRANT_URL=http://localhost:6333 \
	REDIS_ADDR=localhost:6379 \
	MINIO_ENDPOINT=localhost:9000 \
	MINIO_ACCESS_KEY=minioadmin \
	MINIO_SECRET_KEY=minioadmin \
	./$(BINARY)

## test: Run unit tests with race detection and coverage
test:
	go test ./internal/... -race -count=1 -coverprofile=coverage.out
	@go tool cover -func=coverage.out | tail -1

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## docker: Build Docker image (context = monorepo root)
docker:
	@echo "→ Building Docker image $(DOCKER_TAG)..."
	docker build -t $(DOCKER_TAG) -f Dockerfile ../..
	@echo "✓ Built $(DOCKER_TAG)"
	@docker images $(DOCKER_TAG) --format "  Size: {{.Size}}"

## up: Start local dev stack (app + infra)
up:
	docker compose up -d
	@echo "✓ Stack started. Health: http://localhost:11080/healthz"

## down: Stop local dev stack
down:
	docker compose down

## clean: Remove build artifacts
clean:
	rm -rf bin/ coverage.out

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
