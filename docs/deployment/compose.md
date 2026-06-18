# Docker Compose Deployment

Use Docker Compose when you want the fastest self-contained deployment path for local evaluation or a single host.

## What This Path Starts

- Postgres for the current bootstrap runtime
- Redis for the current bootstrap runtime
- `kg-service` built from the repository Dockerfile
- a one-shot migration container that applies the SQL schema before the app starts

## Prerequisites

- Docker with Compose v2
- Access to the repository files on the machine that runs Compose

## Start The Stack

```bash
make deploy-compose
```

The script runs the Compose file under `deploy/compose/` and waits for the application stack to come up.

## Verify The Deployment

```bash
make integration-test
```

For the default bootstrap smoke path, keep the sample environment variables described in the integration validation guide.

## Notes

- The Compose path is the only supported path in this change that provisions Postgres and Redis together with the app.
- The migration container applies the repository SQL schema before the service starts.
- The app still uses the current bootstrap defaults for memory-backed access and ontology slices.
