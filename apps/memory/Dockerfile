# ── UI Build Stage ──────────────────────────────────────────────
FROM node:22-alpine AS ui-builder

WORKDIR /workspace/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci --production=false
COPY ui/ .
RUN npm run build

# ── Go Build Stage ─────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /workspace

# Create a minimal go.work for Docker with only the 2 needed modules
RUN printf 'go 1.25.0\n\nuse (\n\t./apps/memory\n\t./gateway\n)\n' > go.work

# Copy go.mod/go.sum for dependency caching
COPY apps/memory/go.mod apps/memory/go.sum* apps/memory/
COPY gateway/go.mod gateway/go.sum gateway/

# Download dependencies (gateway has the bulk of them)
WORKDIR /workspace/gateway
RUN go mod download
WORKDIR /workspace

# Copy full source code — public packages (adapter/domain/infra/usecase)
# and internal packages for gateway standalone mode
COPY apps/memory/ apps/memory/
COPY gateway/ gateway/

# Copy built UI assets into the Go embed directory
COPY --from=ui-builder /workspace/ui/dist/ apps/memory/internal/ui/ui_dist/

# Build from workspace root
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /workspace/bin/vnp-memory \
    ./apps/memory/cmd/server

# ── Runtime Stage ──────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S vnp && adduser -S vnp -G vnp

WORKDIR /app

# Copy binary and config
COPY --from=builder /workspace/bin/vnp-memory .
COPY --from=builder /workspace/apps/memory/configs/ configs/

# Create data directories
RUN mkdir -p /app/data/nats /app/data/vikingfs && \
    chown -R vnp:vnp /app

USER vnp

# REST API / gRPC / MCP / Health
EXPOSE 8080 8081 8082 8083

ENV VNP_MEMORY_SERVER_LOG_LEVEL=info

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -sf http://localhost:8083/healthz || exit 1

ENTRYPOINT ["./vnp-memory"]
