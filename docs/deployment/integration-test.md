# Integration Validation

Use this script after a deployment to verify the current service health and a core authenticated read path.

## Required Inputs

- `KG_BASE_URL` for the deployed service, such as `http://127.0.0.1:8082`
- `KG_API_KEY` for a valid bootstrap or production key

## Optional Inputs

- `KG_SMOKE_DOMAIN_ID` defaults to `sample-policy`
- `KG_SMOKE_TEMPLATE_NAME` defaults to `action-guide`
- `KG_SMOKE_TEMPLATE_PARAMS` defaults to `{"topic_key":"returns"}`

## Run It

```bash
KG_BASE_URL=http://127.0.0.1:8082 \
KG_API_KEY=kgsk_platform_admin \
make integration-test
```

## What It Checks

- `GET /healthz`
- `GET /v1/access/resolve`
- `GET /v1/kg/read/templates?domain_id=...`
- `POST /v1/kg/read/template/{domain_id}/{template_name}`

## Failure Behavior

- The script exits non-zero if any required request fails.
- The script prints the failing step and the HTTP status when a request is rejected.
- The script keeps the validation target-agnostic so the same flow works for Compose, Kubernetes, and VM deployments.
