---
id: TASK-017
title: E2E Tests — Docker Compose Integration
service: vnp-gateway
version: 1.0.0
status: Ready
priority: P2
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
depends_on: [TASK-013, TASK-014]
estimate: 6h
---

## Mục Tiêu

End-to-end test suite chạy trên Docker Compose stack (gateway + postgres + redis + nats + mock services).

## Phạm Vi

### Files cần tạo
- `gateway/tests/e2e/docker-compose.test.yml` — Test stack
- `gateway/tests/e2e/e2e_test.go` — E2E test scenarios
- `gateway/tests/e2e/mock_server/main.go` — Generic mock gRPC server

### Test Scenarios

1. Full auth lifecycle: register API key → authenticate → call API
2. Rate limit enforcement with real Redis
3. Circuit breaker with failing mock service
4. MCP tool call → mock gRPC → response
5. WebDAV lifecycle: MKCOL → PUT → PROPFIND → DELETE
6. Health cascade with mixed healthy/unhealthy services
7. Prometheus metrics scraping accuracy

### Acceptance Criteria

- [ ] AC-1: `docker compose -f tests/e2e/docker-compose.test.yml up` starts full stack
- [ ] AC-2: `go test ./tests/e2e/... -tags e2e` runs all E2E tests
- [ ] AC-3: Tests complete in < 120s
- [ ] AC-4: No external dependencies beyond Docker
- [ ] AC-5: CI/CD pipeline can run E2E tests (github actions compatible)

## Verification

```bash
docker compose -f tests/e2e/docker-compose.test.yml up -d
go test ./tests/e2e/... -tags e2e -v -timeout 120s
docker compose -f tests/e2e/docker-compose.test.yml down
```
