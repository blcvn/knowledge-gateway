---
id: MERGE-P4-T5
title: "Cleanup: Archive Old Services + Update CI/CD"
phase: P4
service: workspace / CI
priority: P3
status: Done
estimated: 3h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P4-T4]
---

## Mục Tiêu

Dọn dẹp cuối cùng: archive các service directories cũ, cập nhật CI/CD pipeline, và verify toàn bộ hệ thống chạy với 8 services.

## 1. Archive Old Service Directories

Tạo script để move các service cũ vào `services/archived/`:

```bash
#!/bin/bash
# scripts/archive-old-services.sh

ARCHIVE_DIR="services/archived"
mkdir -p "$ARCHIVE_DIR"

OLD_SERVICES=(
    # Cognee (merged into kg-service)
    "cognee-cognify"
    "cognee-ingestion"
    "cognee-pipeline"
    "cognee-search"
    
    # Graphiti (merged into kg-service)
    "graphiti-ingestion"
    "graphiti-knowledge"
    "graphiti-pipeline"
    "graphiti-search"
    "graphiti-store"
    
    # Memobase (merged into memory-service)
    "memobase-context"
    "memobase-engine"
    "memobase-ingestion"
    "memobase-pipeline"
    
    # OpenViking (merged into storage-service)
    "ov-admin"
    "ov-crypto"
    "ov-fs"
    "ov-resource"
    "ov-search"
    "ov-session"
    # ov-storage kept as base structure (renamed to storage-service)
    
    # Supermemory (split: platform + memory + search)
    "sm-analytics"
    "sm-connector"
    "sm-document"
    "sm-engine"
    "sm-mcp"
    "sm-memory"
    "sm-profile"
    "sm-project"
    "sm-search"
    
    # VNP Core (merged into platform/obs/pipeline)
    "vnp-admin"
    "vnp-dashboard"
    "vnp-event"
    "vnp-infra"
    "vnp-observability"
    "vnp-pipelines"
    
    # Zep (merged into memory-service)
    "zep-admin"
    "zep-core"
    "zep-graph"
    "zep-memory"
    "zep-search"
    "zep-thread"
    "zep-user"
    
    # BA Knowledge (merged into pipeline-service)
    "ba-knowledge-service"
    "ba-knowledge-worker"
)

for svc in "${OLD_SERVICES[@]}"; do
    src="services/$svc"
    if [ -d "$src" ]; then
        echo "Archiving: $src"
        mv "$src" "$ARCHIVE_DIR/$svc"
    else
        echo "Already removed or not found: $src"
    fi
done

echo "Done. Archived $(ls $ARCHIVE_DIR | wc -l) services."
```

## 2. Rename Docker Compose Files

```bash
# Rename old docker-compose files for clarity
mv docker-compose.yml docker-compose.scale.yml          # Original 47-service config
mv docker-compose.compact.yml docker-compose.compact.archived.yml  # Old compact (SOL-001)

# New file becomes the default
mv docker-compose.consolidated.yml docker-compose.yml   # 8-service is now default
```

## 3. CI/CD Pipeline Update (`.github/workflows/`)

### `ci.yml` — Unit Tests

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      
      - name: Build all services
        run: go build ./...
      
      - name: Run unit tests
        run: go test -count=1 -timeout 120s ./...
      
      - name: Vet
        run: go vet ./...

  # Matrix build for 8 services
  build-services:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service:
          - gateway
          - services/vnp-platform
          - services/kg-service
          - services/memory-service
          - services/storage-service
          - services/search-service
          - services/pipeline-service
          - services/obs-service
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: Build ${{ matrix.service }}
        run: go build ./${{ matrix.service }}/...
```

### `integration.yml` — Integration Tests

```yaml
# .github/workflows/integration.yml
name: Integration Tests

on:
  push:
    branches: [main]
  workflow_dispatch:

jobs:
  integration-sol003:
    runs-on: ubuntu-latest
    timeout-minutes: 20
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      
      - name: Start consolidated stack
        run: docker compose up -d --wait
        timeout-minutes: 5
      
      - name: Run integration tests
        run: |
          INTEGRATION_ASSUME_UP=true \
          go test -v -timeout 300s -tags integration \
            ./tests/integration/sol003/...
        env:
          AUTH_JWT_PRIVATE_KEY: ${{ secrets.AUTH_JWT_PRIVATE_KEY }}
          ZEP_API_KEY: ""
          ZEP_ENABLED: "false"
      
      - name: Collect logs on failure
        if: failure()
        run: docker compose logs --tail=100
      
      - name: Teardown
        if: always()
        run: docker compose down -v
```

### `docker-build.yml` — Docker Images

```yaml
# .github/workflows/docker-build.yml
name: Docker Build

on:
  push:
    tags: ['v*']

jobs:
  build-push:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - service: gateway
            dockerfile: gateway/Dockerfile
            image: vnp-memory/gateway
          - service: vnp-platform
            dockerfile: services/vnp-platform/Dockerfile
            image: vnp-memory/vnp-platform
          - service: kg-service
            dockerfile: services/kg-service/Dockerfile
            image: vnp-memory/kg-service
          - service: memory-service
            dockerfile: services/memory-service/Dockerfile
            image: vnp-memory/memory-service
          - service: storage-service
            dockerfile: services/storage-service/Dockerfile
            image: vnp-memory/storage-service
          - service: search-service
            dockerfile: services/search-service/Dockerfile
            image: vnp-memory/search-service
          - service: pipeline-service
            dockerfile: services/pipeline-service/Dockerfile
            image: vnp-memory/pipeline-service
          - service: obs-service
            dockerfile: services/obs-service/Dockerfile
            image: vnp-memory/obs-service
    steps:
      - uses: actions/checkout@v4
      - uses: docker/build-push-action@v5
        with:
          context: .
          file: ${{ matrix.dockerfile }}
          push: true
          tags: ghcr.io/${{ matrix.image }}:${{ github.ref_name }}
```

## 4. Makefile Updates

```makefile
# Makefile additions for SOL-003

# === Build ===
.PHONY: build
build:
	@echo "Building all 8 services..."
	@go build ./gateway/...
	@go build ./services/vnp-platform/...
	@go build ./services/kg-service/...
	@go build ./services/memory-service/...
	@go build ./services/storage-service/...
	@go build ./services/search-service/...
	@go build ./services/pipeline-service/...
	@go build ./services/obs-service/...
	@echo "✅ All services built"

# === Test ===
.PHONY: test
test:
	go test -count=1 -timeout 120s ./...

.PHONY: test-integration
test-integration:
	INTEGRATION_ASSUME_UP=true \
	go test -v -timeout 300s -tags integration \
		./tests/integration/sol003/...

# === Docker ===
.PHONY: up
up:
	docker compose up -d --wait

.PHONY: down
down:
	docker compose down

.PHONY: logs
logs:
	docker compose logs -f

.PHONY: ps
ps:
	docker compose ps

# === Dev ===
.PHONY: archive-old-services
archive-old-services:
	bash scripts/archive-old-services.sh
```

## 5. Documentation Updates

Cập nhật các file docs:

```
docs/
├── README.md                  → Update service count (8 services)
├── architecture.md            → Update architecture diagram
├── getting-started.md         → Update quick start (docker compose up)
└── services/                  → Add docs cho 7 new services
    ├── vnp-platform.md
    ├── kg-service.md
    ├── memory-service.md
    ├── storage-service.md
    ├── search-service.md
    ├── pipeline-service.md
    └── obs-service.md
```

## 6. apps/ Directory Handling

```
apps/
├── OpenViking/    → Không xóa — đây là reference Python implementation
├── cognee/        → Không xóa — đây là Python Cognee source
├── graphiti/      → Không xóa — reference
├── memobase/      → Không xóa — reference  
├── memory/        → Không xóa — reference
├── supermemory/   → Không xóa — reference
└── zep/           → Không xóa — reference
```

**Note:** `apps/` là reference implementations, KHÔNG phải Go services. Giữ nguyên, nhưng loại khỏi `go.work` nếu chúng đã được removed trong P4-T1.

## Acceptance Criteria

- [ ] `services/archived/` chứa 42 old service directories
- [ ] `docker-compose.yml` là file 8-service consolidated
- [ ] `docker-compose.scale.yml` là file 47-service (original) cho reference
- [ ] `go.work` không còn reference đến any archived service
- [ ] `.github/workflows/ci.yml` builds 8 services + passes unit tests
- [ ] `.github/workflows/integration.yml` runs e2e tests
- [ ] `make build` builds tất cả 8 services thành công
- [ ] `make up` starts toàn bộ stack
- [ ] `make test-integration` passes
- [ ] Documentation updated (README.md updated với 8 services)
- [ ] CI/CD build time < 10 minutes (down from 47-service build time)

## Final Service Count Verification

```bash
# Verify final state
echo "=== go.work modules ==="
grep "./services" go.work | wc -l
# Expected: 7 (+ 1 for zep-go SDK)

echo "=== Active service directories ==="
ls services/ | grep -v archived | grep -v zep-go | wc -l
# Expected: 8

echo "=== Archived services ==="
ls services/archived/ | wc -l
# Expected: 42

echo "=== Docker services ==="
docker compose config --services | wc -l
# Expected: ~15 (7 backends + gateway + 5 infra + worker)
```

## Ghi Chú

- **Không xóa** `services/archived/` — giữ cho historical reference + potential rollback
- **Không xóa** `apps/` — Python reference implementations vẫn cần cho Cognee
- **sm-auth** code đặc biệt: được move hoàn toàn vào vnp-platform, không archive standalone
- `services/ov-storage` → renamed thành `services/storage-service` (không archive)
- `services/vnp-search-hub` → renamed thành `services/search-service` (không archive)
