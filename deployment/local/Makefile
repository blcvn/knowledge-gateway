###############################################################################
# VNP Memory — Local Development Makefile
###############################################################################

.PHONY: help up down restart logs ps build clean
.DEFAULT_GOAL := help

COMPOSE := docker compose

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

env: ## Create .env from .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env — please edit API keys before starting."; \
	else \
		echo "⚠️  .env already exists, skipping."; \
	fi

build: ## Build all service images
	$(COMPOSE) build

up: ## Start Memory services (Cognee, Graphiti, Zep, OpenViking)
	$(COMPOSE) --profile memory up -d

up-ui: ## Start Memory services + Cognee frontend
	$(COMPOSE) --profile memory --profile ui up -d

up-all: ## Start Memory + monitoring
	$(COMPOSE) --profile memory --profile ui --profile monitoring up -d

# ── KGS Platform ─────────────────────────────────────────────────────────────

up-kgs: ## Start KGS Platform (specialized: PG+Neo4j+Qdrant+Redis)
	$(COMPOSE) --profile kgs up -d

up-kgs-surrealdb: ## Start KGS Platform with SurrealDB backend
	KGS_STORAGE_MODE=surrealdb $(COMPOSE) --profile kgs --profile surrealdb up -d

up-surrealdb: ## Start SurrealDB only (for development)
	$(COMPOSE) --profile surrealdb up -d

up-full: ## Start everything: Memory + KGS + UI + Monitoring
	$(COMPOSE) --profile memory --profile kgs --profile ui --profile monitoring up -d

# ── Lifecycle ─────────────────────────────────────────────────────────────────

down: ## Stop and remove all containers
	$(COMPOSE) --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down

restart: down up ## Restart core services

# ── Logs ──────────────────────────────────────────────────────────────────────

logs: ## Follow logs from all services
	$(COMPOSE) logs -f

logs-cognee: ## Follow Cognee logs
	$(COMPOSE) logs -f cognee

logs-graphiti: ## Follow Graphiti logs
	$(COMPOSE) logs -f graphiti

logs-openviking: ## Follow OpenViking logs
	$(COMPOSE) logs -f openviking

logs-zep: ## Follow Zep logs
	$(COMPOSE) logs -f zep

logs-kgs: ## Follow KGS Platform logs
	$(COMPOSE) logs -f kgs-platform

logs-surrealdb: ## Follow SurrealDB logs
	$(COMPOSE) logs -f surrealdb

# ── Status ────────────────────────────────────────────────────────────────────

ps: ## Show running containers
	$(COMPOSE) --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring ps

clean: ## Stop containers and remove volumes
	$(COMPOSE) --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down -v
	@echo "🧹 All containers and volumes removed."

status: ## Check health of all services
	@echo "=== Service Health ==="
	@$(COMPOSE) --profile kgs --profile surrealdb ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

