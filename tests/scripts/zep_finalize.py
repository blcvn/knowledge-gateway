import os
import subprocess

SERVICES = [
    'zep-admin',
    'zep-core',
    'zep-graph',
    'zep-memory',
    'zep-search',
    'zep-thread',
    'zep-user'
]

BASE_DIR = '/Users/binhnt/Work/blockchain/vnp-memory/services'

# 1. Init Go Modules
GOMOD_TEMPLATE = """module vnp-memory/services/{service}

go 1.23.0

require (
\tgoogle.golang.org/grpc v1.65.0
\tgoogle.golang.org/protobuf v1.34.2
)
"""
for svc in SERVICES:
    go_mod_path = os.path.join(BASE_DIR, svc, 'go.mod')
    if not os.path.exists(go_mod_path):
        with open(go_mod_path, 'w') as f:
            f.write(GOMOD_TEMPLATE.format(service=svc))
            
# 2. Dockerfiles
DOCKERFILE_TEMPLATE = """# Build Stage
FROM golang:1.23.0-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git tzdata ca-certificates
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/{service} ./cmd/server

# Final Stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/bin/{service} .
ENV ENV=production TZ=UTC
EXPOSE 9090 9199
USER nobody:nobody
CMD ["./{service}"]
"""
for svc in SERVICES:
    dockerfile_path = os.path.join(BASE_DIR, svc, 'Dockerfile')
    with open(dockerfile_path, 'w') as f:
        f.write(DOCKERFILE_TEMPLATE.format(service=svc))

# 3. Protos
PROTO_TEMPLATE = """syntax = "proto3";
package vnp.memory.{pkg}.v1;
option go_package = "vnp-memory/services/{service}/api/proto/v1;{pkg}v1";

service {camel}Service {{
  rpc Ping(PingRequest) returns (PingResponse);
}}

message PingRequest {{}}
message PingResponse {{
  string status = 1;
}}
"""
for svc in SERVICES:
    proto_dir = os.path.join(BASE_DIR, svc, 'api', 'proto', 'v1')
    os.makedirs(proto_dir, exist_ok=True)
    pkg = svc.replace('-', '')
    camel = "".join([w.capitalize() for w in svc.split('-')])
    
    proto_file = os.path.join(proto_dir, f"{svc.replace('zep-', '')}.proto")
    with open(proto_file, 'w') as f:
        f.write(PROTO_TEMPLATE.format(service=svc, pkg=pkg, camel=camel))

print("Zep Services Finalized with go.mod, Dockerfile, and Protos.")
