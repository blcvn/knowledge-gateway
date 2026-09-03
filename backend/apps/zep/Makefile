.PHONY: build run test clean docker-build

APP_NAME := zep-monolith
CMD_DIR := ./cmd/zep

build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/$(APP_NAME) $(CMD_DIR)/main.go

run: build
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME)

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -rf bin/

docker-build:
	@echo "Building docker image..."
	cd ../../ && docker build -f apps/zep/Dockerfile -t $(APP_NAME):latest .
