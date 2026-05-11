# Build Stage
FROM golang:1.23.0-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git tzdata ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/memobase-engine ./cmd/server

# Final Stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/memobase-engine .
ENV ENV=production TZ=UTC
EXPOSE 9090 9199
USER nobody:nobody
CMD ["./memobase-engine"]
