.PHONY: build test lint run clean docker help

# === Variables ===
APP_NAME    := vnp-gateway
VERSION     := 0.2.0
GO          := go
GOFLAGS     := -v
LDFLAGS     := -s -w -X main.version=$(VERSION)
DOCKER_IMG  := vnp-gateway:$(VERSION)

# === Default ===
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# === Build ===
build: ## Build the gateway binary
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/main.go

# === Test ===
test: ## Run all tests
	$(GO) test ./... -v -count=1 -timeout=60s

test-domain: ## Run domain unit tests only
	$(GO) test ./internal/domain/... -v -count=1

test-integration: ## Run integration tests only
	$(GO) test ./tests/integration/... -v -count=1

test-cover: ## Run tests with coverage report
	$(GO) test ./... -coverprofile=coverage.out -timeout=60s
	$(GO) tool cover -func=coverage.out | tail -1
	$(GO) tool cover -html=coverage.out -o coverage.html

# === Quality ===
lint: ## Run linters
	$(GO) vet ./...
	@echo "✅ go vet passed"

check-domain: ## Verify domain has zero external deps
	@grep -r '"github.com' internal/domain/ && echo "❌ FAIL: external import in domain" && exit 1 || echo "✅ domain: zero external deps"

# === Run ===
run: ## Run the gateway locally
	AUTH_DEV_MODE=true REST_PORT=8080 HEALTH_PORT=11080 $(GO) run ./cmd/main.go

run-dev: ## Run with auto-reload (requires air)
	air

# === Docker ===
docker: ## Build Docker image
	docker build -t $(DOCKER_IMG) .

docker-run: ## Run Docker container
	docker run --rm -p 8080:8080 -p 8081:8081 -p 8082:8082 -p 11080:11080 \
		-e AUTH_DEV_MODE=true \
		$(DOCKER_IMG)

# === Codegen ===
generate: ## Run Wire code generation
	$(GO) generate ./internal/infra/wire/...

# === Clean ===
clean: ## Clean build artifacts
	rm -rf bin/ coverage.out coverage.html
	$(GO) clean -cache -testcache

# === Info ===
info: ## Show project stats
	@echo "$(APP_NAME) v$(VERSION)"
	@echo "Go files:  $$(find . -name '*.go' | wc -l | tr -d ' ')"
	@echo "Go lines:  $$(find . -name '*.go' -exec cat {} + | wc -l | tr -d ' ')"
	@echo "Packages:  $$(find . -name '*.go' -exec dirname {} \; | sort -u | wc -l | tr -d ' ')"
	@echo "Specs:     $$(find specs -name '*.md' 2>/dev/null | wc -l | tr -d ' ')"
	@echo "Docs:      $$(find docs -name '*.md' 2>/dev/null | wc -l | tr -d ' ')"
