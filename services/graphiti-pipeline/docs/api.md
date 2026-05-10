# graphiti-pipeline — API Reference

> **Service**: `graphiti-pipeline`  
> **Status**: Draft — Proto definitions inherited from pre-consolidation services

---

## gRPC Service Definitions

_This service exposes multiple gRPC service definitions on a single port.
Proto definitions are unchanged from the pre-consolidation services.
See `api/proto/` for canonical proto files._

## Endpoints

_To be documented from proto definitions during implementation._

## Authentication

All endpoints require valid JWT or API key via Gateway.
Tenant isolation enforced via `x-tenant-id` gRPC metadata.
