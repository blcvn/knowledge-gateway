SERVER_IP ?= 10.10.141.23
REMOTE_DIR = /opt/ba-agent-system/services/ai-kg-service/deployment/docker
IMAGE_NAME = kgs-platform:dev
TAR_FILE = kgs-platform.tar

.PHONY: all build package upload deploy clean

# Default behavior: run everything
all: build package upload deploy clean

build:
	@echo "=> Building Docker image..."
	docker compose -f docker-compose.dev.yml build kgs-platform

package:
	@echo "=> Saving Docker image to tar..."
	docker save -o $(TAR_FILE) $(IMAGE_NAME)

upload:
	@echo "=> Uploading files to server $(SERVER_IP)..."
	ssh $(SERVER_IP) "mkdir -p $(REMOTE_DIR)/config"
	scp $(TAR_FILE) $(SERVER_IP):$(REMOTE_DIR)/
	scp docker-compose.dev.yml $(SERVER_IP):$(REMOTE_DIR)/
	scp -r config/* $(SERVER_IP):$(REMOTE_DIR)/config/

deploy:
	@echo "=> Deploying on server $(SERVER_IP)..."
	ssh $(SERVER_IP) "cd $(REMOTE_DIR) && \
		docker load -i $(TAR_FILE) && \
		docker compose -f docker-compose.dev.yml up -d kgs-platform && \
		rm $(TAR_FILE)"

clean:
	@echo "=> Cleaning up local files..."
	rm -f $(TAR_FILE)
