# KGS Docker Deployment - Makefile helpers

COMPOSE=docker compose -f docker-compose.yml

.PHONY: up down ps logs clean build

## Start all services in the background
up:
	@if [ ! -f .env ]; then \
		echo "No .env found — copying from .env.example ..."; \
		cp .env.example .env; \
	fi
	$(COMPOSE) up -d

## Stop and remove containers (keeps volumes)
down:
	$(COMPOSE) down

## Rebuild and restart kgs-platform only (no dependency restart)
rebuild:
	$(COMPOSE) build --no-cache kgs-platform
	$(COMPOSE) up -d --no-deps kgs-platform

## Show status of all containers
ps:
	$(COMPOSE) ps

## Tail logs (optionally pass SERVICE=<name> to filter)
logs:
	$(COMPOSE) logs -f $(SERVICE)

## Remove containers AND volumes (full reset)
clean:
	$(COMPOSE) down -v --remove-orphans

## Show this help
help:
	@grep -E '^##' Makefile | sed 's/## //'

.DEFAULT_GOAL := help
