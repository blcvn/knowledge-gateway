# Deployment Guide

This guide is for application teams who want to run `kg-service` in a specific environment and then validate it with the matching smoke script.

## Choose A Deployment Mode

- Docker Compose: local or single-host validation.
- Kubernetes: cluster deployment with externally reachable dependencies.
- VM: standalone host deployment with a process manager of your choice.

## Common Inputs

- `KG_BASE_URL` for the deployed service, such as `http://127.0.0.1:8082`
- `KG_API_KEY` for a valid bootstrap or tenant app key
- `KG_RUNTIME_PROFILE` for profile-aware Compose or operator validation
- The full operator env contract is documented in [docs/deployment/environment.md](/Users/anhdt/vnpay/knowledge/kg-service/docs/deployment/environment.md)

## Docker Compose

Use Compose when you want the fastest local bootstrap loop.

- Start the integration smoke stack with `make deploy-compose-integration`
- Start the runtime validation stack with `make deploy-compose-runtime-validation`
- Start the dedicated CodeGraph stack with `make deploy-compose-codegraph-runtime`
- Stop the stacks with `make compose-down-integration`, `make compose-down-runtime-validation`, or `make compose-down-codegraph-runtime`

For the runtime validation stack, ensure the selected profile matches the graph/vector pair you want to exercise.

Example:

```bash
KG_RUNTIME_PROFILE=qdrant-nebula make deploy-compose-runtime-validation
```

For local CodeGraph validation, export the HTTP embedding variables and run:

```bash
KG_API_KEY=kgsk_test_alpha_admin make validate-codegraph-runtime
```

## Kubernetes

Use Kubernetes when the dependencies already exist in the cluster or are managed elsewhere.

1. Export the required environment variables for the selected profile.
2. Apply the manifests through `make deploy-k8s`.
3. Wait for the service readiness checks.
4. Run `scripts/validate-runtime-profile.sh` against the exposed base URL.

Useful variables:

- `KG_RUNTIME_PROFILE`
- `KG_GRAPH_ENDPOINT`
- `KG_GRAPH_DATABASE`
- `KG_VECTOR_ENDPOINT`
- `KG_VECTOR_COLLECTION`

See the environment inventory for defaults and when each variable is conditionally required.

## VM

Use the VM path when you run the binary under a system service.

1. Build or copy the `kg-service` binary.
2. Set the same environment variables used by the deployment profile.
3. Start the service through your supervisor or `make deploy-vm`.
4. Run `scripts/validate-runtime-profile.sh`.

## Validation Scripts

- `make integration-test` checks the public health and authenticated read surface.
- `make validate-runtime-profile` runs the profile-aware end-to-end smoke flow.

## Tenant And App Setup

After the service is reachable:

1. Create a tenant with `POST /v1/tenants`.
2. Create an app with `POST /v1/tenants/{tenant_id}/apps`.
3. Save the returned API key.
4. Call `GET /v1/access/resolve` with that key.
5. Create access grants when a second app must see the same domain.
