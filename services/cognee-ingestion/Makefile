.PHONY: build test lint run wire proto docker clean

SERVICE_NAME := cognee-ingestion
GO := go
GOFLAGS := -v

# Build
build:
	$(GO) build $(GOFLAGS) -o bin/$(SERVICE_NAME) ./cmd/server

# Run locally
run:
	$(GO) run ./cmd/server

# Test
test:
	$(GO) test ./... -v -count=1 -race

test-cover:
	$(GO) test ./... -coverprofile=coverage.out -covermode=atomic
	$(GO) tool cover -func=coverage.out

# Lint
lint:
	golangci-lint run ./...

# Wire DI generation
wire:
	$(GO) run github.com/google/wire/cmd/wire ./internal/infra/wire/

# Protobuf generation
proto:
	buf generate

# Docker
docker:
	docker build -t vnp/$(SERVICE_NAME):latest .

docker-run:
	docker run --rm -p 9011:9011 -p 9091:9091 \
		-e DATABASE_URL=postgres://postgres:postgres@host.docker.internal:5432/vnp_memory?sslmode=disable \
		-e NATS_URL=nats://host.docker.internal:4222 \
		-e MINIO_ENDPOINT=host.docker.internal:9000 \
		vnp/$(SERVICE_NAME):latest

# Migrations
migrate-up:
	psql $(DATABASE_URL) -f migrations/001_create_ingestion_tables.sql

# Clean
clean:
	rm -rf bin/ coverage.out

# Development helpers
tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...
	goimports -w .
