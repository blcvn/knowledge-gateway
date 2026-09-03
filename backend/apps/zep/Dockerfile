# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum from the root workspace if using go workspaces
# For now, we copy the apps/zep module and the required local modules
# In a real monorepo setup, you'd copy the whole context and build inside the workspace.
COPY . .

WORKDIR /app/apps/zep
RUN go mod download

# Build the monolith binary
RUN CGO_ENABLED=0 GOOS=linux go build -o zep-monolith ./cmd/zep/main.go

# Final stage
FROM alpine:3.19

WORKDIR /app

# Add tzdata and ca-certificates
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/apps/zep/zep-monolith .

# Default port exposes
EXPOSE 8080 8081 9041 9042 9043 9044 9045 9046

CMD ["./zep-monolith"]
