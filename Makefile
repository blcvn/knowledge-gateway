SHELL := /bin/bash

APP_NAME := kg-service
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP_NAME)
IMAGE ?= kg-service:local

.PHONY: help build run test docker-build deploy-compose deploy-k8s deploy-vm migrate integration-test compose-down

help:
	@printf '%s\n' \
		'Targets:' \
		'  build            Build the service binary into ./bin/kg-service' \
		'  run              Run the service through main.go' \
		'  test             Run the full Go test suite' \
		'  docker-build     Build the container image' \
		'  deploy-compose   Start the Compose stack' \
		'  deploy-k8s       Apply the Kubernetes deployment path' \
		'  deploy-vm        Build and start the VM deployment path' \
		'  migrate          Apply Postgres migrations' \
		'  integration-test Run the repeatable integration validation' \
		'  compose-down     Stop the Compose stack'

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

deploy-k8s:
	./scripts/deploy-k8s.sh

deploy-vm:
	./scripts/deploy-vm.sh

migrate:
	./scripts/migrate-postgres.sh

integration-test:
	./scripts/integration-test.sh

compose-down:
	docker compose -f deploy/compose/docker-compose.yml down
