# =============================================================================
# Memobase App — Multi-stage Dockerfile
# Produces a single static binary (~15-20MB) with zero runtime dependencies.
#
# Build context must be the monorepo root (../../) to access shared pkg/.
# =============================================================================

# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /workspace

# Copy shared packages first (Docker layer cache optimization)
COPY pkg/ pkg/

# Copy the memobase app
COPY apps/memobase/ apps/memobase/

WORKDIR /workspace/apps/memobase

# Download dependencies
RUN go mod download

# Build static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(date +%Y%m%d)" \
    -o /memobase-app \
    ./cmd/memobase/

# Stage 2: Runtime (minimal Alpine)
FROM alpine:3.21

# Security: run as non-root
RUN addgroup -S memobase && adduser -S memobase -G memobase
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary and config
COPY --from=builder /memobase-app /usr/local/bin/memobase-app
COPY --from=builder /workspace/apps/memobase/config.yaml /etc/memobase/config.yaml

# Expose ports
# 8080  - Gateway REST API
# 8082  - MCP server
# 9041  - memobase-ingestion gRPC
# 9042  - memobase-engine gRPC
# 9043  - memobase-context gRPC
# 9044  - memobase-pipeline gRPC
# 9090  - Aggregated health probes
EXPOSE 8080 8082 9041 9042 9043 9044 9090

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:9090/healthz || exit 1

USER memobase
ENTRYPOINT ["memobase-app"]
