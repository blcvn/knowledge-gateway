# Build stage — context must be the repository root so the submodule path resolves.
FROM golang:1.25 AS builder
WORKDIR /app

# Copy go.mod/go.sum first for dependency layer caching.
COPY services/ai-kg-service/kgs-platform/go.mod \
     services/ai-kg-service/kgs-platform/go.sum \
     ./services/ai-kg-service/kgs-platform/

RUN cd services/ai-kg-service/kgs-platform && go mod download

# Copy remaining source.
COPY services/ai-kg-service/kgs-platform/ ./services/ai-kg-service/kgs-platform/

# Build the server binary.
RUN cd services/ai-kg-service/kgs-platform && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /bin/kgs-platform \
    ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates curl tzdata wget

# Create non-root user (matches other services in this repo).
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Runtime directories.
RUN mkdir -p /data/conf /vault/secrets && \
    chown -R appuser:appuser /data /vault/secrets

# Copy binary and only the app config (kgs.rego is OPA-only, not parsed by Kratos).
COPY --from=builder /bin/kgs-platform /bin/kgs-platform
COPY --from=builder /app/services/ai-kg-service/kgs-platform/configs/config.yaml /data/conf/config.yaml

RUN chown appuser:appuser /bin/kgs-platform && \
    chown -R appuser:appuser /data/conf

USER appuser
WORKDIR /data

ENV SERVICE_NAME=ai-kg-service \
    HTTP_PORT=8000 \
    GRPC_PORT=9000 \
    SECRET_FILE=/vault/secrets/config.json

EXPOSE 8000 9000

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -s http://localhost:8000 || exit 1

CMD ["/bin/kgs-platform", "-conf", "/data/conf/config.yaml"]
