# Local Development Guide

This guide explains how to set up and run the VNP Memory project locally using Docker Compose. All deployment commands are managed via a `Makefile` located in the `deployment/local` directory.

## Prerequisites

- **Docker** and **Docker Compose** (v2+)
- **Make** installed on your system

## Initial Setup

1. **Navigate to the local deployment directory:**
   ```bash
   cd deployment/local
   ```

2. **Create the Environment File:**
   ```bash
   make env
   ```
   *This copies `.env.example` to `.env`. Ensure you edit the `.env` file and configure any necessary API keys before proceeding.*

3. **Build the Service Images:**
   ```bash
   make build
   ```

## Starting Services

The local environment uses Docker Compose profiles to let you run specific groups of services.

### Core Memory Services
Start the core Memory services (Cognee, Graphiti, Zep, OpenViking):
```bash
make up
```

### Memory Services with UI
Start the Memory services along with the Cognee frontend:
```bash
make up-ui
```

### Knowledge Graph System (KGS) Platform
Start the specialized KGS platform (PostgreSQL, Neo4j, Qdrant, Redis):
```bash
make up-kgs
```

If you wish to use the **SurrealDB backend** for KGS:
```bash
make up-kgs-surrealdb
```
*(To run SurrealDB only, use `make up-surrealdb`)*

### Full Environment
Start everything (Memory services, KGS, UI, and Monitoring):
```bash
make up-full
```

## Stopping and Cleaning Up

- **Stop all services:**
  ```bash
  make down
  ```
- **Restart core services:**
  ```bash
  make restart
  ```
- **Clean up (Stop containers AND remove volumes):**
  ⚠️ *Warning: This will wipe your local databases and memory state.*
  ```bash
  make clean
  ```

## Monitoring & Logs

### Viewing Logs
You can tail the logs for all running services:
```bash
make logs
```

To tail logs for specific services, use:
- `make logs-cognee`
- `make logs-graphiti`
- `make logs-openviking`
- `make logs-zep`
- `make logs-kgs`
- `make logs-surrealdb`

### Service Health Status
To check the running status of containers:
```bash
make ps
# or
make status
```
