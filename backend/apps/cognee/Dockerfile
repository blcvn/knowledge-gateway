# =============================================================================
# Cognee App — Multi-stage Dockerfile
# Produces a single static binary (~15-20MB) with zero runtime dependencies.
# =============================================================================

# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /workspace

# Copy shared packages first (Docker layer cache optimization)
COPY backend/shared/pkg/ pkg/

# Copy the cognee app
COPY apps/cognee/ apps/cognee/

WORKDIR /workspace/apps/cognee

# Download dependencies
RUN go mod download

# Build static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(date +%Y%m%d)" \
    -o /cognee-app \
    ./cmd/cognee/

# Stage 2: Runtime (minimal Alpine)
FROM alpine:3.21

# Security: run as non-root
RUN addgroup -S cognee && adduser -S cognee -G cognee
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary and config
COPY --from=builder /cognee-app /usr/local/bin/cognee-app
COPY --from=builder /workspace/apps/cognee/config.yaml /etc/cognee/config.yaml

# Expose ports
# 8080  - Gateway REST API
# 9011  - cognee-ingestion gRPC
# 9012  - cognee-cognify gRPC
# 9013  - cognee-search gRPC
# 11080 - Aggregated health probes
EXPOSE 8080 9011 9012 9013 11080

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:11080/healthz || exit 1

USER cognee
ENTRYPOINT ["cognee-app"]
