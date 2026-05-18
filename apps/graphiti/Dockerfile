# =============================================================================
# Graphiti App — Multi-stage Dockerfile
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

# Copy the graphiti app
COPY apps/graphiti/ apps/graphiti/

WORKDIR /workspace/apps/graphiti

# Download dependencies
RUN go mod download

# Build static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$(date +%Y%m%d)" \
    -o /graphiti-app \
    ./cmd/graphiti/

# Stage 2: Runtime (minimal Alpine)
FROM alpine:3.21

# Security: run as non-root
RUN addgroup -S graphiti && adduser -S graphiti -G graphiti
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary and config
COPY --from=builder /graphiti-app /usr/local/bin/graphiti-app
COPY --from=builder /workspace/apps/graphiti/config.yaml /etc/graphiti/config.yaml

# Expose ports
# 8080  - Gateway REST API
# 8082  - MCP server
# 9021  - graphiti-ingestion gRPC
# 9022  - graphiti-search gRPC
# 9023  - graphiti-knowledge gRPC
# 9024  - graphiti-store gRPC
# 9025  - graphiti-pipeline gRPC
# 9090  - Aggregated health probes
EXPOSE 8080 8082 9021 9022 9023 9024 9025 9090

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:9090/healthz || exit 1

USER graphiti
ENTRYPOINT ["graphiti-app"]
