FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o graphiti-search ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/graphiti-search .

EXPOSE 9022
EXPOSE 9095

CMD ["./graphiti-search"]
