# memobase-pipeline — Runbook (Operations Guide)

> **Service**: `memobase-pipeline`  
> **Audience**: SRE, DevOps, On-call engineers  
> **Status**: Draft — To be completed before production deployment

---

## Startup / Shutdown

```bash
# Start
docker-compose up memobase-pipeline

# Graceful shutdown
docker-compose stop memobase-pipeline  # sends SIGTERM, 30s grace period
```

## Health Check

```bash
grpcurl -plaintext localhost:<HEALTH_PORT> grpc.health.v1.Health/Check
```

## Common Errors

_To be documented based on production observations._

## Deployment & Rollback

_To be documented after Kubernetes manifests are finalized._

## Monitoring

_Dashboard links to be added after Grafana setup._

## Escalation

- **L1**: Platform Team on-call
- **L2**: Engine-specific team lead
- **L3**: Software Architect
