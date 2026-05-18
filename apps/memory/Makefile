.PHONY: build run dev docker infra-up infra-down clean test lint

# ── Variables ──────────────────────────────────────────────────
APP_NAME    := vnp-memory
BINARY      := bin/$(APP_NAME)
DOCKER_TAG  := $(APP_NAME):latest
COMPOSE     := docker compose
COMPOSE_INFRA := docker compose -f docker-compose.infra.yml

# ── Build ──────────────────────────────────────────────────────
build:
	@echo "==> Building $(APP_NAME)..."
	cd ../.. && go build -o apps/memory/$(BINARY) ./apps/memory/cmd/server

# ── Run (requires infra running) ──────────────────────────────
run: build
	./$(BINARY)

# ── Dev (start infra + run with hot reload) ───────────────────
dev: infra-up
	@echo "==> Starting $(APP_NAME) in dev mode..."
	cd ../.. && go run ./apps/memory/cmd/server

# ── Docker ─────────────────────────────────────────────────────
docker:
	@echo "==> Building Docker image..."
	cd ../.. && docker build -t $(DOCKER_TAG) -f apps/memory/Dockerfile .

docker-up:
	$(COMPOSE) up -d --build

docker-down:
	$(COMPOSE) down

docker-logs:
	$(COMPOSE) logs -f vnp-memory

# ── Infrastructure ─────────────────────────────────────────────
infra-up:
	@echo "==> Starting infrastructure containers..."
	$(COMPOSE_INFRA) up -d
	@echo "==> Waiting for services to be healthy..."
	@sleep 5
	@echo "==> Infrastructure ready!"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Neo4j:      localhost:7474 (bolt://localhost:7687)"
	@echo "  Qdrant:     localhost:6333"
	@echo "  Redis:      localhost:6379"
	@echo "  MinIO:      localhost:9000 (console: localhost:9001)"

infra-down:
	$(COMPOSE_INFRA) down

infra-reset:
	$(COMPOSE_INFRA) down -v
	@echo "==> All data volumes removed"

# ── Test ───────────────────────────────────────────────────────
test:
	cd ../.. && go test ./apps/memory/...

test-e2e:
	cd ../.. && go test -v -tags=e2e ./apps/memory/tests/...

# ── Lint ───────────────────────────────────────────────────────
lint:
	cd ../.. && golangci-lint run ./apps/memory/...

# ── Tidy ───────────────────────────────────────────────────────
tidy:
	cd ../.. && go mod tidy -C apps/memory

# ── Clean ──────────────────────────────────────────────────────
clean:
	rm -rf bin/
	$(COMPOSE) down -v 2>/dev/null || true
	$(COMPOSE_INFRA) down -v 2>/dev/null || true

# ── Help ───────────────────────────────────────────────────────
help:
	@echo "VNP Memory Monolith — Available commands:"
	@echo ""
	@echo "  make build       Build Go binary"
	@echo "  make run         Build and run"
	@echo "  make dev         Start infra + run locally"
	@echo "  make docker      Build Docker image"
	@echo "  make docker-up   Start full stack (app + infra)"
	@echo "  make docker-down Stop full stack"
	@echo "  make infra-up    Start infrastructure only"
	@echo "  make infra-down  Stop infrastructure"
	@echo "  make infra-reset Stop infra and delete volumes"
	@echo "  make test        Run unit tests"
	@echo "  make test-e2e    Run E2E tests"
	@echo "  make lint        Run linter"
	@echo "  make tidy        Run go mod tidy"
	@echo "  make clean       Remove artifacts and volumes"
