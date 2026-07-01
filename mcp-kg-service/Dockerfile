FROM golang:1.25 AS builder

WORKDIR /app

COPY mcp-kg-service/go.mod mcp-kg-service/go.sum ./mcp-kg-service/
RUN cd mcp-kg-service && go mod download

COPY mcp-kg-service/ ./mcp-kg-service/

RUN cd mcp-kg-service && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /bin/mcp-kg-service ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /bin/mcp-kg-service /bin/mcp-kg-service

EXPOSE 8080

ENTRYPOINT ["/bin/mcp-kg-service"]
