.PHONY: all build run test clean docker-build

APP_NAME = openviking
BUILD_DIR = bin

all: build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/openviking

run: build
	@echo "Running $(APP_NAME)..."
	@./$(BUILD_DIR)/$(APP_NAME)

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

docker-build:
	@echo "Building docker image..."
	@docker build -t $(APP_NAME):latest .
