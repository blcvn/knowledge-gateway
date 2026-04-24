SERVER_IP ?= 10.10.141.23
REMOTE_COMPOSE_DIR = /opt/ba-agent-system/deployment/docker
REMOTE_CONFIG_DIR = /opt/ba-agent-system/services/ai-kg-service/kgs-platform/configs
IMAGE_NAME = kgs-platform:dev
TAR_FILE = kgs-platform.tar
SERVICE_NAME = ai-kg-service
COMPOSE_FILE = docker-compose.dev.yml

.PHONY: all build package upload deploy clean

# Default behavior: build, package, upload and deploy
all: build package upload deploy clean

build:
	@echo "=> Building Docker image for $(SERVICE_NAME)..."
	docker compose -f $(COMPOSE_FILE) build $(SERVICE_NAME)

package:
	@echo "=> Saving Docker image to $(TAR_FILE)..."
	docker save -o $(TAR_FILE) $(IMAGE_NAME)

upload:
	@echo "=> Uploading image and config to server $(SERVER_IP)..."
	ssh $(SERVER_IP) "mkdir -p $(REMOTE_CONFIG_DIR)"
	scp $(TAR_FILE) $(SERVER_IP):$(REMOTE_COMPOSE_DIR)/
	scp ../../kgs-platform/configs/config.yaml $(SERVER_IP):$(REMOTE_CONFIG_DIR)/config.yaml

deploy:
	@echo "=> Deploying $(SERVICE_NAME) on server $(SERVER_IP)..."
	ssh $(SERVER_IP) "cd $(REMOTE_COMPOSE_DIR) && \
		docker load -i $(TAR_FILE) && \
		docker compose -f $(COMPOSE_FILE) up -d $(SERVICE_NAME) && \
		rm $(TAR_FILE)"

clean:
	@echo "=> Cleaning up local files..."
	rm -f $(TAR_FILE)
