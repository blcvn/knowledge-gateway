# ============================================================
# vnp-gateway — Multi-stage Dockerfile
# ============================================================
# Build:   docker build -t vnp-gateway .
# Run:     docker run -p 8080:8080 -p 11080:11080 vnp-gateway
# ============================================================

# --- Stage 1: Build ---
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X main.version=0.2.0" \
    -o /bin/vnp-gateway ./cmd/main.go

# --- Stage 2: Runtime ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1001 vnp \
    && adduser -u 1001 -G vnp -D vnp

COPY --from=builder /bin/vnp-gateway /usr/local/bin/vnp-gateway

USER vnp

# REST, gRPC, MCP, Health
EXPOSE 8080 8081 8082 11080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:11080/healthz || exit 1

ENTRYPOINT ["vnp-gateway"]
