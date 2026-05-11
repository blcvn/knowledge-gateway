FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o /bin/graphiti-knowledge ./services/graphiti-knowledge/cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/graphiti-knowledge /graphiti-knowledge

# gRPC API
EXPOSE 9023
# Health HTTP
EXPOSE 9096

CMD ["/graphiti-knowledge"]
