###############################################################################
# VNP Memory — Dev Server Deployment Makefile (172.20.2.39)
#
# LOCAL COMPILE + BINARY SYNC deployment model:
# - All Go apps are cross-compiled locally for linux/amd64
# - Frontend (UI) is built locally with npm
# - Compiled binaries + UI dist are synced to server via rsync
# - Containers mount binaries from host filesystem
#
# Quick Start:
#   make compile-all      # Cross-compile all Go apps + build UI
#   make deploy-monolith  # Full pipeline: compile + sync + restart monolith
#   make deploy           # Full pipeline: compile all + sync + restart all
###############################################################################

.PHONY: help env \
        compile-memory compile-cognee compile-graphiti compile-openviking \
        compile-zep compile-memobase compile-supermemory compile-kgs \
        compile-ui compile-apps compile-all \
        sync-bin sync-ui sync-config sync-all \
        remote-up remote-up-monolith remote-up-ui remote-up-all \
        remote-down remote-restart remote-ps remote-logs \
        remote-logs-cognee remote-logs-graphiti remote-logs-zep \
        remote-logs-openviking remote-logs-memory \
        deploy deploy-monolith deploy-memory deploy-kgs deploy-config \
        setup-runtime sync-nginx sync-nginx-certbot \
        up up-ui up-all down restart logs ps clean status
.DEFAULT_GOAL := help

COMPOSE := docker compose
DEV_SERVER := 172.20.2.39
DEV_USER := ubuntu
DEPLOY_DIR := /opt/vnp-memory

# Monorepo root (build context for all apps)
MONOREPO_ROOT := ../..

# Output directory for compiled binaries (local)
BIN_DIR := $(CURDIR)/bin

# UI dist directory (local)
UI_DIST_DIR := $(CURDIR)/ui-dist

# Cross-compile flags for Linux amd64
GO_BUILD_FLAGS := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
GO_LDFLAGS := -ldflags="-s -w"

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

# rsync flags
RSYNC := rsync -avz --progress

help: ## Show this help
	@echo ""
	@echo "\033[1;35m╔══════════════════════════════════════════════════════════════╗\033[0m"
	@echo "\033[1;35m║  VNP Memory — Dev Deployment (Local Compile + Binary Sync)  ║\033[0m"
	@echo "\033[1;35m╚══════════════════════════════════════════════════════════════╝\033[0m"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[1;33m⚡ Quick Start:\033[0m"
	@echo "  \033[32mmake deploy-monolith\033[0m  Compile Memory Monolith + UI → sync → restart"
	@echo "  \033[32mmake deploy\033[0m           Compile all services → sync → restart"
	@echo "  \033[32mmake compile-all\033[0m      Cross-compile all Go apps + build UI"
	@echo "  \033[32mmake sync-all\033[0m         Sync binaries + UI + configs to server"
	@echo ""

env: ## Create .env from .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env — please edit API keys before starting."; \
	else \
		echo "⚠️  .env already exists, skipping."; \
	fi

# =============================================================================
# Cross-Compilation — All Go apps compiled locally for linux/amd64
# =============================================================================

compile-memory: compile-ui ## Cross-compile Memory Monolith (with embedded UI)
	@echo "\033[36m⚙  Compiling Memory Monolith (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	@# Embed UI assets into Go binary
	@echo "  → Embedding UI assets..."
	@rm -rf $(MONOREPO_ROOT)/$(MEMORY_APP)/internal/ui/ui_dist
	@mkdir -p $(MONOREPO_ROOT)/$(MEMORY_APP)/internal/ui/ui_dist
	@cp -r $(MONOREPO_ROOT)/ui/dist/* $(MONOREPO_ROOT)/$(MEMORY_APP)/internal/ui/ui_dist/
	@echo "  → Building Go binary..."
	cd $(MONOREPO_ROOT) && $(GO_BUILD_FLAGS) go build \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/vnp-memory \
		./$(MEMORY_APP)/cmd/server
	@echo "\033[32m✓ Compiled: bin/vnp-memory\033[0m"

compile-cognee: ## Cross-compile Cognee (linux/amd64)
	@echo "\033[36m⚙  Compiling Cognee (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	cd $(MONOREPO_ROOT)/$(COGNEE_APP) && $(GO_BUILD_FLAGS) go build \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/cognee \
		./cmd/cognee/
	@echo "\033[32m✓ Compiled: bin/cognee\033[0m"

compile-graphiti: ## Cross-compile Graphiti (linux/amd64)
	@echo "\033[36m⚙  Compiling Graphiti (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	cd $(MONOREPO_ROOT)/$(GRAPHITI_APP) && $(GO_BUILD_FLAGS) go build \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/graphiti \
		./cmd/graphiti/
	@echo "\033[32m✓ Compiled: bin/graphiti\033[0m"

compile-openviking: ## Cross-compile OpenViking (linux/amd64)
	@echo "\033[36m⚙  Compiling OpenViking (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	cd $(MONOREPO_ROOT)/$(OPENVIKING_APP) && $(GO_BUILD_FLAGS) go build \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/openviking \
		./cmd/openviking/
	@echo "\033[32m✓ Compiled: bin/openviking\033[0m"

compile-zep: ## Cross-compile Zep (linux/amd64)
	@echo "\033[36m⚙  Compiling Zep (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	cd $(MONOREPO_ROOT)/$(ZEP_APP) && $(GO_BUILD_FLAGS) go build \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/zep \
		./cmd/zep/
	@echo "\033[32m✓ Compiled: bin/zep\033[0m"

compile-memobase: ## Cross-compile Memobase (linux/amd64)
	@echo "\033[36m⚙  Compiling Memobase (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	cd $(MONOREPO_ROOT)/$(MEMOBASE_APP) && $(GO_BUILD_FLAGS) go build \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/memobase \
		./cmd/memobase/
	@echo "\033[32m✓ Compiled: bin/memobase\033[0m"

compile-supermemory: ## Cross-compile Supermemory (linux/amd64)
	@echo "\033[36m⚙  Compiling Supermemory (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	cd $(MONOREPO_ROOT)/$(SUPERMEMORY_APP) && $(GO_BUILD_FLAGS) go build \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/supermemory \
		./cmd/supermemory/
	@echo "\033[32m✓ Compiled: bin/supermemory\033[0m"

compile-kgs: ## Cross-compile KGS Platform (linux/amd64)
	@echo "\033[36m⚙  Compiling KGS Platform (linux/amd64)...\033[0m"
	@mkdir -p $(BIN_DIR)
	cd $(MONOREPO_ROOT)/kgs-platform && $(GO_BUILD_FLAGS) go build \
		-mod=mod \
		$(GO_LDFLAGS) \
		-o $(BIN_DIR)/kgs-server \
		./cmd/server/
	@echo "\033[32m✓ Compiled: bin/kgs-server\033[0m"

compile-ui: ## Build UI frontend (npm run build)
	@echo "\033[36m⚙  Building UI frontend...\033[0m"
	@mkdir -p $(UI_DIST_DIR)
	cd $(MONOREPO_ROOT)/ui && npm run build
	@echo "\033[32m✓ UI built: $(MONOREPO_ROOT)/ui/dist/\033[0m"

compile-apps: compile-cognee compile-graphiti compile-openviking compile-zep compile-memobase compile-supermemory ## Cross-compile all individual app services

compile-all: compile-apps compile-kgs compile-memory ## Cross-compile ALL services (apps + KGS + Memory Monolith with UI)
	@echo ""
	@echo "\033[32m✅ All services compiled successfully!\033[0m"
	@echo ""
	@ls -lh $(BIN_DIR)/
	@echo ""

# =============================================================================
# Sync — Push compiled binaries and configs to dev server via rsync
# =============================================================================

sync-bin: ## Sync compiled binaries to dev server
	@echo "📦 Syncing binaries to $(DEV_SERVER):$(DEPLOY_DIR)/bin/ ..."
	ssh $(DEV_USER)@$(DEV_SERVER) 'mkdir -p $(DEPLOY_DIR)/bin'
	$(RSYNC) $(BIN_DIR)/ $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/bin/
	@echo "  → Setting executable permissions..."
	ssh $(DEV_USER)@$(DEV_SERVER) 'chmod +x $(DEPLOY_DIR)/bin/*'
	@echo "✅ Binaries synced to $(DEV_SERVER)"

sync-ui: ## Sync UI dist to dev server
	@echo "📦 Syncing UI dist to $(DEV_SERVER):$(DEPLOY_DIR)/ui-dist/ ..."
	ssh $(DEV_USER)@$(DEV_SERVER) 'mkdir -p $(DEPLOY_DIR)/ui-dist'
	$(RSYNC) --delete $(MONOREPO_ROOT)/ui/dist/ $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/ui-dist/
	@echo "✅ UI dist synced to $(DEV_SERVER)"

sync-config: ## Sync docker-compose, configs, and .env to dev server
	@echo "📂 Syncing configuration to $(DEV_SERVER):$(DEPLOY_DIR) ..."
	ssh $(DEV_USER)@$(DEV_SERVER) 'mkdir -p $(DEPLOY_DIR)/config'
	scp docker-compose.yml $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/
	scp Dockerfile.runtime $(DEV_USER)@$(DEV_SERVER):$(DEPLOY_DIR)/
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

sync-all: sync-bin sync-config ## Sync binaries + configs to dev server (no UI)
	@echo "✅ All files synced to $(DEV_SERVER)"

sync-all-ui: sync-bin sync-ui sync-config ## Sync binaries + UI dist + configs to dev server
	@echo "✅ All files (including UI) synced to $(DEV_SERVER)"

# =============================================================================
# Runtime Base Image — Build once on dev server
# =============================================================================

setup-runtime: sync-config ## Build vnp/runtime:latest base image on dev server (run once)
	@echo "🔧 Building runtime base image on $(DEV_SERVER)..."
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker build -t vnp/runtime:latest -f Dockerfile.runtime .'
	@echo "✅ vnp/runtime:latest built on $(DEV_SERVER)"

# =============================================================================
# Remote Docker Compose Operations
# =============================================================================

remote-up: ## Start Memory services on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory up -d --pull never'

remote-up-monolith: ## Start Memory Monolith on dev server (API+UI single binary)
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith up -d --pull never'

remote-up-ui: ## Start Memory + Cognee UI on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile memory --profile ui up -d --pull never'

remote-up-kgs: ## Start KGS Platform on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile kgs up -d --pull never'

remote-up-all: ## Start Memory + Monolith + KGS + UI + Monitoring on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith --profile memory --profile kgs --profile ui --profile monitoring up -d --pull never'

remote-down: ## Stop all services on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down'

remote-restart: remote-down remote-up ## Restart core Memory services on dev server

remote-restart-monolith: ## Restart only Memory Monolith on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile monolith down || true'
	$(MAKE) remote-up-monolith

remote-restart-kgs: ## Restart only KGS Platform on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose --profile kgs down || true'
	$(MAKE) remote-up-kgs

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

remote-logs-kgs: ## Follow KGS Platform logs on dev server
	ssh $(DEV_USER)@$(DEV_SERVER) 'cd $(DEPLOY_DIR) && docker compose logs -f kgs-platform'

# =============================================================================
# Full Deployment Pipelines
# =============================================================================

deploy: compile-all sync-all ## Full deploy: compile all → sync → restart
	@echo "🚀 Starting full deployment to $(DEV_SERVER) ..."
	$(MAKE) remote-down
	$(MAKE) remote-up-ui
	@echo ""
	@echo "✅ Deployment complete! Services running on $(DEV_SERVER)"
	@echo "  Cognee API:     http://$(DEV_SERVER):8000"
	@echo "  Graphiti API:   http://$(DEV_SERVER):8001"
	@echo "  Zep API:        http://$(DEV_SERVER):8002"
	@echo "  OpenViking API: http://$(DEV_SERVER):1933"
	@echo "  Neo4j Browser:  http://$(DEV_SERVER):7474"

deploy-memory: compile-apps sync-bin sync-config ## Deploy individual Memory services (no Monolith, no UI)
	@echo "🚀 Deploying Memory services to $(DEV_SERVER) ..."
	$(MAKE) remote-down
	$(MAKE) remote-up
	@echo "✅ Memory services deployed on $(DEV_SERVER)"

deploy-monolith: compile-memory sync-bin sync-config ## Deploy Memory Monolith (compile + embed UI + sync + restart)
	@echo "🚀 Deploying Memory Monolith to $(DEV_SERVER) ..."
	$(MAKE) remote-restart-monolith
	@echo ""
	@echo "✅ Memory Monolith deployed!"
	@echo "  REST API + UI: http://$(DEV_SERVER):8080"
	@echo "  MCP:           http://$(DEV_SERVER):8082"
	@echo "  Health:        http://$(DEV_SERVER):8083/healthz"
	@echo ""
	@echo "💡 Run 'make sync-nginx' to update nginx on the gateway server."

deploy-kgs: compile-kgs sync-bin sync-config ## Deploy KGS Platform (compile + sync + restart)
	@echo "🚀 Deploying KGS Platform to $(DEV_SERVER) ..."
	$(MAKE) remote-restart-kgs
	@echo "✅ KGS Platform deployed on $(DEV_SERVER)"
	@echo "  HTTP: http://$(DEV_SERVER):8010"
	@echo "  gRPC: http://$(DEV_SERVER):9010"

deploy-config: sync-config remote-restart ## Sync config and restart (no recompile)

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

clean: ## Stop containers, remove volumes, and clean compiled binaries
	$(COMPOSE) --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down -v
	rm -rf $(BIN_DIR)
	@echo "🧹 All containers, volumes, and compiled binaries removed."

status: ## Check health of all services
	@echo "=== Service Health ==="
	@$(COMPOSE) --profile memory --profile ui ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
