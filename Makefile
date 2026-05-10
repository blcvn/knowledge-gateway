# VNP Memory Platform — Makefile
# Usage: make help

SERVICES := vnp-admin vnp-event vnp-platform vnp-search-hub
GATEWAY  := gateway

.PHONY: help proto build test lint migrate docker-up docker-compact clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# === Proto Generation ===

proto: ## Generate Go code from .proto files
	@echo "Generating proto..."
	protoc \
		--proto_path=proto \
		--proto_path=third_party \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/vnp/admin/v1/admin.proto \
		proto/vnp/event/v1/event.proto \
		proto/vnp/searchhub/v1/searchhub.proto

# === Build ===

build: $(SERVICES:%=build-%) build-gateway ## Build all services

build-%: ## Build a specific service
	@echo "Building services/$*..."
	cd services/$* && go build ./cmd/server/

build-gateway: ## Build gateway
	@echo "Building gateway..."
	cd $(GATEWAY) && go build ./cmd/main.go

# === Test ===

test: $(SERVICES:%=test-%) ## Run all unit tests

test-%: ## Run tests for a specific service
	@echo "Testing services/$*..."
	cd services/$* && go test -v -race -count=1 ./...

test-cover: ## Run tests with coverage report
	@for svc in $(SERVICES); do \
		echo "=== $$svc ==="; \
		cd services/$$svc && go test -coverprofile=coverage.out ./... && \
		go tool cover -func=coverage.out && cd ../..; \
	done

# === Lint ===

lint: ## Run linter on all services
	@for svc in $(SERVICES); do \
		echo "Linting $$svc..."; \
		cd services/$$svc && golangci-lint run ./... && cd ../..; \
	done

# === Database ===

migrate: ## Run database migrations
	@echo "Running migrations..."
	@for svc in $(SERVICES); do \
		if [ -d "services/$$svc/migrations" ]; then \
			echo "Migrating $$svc..."; \
			for f in services/$$svc/migrations/*.sql; do \
				psql "$$DATABASE_URL" -f "$$f"; \
			done; \
		fi; \
	done

migrate-up: ## Run migrations against local PostgreSQL
	$(MAKE) migrate DATABASE_URL="postgres://vnp:vnppassword@localhost:5432/vnp_memory?sslmode=disable"

# === Docker ===

docker-up: ## Start all services (individual deployment)
	docker compose up -d --build

docker-compact: ## Start compact deployment (consolidated)
	docker compose -f docker-compose.compact.yml up -d --build

docker-down: ## Stop all services
	docker compose down
	docker compose -f docker-compose.compact.yml down 2>/dev/null || true

docker-logs: ## Tail service logs
	docker compose logs -f

# === Clean ===

clean: ## Remove build artifacts
	@for svc in $(SERVICES); do \
		rm -f services/$$svc/coverage.out; \
	done
	@echo "Cleaned."

# === CI ===

ci: lint test ## Run CI checks (lint + test)
