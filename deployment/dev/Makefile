###############################################################################
# VNP Memory — Dev Server Deployment Makefile (172.20.2.39)
#
# All application services are Go projects built from ../../apps/.
# Build context = monorepo root (../../) so Dockerfiles can access shared pkg/.
###############################################################################

.PHONY: help up down restart logs ps build clean deploy
.PHONY: image-build-memory image-upload-memory
.PHONY: remote-up-monolith deploy-monolith sync-nginx
.DEFAULT_GOAL := help

COMPOSE := docker compose
DEV_SERVER := 172.20.2.39
DEV_USER := ubuntu
DEPLOY_DIR := /opt/vnp-memory

# Monorepo root (build context for all Dockerfiles)
MONOREPO_ROOT := ../..

# App source directories (relative to monorepo root)
MEMORY_APP := apps/memory
COGNEE_APP := apps/cognee
GRAPHITI_APP := apps/graphiti
OPENVIKING_APP := apps/OpenViking
ZEP_APP := apps/zep
MEMOBASE_APP := apps/memobase
SUPERMEMORY_APP := apps/supermemory
COGNEE_FRONTEND_APP := apps/cognee/cognee-frontend

# Gateway server (nginx reverse proxy + certbot)
GATEWAY_SERVER := 172.20.2.16
GATEWAY_USER := ubuntu

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
# Image Building — All Go apps built from monorepo root context
# =============================================================================

image-build-memory: ## Build Memory Monolith Docker image (API + embedded UI)
	docker build -t vnp/memory:latest -f $(MONOREPO_ROOT)/$(MEMORY_APP)/Dockerfile $(MONOREPO_ROOT)

image-build-cognee: ## Build Cognee Docker image from apps/cognee
	docker build -t vnp/cognee:latest -f $(MONOREPO_ROOT)/$(COGNEE_APP)/Dockerfile $(MONOREPO_ROOT)

image-build-graphiti: ## Build Graphiti Docker image from apps/graphiti
	docker build -t vnp/graphiti:latest -f $(MONOREPO_ROOT)/$(GRAPHITI_APP)/Dockerfile $(MONOREPO_ROOT)

image-build-openviking: ## Build OpenViking Docker image from apps/OpenViking
	docker build -t vnp/openviking:latest -f $(MONOREPO_ROOT)/$(OPENVIKING_APP)/Dockerfile $(MONOREPO_ROOT)

image-build-zep: ## Build Zep Docker image from apps/zep
	docker build -t vnp/zep:latest -f $(MONOREPO_ROOT)/$(ZEP_APP)/Dockerfile $(MONOREPO_ROOT)

image-build-memobase: ## Build Memobase Docker image from apps/memobase
	docker build -t vnp/memobase:latest -f $(MONOREPO_ROOT)/$(MEMOBASE_APP)/Dockerfile $(MONOREPO_ROOT)

image-build-supermemory: ## Build Supermemory Docker image from apps/supermemory
	docker build -t vnp/supermemory:latest -f $(MONOREPO_ROOT)/$(SUPERMEMORY_APP)/Dockerfile $(MONOREPO_ROOT)

image-build-cognee-frontend: ## Build Cognee frontend Docker image
	docker build -t vnp/cognee-frontend:latest $(MONOREPO_ROOT)/$(COGNEE_FRONTEND_APP)

image-build-all: image-build-memory image-build-cognee image-build-graphiti image-build-openviking image-build-zep image-build-memobase image-build-supermemory image-build-cognee-frontend ## Build all images

# =============================================================================
# Image Upload — build + export tar.gz + SCP + docker load on remote
# =============================================================================

define upload_image
	@echo "📦 Exporting vnp/$(1):latest ..."
	docker save vnp/$(1):latest | gzip > /tmp/vnp-$(1).tar.gz
	@echo "🚀 Uploading to $(DEV_SERVER) ..."
	scp /tmp/vnp-$(1).tar.gz $(DEV_USER)@$(DEV_SERVER):/tmp/
	ssh $(DEV_USER)@$(DEV_SERVER) 'docker load < /tmp/vnp-$(1).tar.gz && rm /tmp/vnp-$(1).tar.gz'
	rm -f /tmp/vnp-$(1).tar.gz
	@echo "✅ vnp/$(1):latest loaded on $(DEV_SERVER)"
endef

image-upload-memory: image-build-memory ## Build & upload Memory Monolith image to dev server
	$(call upload_image,memory)

image-upload-cognee: image-build-cognee ## Build & upload Cognee image to dev server
	$(call upload_image,cognee)

image-upload-graphiti: image-build-graphiti ## Build & upload Graphiti image to dev server
	$(call upload_image,graphiti)

image-upload-openviking: image-build-openviking ## Build & upload OpenViking image to dev server
	$(call upload_image,openviking)

image-upload-zep: image-build-zep ## Build & upload Zep image to dev server
	$(call upload_image,zep)

image-upload-memobase: image-build-memobase ## Build & upload Memobase image to dev server
	$(call upload_image,memobase)

image-upload-supermemory: image-build-supermemory ## Build & upload Supermemory image to dev server
	$(call upload_image,supermemory)

image-upload-cognee-frontend: image-build-cognee-frontend ## Build & upload Cognee frontend image
	$(call upload_image,cognee-frontend)

image-upload-all: image-upload-memory image-upload-cognee image-upload-graphiti image-upload-openviking image-upload-zep image-upload-memobase image-upload-supermemory image-upload-cognee-frontend ## Build & upload all images

# =============================================================================
# KGS Platform — Dedicated Dockerfile in deployment/dev/
# =============================================================================

image-build-kgs: ## Build KGS Platform Docker image
	docker build -t vnp/kgs-platform:latest -f Dockerfile.kgs $(MONOREPO_ROOT)

image-upload-kgs: image-build-kgs ## Build & upload KGS Platform image
	$(call upload_image,kgs-platform)

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
	scp config/memory.yaml $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/config/
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
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory up -d --pull never'

remote-up-monolith: ## Start Memory Monolith on dev server (API+UI single binary)
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith up -d --pull never'

remote-up-ui: ## Start Memory + Cognee UI on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory --profile ui up -d --pull never'

remote-up-all: ## Start Memory + Monolith + KGS + UI + Monitoring on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith --profile memory --profile kgs --profile ui --profile monitoring up -d --pull never'

remote-down: ## Stop all services on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down'

remote-restart: remote-down remote-up ## Restart core Memory services on dev server

remote-ps: ## Show running containers on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring ps'

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

remote-logs-memory: ## Follow Memory Monolith logs on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose logs -f vnp-memory'

# =============================================================================
# Full Deployment Pipeline
# =============================================================================

deploy: ## Full deploy: build all Go images + upload + sync config + start
	@echo "🚀 Starting full deployment to $(DEV_SERVER) ..."
	$(MAKE) image-upload-all
	$(MAKE) sync-config
	$(MAKE) remote-down
	$(MAKE) remote-up-ui
	@echo "✅ Deployment complete! Services running on $(DEV_SERVER)"

deploy-memory: ## Deploy only Memory services (no frontend build)
	@echo "🚀 Deploying Memory services to $(DEV_SERVER) ..."
	$(MAKE) image-upload-cognee
	$(MAKE) image-upload-graphiti
	$(MAKE) image-upload-openviking
	$(MAKE) image-upload-zep
	$(MAKE) image-upload-memobase
	$(MAKE) image-upload-supermemory
	$(MAKE) sync-config
	$(MAKE) remote-down
	$(MAKE) remote-up
	@echo "✅ Memory services deployed on $(DEV_SERVER)"

deploy-config: sync-config remote-restart ## Sync config and restart (no image rebuild)

# =============================================================================
# Monolith Deployment — Memory Monolith with embedded UI
# =============================================================================

deploy-monolith: ## Deploy Memory Monolith (build + upload + start) + sync nginx
	@echo "🚀 Deploying Memory Monolith to $(DEV_SERVER) ..."
	$(MAKE) image-upload-memory
	$(MAKE) sync-config
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith down || true'
	$(MAKE) remote-up-monolith
	@echo ""
	@echo "✅ Memory Monolith deployed!"
	@echo "  REST API + UI: http://$(DEV_SERVER):8080"
	@echo "  MCP:           http://$(DEV_SERVER):8082"
	@echo "  Health:        http://$(DEV_SERVER):8083/healthz"
	@echo ""
	@echo "💡 Run 'make sync-nginx' to update nginx on the gateway server."

# =============================================================================
# Gateway Server — Nginx reverse proxy (172.20.2.16 / c6.openledger.vn)
# =============================================================================

sync-nginx: ## Sync nginx config to gateway server and reload
	@echo "📂 Syncing nginx config to $(GATEWAY_SERVER) ..."
	scp nginx/c6-openledger-vn.conf $(GATEWAY_USER)@$(GATEWAY_SERVER):/tmp/c6-openledger-vn.conf
	ssh $(GATEWAY_USER)@$(GATEWAY_SERVER) '\
		docker cp /tmp/c6-openledger-vn.conf nginx:/etc/nginx/conf.d/c6-openledger-vn.conf && \
		docker exec nginx nginx -t && \
		docker exec nginx nginx -s reload && \
		rm -f /tmp/c6-openledger-vn.conf'
	@echo "✅ Nginx config updated and reloaded on $(GATEWAY_SERVER)"
	@echo "  https://c6.openledger.vn → $(DEV_SERVER):8080"

sync-nginx-certbot: ## Generate SSL cert for c6.openledger.vn via certbot
	@echo "🔐 Requesting SSL certificate for c6.openledger.vn ..."
	ssh $(GATEWAY_USER)@$(GATEWAY_SERVER) '\
		docker exec certbot certbot certonly \
			--webroot -w /var/www/certbot \
			-d c6.openledger.vn \
			--non-interactive --agree-tos \
			--email admin@openledger.vn'
	@echo "✅ Certificate issued. Run 'make sync-nginx' to reload nginx."

# =============================================================================
# Local development (run on this machine)
# =============================================================================

up: ## Start Memory services locally
	$(COMPOSE) --profile memory up -d --pull never

up-ui: ## Start Memory services + Cognee frontend locally
	$(COMPOSE) --profile memory --profile ui up -d --pull never

up-all: ## Start all services locally
	$(COMPOSE) --profile memory --profile kgs --profile ui --profile monitoring up -d --pull never

down: ## Stop and remove all containers locally
	$(COMPOSE) --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down

restart: down up ## Restart core services locally

logs: ## Follow logs locally
	$(COMPOSE) logs -f

ps: ## Show running containers locally
	$(COMPOSE) --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring ps

clean: ## Stop containers and remove volumes locally
	$(COMPOSE) --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down -v
	@echo "🧹 All containers and volumes removed."

status: ## Check health of all services
	@echo "=== Service Health ==="
	@$(COMPOSE) --profile memory --profile ui ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
