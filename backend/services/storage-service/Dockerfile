FROM golang:1.23-alpine AS builder
WORKDIR /workspace

# Copy shared packages
COPY backend/shared/pkg/ ./pkg/

# Copy service source
COPY backend/services/storage-service/ ./services/storage-service/

# Build
WORKDIR /workspace/services/storage-service
RUN go build -o /storage-service ./cmd/server/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /storage-service /app/storage-service

# Create default data directory
RUN mkdir -p /data/storage

ENV GRPC_PORT=9090
ENV HEALTH_PORT=9140
ENV FS_BASE_DIR=/data/storage

EXPOSE 9090 9140

ENTRYPOINT ["/app/storage-service"]
