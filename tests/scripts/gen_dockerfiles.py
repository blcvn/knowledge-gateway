import os

SERVICES = [
    'sm-analytics',
    'sm-auth',
    'sm-connector',
    'sm-document',
    'sm-engine',
    'sm-mcp',
    'sm-memory',
    'sm-profile',
    'sm-project',
    'sm-search'
]

BASE_DIR = '/Users/binhnt/Work/blockchain/vnp-memory/services'

DOCKERFILE_TEMPLATE = """# Build Stage
FROM golang:1.23.0-alpine AS builder

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git tzdata ca-certificates

# Copy go mod files
COPY go.mod ./
# (Optional) COPY go.sum ./ if it exists
RUN go mod download

# Copy source code
COPY . .

# Build the executable with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/{service} ./cmd/server

# Final Stage
FROM alpine:3.18

WORKDIR /app

# Copy CA certificates for HTTPS/gRPC
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy compiled binary
COPY --from=builder /app/bin/{service} .

# Set environment
ENV ENV=production
ENV TZ=UTC

# Expose gRPC port and Health Probe port
EXPOSE 9090 9199

# Run binary
USER nobody:nobody
CMD ["./{service}"]
"""

for svc in SERVICES:
    svc_dir = os.path.join(BASE_DIR, svc)
    dockerfile_path = os.path.join(svc_dir, 'Dockerfile')
    
    print(f"Creating Dockerfile for {svc}...")
    with open(dockerfile_path, 'w') as f:
        f.write(DOCKERFILE_TEMPLATE.format(service=svc))
            
print("Dockerfiles generated for all services.")
