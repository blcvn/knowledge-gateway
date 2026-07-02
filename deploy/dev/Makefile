SHELL := /bin/bash

# =============================================================================
# Target server
# Binary is compiled locally, rsync'd to server, and mounted into alpine:3.19
# =============================================================================
SERVER_HOST  ?= 172.20.2.39
SERVER_USER  ?= ubuntu
SERVER_PORT  ?= 22
SERVER_DIR   ?= /opt/kg-service/dev

SSH_OPTS     := -p $(SERVER_PORT) -o StrictHostKeyChecking=no
SSH          := ssh $(SSH_OPTS) $(SERVER_USER)@$(SERVER_HOST)
RSYNC_OPTS   := -avz --progress -e "ssh $(SSH_OPTS)"

# =============================================================================
# Local paths (relative to project root, two levels up from deploy/dev)
# =============================================================================
PROJECT_ROOT := $(shell git -C "$(dir $(abspath $(lastword $(MAKEFILE_LIST))))" rev-parse --show-toplevel)
BIN_DIR      := $(CURDIR)/bin
BINARY       := $(BIN_DIR)/kg-service
MIGRATIONS   := $(PROJECT_ROOT)/migrations

# =============================================================================
# Targets
# =============================================================================
.PHONY: help build sync deploy up down clean logs ps restart status

help:
	@printf '%s\n' \
		'Deploy targets (server: $(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR))' \
		'' \
		'  build    Cross-compile binary for linux/amd64' \
		'  sync     Rsync binary + compose files + migrations to server' \
		'  deploy   build → sync → docker compose up on server  (default workflow)' \
		'' \
		'  up       docker compose up on server (no image build needed)' \
		'  down     docker compose down on server' \
		'  clean    docker compose down -v on server (removes volumes!)' \
		'  logs     Follow logs of all services on server' \
		'  ps       Show running containers on server' \
		'  restart  Restart only kg-service on server' \
		'  status   Show service health on server' \
		'' \
		'Note: kg-service uses alpine:3.19 with the binary mounted — no docker build on server.' \
		'' \
		'Override defaults:' \
		'  make deploy SERVER_USER=root SERVER_PORT=2222'

# ---------------------------------------------------------------------------
# 1. Cross-compile for linux/amd64
# ---------------------------------------------------------------------------
build:
	@echo "==> Building linux/amd64 binary..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath -ldflags="-s -w" -o $(BINARY) $(PROJECT_ROOT)
	@echo "    Binary: $(BINARY) ($$(du -sh $(BINARY) | cut -f1))"

# ---------------------------------------------------------------------------
# 2. Sync all artefacts to server
# ---------------------------------------------------------------------------
sync: _check-binary
	@echo "==> Creating remote directory $(SERVER_DIR)..."
	$(SSH) "mkdir -p $(SERVER_DIR)/bin $(SERVER_DIR)/migrations"
	@echo "==> Syncing binary..."
	rsync $(RSYNC_OPTS) $(BINARY) $(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/bin/kg-service
	@echo "==> Syncing compose files..."
	rsync $(RSYNC_OPTS) \
		$(CURDIR)/docker-compose.yml \
		$(CURDIR)/.env.example \
		$(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/
	@echo "==> Syncing migrations..."
	rsync $(RSYNC_OPTS) --delete \
		$(MIGRATIONS)/ \
		$(SERVER_USER)@$(SERVER_HOST):$(SERVER_DIR)/migrations/
	@echo "==> Ensuring .env exists on server..."
	$(SSH) "[ -f $(SERVER_DIR)/.env ] || cp $(SERVER_DIR)/.env.example $(SERVER_DIR)/.env && echo '    .env created from .env.example — edit it on the server if needed'"

# ---------------------------------------------------------------------------
# 3. Full deploy: build → sync → up
# ---------------------------------------------------------------------------
deploy: build sync
	@echo "==> Starting services on server..."
	$(SSH) "cd $(SERVER_DIR) && docker compose up -d"
	@echo "==> Deploy complete. Service: http://$(SERVER_HOST):8082"

# ---------------------------------------------------------------------------
# Remote compose helpers
# ---------------------------------------------------------------------------
up:
	$(SSH) "cd $(SERVER_DIR) && docker compose up -d"

down:
	$(SSH) "cd $(SERVER_DIR) && docker compose down"

clean:
	$(SSH) "cd $(SERVER_DIR) && docker compose down -v"

logs:
	$(SSH) "cd $(SERVER_DIR) && docker compose logs -f"

ps:
	$(SSH) "cd $(SERVER_DIR) && docker compose ps"

restart:
	$(SSH) "cd $(SERVER_DIR) && docker compose restart kg-service"

status:
	$(SSH) "cd $(SERVER_DIR) && docker compose ps && echo '' && docker compose logs --tail=20 kg-service"

# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------
_check-binary:
	@if [ ! -f $(BINARY) ]; then \
		echo "ERROR: binary not found at $(BINARY). Run 'make build' first."; \
		exit 1; \
	fi
