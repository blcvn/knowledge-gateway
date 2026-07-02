SHELL := /bin/bash

APP_NAME := kg-service
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP_NAME)
IMAGE ?= kg-service:local

.PHONY: help build run test docker-build deploy-compose deploy-compose-integration deploy-compose-runtime-validation deploy-compose-codegraph-runtime deploy-k8s deploy-vm migrate integration-test validate-runtime-profile validate-codegraph-runtime compose-down compose-down-integration compose-down-runtime-validation compose-down-codegraph-runtime codegraph-refresh codegraph-sync-build codegraph-sync-sync codegraph-sync-sync-dry codegraph-sync-mcp bootstrap-codegraph-ontology verify-codegraph-ontology

help:
	@printf '%s\n' \
		'Targets:' \
		'  build            Build the service binary into ./bin/kg-service' \
		'  run              Run the service through main.go' \
		'  test             Run the full Go test suite' \
		'  docker-build     Build the container image' \
		'  deploy-compose   Start the Compose stack' \
		'  deploy-compose-integration Start the integration Compose stack' \
		'  deploy-compose-runtime-validation Start the runtime validation Compose stack' \
		'  deploy-compose-codegraph-runtime Start the CodeGraph Compose stack' \
		'  deploy-k8s       Apply the Kubernetes deployment path' \
		'  deploy-vm        Build and start the VM deployment path' \
		'  migrate          Apply Postgres migrations' \
		'  integration-test Run the repeatable integration validation' \
		'  validate-runtime-profile Run the profile-aware deployment validation' \
		'  validate-codegraph-runtime Boot and validate the CodeGraph Compose flow' \
		'  compose-down     Stop the Compose stack' \
		'  compose-down-codegraph-runtime Stop the CodeGraph Compose stack' \
		'  codegraph-refresh Refresh the local CodeGraph incremental index' \
		'  codegraph-sync-build Build the codegraph-sync bridge binary' \
		'  codegraph-sync-sync Run a full codegraph-sync pass' \
		'  codegraph-sync-sync-dry Run a dry-run codegraph-sync pass' \
		'  codegraph-sync-mcp Start the codegraph-sync MCP server' \
		'  bootstrap-codegraph-ontology Bootstrap the code-graph ontology domain' \
		'  verify-codegraph-ontology Verify the code-graph ontology domain'

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) .

run:
	go run .

test:
	go test ./...

docker-build:
	docker build -t $(IMAGE) .

deploy-compose:
	./scripts/deploy-compose.sh

deploy-compose-integration:
	./scripts/deploy-compose-integration.sh

deploy-compose-runtime-validation:
	./scripts/deploy-compose-runtime-validation.sh

deploy-compose-codegraph-runtime:
	./scripts/deploy-compose-codegraph-runtime.sh $(ARGS)

deploy-k8s:
	./scripts/deploy-k8s.sh

deploy-vm:
	./scripts/deploy-vm.sh

migrate:
	./scripts/migrate-postgres.sh

integration-test:
	./scripts/integration-test.sh

validate-runtime-profile:
	./scripts/validate-runtime-profile.sh

validate-codegraph-runtime:
	./scripts/validate-codegraph-runtime.sh $(ARGS)

compose-down:
	docker compose -f deploy/compose/docker-compose.yml down

compose-down-integration:
	docker compose -f deploy/compose/integration-test/docker-compose.yml down

compose-down-runtime-validation:
	docker compose -f deploy/compose/runtime-validation/docker-compose.yml down

compose-down-codegraph-runtime:
	docker compose -f deploy/compose/codegraph-runtime/docker-compose.yml down

codegraph-refresh:
	./scripts/codegraph-refresh.sh

codegraph-sync-build:
	./codegraph-sync/build

codegraph-sync-sync:
	./codegraph-sync/sync

codegraph-sync-sync-dry:
	./codegraph-sync/sync:dry

codegraph-sync-mcp:
	./codegraph-sync/mcp

bootstrap-codegraph-ontology:
	./scripts/bootstrap-codegraph-ontology.sh

verify-codegraph-ontology:
	./scripts/verify-codegraph-ontology.sh
