# ============================================================================
# Multi-stage Dockerfile for cognee-ingestion
# Produces a minimal scratch-based image ≤ 50MB
# ============================================================================

# --- Stage 1: Build ---
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.version=$(cat VERSION 2>/dev/null || echo dev)" \
    -o /bin/cognee-ingestion ./cmd/server

# --- Stage 2: Runtime ---
FROM scratch

# Import CA certs and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /bin/cognee-ingestion /cognee-ingestion

# Copy migrations for init containers
COPY --from=builder /src/migrations /migrations

# Default env
ENV GRPC_PORT=9011
ENV HEALTH_PORT=9091
ENV LOG_LEVEL=info
ENV LOG_FORMAT=json

EXPOSE 9011 9091

ENTRYPOINT ["/cognee-ingestion"]
