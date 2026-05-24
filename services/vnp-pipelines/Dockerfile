FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /vnp-pipelines ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /vnp-pipelines /usr/local/bin/vnp-pipelines
EXPOSE 9044
ENTRYPOINT ["vnp-pipelines"]
