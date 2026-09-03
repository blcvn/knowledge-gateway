FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy the entire monorepo so that replace directives in go.mod work
COPY . .

# Move to the app directory
WORKDIR /app/apps/OpenViking

RUN go build -o /app/bin/openviking ./cmd/openviking

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/bin/openviking .

EXPOSE 8080 8081 8082 8000
CMD ["./openviking"]
