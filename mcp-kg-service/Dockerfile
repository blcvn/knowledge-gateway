FROM golang:1.25 AS builder

WORKDIR /app

COPY services/ai-kg-service/mcp-kg-service/go.mod services/ai-kg-service/mcp-kg-service/go.sum ./services/ai-kg-service/mcp-kg-service/
RUN cd services/ai-kg-service/mcp-kg-service && go mod download

COPY services/ai-kg-service/mcp-kg-service/ ./services/ai-kg-service/mcp-kg-service/

RUN cd services/ai-kg-service/mcp-kg-service && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/mcp-kg-service ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /bin/mcp-kg-service /bin/mcp-kg-service

EXPOSE 8080

ENTRYPOINT ["/bin/mcp-kg-service"]
