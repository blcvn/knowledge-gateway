FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -ldflags="-w -s" -o server cmd/server/main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 9012 9092
CMD ["./server"]
