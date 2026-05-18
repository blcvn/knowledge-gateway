# VNP Memory Platform — Makefile
# Usage: make help
#
# Quick Start:
#   make dev          — Start infra + backend + UI (full-stack dev)
#   make memory-dev   — Start infra + backend only
#   make ui-dev       — Start UI dev server only (needs backend running)

SERVICES := vnp-admin vnp-event vnp-platform vnp-search-hub
GATEWAY  := gateway

# ── Memory Monolith ────────────────────────────────────────────
MEMORY_APP      := apps/memory
MEMORY_BINARY   := $(MEMORY_APP)/bin/vnp-memory
MEMORY_COMPOSE  := docker compose -f $(MEMORY_APP)/docker-compose.infra.yml

# ── UI ─────────────────────────────────────────────────────────
UI_DIR          := ui
UI_PORT         := 5173
BACKEND_PORT    := 8080

.PHONY: help dev dev-stop \
        memory-build memory-run memory-dev memory-logs memory-full \
        infra-up infra-down infra-reset infra-status \
        ui-install ui-dev ui-build ui-embed ui-preview \
        proto build test lint migrate \
        docker-up docker-compact docker-down docker-logs \
        clean clean-all status ci

help: ## Show available targets
	@echo ""
	@echo "\033[1;35m╔══════════════════════════════════════════════════════╗\033[0m"
	@echo "\033[1;35m║          VNP Memory Platform — Makefile              ║\033[0m"
	@echo "\033[1;35m╚══════════════════════════════════════════════════════╝\033[0m"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[1;33m⚡ Quick Start:\033[0m"
	@echo "  \033[32mmake dev\033[0m          Start full-stack (infra + backend + UI dev server)"
	@echo "  \033[32mmake memory-dev\033[0m   Start backend only (infra + Go monolith)"
	@echo "  \033[32mmake ui-dev\033[0m       Start UI only (needs backend running)"
	@echo "  \033[32mmake memory-full\033[0m  Build single binary (API + embedded UI)"
	@echo ""

# ═══════════════════════════════════════════════════════════════
# ██  FULL-STACK DEVELOPMENT
# ═══════════════════════════════════════════════════════════════

dev: infra-up ## Start full-stack dev (infra + backend + UI)
	@echo ""
	@echo "\033[1;35m🚀 Starting VNP Memory Full-Stack Dev...\033[0m"
	@echo ""
	@echo "  \033[36m→\033[0m Backend API:  http://localhost:$(BACKEND_PORT)"
	@echo "  \033[36m→\033[0m UI Console:   http://localhost:$(UI_PORT)"
	@echo "  \033[36m→\033[0m Health Check: http://localhost:8083/healthz"
	@echo ""
	@trap 'kill %1 %2 2>/dev/null; echo "\n\033[33m⏹  Stopped all services.\033[0m"' INT TERM; \
		(cd $(CURDIR) && go run ./$(MEMORY_APP)/cmd/server 2>&1 | sed 's/^/\x1b[34m[backend]\x1b[0m /') & \
		sleep 2 && \
		(cd $(UI_DIR) && npx vite --port $(UI_PORT) --host 2>&1 | sed 's/^/\x1b[35m[ui]\x1b[0m      /') & \
		wait

dev-stop: ## Stop background dev services
	@echo "Stopping dev services..."
	@-pkill -f "go run ./$(MEMORY_APP)/cmd/server" 2>/dev/null || true
	@-pkill -f "vite.*--port $(UI_PORT)" 2>/dev/null || true
	@echo "\033[32m✓ Dev services stopped.\033[0m"

# ═══════════════════════════════════════════════════════════════
# ██  MEMORY MONOLITH BACKEND
# ═══════════════════════════════════════════════════════════════

memory-build: ## Build the memory monolith binary
	@echo "\033[36m⚙  Building memory monolith...\033[0m"
	go build -o $(MEMORY_BINARY) ./$(MEMORY_APP)/cmd/server
	@echo "\033[32m✓ Built: $(MEMORY_BINARY)\033[0m"

memory-run: memory-build ## Build and run memory monolith
	@echo "\033[36m▶  Running memory monolith...\033[0m"
	./$(MEMORY_BINARY)

memory-dev: infra-up ## Start infra + run backend in dev mode
	@echo ""
	@echo "\033[1;35m🔧 Starting Memory Backend Dev...\033[0m"
	@echo "  \033[36m→\033[0m REST API:     http://localhost:$(BACKEND_PORT)"
	@echo "  \033[36m→\033[0m MCP Server:   http://localhost:8082"
	@echo "  \033[36m→\033[0m Health Check: http://localhost:8083/healthz"
	@echo ""
	go run ./$(MEMORY_APP)/cmd/server

memory-test: ## Run memory monolith tests
	go test ./$(MEMORY_APP)/...

memory-lint: ## Lint memory monolith code
	golangci-lint run ./$(MEMORY_APP)/...

memory-logs: ## Tail memory backend logs (Docker mode)
	docker compose -f $(MEMORY_APP)/docker-compose.yml logs -f vnp-memory

# ═══════════════════════════════════════════════════════════════
# ██  INFRASTRUCTURE (PostgreSQL, Neo4j, Qdrant, Redis, MinIO)
# ═══════════════════════════════════════════════════════════════

infra-up: ## Start infrastructure containers
	@echo "\033[36m⚙  Starting infrastructure...\033[0m"
	$(MEMORY_COMPOSE) up -d
	@echo "Waiting for services..."
	@sleep 5
	@echo ""
	@echo "\033[32m✓ Infrastructure ready!\033[0m"
	@echo "  PostgreSQL:  localhost:5432"
	@echo "  Neo4j:       localhost:7474 (bolt://localhost:7687)"
	@echo "  Qdrant:      localhost:6333"
	@echo "  Redis:       localhost:6379"
	@echo "  MinIO:       localhost:9000 (console: localhost:9001)"
	@echo ""

infra-down: ## Stop infrastructure containers
	@echo "\033[33m⏹  Stopping infrastructure...\033[0m"
	$(MEMORY_COMPOSE) down
	@echo "\033[32m✓ Infrastructure stopped.\033[0m"

infra-reset: ## Stop infra and delete all data volumes
	@echo "\033[31m⚠  Resetting infrastructure (all data will be lost)...\033[0m"
	$(MEMORY_COMPOSE) down -v
	@echo "\033[32m✓ Infrastructure reset complete.\033[0m"

infra-status: ## Show infrastructure container status
	@echo "\033[36m📊 Infrastructure Status:\033[0m"
	@$(MEMORY_COMPOSE) ps

# ═══════════════════════════════════════════════════════════════
# ██  UI DEVELOPMENT (Vite + React + TailwindCSS)
# ═══════════════════════════════════════════════════════════════

ui-install: ## Install UI dependencies
	@echo "\033[36m📦 Installing UI dependencies...\033[0m"
	cd $(UI_DIR) && npm install
	@echo "\033[32m✓ UI dependencies installed.\033[0m"

ui-dev: ## Start UI dev server (proxy → backend:8080)
	@echo ""
	@echo "\033[1;35m🎨 Starting UI Dev Server...\033[0m"
	@echo "  \033[36m→\033[0m UI Console:  http://localhost:$(UI_PORT)"
	@echo "  \033[36m→\033[0m API Proxy:   /v1/* → http://localhost:$(BACKEND_PORT)"
	@echo ""
	cd $(UI_DIR) && npx vite --port $(UI_PORT) --host

ui-build: ## Build UI for production
	@echo "\033[36m📦 Building UI for production...\033[0m"
	cd $(UI_DIR) && npm run build
	@echo "\033[32m✓ UI built: $(UI_DIR)/dist/\033[0m"

ui-embed: ui-build ## Copy UI dist into Go embed directory
	@echo "\033[36m📎 Embedding UI assets into Go binary path...\033[0m"
	rm -rf $(MEMORY_APP)/internal/ui/ui_dist
	mkdir -p $(MEMORY_APP)/internal/ui/ui_dist
	cp -r $(UI_DIR)/dist/* $(MEMORY_APP)/internal/ui/ui_dist/
	@echo "\033[32m✓ UI assets copied to $(MEMORY_APP)/internal/ui/ui_dist/\033[0m"

ui-preview: ui-build ## Preview production UI build
	@echo "\033[36m👁  Previewing production build...\033[0m"
	cd $(UI_DIR) && npx vite preview --port 4173

# ── Full-Stack Production Build ────────────────────────────────
memory-full: ui-embed ## Build single binary with embedded UI
	@echo "\033[36m🔨 Building full-stack monolith (API + UI)...\033[0m"
	go build -o $(MEMORY_BINARY) ./$(MEMORY_APP)/cmd/server
	@echo "\033[32m✓ Full-stack binary: $(MEMORY_BINARY)\033[0m"
	@echo "  Run with: ./$(MEMORY_BINARY)"
	@echo "  API:  http://localhost:8080/v1/"
	@echo "  UI:   http://localhost:8080/"

# ═══════════════════════════════════════════════════════════════
# ██  PROTO GENERATION
# ═══════════════════════════════════════════════════════════════

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

# ═══════════════════════════════════════════════════════════════
# ██  MICROSERVICES (legacy individual services)
# ═══════════════════════════════════════════════════════════════

build: $(SERVICES:%=build-%) build-gateway ## Build all microservices

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

# ═══════════════════════════════════════════════════════════════
# ██  STATUS & CLEAN
# ═══════════════════════════════════════════════════════════════

status: ## Show status of all running components
	@echo ""
	@echo "\033[1;35m📊 VNP Memory Platform Status\033[0m"
	@echo "\033[90m──────────────────────────────────────────\033[0m"
	@echo -n "  Infrastructure: "; \
		if $(MEMORY_COMPOSE) ps --quiet 2>/dev/null | grep -q .; then \
			echo "\033[32m● Running\033[0m"; \
		else \
			echo "\033[31m○ Stopped\033[0m"; \
		fi
	@echo -n "  Backend:        "; \
		if curl -sf http://localhost:$(BACKEND_PORT)/v1/admin/health >/dev/null 2>&1 || \
		   curl -sf http://localhost:8083/readyz >/dev/null 2>&1; then \
			echo "\033[32m● Running\033[0m (port $(BACKEND_PORT))"; \
		else \
			echo "\033[31m○ Stopped\033[0m"; \
		fi
	@echo -n "  UI Dev Server:  "; \
		if curl -sf http://localhost:$(UI_PORT) >/dev/null 2>&1; then \
			echo "\033[32m● Running\033[0m (port $(UI_PORT))"; \
		else \
			echo "\033[31m○ Stopped\033[0m"; \
		fi
	@echo ""

clean: ## Remove build artifacts
	@for svc in $(SERVICES); do \
		rm -f services/$$svc/coverage.out; \
	done
	rm -rf $(MEMORY_APP)/bin/
	@echo "\033[32m✓ Cleaned.\033[0m"

clean-all: clean infra-down ## Remove artifacts + stop infrastructure
	rm -rf $(UI_DIR)/dist/ $(UI_DIR)/node_modules/.vite/
	rm -rf $(MEMORY_APP)/internal/ui/ui_dist/
	@echo "\033[32m✓ Full cleanup complete.\033[0m"

# === CI ===

ci: lint test ## Run CI checks (lint + test)
