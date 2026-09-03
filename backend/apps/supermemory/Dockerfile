FROM golang:1.23-alpine AS builder

WORKDIR /build

# Build the monolith
COPY . .
RUN go build -o bin/supermemory-app ./apps/supermemory/cmd/supermemory/

FROM alpine:latest

WORKDIR /app
COPY --from=builder /build/bin/supermemory-app /app/supermemory-app

EXPOSE 8080 9090

ENTRYPOINT ["/app/supermemory-app"]
