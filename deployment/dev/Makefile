###############################################################################
# VNP Memory — Dev Server Deployment Makefile (172.20.2.39)
###############################################################################

.PHONY: help up down restart logs ps build clean deploy
.DEFAULT_GOAL := help

COMPOSE := docker compose
DEV_SERVER := 172.20.2.39
DEV_USER := ubuntu
DEPLOY_DIR := /opt/vnp-memory
COGNEE_SRC := ../../services/cognee
GRAPHITI_SRC := ../../services/graphiti
COGNEE_FRONTEND_SRC := ../../services/cognee/cognee-frontend

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

env: ## Create .env from .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env — please edit API keys before starting."; \
	else \
		echo "⚠️  .env already exists, skipping."; \
	fi

# =============================================================================
# Image Building (run locally — images are then uploaded to dev server)
# =============================================================================

image-build-cognee: ## Build Cognee Docker image locally
	docker build -t vnp/cognee:latest $(COGNEE_SRC)

image-build-graphiti: ## Build Graphiti Docker image locally
	docker build -t vnp/graphiti:latest $(GRAPHITI_SRC)

image-build-cognee-frontend: ## Build Cognee frontend Docker image locally
	docker build -t vnp/cognee-frontend:latest $(COGNEE_FRONTEND_SRC)

image-build-all: image-build-cognee image-build-graphiti image-build-cognee-frontend ## Build all images

# =============================================================================
# Image Upload (build + export tar.gz + SCP + docker load on remote)
# =============================================================================

image-upload-cognee: image-build-cognee ## Build & upload Cognee image to dev server
	@echo "📦 Exporting vnp/cognee:latest ..."
	docker save vnp/cognee:latest | gzip > /tmp/vnp-cognee.tar.gz
	@echo "🚀 Uploading to $(DEV_SERVER) ..."
	scp /tmp/vnp-cognee.tar.gz $(DEV_USER)@$(DEV_SERVER):/tmp/
	ssh $(DEV_USER)@$(DEV_SERVER) 'docker load < /tmp/vnp-cognee.tar.gz && rm /tmp/vnp-cognee.tar.gz'
	rm -f /tmp/vnp-cognee.tar.gz
	@echo "✅ vnp/cognee:latest loaded on $(DEV_SERVER)"

image-upload-graphiti: image-build-graphiti ## Build & upload Graphiti image to dev server
	@echo "📦 Exporting vnp/graphiti:latest ..."
	docker save vnp/graphiti:latest | gzip > /tmp/vnp-graphiti.tar.gz
	@echo "🚀 Uploading to $(DEV_SERVER) ..."
	scp /tmp/vnp-graphiti.tar.gz $(DEV_USER)@$(DEV_SERVER):/tmp/
	ssh $(DEV_USER)@$(DEV_SERVER) 'docker load < /tmp/vnp-graphiti.tar.gz && rm /tmp/vnp-graphiti.tar.gz'
	rm -f /tmp/vnp-graphiti.tar.gz
	@echo "✅ vnp/graphiti:latest loaded on $(DEV_SERVER)"

image-upload-cognee-frontend: image-build-cognee-frontend ## Build & upload Cognee frontend image
	@echo "📦 Exporting vnp/cognee-frontend:latest ..."
	docker save vnp/cognee-frontend:latest | gzip > /tmp/vnp-cognee-frontend.tar.gz
	@echo "🚀 Uploading to $(DEV_SERVER) ..."
	scp /tmp/vnp-cognee-frontend.tar.gz $(DEV_USER)@$(DEV_SERVER):/tmp/
	ssh $(DEV_USER)@$(DEV_SERVER) 'docker load < /tmp/vnp-cognee-frontend.tar.gz && rm /tmp/vnp-cognee-frontend.tar.gz'
	rm -f /tmp/vnp-cognee-frontend.tar.gz
	@echo "✅ vnp/cognee-frontend:latest loaded on $(DEV_SERVER)"

image-upload-all: image-upload-cognee image-upload-graphiti image-upload-cognee-frontend ## Build & upload all images

# =============================================================================
# Configuration Sync (push configs to remote server)
# =============================================================================

sync-config: ## Sync docker-compose, configs, and .env to dev server
	@echo "📂 Syncing configuration to $(DEV_SERVER):$(DEPLOY_DIR) ..."
	ssh $(DEV_USER)@$(DEV_SERVER) 'mkdir -p $(DEPLOY_DIR)/config'
	scp docker-compose.yml $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/
	scp init-db.sql $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/
	scp config/zep.yaml $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/config/
	scp config/kgs.yaml $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/config/
	@if [ -f .env ]; then \
		scp .env $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/; \
		echo "✅ Synced .env + configs to $(DEV_SERVER)"; \
	else \
		scp .env.example $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/.env; \
		echo "⚠️  No .env found, synced .env.example as .env"; \
	fi

# =============================================================================
# Remote Docker Compose Operations
# =============================================================================

remote-up: ## Start Memory services on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory up -d'

remote-up-ui: ## Start Memory + Cognee UI on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory --profile ui up -d'

remote-up-all: ## Start Memory + KGS + UI + Monitoring on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory --profile kgs --profile ui --profile monitoring up -d'

remote-down: ## Stop all services on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down'

remote-restart: remote-down remote-up ## Restart core Memory services on dev server

remote-ps: ## Show running containers on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring ps'

remote-logs: ## Follow logs on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose logs -f --tail=100'

remote-logs-cognee: ## Follow Cognee logs on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose logs -f cognee'

remote-logs-graphiti: ## Follow Graphiti logs on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose logs -f graphiti'

remote-logs-zep: ## Follow Zep logs on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose logs -f zep'

remote-logs-openviking: ## Follow OpenViking logs on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose logs -f openviking'

# =============================================================================
# Full Deployment Pipeline
# =============================================================================

deploy: ## Full deploy: build images + upload + sync config + start
	@echo "🚀 Starting full deployment to $(DEV_SERVER) ..."
	$(MAKE) image-upload-all
	$(MAKE) sync-config
	$(MAKE) remote-down
	$(MAKE) remote-up-ui
	@echo "✅ Deployment complete! Services running on $(DEV_SERVER)"

deploy-memory: ## Deploy only Memory services (no KGS, no frontend build)
	@echo "🚀 Deploying Memory services to $(DEV_SERVER) ..."
	$(MAKE) image-upload-cognee
	$(MAKE) image-upload-graphiti
	$(MAKE) sync-config
	$(MAKE) remote-down
	$(MAKE) remote-up
	@echo "✅ Memory services deployed on $(DEV_SERVER)"

deploy-config: sync-config remote-restart ## Sync config and restart (no image rebuild)

# =============================================================================
# Local development (run on this machine)
# =============================================================================

up: ## Start Memory services locally
	$(COMPOSE) --profile memory up -d

up-ui: ## Start Memory services + Cognee frontend locally
	$(COMPOSE) --profile memory --profile ui up -d

up-all: ## Start all services locally
	$(COMPOSE) --profile memory --profile kgs --profile ui --profile monitoring up -d

down: ## Stop and remove all containers locally
	$(COMPOSE) --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down

restart: down up ## Restart core services locally

logs: ## Follow logs locally
	$(COMPOSE) logs -f

ps: ## Show running containers locally
	$(COMPOSE) --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring ps

clean: ## Stop containers and remove volumes locally
	$(COMPOSE) --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down -v
	@echo "🧹 All containers and volumes removed."

status: ## Check health of all services
	@echo "=== Service Health ==="
	@$(COMPOSE) --profile memory --profile ui ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
