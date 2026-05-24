FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /vnp-infra ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /vnp-infra /usr/local/bin/vnp-infra
EXPOSE 9045
ENTRYPOINT ["vnp-infra"]
