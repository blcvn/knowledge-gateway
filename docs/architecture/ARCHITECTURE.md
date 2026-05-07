# System Architecture

This document describes the high-level architecture, technology stack, and the API Gateway model of the Zero-Trust Microservices Platform.

## 1. Architecture Overview

The system is designed as a **Zero-Trust Microservices Architecture**. It prioritizes security, observability, and scalability.

### Key Principles
*   **Zero Request Trust:** All traffic, internal and external, is authenticated, authorized, and encrypted.
*   **mTLS Everywhere:** Service-to-service communication is secured via Mutual TLS, with certificates managed and rotated by HashiCorp Vault.
*   **Sidecar Pattern:** Infrastructure concerns (secrets, proxies) are handled by sidecars (e.g., Vault Agent).
*   **Centralized Identity:** A dedicated Auth Service handles identity management, integrating with external providers (Google) and internal systems.

### High-Level Diagram

```mermaid
graph TD
    Client[Client App / User] -->|HTTPS| HAProxy[HAProxy LB]
    HAProxy -->|HTTP| Kong[Kong Gateway]
    
    subgraph "Infrastructure"
        Vault[HashiCorp Vault]
        Consul[Consul (Optional)]
        Prometheus[Prometheus]
        Jaeger[Jaeger]
        Loki[Loki]
    end

    subgraph "Microservices Cluster"
        Kong -->|mTLS| AuthService[Auth Service]
        Kong -->|mTLS| AuthorService[Author Service]
        Kong -->|mTLS| ExampleService[Example Service]
        
        AuthService -->|gRPC/mTLS| DB_Auth[(Postgres Auth)]
        AuthorService -->|gRPC/mTLS| DB_Author[(Postgres Author)]
        
        AuthService -.->|Logs/Metrics| Infrastructure
        AuthorService -.->|Logs/Metrics| Infrastructure
    end

    Kong -.->|Check Policy| OPA[Open Policy Agent]
```

---

## 2. Technology Stack

### Backend
*   **Language:** Go (Golang) 1.24+
*   **Framework:** **Go-Kratos** (v2) - Provides a robust foundation for microservices with built-in support for HTTP/gRPC, middleware, and tracing.
*   **ORM:** **GORM** - For database interactions.

### Infrastructure & Security
*   **API Gateway:** **Kong Gateway** (3.4) - Manages ingress, routing, and rate limiting.
*   **Load Balancer:** **HAProxy** (2.8) - Entry point load balancing.
*   **Secret Management:** **HashiCorp Vault** (1.15) - Centralized secret management, PKI (Certificate Authority) for mTLS.
*   **Authorization:** **OPA (Open Policy Agent)** - Fine-grained policy enforcement.

### Data Storage
*   **Relational Database:** **PostgreSQL** (15) - Primary data store per service.
*   **Cache:** **Redis** (7) - Distributed caching and token blocklisting.

### Observability
*   **Metrics:** **Prometheus** - Scrapes metrics from services.
*   **Tracing:** **Jaeger** - Distributed tracing via OpenTelemetry (OTLP).
*   **Logging:** **Loki** & **Promtail** - Centralized log aggregation.
*   **Visualization:** **Grafana** - Dashboards for all observability data.

---

## 3. Gateway Model & API Pattern

The system uses a hybrid approach combining a centralized API Gateway (Kong) with service-local transcoding (`grpc-gateway`).

### 3.1 The Flow
1.  **Ingress:** A request hits **HAProxy** (Port 8000/8443).
2.  **Routing:** HAProxy forwards to **Kong Gateway**.
3.  **Policy Check:** Kong (via plugins) may consult **OPA** or the **Auth Service** to validate credentials and permissions.
4.  **Forwarding:** Kong routes the request to the appropriate microservice (e.g., `auth-service`) via **HTTP** (secured by mTLS).
5.  **Transcoding:** The microservice receives the HTTP request. The embedded **`grpc-gateway`** interceptor converts the JSON/HTTP request into a gRPC message.
6.  **Processing:** The request is handled by the Go gRPC handler.
7.  **Response:** The response is converted back to JSON and returned up the chain.

### 3.2 Component Details

#### Kong Gateway
*   Acts as the central control point.
*   Handles cross-cutting concerns: CORS, Rate Limiting, RequestID generation.
*   Configured declaratively (`kong.yml`).

#### Service Layer (Go-Kratos + grpc-gateway)
Each service exposes two ports:
*   **gRPC Port (e.g., 9090):** For efficient inter-service communication.
*   **HTTP Port (e.g., 8080):** For external access via the gateway.
    *   Uses `grpc-gateway` to expose gRPC endpoints as RESTful JSON APIs.
    *   Host Swagger/OpenAPI documentation endpoints (optional).

### 3.3 Security Model within Gateway
*   **mTLS:** Kong uses client certificates (issued by Vault) to authenticate itself to backend services. Services verify that requests come from Kong or other trusted internal services.
*   **Token Validation:**
    *   **Access Tokens:** JWT (JSON Web Tokens). Supported validation at Gateway level (Kong JWT plugin) or Service level (Middleware).
    *   **Context Propagation:** User identity (UserID, Roles, TenantID) is extracted from the token and passed to downstream services via HTTP Headers (e.g., `X-User-ID`, `X-Tenant-ID`).

---
