# API Tester (kgs-platform)

CLI smoke/integration tester cho `kgs-platform` API.

Tester sẽ chạy lần lượt nhiều API quan trọng:
- Health/ready
- App registry (create/get/list/issue key/revoke key)
- Auth + namespace checks
- Ontology/rules/policies
- Graph (node/edge/batch/search/traversal/coverage/traceability)
- Overlay/version/view APIs

Kết thúc chạy sẽ in summary `passed/failed/skipped` và trả exit code:
- `0`: không có step fail
- `1`: có ít nhất 1 step fail

## Prerequisites

1. `ai-kg-service` đang chạy và reachable.
2. Các dependency backend (Postgres/Redis/Neo4j/Qdrant/NATS/OPA) đã lên.
3. Go toolchain đã cài.

## Quick Start

### 1) Chạy từ source

```bash
cd services/ai-kg-service/kgs-platform
GOWORK=off go run ./cmd/api-tester \
  -base-url http://localhost:18000 \
  -sync-opa-policy=false
```

Gợi ý:
- Với stack root `deployment/docker/docker-compose.dev.yml`, `ai-kg-service` đang expose HTTP ở `18000`.
- `-sync-opa-policy=false` vì `kgs-opa` không expose port host trong stack này.

### 2) Chạy verbose

```bash
cd services/ai-kg-service/kgs-platform
GOWORK=off go run ./cmd/api-tester \
  -base-url http://localhost:18000 \
  -sync-opa-policy=false \
  -verbose=true
```

## Flags

- `-base-url` (default: `http://localhost:8000`): base URL của kgs-platform HTTP API.
- `-timeout` (default: `20s`): timeout cho mỗi HTTP request.
- `-verbose` (default: `false`): in thêm response body cho từng step.
- `-fail-fast` (default: `false`): dừng ngay khi step đầu tiên fail.
- `-auth-app-id` (optional): dùng app đã tồn tại để issue key (thay vì app vừa create).
- `-sync-opa-policy` (default: `true`): push policy allow test vào OPA.
- `-opa-url` (default: `http://localhost:8181`): URL OPA để sync policy.
- `-org-id` (optional): set `X-Org-ID` cho authenticated requests.

## Recommended Commands

### Chạy ổn định cho stack root compose

```bash
cd services/ai-kg-service/kgs-platform
GOWORK=off go run ./cmd/api-tester \
  -base-url http://localhost:18000 \
  -sync-opa-policy=false \
  -fail-fast=true
```

### Nếu OPA có expose host port 8181

```bash
cd services/ai-kg-service/kgs-platform
GOWORK=off go run ./cmd/api-tester \
  -base-url http://localhost:18000 \
  -sync-opa-policy=true \
  -opa-url http://localhost:8181
```

## Troubleshooting

- `connection refused`:
  - Kiểm tra `ai-kg-service` đã lên và đúng port (`18000` trong root compose).
- Fail ở step `Setup OPA allow policy`:
  - Dùng `-sync-opa-policy=false`, hoặc expose OPA ra host và set `-opa-url`.
- Nhiều `SKIP`:
  - Một số endpoint spec mới có thể chưa expose đầy đủ trong build hiện tại; tester sẽ fallback hoặc skip có lý do.

## Build Binary

```bash
cd services/ai-kg-service/kgs-platform
GOWORK=off GOCACHE=/tmp/gocache go build -o ./bin/api-tester ./cmd/api-tester
./bin/api-tester -base-url http://localhost:18000 -sync-opa-policy=false
```

