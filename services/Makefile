SERVICES := graphiti-knowledge ba-knowledge-service sm-connector cognee-cognify ov-resource sm-engine zep-memory ov-storage cognee-pipeline vnp-search-hub sm-search sm-mcp vnp-event sm-analytics vnp-platform ba-knowledge-worker memobase-context vnp-gateway memobase-pipeline ov-crypto zep-search graphiti-ingestion graphiti-store zep-user graphiti-search cognee-ingestion ov-session sm-project zep-graph ov-fs memobase-ingestion sm-profile graphiti-pipeline zep-core vnp-admin ov-admin zep-thread sm-memory sm-document zep-admin sm-auth memobase-engine ov-search cognee-search

.PHONY: all build test lint docker tidy

all: build

build:
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		cd $$svc && go build -o bin/$$svc ./cmd/server && cd ..; \
	done

test:
	@for svc in $(SERVICES); do \
		echo "Testing $$svc..."; \
		cd $$svc && go test ./... -v && cd ..; \
	done

tidy:
	@for svc in $(SERVICES); do \
		echo "Tidying $$svc..."; \
		cd $$svc && go mod tidy && cd ..; \
	done

docker:
	@for svc in $(SERVICES); do \
		echo "Building Docker image for $$svc..."; \
		cd $$svc && docker build -t vnp-memory/$$svc:latest . && cd ..; \
	done
