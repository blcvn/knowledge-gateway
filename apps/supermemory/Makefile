# =============================================================================
# Supermemory Monolith App — Makefile
# =============================================================================

APP_NAME    := supermemory-app
BINARY      := bin/$(APP_NAME)
DOCKER_TAG  := $(APP_NAME):latest

.PHONY: build run clean docker up down

## build: Compile the binary
build:
	@echo "→ Building $(APP_NAME)..."
	go build -o $(BINARY) ./cmd/supermemory/
	@echo "✓ Built $(BINARY)"

## run: Build and run locally with dev mode
run: build
	@echo "→ Starting $(APP_NAME) (dev mode)..."
	./$(BINARY)

## clean: Remove build artifacts
clean:
	rm -rf bin/

## docker: Build Docker image
docker:
	docker build -t $(DOCKER_TAG) -f Dockerfile ../..

## up: Start local dev stack (infra)
up:
	docker compose up -d

## down: Stop local dev stack
down:
	docker compose down
