FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /vnp-observability ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /vnp-observability /usr/local/bin/vnp-observability
EXPOSE 9046
ENTRYPOINT ["vnp-observability"]
