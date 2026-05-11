# Build Stage
FROM golang:1.23.0-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git tzdata ca-certificates
# Copy global packages
COPY pkg/ ./pkg/
# Copy service
COPY services/ba-knowledge-worker/ ./services/ba-knowledge-worker/
WORKDIR /app/services/ba-knowledge-worker
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/ba-knowledge-worker ./cmd/server

# Final Stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/ba-knowledge-worker .
ENV ENV=production TZ=UTC
EXPOSE 9090 9199
USER nobody:nobody
CMD ["./ba-knowledge-worker"]
